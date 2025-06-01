# Modular GC Architecture - Selective GC Control via Shared Objects

## Breakthrough Concept

Instead of fighting Go's "all-or-nothing" GC, we compile different parts of our plugin as separate shared objects (.so files) with different GC settings. The C bridge orchestrates between them.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        VST3 Host                             │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                   C Bridge (vst3.so)                         │
│  - Implements VST3 API                                       │
│  - Loads Go modules dynamically                              │
│  - Routes calls to appropriate modules                       │
│  - Manages shared memory regions                             │
└──────┬──────────────────┬──────────────────┬────────────────┘
       │                  │                  │
       │                  │                  │
┌──────▼────────┐ ┌──────▼────────┐ ┌──────▼────────┐
│ rt_engine.so  │ │ control.so    │ │ ui_state.so   │
│               │ │               │ │               │
│ GOGC=off      │ │ GOGC=100      │ │ GOGC=100      │
│ No GC pauses  │ │ Normal GC     │ │ Normal GC     │
│               │ │               │ │               │
│ - DSP process │ │ - Parameters  │ │ - Presets     │
│ - Voice mix   │ │ - MIDI map    │ │ - UI updates  │
│ - Effects     │ │ - Automation  │ │ - State save  │
└───────────────┘ └───────────────┘ └───────────────┘
```

## Implementation Details

### 1. Build System

```makefile
# Makefile for multi-module plugin

