# Plugin Wrapper Layer

## Overview

The Plugin Wrapper (`pkg/plugin`) serves as the bridge between the minimal C bridge and the Go framework. It handles the low-level VST3 protocol details while presenting a clean Go API to plugin developers.

## Architecture

```
┌─────────────────────────┐
│   Plugin Developer      │
├─────────────────────────┤
│   Framework Core        │
├─────────────────────────┤
│   Plugin Wrapper ←──────┤ This layer
├─────────────────────────┤
│   C Bridge              │
└─────────────────────────┘
```

## Key Components

### Component Wrapper

The `componentWrapper` is the central orchestrator that implements VST3 interfaces:

```go
type componentWrapper struct {
    plugin      Plugin
    processor   Processor
    info        plugin.Info
    
    // State
    active      bool
    processing  bool
    sampleRate  float64
    blockSize   int32
    
    // Buffers
    inputBuses  []BusBuffers
    outputBuses []BusBuffers
}
```

### Plugin Registration

#### Global Registry
```go
var (
    registeredPlugins []Plugin
    pluginMutex      sync.RWMutex
)

func Register(p Plugin) {
    pluginMutex.Lock()
    defer pluginMutex.Unlock()
    registeredPlugins = append(registeredPlugins, p)
}
```

#### Factory Info
```go
type FactoryInfo struct {
    Vendor string
    URL    string
    Email  string
}

func SetFactoryInfo(info FactoryInfo) {
    factoryInfo = info
}
```

### C Bridge Integration

#### Export Functions
```go
//export GoGetFactoryInfo
func GoGetFactoryInfo(info *C.struct_FactoryInfo) {
    info.vendor = C.CString(factoryInfo.Vendor)
    info.url = C.CString(factoryInfo.URL)
    info.email = C.CString(factoryInfo.Email)
    info.flags = C.Steinberg_uint32(0x10) // Unicode
}

//export GoCreateInstance
func GoCreateInstance(cid *C.char, iid *C.char) unsafe.Pointer {
    classID := C.GoString(cid)
    
    // Find registered plugin
    plugin := findPluginByID(classID)
    if plugin == nil {
        return nil
    }
    
    // Create wrapper
    wrapper := &componentWrapper{
        plugin:    plugin,
        processor: plugin.CreateProcessor(),
    }
    
    // Register and return handle
    handle := registerComponent(wrapper)
    return unsafe.Pointer(handle)
}
```

## Interface Implementations

### IComponent

#### Initialize
```go
//export GoComponent_Initialize
func GoComponent_Initialize(handle C.uintptr_t, context unsafe.Pointer) C.Steinberg_tresult {
    wrapper := getComponent(handle)
    if wrapper == nil {
        return C.Steinberg_kResultFalse
    }
    
    // Initialize processor
    if err := wrapper.processor.Initialize(44100, 512); err != nil {
        return C.Steinberg_kResultFalse
    }
    
    return C.Steinberg_kResultOk
}
```

#### Bus Management
```go
//export GoComponent_GetBusCount
func GoComponent_GetBusCount(handle C.uintptr_t, type_ C.Steinberg_Vst_MediaType, 
    dir C.Steinberg_Vst_BusDirection) C.Steinberg_int32 {
    
    wrapper := getComponent(handle)
    buses := wrapper.processor.GetBuses()
    
    if type_ == C.Steinberg_Vst_kAudio {
        if dir == C.Steinberg_Vst_kInput {
            return C.Steinberg_int32(buses.NumInputs())
        }
        return C.Steinberg_int32(buses.NumOutputs())
    }
    
    return 0  // No event buses yet
}
```

### IAudioProcessor

#### Setup Processing
```go
//export GoComponent_SetupProcessing
func GoComponent_SetupProcessing(handle C.uintptr_t, 
    setup *C.struct_Steinberg_Vst_ProcessSetup) C.Steinberg_tresult {
    
    wrapper := getComponent(handle)
    
    // Store setup
    wrapper.sampleRate = float64(setup.sampleRate)
    wrapper.blockSize = int32(setup.maxSamplesPerBlock)
    
    // Reinitialize processor
    err := wrapper.processor.Initialize(
        wrapper.sampleRate, 
        wrapper.blockSize,
    )
    
    if err != nil {
        return C.Steinberg_kResultFalse
    }
    
    return C.Steinberg_kResultOk
}
```

