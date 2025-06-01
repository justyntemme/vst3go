# VST3Go Synthesizer Implementation Strategy

## Prerequisites

### Required MIDI Support (Phase 2 from TODO.md)
Before implementing the synthesizer, the following MIDI infrastructure must be completed:

1. **Core MIDI Infrastructure** (`pkg/midi/`)
   - MIDI event types (Note On/Off, CC, Pitch Bend, Aftertouch, Program Change)
   - MIDI event queue with sample-accurate timing
   - MIDI learn system for parameter mapping
   - MPE (MIDI Polyphonic Expression) support

2. **Voice Management** (`pkg/framework/voice/`)
   - Voice allocator with multiple modes (round-robin, oldest, quietest)
   - Voice stealing algorithms
   - Portamento/glide system
   - Unison/detune management

3. **Event Processing Updates** (`pkg/framework/process/`)
   - Event input queue in process.Context
   - Event output queue (for MIDI effects)
   - Sample-accurate event processing

## MegaSynth - Professional Subtractive Synthesizer

A comprehensive synthesizer that demonstrates all VST3Go capabilities and serves as a reference implementation for instrument plugins.

### Architecture Overview

```
MIDI Input → Voice Manager → Voice Processing → Global Effects → Output
                ↓
         [Per Voice:]
         Oscillators → Mix → Filter → Amp → Pan
              ↓         ↓       ↓       ↓
            LFO1     LFO2    ADSR1   ADSR2
```

### Core Components

#### 1. Voice Architecture
```go
type SynthVoice struct {
    // Oscillators
    osc1        *dsp.Oscillator
    osc2        *dsp.Oscillator
    subOsc      *dsp.Oscillator
    noiseGen    *dsp.NoiseGenerator
    
    // Mixing
    osc1Level   float32
    osc2Level   float32
    subLevel    float32
    noiseLevel  float32
    
    // Filter
    filter      *dsp.SVFilter
    filterEnv   *dsp.ADSR
    
    // Amplifier
    ampEnv      *dsp.ADSR
    
    // Modulation
    lfo1        *dsp.LFO
    lfo2        *dsp.LFO
    
    // Voice state
    note        int
    velocity    float32
    active      bool
    releasing   bool
}
```

#### 2. Global Architecture
```go
type MegaSynth struct {
    plugin.Base
    
    // Voice management
    voices      [32]*SynthVoice
    voiceAlloc  *voice.Allocator
    
    // Global modulation
    globalLFO   *dsp.LFO
    
    // Effects chain
    chorus      *dsp.Chorus
    phaser      *dsp.Phaser
    delay       *dsp.Delay
    reverb      *dsp.FDNReverb
    
    // Master section
    compressor  *dsp.Compressor
    limiter     *dsp.Limiter
    
    // Analysis
    spectrum    *dsp.SpectrumAnalyzer
    meters      *dsp.LevelMeter
}
```

### Feature Set

#### Oscillator Section (Per Voice)
- **Oscillator 1 & 2**
  - Waveforms: Sine, Triangle, Saw, Square, Pulse
  - Band-limited with anti-aliasing
  - Octave: -2 to +2
  - Semi: -12 to +12
  - Fine: -100 to +100 cents
  - Phase: 0-360°
  - Pulse width: 5-95% (for pulse wave)
  
- **Sub Oscillator**
  - Sine or Square wave
  - -1 or -2 octaves
  - Level control

- **Noise Generator**
  - White, Pink, Brown noise
  - Level control

#### Filter Section
- **Multi-mode State Variable Filter**
  - Types: Low-pass, High-pass, Band-pass, Notch
  - Cutoff: 20Hz - 20kHz
  - Resonance: 0-100%
  - Key tracking: 0-100%
  - Envelope amount: -100% to +100%
  - Velocity sensitivity: 0-100%

#### Envelope Generators
- **Filter ADSR**
  - Attack: 0-10s
  - Decay: 0-10s
  - Sustain: 0-100%
  - Release: 0-10s
  - Velocity → Attack/Decay modulation

- **Amplitude ADSR**
  - Same parameters as filter
  - Velocity → Level mapping curve

#### LFO Section
- **LFO 1 & 2 (Per Voice)**
  - Waveforms: Sine, Triangle, Square, Saw, S&H
  - Rate: 0.01Hz - 50Hz
  - Sync: Off, 1/16 to 1 bar
  - Phase: 0-360°
  - Fade in: 0-5s
  - Targets: Pitch, Filter, Amp, Pan, PWM

- **Global LFO**
  - Same features but affects all voices
  - Additional target: Effect parameters

#### Modulation Matrix
```go
type ModulationRoute struct {
    Source      ModSource  // LFO1, LFO2, Env1, Env2, Velocity, etc.
    Destination ModDest    // Pitch, Filter, Amp, Pan, etc.
    Amount      float32    // -100% to +100%
    Via         ModSource  // Optional modulation of amount
}
```

#### Effects Section

##### Insert Effects (Per Voice Group)
- **Distortion/Saturation**
  - Types: Soft clip, Hard clip, Tube, Tape
  - Drive: 0-100%
  - Tone: Dark to Bright
  - Mix: 0-100%

##### Send Effects (Global)
- **Chorus** (from dsp.Chorus)
  - Rate, Depth, Delay, Feedback
  - 1-4 voices
  - Stereo spread

