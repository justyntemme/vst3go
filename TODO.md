# VST3Go - Unified Development Strategy

## Project Overview

VST3Go provides a Go framework for building VST3 audio plugins. We follow a "move fast, break things" philosophy to rapidly iterate towards a clean, idiomatic Go API that makes audio plugin development accessible to Go developers.

### Core Principles

1. **Minimal C Bridge** - C layer is just a thin wrapper, all business logic lives in Go
2. **Zero Allocations** - No memory allocations in the audio processing path
3. **Developer Experience** - Make the 80% use case trivial, the 20% possible
4. **Go Idiomatic** - Feel like a native Go library, not a C++ wrapper

## Current Status

### ✅ What's Working

**Architecture**
- Minimal C bridge layer (bridge.c, component.c) - just function routing
- Rich Go framework with clean separation of concerns
- Zero-allocation audio processing with pre-allocated buffers
- Thread-safe parameter system using atomic operations

**Framework Packages**
- `pkg/framework/plugin` - Plugin metadata and interfaces
- `pkg/framework/param` - Parameter management with fluent builder API
- `pkg/framework/process` - Audio processing context
- `pkg/framework/bus` - Bus configuration management
- `pkg/framework/state` - State persistence (structure in place)

**DSP Packages**
- `pkg/dsp/buffer` - Common buffer operations
- `pkg/dsp/filter` - Biquad and State Variable filters
- `pkg/dsp/oscillator` - Basic and band-limited oscillators
- `pkg/dsp/envelope` - ADSR, AR, and envelope followers
- `pkg/dsp/delay` - Various delay line implementations

**Working Examples**
- **SimpleGain** - Basic gain control (115 lines, down from 147)
- **SimpleDelay** - Delay effect with feedback
- **MultiModeFilter** - State variable filter with morphing

All examples build successfully and pass VST3 validation tests.

## Immediate Priorities

### 1. Parameter Automation from Host 🔜
Currently parameters can be set but changes from the host aren't processed.

```go
// TODO in pkg/plugin/component.go
func (w *componentWrapper) process(data *C.struct_Steinberg_Vst_ProcessData) C.Steinberg_tresult {
    // TODO: Process parameter changes from data.inputParameterChanges
    // This needs to read the parameter change queue and apply changes
}
```

### 2. State Save/Load Implementation 🔜
Framework structure exists but implementation is incomplete.

```go
// pkg/framework/state/manager.go needs:
- Actual serialization/deserialization
- Version handling
- Stream wrapper for IBStream
```

### 3. Process Context Support 🔜
Musical time, tempo, and transport information.

```go
// Add to process.Context:
- Tempo/BPM
- Time signature
- Transport state (playing/stopped)
- Sample position
- Musical position
```

### 4. Parameter Value Strings 🔜
Allow parameters to display formatted values.

```go
// Add to parameter system:
- Value to string conversion
- String to value parsing
- Custom formatting (e.g., "440 Hz", "-6.0 dB")
```

## Development Roadmap

### Phase 4: Developer Experience (Current) 🚧

**Goal**: Make plugin development as simple as possible

1. **Plugin Templates**
   ```go
   // Simple as:
   type MyPlugin struct {
       plugin.Base
   }
   
   func (p *MyPlugin) ProcessAudio(ctx *process.Context) {
       // Your DSP here
   }
   ```

2. **Common Effects Library**
   - Reverb template
   - Compressor template
   - EQ template
   - Synthesizer template

3. **Development Tools**
   - Hot reload support
   - Performance profiler
   - Parameter automation recorder
   - Debug visualizer

### Phase 5: MIDI & Events 🔜

**Goal**: Support instrument plugins

1. **Event Processing**
   - Note on/off events
   - MIDI CC handling
   - Pitch bend
   - Basic MPE support

2. **Voice Management**
   - Voice allocator
   - Note stealing
   - Envelope triggering

### Phase 6: Extended Features 📅

**Goal**: Professional plugin capabilities

1. **Advanced Bus Support**
   - Side-chain inputs
   - Multi-channel configurations
   - Surround formats (5.1, 7.1)

2. **Advanced Parameters**
   - Parameter groups
   - Linked parameters
   - Meta parameters

3. **Performance Features**
   - SIMD optimizations
   - Multi-core processing
   - Lookahead buffers

## VST3 Feature Implementation Status

### Core Features
- ✅ IComponent - Basic component interface
- ✅ IAudioProcessor - Audio processing
- ✅ IEditController - Parameter control
- ✅ 32-bit float processing
- ✅ Basic stereo I/O
- ✅ Parameter definition and storage
- ✅ Thread-safe parameter access
- 🚧 Parameter changes from host
- 🚧 State save/load
- ❌ 64-bit double processing
- ❌ Multi-bus support
- ❌ MIDI event processing
- ❌ IEditController2 - Extended parameter features
- ❌ IUnitInfo - Unit/preset organization

### Not Planned for v1.0
- GUI support (IPlugView) - Use generic host UI
- VST2 wrapper - Focus on VST3 only
- AAX/AU wrappers - VST3 first
- Windows/macOS support - Linux first, others later

## Code Examples

### Current API (Working)
```go
type DelayProcessor struct {
    params *param.Registry
    delay  *dsp.Line
}

func (p *DelayProcessor) ProcessAudio(ctx *process.Context) {
    delayMs := ctx.ParamPlain(ParamDelayTime)
    mix := float32(ctx.Param(ParamMix))
    
    for ch := 0; ch < ctx.NumChannels(); ch++ {
        p.delay.ProcessBufferMix(
            ctx.ChannelBuffer(ch), 
            delayMs, 
            mix,
        )
    }
}
```

### Target API (Simpler)
```go
type DelayPlugin struct {
    plugin.Base
    delay *dsp.Delay
}

func (p *DelayPlugin) Process(audio *plugin.Audio) {
    p.delay.Process(audio, p.Params.DelayTime, p.Params.Mix)
}
```

## Implementation Guidelines

### Zero Allocation Rules
1. Pre-allocate all buffers in `Initialize()`
2. Use object pools for temporary buffers
3. Avoid slice append in audio path
4. No string operations in audio path
5. Use atomic operations for parameters

### Testing Requirements
1. All plugins must pass VST3 validator
2. Benchmark tests for allocation checking
3. Integration tests with test host
4. Cross-platform build verification

## Getting Started

### Building
```bash
make all-examples  # Build all example plugins
make test-validate # Run VST3 validator
make install       # Install to ~/.vst3
```

### Creating a Plugin
1. Copy an example plugin
2. Modify the DSP processing
3. Adjust parameters as needed
4. Build and test

## Resources

- [VST3 SDK Documentation](https://steinbergmedia.github.io/vst3_dev_portal/)
- [VST3 C API Header](./include/vst3/vst3_c_api.h)
- [Example Plugins](./examples/)
- [Architecture Guide](./docs/architecture.md) (TODO)

## Success Metrics

### v1.0 Requirements
- ✅ Pass VST3 validator
- ✅ Zero allocations in audio path
- ✅ < 200 lines for basic effects
- 🚧 Parameter automation working
- 🚧 State persistence working
- 🚧 Used in production by at least one user
- 📅 Documentation complete
- 📅 10+ example plugins

### Post v1.0 Goals
- Cross-platform support
- MIDI/Instrument support
- Performance competitive with C++
- Active community
- Plugin marketplace

---

*This document is the single source of truth for VST3Go development. Update it as features are completed.*