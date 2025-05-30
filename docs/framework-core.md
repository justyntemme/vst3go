# Framework Core (Layer 2)

## Overview

The Framework Core provides Go-idiomatic abstractions for VST3 concepts. This layer transforms the low-level VST3 C API into a clean, type-safe Go API that feels natural to Go developers.

## Package Structure

```
pkg/framework/
├── plugin/      # Plugin metadata and interfaces
├── param/       # Parameter management system
├── process/     # Audio processing context
├── bus/         # Audio bus configuration
└── state/       # State persistence
```

## Plugin Package (`pkg/framework/plugin`)

### Purpose
Defines the core plugin interfaces and metadata structures.

### Key Components

#### Plugin Interface
```go
type Plugin interface {
    GetInfo() Info
    CreateProcessor() Processor
}
```

#### Plugin Info
```go
type Info struct {
    ID       string  // Unique plugin identifier
    Name     string  // Display name
    Version  string  // Semantic version
    Vendor   string  // Company/developer name
    Category string  // VST3 category (e.g., "Fx|Reverb")
}
```

#### Processor Interface
```go
type Processor interface {
    Initialize(sampleRate float64, maxBlockSize int32) error
    ProcessAudio(ctx *process.Context)
    SetActive(active bool) error
    GetParameters() *param.Registry
    GetBuses() *bus.Configuration
    GetLatencySamples() int32
    GetTailSamples() int32
}
```

### Design Decisions
- Separate Plugin (metadata) from Processor (DSP)
- Simple interfaces that map to VST3 concepts
- Error returns for fallible operations
- No hidden allocations

## Parameter Package (`pkg/framework/param`)

### Purpose
Thread-safe parameter management with zero allocations in audio path.

### Key Components

#### Parameter Type
```go
type Parameter struct {
    ID           ParamID
    Name         string
    DefaultValue float64
    Min          float64
    Max          float64
    Unit         string
    flags        uint32
    value        uint64  // Atomic storage
}
```

#### Atomic Operations
```go
func (p *Parameter) GetValue() float64 {
    bits := atomic.LoadUint64(&p.value)
    return math.Float64frombits(bits)
}

func (p *Parameter) SetValue(v float64) {
    bits := math.Float64bits(v)
    atomic.StoreUint64(&p.value, bits)
}
```

#### Parameter Builder
```go
param.New(0, "Gain").
    Range(-24, 24).
    Default(0).
    Unit("dB").
    Build()
```

#### Registry
```go
type Registry struct {
    params map[ParamID]*Parameter
    order  []ParamID  // Maintains registration order
    mu     sync.RWMutex
}
```

### Thread Safety
- Lock-free parameter value access
- Registry locked only during setup/query
- No allocations during audio processing

## Process Package (`pkg/framework/process`)

### Purpose
Provides clean audio processing context with pre-allocated buffers.

### Key Components

#### Process Context
```go
type Context struct {
    Input      [][]float32  // Input channel buffers
    Output     [][]float32  // Output channel buffers
    SampleRate float64
    
    // Private fields
    numSamples int32
    params     *param.Registry
    workBuffer []float32    // Pre-allocated
}
```

#### Zero-Allocation Methods
```go
func (c *Context) NumSamples() int {
    return int(c.numSamples)
}

func (c *Context) WorkBuffer() []float32 {
    return c.workBuffer[:c.numSamples]
}

func (c *Context) Param(id ParamID) float64 {
    return c.params.GetValue(id)  // Atomic read
}
```

### Buffer Management
- All buffers pre-allocated
- Slicing for size adjustment
- No allocations in process calls

## Bus Package (`pkg/framework/bus`)

### Purpose
Audio bus configuration and management.

### Key Components

#### Bus Types
```go
type Type int
const (
    Main Type = iota
    Aux
)

type MediaType int
const (
    Audio MediaType = iota
    Event
)
```

#### Bus Definition
```go
type Bus struct {
    Name         string
    Type         Type
    MediaType    MediaType
    ChannelCount int32
    Active       bool
}
```

#### Configuration
```go
type Configuration struct {
    inputs  []Bus
    outputs []Bus
}

func NewStereoConfiguration() *Configuration {
    return &Configuration{
        inputs:  []Bus{{Name: "Input", ChannelCount: 2, Active: true}},
        outputs: []Bus{{Name: "Output", ChannelCount: 2, Active: true}},
    }
}
```

### Common Configurations
- `NewStereoConfiguration()` - Stereo in/out
- `NewMonoConfiguration()` - Mono in/out
- `NewSurroundConfiguration()` - 5.1/7.1
- Custom configurations via builder

## State Package (`pkg/framework/state`)

### Purpose
Plugin state persistence (presets, sessions).

### Key Components

#### State Manager
```go
type Manager struct {
    version  int
    params   *param.Registry
}
```

#### Serialization Interface
```go
type Serializer interface {
    WriteInt32(v int32) error
    WriteFloat64(v float64) error
    WriteString(s string) error
    WriteBytes(b []byte) error
}
```

#### State Operations
```go
func (m *Manager) Save(s Serializer) error {
    // Write version
    s.WriteInt32(m.version)
    
    // Write parameters
    for _, p := range m.params.All() {
        s.WriteInt32(p.ID)
        s.WriteFloat64(p.GetValue())
    }
}

func (m *Manager) Load(d Deserializer) error {
    // Version check
    version := d.ReadInt32()
    if version != m.version {
        return ErrVersionMismatch
    }
    
    // Read parameters
    // ...
}
```

### Versioning Strategy
- Explicit version field
- Forward compatibility considerations
- Graceful handling of missing data

## Integration with C Bridge

### Component Wrapper
The framework provides a wrapper that connects to the C bridge:

```go
type componentWrapper struct {
    plugin    Plugin
    processor Processor
    // ... state
}

//export GoCreateInstance
func GoCreateInstance(cid, iid *C.char) unsafe.Pointer {
    // Find plugin by ID
    plugin := findPlugin(C.GoString(cid))
    
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

## Best Practices

### Memory Management
1. Pre-allocate in `Initialize()`
2. Use object pools for temporary data
3. Avoid allocations in process path
4. Use atomic operations for parameters

### Error Handling
1. Return errors from fallible operations
2. Use named error variables
3. Provide context in error messages
4. Fail fast during initialization

### Thread Safety
1. Parameters: atomic operations
2. Configuration: read-write mutex
3. Processing: no shared mutable state
4. Clear ownership of data

### API Design
1. Simple interfaces
2. Builder pattern for complex objects
3. Sensible defaults
4. Escape hatches for advanced use

## Extension Points

### Custom Parameters
```go
type MyCustomParam struct {
    param.Parameter
    // Additional fields
}

func (p *MyCustomParam) FormatValue() string {
    // Custom formatting
}
```

### Process Extensions
```go
type ExtendedContext struct {
    *process.Context
    // Additional context
}
```

### Bus Configurations
```go
func NewCustomConfiguration() *Configuration {
    // Define custom bus layout
}
```

## Performance Considerations

### Cache Efficiency
- Related data grouped together
- Hot data in contiguous memory
- Minimal pointer indirection

### Lock-Free Design
- Atomic parameter access
- Pre-allocated buffers
- No allocations in audio path

### Optimization Opportunities
- SIMD-friendly data layout
- Branch-free parameter scaling
- Efficient buffer operations