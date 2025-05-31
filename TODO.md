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

### 1. Parameter Automation from Host ✅ DONE
Parameter changes from the host are now processed correctly.

### 2. Parameter Value Strings ✅ DONE  
Parameter formatting and parsing is implemented with custom formatters.

### 3. Component Handler for Parameter Change Notifications ✅ DONE
Component handler is now stored and notification methods are implemented.

```go
// Implemented in:
// - pkg/plugin/wrapper_controller.go: SetComponentHandler stores the handler
// - pkg/plugin/wrapper.go: notification methods (notifyParamBeginEdit, etc.)
// - pkg/plugin/component.go: SetParamNormalizedWithNotification method
```

**Refactoring opportunities** for plugins to use parameter notifications:
- Auto-gain/normalization plugins can notify host of gain adjustments
- Compressors can update gain reduction meter parameters
- Envelope followers can update visual feedback parameters  
- Adaptive filters can notify of automatic parameter changes
- LFO/modulation sources can show current modulation values
- Tempo-synced effects can update when host tempo changes

**Note**: Plugins need access to the component to call SetParamNormalizedWithNotification.
Consider adding a SetComponent method to the Processor interface or passing through context.

### 4. Sample-Accurate Parameter Automation ✅ DONE
Parameter changes are now processed at exact sample boundaries for precise automation.

```go
// Implemented in:
// - pkg/framework/process/context.go: Parameter change collection and sorting
// - pkg/plugin/component.go: processSampleAccurate method processes audio in chunks
// - Zero-allocation design using sub-slices of existing buffers
```

**Implementation details**:
- Parameter changes are collected during the parameter processing phase
- Changes are sorted by sample offset
- Audio is processed in chunks between parameter changes
- Each parameter change is applied at its exact sample position
- No allocations in the audio path - uses buffer sub-slicing

### 5. State Save/Load Implementation ✅ DONE
Complete state persistence with support for custom plugin data.

```go
// Implemented in:
// - pkg/framework/state/manager.go: Full serialization/deserialization
// - pkg/plugin/plugin.go: StatefulProcessor interface for custom state
// - pkg/plugin/component.go: Integration with StatefulProcessor
```

**Features**:
- Automatic parameter state persistence
- Optional custom state save/load via StatefulProcessor interface
- Version handling and forward compatibility
- Magic header validation ("VST3GO")

**Example usage**:
```go
// Implement StatefulProcessor to save custom data
type DelayProcessor struct {
    // ... fields ...
}

func (p *DelayProcessor) SaveCustomState(w io.Writer) error {
    // Save delay buffer position, contents, etc.
    binary.Write(w, binary.LittleEndian, p.writePos)
    // ... save other custom data
    return nil
}

func (p *DelayProcessor) LoadCustomState(r io.Reader) error {
    // Load delay buffer position, contents, etc.
    binary.Read(r, binary.LittleEndian, &p.writePos)
    // ... load other custom data
    return nil
}
```

### 6. Process Context Support ✅ DONE
Musical time, tempo, and transport information are now available to plugins.

```go
// Implemented in:
// - pkg/vst3/buffers.go: Complete ProcessContext wrapper with all VST3 fields
// - pkg/framework/process/context.go: TransportInfo struct with helper methods
// - pkg/plugin/component.go: Extracts transport info from VST3 process context
```

**Features**:
- Transport state (playing, recording, cycling)
- Tempo in BPM
- Time signature
- Musical position in quarter notes
- Bar position
- Cycle/loop points
- Sample-accurate timing
- Helper methods for common calculations

