# Simple Synthesizer Example

This example demonstrates a basic sine wave synthesizer with MIDI support using the VST3Go framework.

## Features

- **16-voice polyphony** - Play up to 16 notes simultaneously
- **ADSR envelope** - Shape the amplitude of each note
- **MIDI support** - Responds to Note On/Off and sustain pedal
- **Voice stealing** - Automatically handles more than 16 simultaneous notes
- **Parameter automation** - All parameters can be automated by the host

## Parameters

1. **Attack** (0.001 - 2.0 seconds) - Time for the note to reach full volume
2. **Decay** (0.001 - 2.0 seconds) - Time to decay from peak to sustain level  
3. **Sustain** (0 - 100%) - Level held while note is sustained
4. **Release** (0.001 - 5.0 seconds) - Time for the note to fade out after release
5. **Volume** (0 - 100%) - Master output volume

## Architecture

The synthesizer consists of three main components:

### 1. SynthVoice (voice.go)
Each voice contains:
- A sine wave oscillator
- An ADSR envelope generator
- Voice state management

### 2. SimpleSynth (synth.go)
The main plugin class that:
- Manages 16 voices
- Handles MIDI events
- Processes parameter changes
- Mixes voice outputs to stereo

### 3. Voice Allocator
Uses the framework's voice allocator to:
- Assign incoming MIDI notes to voices
- Handle voice stealing when all voices are busy
- Support sustain pedal

## Building

```bash
go build -o simplesynth
```

## Running

```bash
./simplesynth
```

This will run the VST3 validation tests and display plugin information.

## Extending the Synthesizer

This example provides a foundation that can be extended with:

### Additional Oscillators
- Add more waveforms (saw, square, triangle)
- Implement band-limited oscillators
- Add sub-oscillator

### Filters
- Low-pass, high-pass, band-pass filters
- Filter envelope
- Resonance control

### Modulation
- LFOs for vibrato and tremolo
- Pitch bend support
- Modulation wheel routing

### Effects
- Chorus
- Delay
- Reverb

### Advanced Features
- Unison mode
- Portamento/glide
- Additional envelope generators
- Modulation matrix

## Code Structure

The code is organized to be easily extensible:

1. **Voice Interface** - The voice implements the framework's Voice interface, making it compatible with the voice allocator
2. **Parameter Management** - Uses the framework's parameter registry for host automation
3. **MIDI Processing** - Leverages the framework's MIDI event system
4. **Bus Configuration** - Properly configured for instrument plugins (no audio input, stereo output, event input)

## Next Steps

To transform this into the full MegaSynth:
1. Add multiple oscillators per voice
2. Implement filters and filter envelope
3. Add LFOs and modulation routing
4. Implement effects chain
5. Add preset management
6. Optimize for performance