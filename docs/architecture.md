# VST3Go Architecture

## Overview

VST3Go is a layered framework designed to make VST3 plugin development accessible to Go developers while maintaining the performance requirements of real-time audio processing. The architecture follows a strict separation of concerns with a minimal C bridge and rich Go framework layers.

## Core Principles

### 1. Minimal C Bridge
The C layer serves purely as a translation layer between the VST3 C API and Go. It contains no business logic, no state management, and no framework features. All it does is route function calls from the host to Go and back.

### 2. Zero Allocations in Audio Path
Real-time audio processing requires predictable performance. All memory allocations happen during initialization, not during audio processing. This is achieved through:
- Pre-allocated buffers
- Object pools for temporary data
- Atomic operations for parameter access
- Lock-free data structures where possible

### 3. Layered Architecture
The framework is organized into four distinct layers, each with a specific purpose:

```
┌─────────────────────────────────────────┐
│         Layer 4: Developer Tools        │  <- Templates, generators, helpers
├─────────────────────────────────────────┤
│         Layer 3: DSP Library            │  <- Filters, oscillators, effects
├─────────────────────────────────────────┤
│      Layer 2: Go Framework Core         │  <- Plugin abstractions, parameters
├─────────────────────────────────────────┤
│        Layer 1: C Bridge                │  <- Minimal VST3 C API wrapper
└─────────────────────────────────────────┘
```

## Architecture Layers

### Layer 1: C Bridge (`/bridge`)
**Purpose**: Direct, minimal mapping of VST3 C API to Go

**Components**:
- `bridge.c/h` - Plugin factory and entry point
- `component.c/h` - Component interface routing

**Design**:
- No business logic
- Direct function mapping
- Handle-based object management
- Manifest-driven plugin discovery

[Read more about the C Bridge →](./c-bridge.md)

### Layer 2: Go Framework Core (`/pkg/framework`)
**Purpose**: Go-idiomatic abstractions for VST3 concepts

**Packages**:
- `plugin` - Plugin metadata and base types
- `param` - Parameter management with atomic operations
- `process` - Audio processing context
- `bus` - Audio bus configuration
- `state` - State persistence

[Read more about the Framework Core →](./framework-core.md)

### Layer 3: DSP Library (`/pkg/dsp`)
**Purpose**: Comprehensive audio processing utilities

**Packages**:
- `buffer` - Buffer operations (copy, mix, scale)
- `filter` - Various filter implementations
- `oscillator` - Signal generators
- `envelope` - Envelope generators
- `delay` - Delay line implementations

[Read more about the DSP Library →](./dsp-library.md)

### Layer 4: Developer Tools
**Purpose**: Make plugin development faster and easier

**Components**:
- Plugin templates
- Code generators
- Debug utilities
- Performance profilers

[Read more about Developer Tools →](./developer-tools.md)

## Data Flow

### Audio Processing Path
```
Host → C Bridge → Go Component → Process Context → Plugin Code → DSP → Output
         ↓                              ↓
    (no allocation)            (pre-allocated buffers)
```

### Parameter Changes
```
Host → C Bridge → Parameter Queue → Atomic Update → Audio Thread Read
                        ↓
                  (lock-free)
```

## Memory Management

### Pre-allocation Strategy
All buffers are allocated during the `Initialize` phase:

```go
func (p *Processor) Initialize(sampleRate float64, maxBlockSize int32) error {
    // Allocate all buffers here
    p.workBuffer = make([]float32, maxBlockSize)
    p.delayLine = make([]float32, int(sampleRate * maxDelaySeconds))
    // etc...
}
```

### Object Pools
Temporary buffers are managed through pre-allocated pools:

```go
type BufferPool struct {
    buffers [][]float32
    index   int32 // atomic
}

func (p *BufferPool) Get() []float32 {
    // Returns pre-allocated buffer, no allocation
}
```

## Thread Safety

### Parameter System
Parameters use atomic operations for lock-free access:
- Write thread: Host automation, UI
- Read thread: Audio processing
- No locks in audio path

### State Management
Component state uses appropriate synchronization:
- Mutex for configuration changes
- Atomic operations for real-time values
- Copy-on-write for complex state

## Platform Support

### Cross-Platform Strategy
- Build tags for platform-specific Go code
- Conditional compilation for C code
- Platform-specific file naming conventions
- Unified API across platforms

### Platform Files
```
component.go         # Common interface
component_linux.go   # Linux-specific
component_windows.go # Windows-specific
component_darwin.go  # macOS-specific
```

## Performance Considerations

### CPU Cache Optimization
- Data structures aligned for cache lines
- Hot data grouped together
- Minimal pointer chasing in audio path

### SIMD Readiness
- Buffer operations use simple loops
- Data layout compatible with SIMD
- Future optimization path clear

## Extension Points

### Custom DSP
Plugins can implement custom DSP while using framework utilities:
```go
func (p *MyPlugin) ProcessAudio(ctx *process.Context) {
    // Use framework DSP
    p.filter.Process(ctx.Input[0])
    
    // Custom processing
    myCustomEffect(ctx.Output[0])
}
```

### Parameter Extensions
Custom parameter types can be built on the base system:
```go
type FrequencyParam struct {
    param.Parameter
    // Custom frequency-specific behavior
}
```

## Error Handling

### Initialization Errors
- Explicit error returns
- Graceful degradation where possible
- Clear error messages

### Runtime Errors
- No panics in audio path
- Defensive programming
- Silent failure with safe defaults

## Testing Strategy

### Unit Tests
- Package-level testing
- Mock interfaces for isolation
- Benchmark tests for performance

### Integration Tests
- VST3 validator compliance
- Host compatibility testing
- Cross-platform verification

## Future Extensibility

The architecture is designed to support future additions:
- MIDI processing (Layer 2 extension)
- GUI support (New layer above Layer 4)
- Additional DSP algorithms (Layer 3 modules)
- Performance optimizations (Layer implementation details)

---

For detailed information about each layer, see the linked documentation.