**Example usage**:
```go
func (p *DelayProcessor) ProcessAudio(ctx *process.Context) {
    if ctx.Transport.IsPlaying && ctx.Transport.HasTempo {
        // Sync delay time to tempo
        samplesPerBeat := ctx.Transport.GetSamplesPerBeat(ctx.SampleRate)
        delayTime := samplesPerBeat * 0.5 // Eighth note delay
        
        // Get current position
        bars, beats := ctx.Transport.GetBarsBeats()
        
        // Check if we're on a beat
        if ctx.Transport.IsOnBeat(0.01) {
            // Trigger something on the beat
        }
    }
}
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

3. **Example Synthesizer**
   - Simple subtractive synth
   - Demonstrates MIDI handling
   - Uses DSP package components

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

### Phase 7: Platform-Specific Features 📅

**Goal**: Full cross-platform support

1. **Windows Support**
   - Windows-specific view handling
   - COM integration
   - Platform-specific optimizations

2. **macOS Support**
   - Core Audio integration
   - Metal Performance Shaders
   - macOS-specific bundle structure

3. **Linux Enhancements**
   - Jack support
   - Real-time kernel optimizations

### Future: GUI Support (When Manually Approved) 🔒

**Note**: GUI implementation will only begin when explicitly requested

1. **View System**
   - IPlugView implementation
   - Platform-specific window creation
   - Event handling

2. **Graphics Integration**
   - OpenGL/Metal/DirectX support
   - Hardware acceleration
   - DPI scaling

## VST3 Feature Implementation Status

### Core Features
- ✅ IComponent - Basic component interface
- ✅ IAudioProcessor - Audio processing
- ✅ IEditController - Parameter control
- ✅ 32-bit float processing
- ✅ Basic stereo I/O
- ✅ Parameter definition and storage
- ✅ Thread-safe parameter access
- ✅ Parameter changes from host
- ✅ Parameter value formatting and parsing
- 🚧 State save/load
- ❌ 64-bit double processing
- ❌ Multi-bus support
- ❌ MIDI event processing
- ❌ IEditController2 - Extended parameter features
- ❌ IUnitInfo - Unit/preset organization

### Platform Support Strategy
- **All platforms from the start** - Linux, Windows, macOS
- Use build tags and conditional compilation:
  ```go
  // +build linux
  
  // +build windows
  
  // +build darwin
  ```
- Platform-specific code in separate files:
  - `component_linux.go`
  - `component_windows.go`
  - `component_darwin.go`
- C code with proper `#ifdef`:
  ```c
  #ifdef _WIN32
    // Windows specific
  #elif __APPLE__
    // macOS specific
  #else
    // Linux specific
  #endif
  ```

### Deferred Until Manually Approved
- GUI support (IPlugView) - Will be added when explicitly requested
- VST2 wrapper - Focus on VST3 only
- AAX/AU wrappers - VST3 first

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

### Architectural Principles (from guardrails.md)