- **Phaser** (from dsp.Phaser)
  - 4/6/8 stages
  - Rate, Depth, Feedback
  - Stereo operation

- **Delay** (from dsp.Delay)
  - Sync to tempo
  - Feedback with HP/LP filters
  - Ping-pong mode
  - Modulation

- **Reverb** (from dsp.FDNReverb)
  - Room size
  - Damping
  - Pre-delay
  - Early/Late mix

##### Master Effects
- **Compressor** (from dsp.Compressor)
  - Threshold, Ratio, Attack, Release
  - Makeup gain
  - Mix (parallel compression)

- **Limiter** (from dsp.Limiter)
  - Ceiling level
  - Release time
  - True peak detection

#### Additional Features

##### Performance Controls
- **Pitch Bend**
  - Range: ±2 to ±24 semitones
  - Per-voice or global

- **Modulation Wheel**
  - Assignable to multiple destinations
  - Amount per destination

- **Aftertouch**
  - Channel or polyphonic
  - Assignable destinations

##### Voice Management
- **Polyphony**
  - 1-32 voices
  - Voice stealing modes
  - Priority modes (last, lowest, highest)

- **Unison**
  - 1-8 unison voices
  - Detune spread
  - Stereo spread
  - Analog drift simulation

- **Portamento**
  - Time: 0-1000ms
  - Modes: Always, Legato only
  - Constant time or rate

##### Preset Management
- Factory presets demonstrating capabilities
- User preset storage
- Preset morphing
- Category tagging

### Implementation Phases

#### Phase 1: Basic Voice Engine
1. Single oscillator per voice
2. Simple ADSR for amplitude
3. Basic voice allocation
4. MIDI note on/off handling

#### Phase 2: Full Oscillator Section
1. Multiple oscillators with mixing
2. Sub oscillator and noise
3. Basic modulation (pitch bend, mod wheel)

#### Phase 3: Filter and Envelopes
1. SVF filter implementation
2. Filter envelope
3. Velocity sensitivity
4. Key tracking

#### Phase 4: LFO and Modulation
1. Per-voice LFOs
2. Global LFO
3. Basic modulation routing
4. Sync to host tempo

#### Phase 5: Effects Chain
1. Insert effects (distortion)
2. Send effects (chorus, delay, reverb)
3. Master effects (compressor, limiter)
4. Effect bypass and mix controls

#### Phase 6: Advanced Features
1. Modulation matrix
2. Unison and voice modes
3. MPE support
4. Preset system

#### Phase 7: Polish and Optimization
1. CPU optimization
2. Voice stealing refinement
3. Parameter smoothing
4. Preset library

### Performance Considerations

#### CPU Optimization
- Voice pooling to avoid allocations
- Inactive voice culling
- Conditional processing (bypass unused effects)
- SIMD optimization for filter/oscillator code

#### Memory Management
- Pre-allocated voice pool
- Reusable temporary buffers
- Fixed-size modulation routing table

#### Real-time Safety
- Lock-free voice allocation
- Atomic parameter updates
- No allocations in audio thread

### Example Code Structure

```go
// pkg/examples/megasynth/main.go
package main

import (
    "github.com/justyntemme/vst3go/pkg/framework/plugin"
    "github.com/justyntemme/vst3go/pkg/framework/param"
    "github.com/justyntemme/vst3go/pkg/framework/voice"
    "github.com/justyntemme/vst3go/pkg/dsp/oscillator"
    "github.com/justyntemme/vst3go/pkg/dsp/filter"
    "github.com/justyntemme/vst3go/pkg/dsp/envelope"
)

type MegaSynth struct {
    plugin.Base
    
    voices     []*SynthVoice
    voiceAlloc *voice.Allocator
    params     *param.Registry
}

func (s *MegaSynth) ProcessEvent(event midi.Event) {
    switch e := event.(type) {
    case *midi.NoteOn:
        voice := s.voiceAlloc.Allocate(e.Note)
        voice.Start(e.Note, e.Velocity)
    case *midi.NoteOff:
        s.voiceAlloc.Release(e.Note)
    case *midi.ControlChange:
        s.HandleCC(e.Controller, e.Value)
    }
}

func (s *MegaSynth) ProcessAudio(ctx *process.Context) {
    // Clear output buffers
    ctx.ClearOutputs()
    
    // Process each active voice
    for _, voice := range s.voices {
        if voice.active {
            voice.Process(ctx)
        }
    }
    
    // Apply global effects
    s.ProcessEffects(ctx)
}
```

### Testing Strategy

1. **Unit Tests**
   - Each DSP component in isolation
   - Voice allocation logic
   - Parameter scaling and smoothing

2. **Integration Tests**
   - Full signal path testing
   - MIDI event handling
   - Preset recall

3. **Performance Tests**
   - CPU usage per voice
   - Memory allocation verification
   - Real-time safety validation

4. **Audio Quality Tests**
   - Aliasing measurements
   - Filter response curves
   - Effect quality verification

### Documentation Requirements

1. **User Manual**
   - Parameter descriptions
   - Signal flow diagram
   - Preset descriptions
   - Performance tips

2. **Developer Guide**
   - Architecture overview
   - Extension points
   - DSP algorithm details
   - Optimization techniques

This synthesizer will serve as both a professional instrument and a comprehensive example of VST3Go's capabilities, demonstrating best practices for building complex audio plugins in Go.