#### Process Audio
```go
//export GoComponent_Process
func GoComponent_Process(handle C.uintptr_t, 
    data *C.struct_Steinberg_Vst_ProcessData) C.Steinberg_tresult {
    
    wrapper := getComponent(handle)
    if !wrapper.active || !wrapper.processing {
        return C.Steinberg_kResultOk
    }
    
    // Create process context
    ctx := wrapper.createProcessContext(data)
    
    // Process audio
    wrapper.processor.ProcessAudio(ctx)
    
    // TODO: Handle parameter changes from host
    // TODO: Handle events
    
    return C.Steinberg_kResultOk
}
```

### IEditController

#### Parameter Management
```go
//export GoController_GetParameterCount
func GoController_GetParameterCount(handle C.uintptr_t) C.Steinberg_int32 {
    wrapper := getComponent(handle)
    params := wrapper.processor.GetParameters()
    return C.Steinberg_int32(params.Count())
}

//export GoController_GetParameterInfo
func GoController_GetParameterInfo(handle C.uintptr_t, paramIndex C.Steinberg_int32,
    info *C.struct_Steinberg_Vst_ParameterInfo) C.Steinberg_tresult {
    
    wrapper := getComponent(handle)
    params := wrapper.processor.GetParameters()
    
    param := params.GetByIndex(int(paramIndex))
    if param == nil {
        return C.Steinberg_kResultFalse
    }
    
    // Fill parameter info
    info.id = C.Steinberg_Vst_ParamID(param.ID)
    copyStringToUTF16(param.Name, &info.title[0], 128)
    copyStringToUTF16(param.Unit, &info.units[0], 128)
    
    info.stepCount = 0  // Continuous
    info.defaultNormalizedValue = C.Steinberg_Vst_ParamValue(
        param.Normalize(param.DefaultValue))
    info.unitId = C.Steinberg_Vst_UnitID(0)  // Root unit
    info.flags = C.Steinberg_int32(C.Steinberg_Vst_ParameterInfo_kCanAutomate)
    
    return C.Steinberg_kResultOk
}
```

## Buffer Management

### Audio Buffer Mapping
```go
func (w *componentWrapper) createProcessContext(
    data *C.struct_Steinberg_Vst_ProcessData) *process.Context {
    
    ctx := &process.Context{
        SampleRate: w.sampleRate,
        numSamples: int32(data.numSamples),
    }
    
    // Map input buffers
    if data.inputs != nil && data.numInputs > 0 {
        inputs := (*[8]C.struct_Steinberg_Vst_AudioBusBuffers)(
            unsafe.Pointer(data.inputs))[:data.numInputs]
        
        for i, bus := range inputs {
            if bus.numChannels > 0 {
                channels := mapChannelBuffers32(&bus)
                ctx.Input = append(ctx.Input, channels...)
            }
        }
    }
    
    // Map output buffers (similar)
    // ...
    
    return ctx
}

func mapChannelBuffers32(bus *C.struct_Steinberg_Vst_AudioBusBuffers) [][]float32 {
    if bus.channelBuffers32 == nil {
        return nil
    }
    
    // Cast to array of float32 pointers
    channelPtrs := (*[16]*C.float)(unsafe.Pointer(bus.channelBuffers32))[
        :bus.numChannels:bus.numChannels]
    
    // Create slice views (no copy)
    channels := make([][]float32, bus.numChannels)
    for i, ptr := range channelPtrs {
        if ptr != nil {
            channels[i] = (*[1 << 30]float32)(unsafe.Pointer(ptr))[
                :w.blockSize:w.blockSize]
        }
    }
    
    return channels
}
```

## State Management

