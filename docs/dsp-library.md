# DSP Library (Layer 3)

## Overview

The DSP Library provides a comprehensive set of audio processing utilities designed for real-time performance. All components follow zero-allocation principles and are optimized for cache efficiency.

## Package Structure

```
pkg/dsp/
├── buffer/      # Buffer operations
├── filter/      # Filter implementations
├── oscillator/  # Signal generators
├── envelope/    # Envelope generators
└── delay/       # Delay lines
```

## Buffer Package (`pkg/dsp/buffer`)

### Purpose
Common buffer operations optimized for audio processing.

### Operations

#### Basic Operations
```go
// Clear zeroes a buffer
func Clear(buffer []float32)

// Copy from source to destination
func Copy(dst, src []float32)

// Add source to destination
func Add(dst, src []float32)

// Scale buffer by constant
func Scale(buffer []float32, scale float32)
```

#### Advanced Operations
```go
// Mix two buffers with crossfade
func Mix(dst, src1, src2 []float32, mix float32)

// Add scaled source to destination
func AddScaled(dst, src []float32, scale float32)

// Find peak absolute value
func Peak(buffer []float32) float32

// Calculate RMS value
func RMS(buffer []float32) float32
```

#### Clipping Operations
```go
// Hard clip to range [-limit, limit]
func Clip(buffer []float32, limit float32)

// Soft saturation clipping
func SoftClip(buffer []float32, threshold float32)
```

### Design Principles
- In-place operations where possible
- No allocations
- Simple loops for SIMD optimization
- Bounds checking in debug mode only

## Filter Package (`pkg/dsp/filter`)

### Biquad Filter

#### Structure
```go
type Biquad struct {
    // Coefficients
    a0, a1, a2 float32
    b0, b1, b2 float32
    
    // State per channel
    x1, x2 []float32
    y1, y2 []float32
}
```

#### Filter Types
```go
// Configure as lowpass
func (b *Biquad) SetLowpass(sampleRate, frequency, q float64)

// Configure as highpass
func (b *Biquad) SetHighpass(sampleRate, frequency, q float64)

// Configure as bandpass
func (b *Biquad) SetBandpass(sampleRate, frequency, q float64)

// Configure as notch
func (b *Biquad) SetNotch(sampleRate, frequency, q float64)

// Configure as peaking EQ
func (b *Biquad) SetPeakingEQ(sampleRate, frequency, q, gainDB float64)

// Configure as shelf
func (b *Biquad) SetLowShelf(sampleRate, frequency, q, gainDB float64)
func (b *Biquad) SetHighShelf(sampleRate, frequency, q, gainDB float64)
```

#### Processing
```go
// Process single channel
func (b *Biquad) Process(buffer []float32, channel int)

// Process multiple channels
func (b *Biquad) ProcessMulti(buffers [][]float32)
```

### State Variable Filter

#### Structure
```go
type SVF struct {
    g float32  // frequency coefficient
    k float32  // damping coefficient (1/Q)
    
    // State per channel
    ic1eq []float32
    ic2eq []float32
}
```

#### Features
- Simultaneous multimode outputs
- Zero-delay feedback topology
- Stable at all frequencies
- Smooth parameter changes

#### Usage
```go
// Set parameters
svf.SetFrequencyAndQ(sampleRate, 1000.0, 2.0)

// Get all outputs
outputs := svf.ProcessSample(input, channel)
// outputs.Lowpass, .Highpass, .Bandpass, .Notch

// Process specific mode
svf.ProcessLowpass(buffer, channel)
```

#### Multi-Mode Filter
```go
type MultiModeSVF struct {
    SVF
    mode float32  // 0-1 morphs between modes
}

// Smooth morphing between filter types
filter.SetMode(0.0)   // Lowpass
filter.SetMode(0.25)  // Bandpass
filter.SetMode(0.5)   // Highpass
filter.SetMode(0.75)  // Notch
```

## Oscillator Package (`pkg/dsp/oscillator`)

### Basic Oscillator

#### Structure
```go
type Oscillator struct {
    sampleRate float64
    frequency  float64
    phase      float64
    phaseInc   float64
}
```

#### Waveforms
```go
// Generate single samples
func (o *Oscillator) Sine() float32
func (o *Oscillator) Saw() float32
func (o *Oscillator) Square() float32
func (o *Oscillator) Triangle() float32
func (o *Oscillator) Pulse(width float64) float32

// Process buffers
func (o *Oscillator) ProcessSine(buffer []float32)
func (o *Oscillator) ProcessSaw(buffer []float32)
// etc...
```

### Band-Limited Oscillators

#### BLIT (Band-Limited Impulse Train)
```go
type BLITOscillator struct {
    sampleRate float64
    frequency  float64
    phase      float64
    m          int     // number of harmonics
}

// Generates alias-free impulses
func (b *BLITOscillator) BLIT() float32
```

