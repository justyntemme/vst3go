# VST3Go - Development Roadmap

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
- Component handler for parameter change notifications
- Sample-accurate parameter automation
- Complete state persistence with custom data support
- Full transport and tempo synchronization

**Framework Packages**
- `pkg/framework/plugin` - Plugin metadata and interfaces
- `pkg/framework/param` - Parameter management with fluent builder API
- `pkg/framework/process` - Audio processing context with transport info
- `pkg/framework/bus` - Bus configuration management
- `pkg/framework/state` - State persistence with custom data support

**DSP Packages**
- `pkg/dsp/buffer` - Common buffer operations
- `pkg/dsp/filter` - Biquad and State Variable filters
- `pkg/dsp/oscillator` - Basic and band-limited oscillators
- `pkg/dsp/envelope` - ADSR, AR, and envelope followers
- `pkg/dsp/delay` - Various delay line implementations

**Working Examples**
- **SimpleGain** - Basic gain control
- **SimpleDelay** - Delay effect with feedback
- **MultiModeFilter** - State variable filter with morphing

All examples build successfully and pass VST3 validation tests.

## Development Priorities

### Phase 1: DSP Library Enhancement 🚧

**Goal**: Provide comprehensive DSP building blocks for plugin developers

1. **Dynamics Processing**
   ```go
   // pkg/dsp/dynamics/
   - Compressor with lookahead
   - Limiter with true peak detection
   - Gate with hysteresis
   - Expander
   - Envelope detector with multiple modes
   ```

2. **Modulation Effects**
   ```go
   // pkg/dsp/modulation/
   - LFO with multiple waveforms and sync
   - Chorus with multiple voices
   - Flanger with feedback
   - Phaser with multiple stages
   - Ring modulator
   - Tremolo and vibrato
   ```

3. **Reverb Algorithms**
   ```go
   // pkg/dsp/reverb/
   - Schroeder reverb
   - Freeverb implementation
   - FDN (Feedback Delay Network) reverb
   - Early reflections processor
   - Convolution reverb support
   ```

4. **Distortion & Saturation**
   ```go
   // pkg/dsp/distortion/
   - Waveshaping with multiple curves
   - Tube saturation emulation
   - Tape saturation emulation
   - Bit crushing and sample rate reduction
   - Asymmetric clipping
   ```

5. **Analysis Tools**
   ```go
   // pkg/dsp/analysis/
   - FFT wrapper with window functions
   - Spectrum analyzer
   - Peak/RMS/LUFS meters
   - Correlation meter
   - Phase scope
   ```

6. **Utility DSP**
   ```go
   // pkg/dsp/utility/
   - Noise generators (white, pink, brown)
   - DC blocker
   - Interpolation (cubic, sinc, all-pass)
   - Crossfade utilities
   - Window functions (Hann, Hamming, Blackman)
   - Dithering algorithms
   ```

### Phase 2: MIDI & Event Support 🔜

**Goal**: Enable instrument plugin development

1. **Core MIDI Infrastructure**
   ```go
   // pkg/midi/
   - MIDI event types (Note On/Off, CC, Pitch Bend, etc.)
   - MIDI event queue with sample-accurate timing
   - MIDI learn system
   - MPE (MIDI Polyphonic Expression) support
   ```

2. **Voice Management**
   ```go
   // pkg/framework/voice/
   - Voice allocator with multiple modes
   - Voice stealing algorithms
   - Portamento/glide system
   - Unison/detune management
   ```

3. **Event Processing**
   ```go
   // Update process.Context:
   - Event input queue
   - Event output queue (for MIDI effects)
   - Sample-accurate event processing
   ```

### Phase 3: Advanced Bus Support 🔜

**Goal**: Professional routing capabilities

1. **Multi-channel Configurations**
   - Stereo, 5.1, 7.1, Ambisonics
   - Side-chain input support
   - Multiple input/output buses
   - Dynamic bus activation

2. **Bus Templates**
   ```go
   // Common configurations
   - EffectStereo (1 in, 1 out)
   - EffectStereoSidechain (2 in, 1 out)
   - SurroundPanner (1 in, 1 surround out)
   - MixerChannel (1 in, multiple out)
   ```

### Phase 4: Developer Tools & Experience 🔜

**Goal**: Make plugin development delightful

1. **Project Generator**
   ```bash
   vst3go new effect --name "MyReverb" --company "MyCompany"
   vst3go new instrument --name "MySynth" --voices 16
   ```

2. **Hot Reload System**
   - Watch for code changes
   - Rebuild and reload in test host
   - Preserve parameter state