### State Save
```go
//export GoComponent_GetState
func GoComponent_GetState(handle C.uintptr_t, state unsafe.Pointer) C.Steinberg_tresult {
    wrapper := getComponent(handle)
    
    // Create stream wrapper
    stream := &bstream{handle: (*C.struct_Steinberg_IBStream)(state)}
    
    // Write version
    if err := stream.WriteInt32(1); err != nil {
        return C.Steinberg_kResultFalse
    }
    
    // Write parameters
    params := wrapper.processor.GetParameters()
    for _, param := range params.All() {
        stream.WriteInt32(int32(param.ID))
        stream.WriteFloat64(param.GetValue())
    }
    
    return C.Steinberg_kResultOk
}
```

### State Load
```go
//export GoComponent_SetState
func GoComponent_SetState(handle C.uintptr_t, state unsafe.Pointer) C.Steinberg_tresult {
    wrapper := getComponent(handle)
    
    // Create stream wrapper
    stream := &bstream{handle: (*C.struct_Steinberg_IBStream)(state)}
    
    // Read version
    version, err := stream.ReadInt32()
    if err != nil || version != 1 {
        return C.Steinberg_kResultFalse
    }
    
    // Read parameters
    // ...
    
    return C.Steinberg_kResultOk
}
```

## Thread Safety

### Parameter Access
All parameter access is thread-safe through atomic operations:
```go
// Audio thread (real-time)
value := param.GetValue()  // Atomic read

// UI thread
param.SetValue(newValue)   // Atomic write
```

### Component State
Non-real-time state uses mutexes:
```go
type componentWrapper struct {
    // ...
    stateMu sync.RWMutex
}

func (w *componentWrapper) SetActive(active bool) {
    w.stateMu.Lock()
    defer w.stateMu.Unlock()
    
    w.active = active
    w.processor.SetActive(active)
}
```

## Error Handling

### Graceful Degradation
```go
func (w *componentWrapper) Process(data *C.ProcessData) C.tresult {
    // Recover from panics
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Process panic: %v", r)
            // Output silence
            w.outputSilence(data)
        }
    }()
    
    // Normal processing
    return w.processAudio(data)
}
```

### Validation
```go
func (w *componentWrapper) validateBuffers(data *C.ProcessData) bool {
    if data.numSamples <= 0 || data.numSamples > w.blockSize {
        return false
    }
    
    if data.numInputs > 0 && data.inputs == nil {
        return false
    }
    
    // More validation...
    return true
}
```

## Platform Differences

### String Handling
```go
// UTF16 for VST3 strings
func copyStringToUTF16(src string, dst *C.Steinberg_char16, maxLen int) {
    runes := utf16.Encode([]rune(src))
    n := len(runes)
    if n > maxLen-1 {
        n = maxLen - 1
    }
    
    for i := 0; i < n; i++ {
        *(*C.Steinberg_char16)(unsafe.Pointer(
            uintptr(unsafe.Pointer(dst)) + uintptr(i*2))) = C.Steinberg_char16(runes[i])
    }
    
    // Null terminate
    *(*C.Steinberg_char16)(unsafe.Pointer(
        uintptr(unsafe.Pointer(dst)) + uintptr(n*2))) = 0
}
```

## Performance Optimizations

### Buffer Caching
```go
type componentWrapper struct {
    // Pre-allocated process context
    processCtx *process.Context
}

func (w *componentWrapper) initializeBuffers() {
    w.processCtx = &process.Context{
        Input:      make([][]float32, 2),
        Output:     make([][]float32, 2),
        workBuffer: make([]float32, w.blockSize),
    }
}
```

### Fast Parameter Lookup
```go
// Build parameter ID to index map for O(1) lookup
type paramMap map[ParamID]int

func buildParamMap(params *param.Registry) paramMap {
    m := make(paramMap)
    for i, p := range params.All() {
        m[p.ID] = i
    }
    return m
}
```

## Future Enhancements

1. **MIDI Support**: Implement event list processing
2. **Parameter Changes**: Process automation from host
3. **Multiple Buses**: Support complex I/O configurations
4. **State Versioning**: Handle preset compatibility
5. **Performance Monitoring**: Built-in profiling support