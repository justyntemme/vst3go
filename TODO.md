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

**Architecture** - Complete minimal C bridge, zero-allocation processing, thread-safe parameters, state persistence

**Framework Packages** - All core packages implemented (plugin, param, process, bus, state)

**DSP Library** - Complete implementations: filters, oscillators, envelopes, delays, dynamics, modulation, reverb

**Working Examples** - gain, delay, filter (all pass VST3 validation)

## Development Priorities

### Phase 1: DSP Library Enhancement 🚧

**Goal**: Provide comprehensive DSP building blocks for plugin developers

**NOTE**: DSP library is complete but example plugins need to be created to showcase capabilities.

#### Completed DSP Components:

**1. Dynamics** ✅ DONE - Envelope detector, Compressor, Limiter, Gate, Expander

**2. Modulation** ✅ DONE - LFO, Chorus, Flanger, Phaser, Ring Modulator, Tremolo

**3. Reverb** ✅ DONE - Schroeder, Freeverb, FDN algorithms

**4. Distortion & Saturation** ✅ DONE - Waveshaper, Tube, Tape, Bitcrusher, MultiDistortion example

**5. Analysis Tools** ✅ DONE - FFT, Spectrum analyzer, Meters (Peak/RMS/LUFS), Correlation meter, Phase scope

**6. Utility DSP** ✅ DONE - Gain, Mix, Pan, Interpolation utilities

### Phase 2: MIDI & Event Support 🚧 TODO

**Goal**: Enable instrument plugin development