#### Band-Limited Sawtooth
```go
type BandLimitedSaw struct {
    blit    *BLITOscillator
    leaky   float64  // integrator coefficient
    state   float64  // integrator state
    dcBlock float64  // DC blocker state
}

// Alias-free sawtooth
func (s *BandLimitedSaw) Process(buffer []float32)
```

### Design Considerations
- Phase accumulators with wrapping
- Pre-calculated increments
- Band limiting for alias prevention
- Efficient phase accumulation

## Envelope Package (`pkg/dsp/envelope`)

### ADSR Envelope

#### Structure
```go
type ADSR struct {
    attack  float64
    decay   float64
    sustain float64
    release float64
    
    // Pre-calculated coefficients
    attackCoef  float64
    decayCoef   float64
    releaseCoef float64
    
    // State
    stage  Stage
    value  float64
    target float64
}
```

#### Usage
```go
// Configure envelope
env.SetADSR(0.01, 0.1, 0.7, 0.3)  // A, D, S, R

// Trigger note on
env.Trigger()

// Trigger note off
env.Release()

// Generate envelope
value := env.Next()

// Process buffer
env.ProcessMultiply(buffer)  // Apply envelope to audio
```

### AR Envelope
Simplified attack-release envelope:
```go
type AR struct {
    attack  float64
    release float64
    active  bool
    value   float64
}
```

### Envelope Follower
For dynamics processing:
```go
type Follower struct {
    attack     float64
    release    float64
    attackCoef float64
    releaseCoef float64
    envelope   float64
}

// Extract envelope from signal
follower.Process(input, output)
```

### Design Features
- Exponential curves for natural sound
- Pre-calculated coefficients
- State machine for stages
- Separate trigger/release

## Delay Package (`pkg/dsp/delay`)

### Basic Delay Line

#### Structure
```go
type Line struct {
    buffer     []float32
    bufferSize int
    writePos   int
    sampleRate float64
}
```

#### Operations
```go
// Write and read
func (d *Line) Write(sample float32)
func (d *Line) Read(delaySamples float64) float32

// Combined operation
func (d *Line) Process(input float32, delaySamples float64) float32

// Buffer processing
func (d *Line) ProcessBuffer(buffer []float32, delaySamples float64)
func (d *Line) ProcessBufferMix(buffer []float32, delaySamples float64, mix float32)
```

### Specialized Delays

#### Allpass Delay
For reverb diffusion:
```go
type AllpassDelay struct {
    Line
    feedback float32
}

func (a *AllpassDelay) Process(input float32, delaySamples float64) float32
```

#### Comb Filter
For reverb and effects:
```go
type CombDelay struct {
    Line
    feedback float32
    damp     float32
    dampVal  float32
}
```

#### Multi-Tap Delay
For complex echo effects:
```go
type TapOutput struct {
    DelaySamples float64
    Gain         float32
    Pan          float32
}

func (m *MultiTapDelay) ProcessMultiTap(
    input float32, 
    taps []TapOutput, 
    outL, outR *float32,
)
```

#### Modulated Delay
For chorus/flanger effects:
```go
type ModulatedDelay struct {
    Line
    lfoPhase   float64
    lfoRate    float64
    lfoDepth   float64
    centerDelay float64
}
```

### Implementation Details
- Circular buffer with wrapping
- Linear interpolation for fractional delays
- Pre-allocated buffers
- Efficient modulo operations

## Performance Optimization

### Memory Layout
- Contiguous arrays for cache efficiency
- Aligned data structures
- Minimal pointer indirection

### Computation
- Pre-calculated coefficients
- Avoided divisions in loops
- Branch-free implementations
- SIMD-friendly operations

### Real-Time Safety
- No allocations after init
- No system calls
- Predictable execution time
- Lock-free where possible

## Usage Examples

### Building a Simple Reverb
```go
type SimpleReverb struct {
    predelay   *delay.Line
    allpass    [4]*delay.AllpassDelay
    comb       [4]*delay.CombDelay
    damping    float32
}

func (r *SimpleReverb) Process(input, output []float32) {
    // Early reflections
    early := r.predelay.Process(input[0], 20.0)
    
    // Diffusion network
    signal := early
    for _, ap := range r.allpass {
        signal = ap.Process(signal, ap.delayTime)
    }
    
    // Parallel combs
    var combOut float32
    for _, c := range r.comb {
        combOut += c.Process(signal, c.delayTime)
    }
    
    output[0] = combOut * 0.25
}
```

### Multi-Band Compressor
```go
type MultiBandCompressor struct {
    splitFilters [2]*filter.Biquad
    followers    [3]*envelope.Follower
    gains        [3]float32
}
```

## Best Practices

1. **Pre-allocate Everything**: All buffers in Initialize()
2. **Avoid Branches**: Use arithmetic instead of if statements
3. **Cache Parameters**: Store frequently used values
4. **Profile First**: Measure before optimizing
5. **Keep It Simple**: Clear code often performs better

## Future Extensions

- SIMD implementations
- More filter types (Moog, Chebyshev)
- FFT-based processing
- Convolution engine
- Physical modeling components