# RT Module - GC disabled
rt_engine.so: cmd/rt_engine/*.go
	GOGC=off go build -buildmode=c-shared \
		-ldflags="-s -w" \
		-o build/rt_engine.so \
		./cmd/rt_engine

# Control Module - Normal GC
control.so: cmd/control/*.go
	go build -buildmode=c-shared \
		-o build/control.so \
		./cmd/control

# Main VST3 plugin - C bridge
myplugin.vst3: bridge/*.c build/*.so
	gcc -shared -fPIC \
		-o myplugin.vst3/Contents/x86_64-linux/myplugin.so \
		bridge/*.c \
		-ldl  # for dlopen
```

### 2. C Bridge - Dynamic Module Loading

```c
// bridge/module_loader.c

typedef struct {
    void* rt_handle;
    void* control_handle;
    void* ui_handle;
    
    // Function pointers from RT module (GC disabled)
    void (*rt_process)(float**, float**, int);
    void (*rt_handle_event)(Event*);
    
    // Function pointers from Control module (GC enabled)
    void (*ctrl_set_parameter)(int, float);
    void* (*ctrl_get_state)();
    
} PluginModules;

PluginModules* load_modules() {
    PluginModules* modules = calloc(1, sizeof(PluginModules));
    
    // Load RT module (GOGC=off)
    modules->rt_handle = dlopen("./rt_engine.so", RTLD_NOW);
    modules->rt_process = dlsym(modules->rt_handle, "Process");
    modules->rt_handle_event = dlsym(modules->rt_handle, "HandleEvent");
    
    // Load Control module (normal GC)
    modules->control_handle = dlopen("./control.so", RTLD_NOW);
    modules->ctrl_set_parameter = dlsym(modules->control_handle, "SetParameter");
    
    return modules;
}
```

### 3. RT Engine Module (GOGC=off)

```go
// cmd/rt_engine/main.go
package main

import "C"
import (
    "unsafe"
    "runtime/debug"
)

func init() {
    // Disable GC for this module
    debug.SetGCPercent(-1)
}

// Shared memory for communication
var sharedMem *SharedMemory

//export InitRT
func InitRT(mem unsafe.Pointer) {
    sharedMem = (*SharedMemory)(mem)
    
    // Pre-allocate everything
    initializeVoices()
    initializeBuffers()
    initializeEffects()
}

//export Process
func Process(inputs **C.float, outputs **C.float, numSamples C.int) {
    // This runs with no GC - can use full Go!
    // Convert C pointers to Go slices
    inSlices := convertInputs(inputs, numSamples)
    outSlices := convertOutputs(outputs, numSamples)
    
    // Process audio - no GC will interrupt this
    processVoices(outSlices, int(numSamples))
    applyEffects(inSlices, outSlices, int(numSamples))
}

//export HandleEvent
func HandleEvent(event *C.Event) {
    // Handle MIDI events in real-time
    switch event.type {
    case NOTE_ON:
        allocateVoice(int(event.note), float32(event.velocity))
    case NOTE_OFF:
        releaseVoice(int(event.note))
    }
}

// We can use normal Go idioms!
func processVoices(output [][]float32, numSamples int) {
    for _, voice := range activeVoices {
        if voice.IsActive() {
            // Even method calls work fine - no GC!
            voice.Process(tempBuffer[:numSamples])
            mixInto(output, tempBuffer[:numSamples])
        }
    }
}
```

### 4. Control Module (Normal GC)

```go
// cmd/control/main.go
package main

import "C"

// This module has normal GC - can allocate freely
var presetManager *PresetManager
var automationSystem *AutomationSystem
var midiMapper *MIDIMapper

//export InitControl
func InitControl(sharedMem unsafe.Pointer) {
    // Can use normal Go with allocations
    presetManager = NewPresetManager()
    automationSystem = NewAutomationSystem()
    midiMapper = NewMIDIMapper()
    
    // Load default presets (can read files, parse JSON, etc)
    presetManager.LoadDefaults()
}

//export SetParameter
func SetParameter(id C.int, value C.float) {
    // This can allocate, use maps, etc
    param := parameterRegistry[int(id)]
    if param == nil {
        return
    }
    
    // Update shared memory for RT module
    updateSharedParam(int(id), float32(value))
    
    // Record automation (can allocate)
    automationSystem.Record(param, float64(value))
    
    // Notify UI (can use channels)
    uiUpdateChan <- ParamUpdate{ID: int(id), Value: float64(value)}
}

//export SaveState
func SaveState() *C.char {
    // Can use encoding/json, maps, etc
    state := map[string]interface{}{
        "version": "1.0",
        "presets": presetManager.GetAll(),
        "parameters": getAllParameters(),
    }
    
    data, _ := json.Marshal(state)
    return C.CString(string(data))
}
```

### 5. Shared Memory Communication

```go
// shared/memory.go - included in both modules
package shared

// #include <stdatomic.h>
import "C"

type SharedMemory struct {
    // Parameters (Control writes, RT reads)
    Parameters [256]C.atomic_float
    
    // Voice states (RT writes, Control reads)
    VoiceStates [128]C.atomic_int
    
    // Level meters (RT writes, UI reads)
    OutputLevels [2]C.atomic_float
    VoiceLevels [128]C.atomic_float
    
    // Event queue (Control writes, RT reads)
    EventQueue EventQueue
}
```

### 6. Developer Experience

Developers create three packages:

```
myplugin/
├── rt/          # Real-time code (no GC)
│   ├── voices.go
│   ├── effects.go
│   └── mixer.go
├── control/     # Control logic (normal GC)
│   ├── presets.go
│   ├── automation.go
│   └── midi.go
├── shared/      # Shared definitions
│   ├── types.go
│   └── memory.go
└── build.yaml   # Build configuration
```

Build configuration:
```yaml
# build.yaml
modules:
  rt:
    path: ./rt
    gc: disabled
    exports: [Process, HandleEvent]
    
  control:
    path: ./control
    gc: normal
    exports: [SetParameter, SaveState, LoadState]
    
  ui:
    path: ./ui
    gc: normal
    exports: [GetUIState, HandleUIEvent]

shared_memory:
  size: 1MB
  layout: ./shared/memory.go
```

### 7. Advanced: Hot-Reloading

Since modules are separate .so files, we can hot-reload the control module:

```c
// Reload control module without stopping audio
void reload_control_module(PluginModules* modules) {
    // Save state
    char* state = modules->ctrl_save_state();
    
    // Unload old module
    dlclose(modules->control_handle);
    
    // Load new version
    modules->control_handle = dlopen("./control.so", RTLD_NOW);
    modules->ctrl_set_parameter = dlsym(modules->control_handle, "SetParameter");
    modules->ctrl_load_state = dlsym(modules->control_handle, "LoadState");
    
    // Restore state
    modules->ctrl_load_state(state);
}
```

## Benefits

### 1. **True GC Control**
- RT module has NO GC - zero pauses guaranteed
- Control modules have normal GC - full Go convenience
- Each module optimized for its use case

### 2. **Full Go Language Features**
In the RT module (even with GOGC=off):
- Methods and interfaces work
- Slices and arrays work
- Goroutines work (just no GC)
- All of Go's syntax

In control modules:
- Full GC benefits
- Can use any Go package
- Normal Go development

### 3. **Clean Separation**
- Audio processing isolated from control logic
- Can update control logic without touching audio
- Different teams can work on different modules

### 4. **Performance**
- RT module can be highly optimized
- Control module can prioritize developer productivity
- No cross-module function calls during audio processing

## Challenges & Solutions

### Challenge 1: Module Communication
**Solution**: Shared memory with atomic operations
```go
// RT module reads
cutoff := atomic.LoadFloat32(&shared.Parameters[PARAM_CUTOFF])

// Control module writes
atomic.StoreFloat32(&shared.Parameters[PARAM_CUTOFF], newValue)
```

### Challenge 2: Debugging
**Solution**: Each module can be debugged separately
```bash
# Debug control module with normal Go tools
dlv debug ./cmd/control

# Profile RT module without GC interference  
go tool pprof ./rt_engine.so cpu.prof
```

### Challenge 3: Deployment
**Solution**: Bundle all .so files in VST3 package
```
MyPlugin.vst3/
├── Contents/
│   ├── x86_64-linux/
│   │   ├── MyPlugin.so      # Main C bridge
│   │   ├── rt_engine.so     # RT module  
│   │   ├── control.so       # Control module
│   │   └── ui.so           # UI module
│   └── Resources/
│       └── moduleinfo.json  # Module metadata
```

## Example: Complete Synthesizer

### RT Module (cmd/rt/main.go)
```go
package main

import "C"
import "sync/atomic"

var voices [128]Voice
var shared *SharedMemory

//export Process
func Process(output **C.float, numSamples C.int) {
    // Clear output
    samples := int(numSamples)
    out := convertOutput(output, samples)
    
    // Process all voices - no GC will interrupt
    for i := range voices {
        if voices[i].active {
            voices[i].ProcessInto(out, samples)
        }
    }
    
    // Update levels for UI
    updateLevels(out)
}

//export NoteOn
func NoteOn(note, velocity C.uchar) {
    // Find free voice - no allocations
    for i := range voices {
        if !voices[i].active {
            voices[i].Start(int(note), float32(velocity)/127.0)
            atomic.StoreInt32(&shared.VoiceStates[i], 1)
            break
        }
    }
}
```

### Control Module (cmd/control/main.go)
```go
package main

import "C"
import "encoding/json"

var presets map[string]Preset // Can use maps!

//export LoadPreset
func LoadPreset(name *C.char) {
    presetName := C.GoString(name)
    
    // Can allocate, read files, parse JSON
    if preset, ok := presets[presetName]; ok {
        // Update all parameters
        for id, value := range preset.Parameters {
            atomic.StoreFloat32(&shared.Parameters[id], value)
        }
        
        // Can even use goroutines
        go notifyPresetLoaded(presetName)
    }
}
```

## Conclusion

This modular approach gives us:
1. **Absolute GC control** - RT modules have zero GC
2. **Full Go features** - Both in RT and control modules
3. **Clean architecture** - Natural separation of concerns
4. **Production ready** - Similar to how many DAWs work internally

The key insight: **Different parts of a plugin have different requirements**. By compiling them as separate modules, each part gets exactly what it needs.