1. **C Bridge Philosophy**: Keep it minimal
   - Just routing, no business logic
   - Direct VST3 C API to Go mapping
   - All framework features in Go layers
   - Use manifest discovery (no Go registration in C bridge)
   - Consult [VST3 documentation](https://steinbergmedia.github.io) for interface contracts

2. **Layered Architecture**:
   - Layer 1: Minimal C bridge
   - Layer 2: Go-idiomatic abstractions
   - Layer 3: DSP utilities
   - Layer 4: Developer conveniences

3. **No Over-Abstraction**:
   - VST3 concepts remain accessible
   - Provide escape hatches
   - Make simple tasks simple

4. **Move Fast, Break Things**:
   - No backwards compatibility concerns
   - Delete old code when refactoring
   - Iterate quickly

### Zero Allocation Rules
1. Pre-allocate all buffers in `Initialize()`
2. Pre-allocate object pools in `Initialize()`:
   ```go
   // In Initialize():
   pool := &WorkBufferPool{
       buffers: make([][]float32, maxConcurrency),
   }
   for i := range pool.buffers {
       pool.buffers[i] = make([]float32, maxBlockSize)
   }
   
   // In ProcessAudio():
   work := pool.Get()  // Just returns pre-allocated buffer
   defer pool.Put(work) // Returns to pool, no deallocation
   ```
3. Avoid slice append in audio path
4. No string operations in audio path
5. Use atomic operations for parameters
6. All allocations happen during initialization, not processing

### Cross-Platform Development
1. **Build Tags**: Use for platform-specific Go code
   ```go
   // +build windows,amd64
   ```

2. **Conditional Compilation**: Use for C code
   ```c
   #ifdef _WIN32
     // Windows code
   #endif
   ```

3. **Platform Files**: Separate implementations
   - Common interface in `component.go`
   - Platform code in `component_<platform>.go`

4. **Testing**: Must test on all target platforms

### Testing Requirements
1. All plugins must pass VST3 validator
2. Benchmark tests for allocation checking
3. Integration tests with test host
4. Cross-platform build verification
5. Platform-specific test targets in Makefile

## Getting Started

### Building
```bash
# Build all example plugins (auto-detects platform)
make all-examples

# Build for specific platform
make all-examples GOOS=windows
make all-examples GOOS=darwin

# Build specific plugin
make gain
make filter

# Run VST3 validation
make test-validate PLUGIN_NAME=SimpleGain

# Install to platform-specific VST3 directory
make install  # Uses ~/.vst3 on Linux, appropriate dirs on Windows/macOS
```

### Creating a Plugin
1. Copy an example plugin
2. Modify the DSP processing
3. Adjust parameters as needed
4. Build and test

## Resources

- [VST3 SDK Documentation](https://steinbergmedia.github.io/vst3_dev_portal/)
- [VST3 Developer Portal](https://steinbergmedia.github.io) - Complete VST3 architecture and development principles
  - Architecture overview and design patterns
  - Interface specifications and expected behaviors
  - Best practices for plugin development
  - Guidelines for host/plugin communication
- [VST3 C API Header](./include/vst3/vst3_c_api.h)
- [Example Plugins](./examples/)
- [Architecture Guide](./docs/architecture.md)

## Success Metrics

### v1.0 Requirements
- ✅ Pass VST3 validator
- ✅ Zero allocations in audio path
- ✅ < 200 lines for basic effects
- ✅ Follows architectural guardrails
- ✅ Cross-platform support (Linux, Windows, macOS)
- ✅ Parameter automation working
- 🚧 State persistence working
- 🚧 Used in production by at least one user
- 📅 Documentation complete
- 📅 10+ example plugins

### Post v1.0 Goals
- MIDI/Instrument support
- Performance competitive with C++
- Active community
- Plugin marketplace
- GUI support (when approved)

## Code Quality & Refactoring Opportunities

### Areas Identified for Cleanup

1. **Sample-Accurate Processing Optimization** (`pkg/plugin/component.go:399`)
   - Current implementation uses `append` which may allocate
   - Consider pre-allocating slice headers for sub-buffer views
   - Could optimize by reusing slice headers between process calls

2. **Parameter Change Buffer Size** (`pkg/framework/process/context.go:42`)
   - Currently hardcoded to 128 parameter changes
   - Should be configurable or dynamically sized based on plugin needs
   - Add bounds checking and warning when limit exceeded

3. **Debug Output Cleanup**
   - Multiple `fmt.Printf` statements throughout parameter handling
   - Should use proper logging framework with levels
   - Allow debug output to be enabled/disabled via configuration

4. **Process Method Refactoring** (`pkg/plugin/component.go:197`)
   - Process function has 74 statements (lint suggests max 50)
   - Could extract transport info update into separate method
   - Could extract buffer mapping into separate method

5. **Musical Constants** (`pkg/framework/process/context.go`)
   - Define constants for musical values:
     - `QuarterNotesPerWhole = 4.0`
     - `SecondsPerMinute = 60.0`
     - `DefaultParamChangeBufferSize = 128`

---

*This document is the single source of truth for VST3Go development. Update it as features are completed.*