1. **Core MIDI Infrastructure** 🚧 TODO
   ```go
   // pkg/midi/
   - MIDI event types (Note On/Off, CC, Pitch Bend, etc.) 🚧
   - MIDI event queue with sample-accurate timing 🚧
   - MIDI learn system 🔜 (future enhancement)
   - MPE (MIDI Polyphonic Expression) support 🔜 (future enhancement)
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

### Phase 3: Advanced Bus Support ✅ DONE

**Multi-channel routing** - All speaker configurations, sidechain, multi-bus, dynamic activation
**Bus Templates** - 15+ common configurations implemented
**Examples** - sidechain and surround plugins

### Phase 4: Developer Tools & Experience 🔜

**Goal**: Make plugin development delightful

1. **Project Generator**
   ```bash
   vst3go new effect --name "MyReverb" --company "MyCompany"
   vst3go new instrument --name "MySynth" --voices 16
   ```
### Phase 5: Cross-Platform Support 🔒 DEFERRED

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
   - Add bounds checking and warning when limit exceeded - How can we do this while keeping allocations at start and not during processing

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
- ✅ Follows architectural guardrails
- 🔒 Cross-platform support (Linux only for now, others deferred)
- ✅ Parameter automation working
- ✅ State persistence working
- 🚧 Comprehensive DSP library (missing distortion effects)
- 🚧 MIDI support for instruments
- 🚧 Developer tools and templates
- 📅 Documentation complete
- 📅 15+ example plugins
- 📅 Simple synthesizer example

## Example Plugins Roadmap

### Phase 1: Dynamics Examples 🔜

**1. MasterCompressor** - Professional Compressor Plugin ✅ DONE
- Demonstrates: Compressor, Envelope Detector, Stereo Linking
- Features:
  - Threshold, Ratio, Attack, Release, Knee controls
  - Makeup gain with auto-gain option
  - Lookahead for transparent compression
  - Sidechain filter (high-pass)
  - Gain reduction meter
  - Stereo linked processing
- Implementation Guide:
  ```go
  // Use pkg/dsp/dynamics/compressor.go
  // Add sidechain HPF using simple filter
  // Parameter ranges: threshold -60 to 0 dB, ratio 1:1 to 20:1
  // Attack 0.1-100ms, Release 10-1000ms
  ```

**2. StudioGate** - Noise Gate Plugin ✅ DONE
- Demonstrates: Gate with hysteresis, hold time, range control
- Features:
  - Threshold with hysteresis display
  - Attack/Hold/Release envelope
  - Range control (max attenuation)
  - Sidechain filter (HPF for kick drum gating)
  - Gate state LED (open/closed)
- Implementation Guide:
  ```go
  // Use pkg/dsp/dynamics/gate.go
  // Add visual feedback for gate state
  // Parameter ranges: threshold -80 to 0 dB, hysteresis 0-10 dB
  // Hold 0-100ms, Range -80 to 0 dB
  ```

**3. TransientShaper** - Expander/Transient Designer ✅ DONE
- Demonstrates: Expander for enhancing transients
- Features:
  - Downward expansion for punch
  - Attack and sustain controls
  - Parallel processing (mix)
  - Output gain
- Implementation Guide:
  ```go
  // Use pkg/dsp/dynamics/expander.go
  // Add parallel processing path
  // Focus on drum enhancement use case
  ```

**4. MasterLimiter** - Brick-wall Limiter
- Demonstrates: Limiter with true peak detection
- Features:
  - Ceiling control
  - Release time
  - True peak detection on/off
  - Lookahead for transparency
  - Gain reduction meter
- Implementation Guide:
  ```go
  // Use pkg/dsp/dynamics/limiter.go
  // Simple interface focused on mastering
  // Ceiling -3 to 0 dB, Release 1-100ms
  ```

### Phase 2: Modulation Examples 🔜

**5. VintageChorus** - Classic Chorus Effect
- Demonstrates: Multi-voice chorus with LFO modulation
- Features:
  - Rate, Depth, Delay, Mix controls
  - 1-4 voice selection
  - Stereo spread
  - Feedback for richer sound
- Implementation Guide:
  ```go
  // Use pkg/dsp/modulation/chorus.go
  // Preset system for classic sounds
  // Rate 0.1-10 Hz, Depth 0-10ms, Delay 10-50ms
  ```

**6. JetFlanger** - Flanger Effect (when implemented)
- Demonstrates: Flanger with feedback
- Features:
  - Rate, Depth, Feedback, Mix
  - Manual control for static flanging
  - Negative feedback option
- Implementation Guide:
  ```go
  // Use pkg/dsp/modulation/flanger.go (to be created)
  // Very short delays (0.5-10ms)
  // High feedback for jet sounds
  ```

### Phase 3: Synthesis Examples 🔜

**7. SimpleSynth** - Basic Subtractive Synthesizer ✅ DONE
- Demonstrates: Note events, voice allocation, MIDI processing
- Features:
  - Oscillator with multiple waveforms (saw, square, sine, triangle)
  - ADSR envelope for amplitude
  - Filter with cutoff and resonance
  - Voice allocation (8 voices polyphonic)
  - Pitch bend support
- Implementation Guide:
  ```go
  // Use pkg/dsp/synthesis/oscillator.go
  // Use pkg/dsp/synthesis/envelope.go
  // Basic voice allocation system
  // MIDI note on/off handling
  ```

### Phase 4: Multi-Effect Examples 🔜

**8. VocalStrip** - Channel Strip for Vocals
- Demonstrates: Combining multiple processors
- Features:
  - Gate → Compressor → EQ → Limiter chain
  - Each section bypassable
  - Preset management
- Implementation Guide:
  ```go
  // Combine gate, compressor, filter, limiter
  // Show proper gain staging
  // Focus on vocal processing presets
  ```

**9. DrumBus** - Drum Bus Processor
- Demonstrates: Parallel compression, transient shaping
- Features:
  - Parallel compressor with HPF sidechain
  - Transient shaper (expander)
  - Glue compression
  - Mix control
- Implementation Guide:
  ```go
  // Combine compressor and expander
  // Parallel processing architecture
  // Optimized for drum busses
  ```

### Implementation Priority
1. Start with MasterCompressor as it's the most commonly used
2. Then StudioGate to show the gate implementation
3. VintageChorus to demonstrate modulation
4. Continue based on DSP library progress

### Example Plugin Guidelines
- Focus on one primary DSP feature
- Include comprehensive parameter ranges
- Add presets where appropriate
- Include usage comments
- Ensure VST3 validation passes

### Existing Examples Refactoring ✅ DONE
- delay, filter, gain examples updated to use DSP library where applicable

### GUI Support 🔒 DEFERRED

**Status**: GUI implementation (IPlugView) is deferred until manually approved. The framework will remain audio-only until further notice.

## Resources

- [VST3 SDK Documentation](https://steinbergmedia.github.io/vst3_dev_portal/)
- [VST3 Developer Portal](https://steinbergmedia.github.io)
- [VST3 C API Header](./include/vst3/vst3_c_api.h)
- [Example Plugins](./examples/)
- [Architecture Guide](./docs/architecture.md)

---

*This document is the single source of truth for VST3Go development. Update it as features are completed.*