3. **Debug Visualizer**
   - Real-time parameter monitoring
   - Audio scope and spectrum
   - CPU usage profiling
   - Memory allocation tracking

4. **Preset Management**
   ```go
   // pkg/framework/preset/
   - Preset save/load system
   - Bank management
   - Factory preset embedding
   - Preset morphing support
   ```

5. **Testing Framework**
   ```go
   // pkg/testing/
   - Mock host implementation
   - Automated parameter testing
   - Audio comparison utilities
   - Performance benchmarking
   ```

### Phase 5: Cross-Platform Support 🔜

**Goal**: True cross-platform deployment

1. **Platform Abstraction**
   ```go
   // Platform-specific implementations
   - component_windows.go (COM support)
   - component_darwin.go (Core Foundation)
   - component_linux.go (current implementation)
   ```

2. **Build System Enhancement**
   - Cross-compilation support
   - Bundle generation for macOS
   - Installer generation for Windows
   - Debian package generation

3. **CI/CD Pipeline**
   - GitHub Actions for all platforms
   - Automated testing on each platform
   - Binary releases for all platforms

### Phase 6: Simple Synthesizer Example 🎹

**Goal**: Demonstrate the framework's capabilities with a complete instrument

**SimpleSynth** - A subtractive synthesizer showcasing:
- 2 Oscillators with multiple waveforms
- ADSR envelope for amplitude
- ADSR envelope for filter
- State variable filter with envelope modulation
- LFO for vibrato and filter modulation
- Voice management (8-16 voices)
- MIDI learn for all parameters
- Preset support
- < 500 lines of code

```go
type SimpleSynth struct {
    plugin.Base
    voices   []*SynthVoice
    lfo      *dsp.LFO
    reverb   *dsp.Reverb
}

func (s *SimpleSynth) ProcessEvent(event midi.Event) {
    // Handle MIDI events
}

func (s *SimpleSynth) ProcessAudio(ctx *process.Context) {
    // Mix active voices
    // Apply global effects
}
```

## Implementation Guidelines

### Code Quality Standards

1. **Zero Allocation Rule**
   - Pre-allocate all buffers in Initialize()
   - Use object pools for temporary objects
   - Verify with benchmarks

2. **Thread Safety**
   - Use atomic operations for parameters
   - Minimize mutex usage in audio path
   - Document thread safety guarantees

3. **Testing Requirements**
   - Unit tests for all DSP algorithms
   - Benchmark tests for performance
   - Integration tests with test host
   - Cross-platform build verification

### API Design Principles

1. **Idiomatic Go**
   ```go
   // Good: Go-style API
   filter := dsp.NewBiquad(dsp.LowPass, 1000, 0.7)
   
   // Avoid: C++ style
   filter := dsp.NewBiquadFilter()
   filter.SetType(dsp.FILTER_TYPE_LOWPASS)
   filter.SetFrequency(1000)
   ```

2. **Builder Pattern for Complex Objects**
   ```go
   reverb := dsp.NewReverb().
       WithRoomSize(0.8).
       WithDamping(0.5).
       WithPreDelay(20 * time.Millisecond).
       Build()
   ```

3. **Functional Options**
   ```go
   osc := dsp.NewOscillator(
       dsp.WithWaveform(dsp.Sawtooth),
       dsp.WithAntiAliasing(true),
   )
   ```

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

## Success Metrics

### v1.0 Requirements
- ✅ Pass VST3 validator
- ✅ Zero allocations in audio path
- ✅ < 200 lines for basic effects
- ✅ Follows architectural guardrails
- ✅ Cross-platform support (Linux, Windows, macOS)
- ✅ Parameter automation working
- ✅ State persistence working
- 🚧 Comprehensive DSP library
- 🚧 MIDI support for instruments
- 🚧 Developer tools and templates
- 📅 Documentation complete
- 📅 15+ example plugins
- 📅 Simple synthesizer example

### Post v1.0 Goals
- Performance competitive with C++
- Active community
- Plugin marketplace
- Visual plugin builder
- AI-assisted DSP development

## Resources

- [VST3 SDK Documentation](https://steinbergmedia.github.io/vst3_dev_portal/)
- [VST3 Developer Portal](https://steinbergmedia.github.io)
- [VST3 C API Header](./include/vst3/vst3_c_api.h)
- [Example Plugins](./examples/)
- [Architecture Guide](./docs/architecture.md)

---

*This document is the single source of truth for VST3Go development. Update it as features are completed.*