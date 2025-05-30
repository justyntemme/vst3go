// Package oscillator provides audio oscillators for synthesis
package oscillator

import "math"

// Oscillator generates periodic waveforms
type Oscillator struct {
	sampleRate float64
	frequency  float64
	phase      float64
	phaseInc   float64
}

// New creates a new oscillator
func New(sampleRate float64) *Oscillator {
	return &Oscillator{
		sampleRate: sampleRate,
		frequency:  440.0,
		phase:      0.0,
		phaseInc:   440.0 / sampleRate,
	}
}

// SetFrequency sets the oscillator frequency
func (o *Oscillator) SetFrequency(freq float64) {
	o.frequency = freq
	o.phaseInc = freq / o.sampleRate
}

// SetPhase sets the oscillator phase (0-1)
func (o *Oscillator) SetPhase(phase float64) {
	o.phase = phase - math.Floor(phase) // Wrap to 0-1
}

// Reset resets the oscillator phase to 0
func (o *Oscillator) Reset() {
	o.phase = 0.0
}

// updatePhase advances the phase and wraps it
func (o *Oscillator) updatePhase() {
	o.phase += o.phaseInc
	if o.phase >= 1.0 {
		o.phase -= math.Floor(o.phase)
	}
}

// Sine generates a sine wave sample
func (o *Oscillator) Sine() float32 {
	sample := float32(math.Sin(2.0 * math.Pi * o.phase))
	o.updatePhase()
	return sample
}

// Saw generates a sawtooth wave sample
func (o *Oscillator) Saw() float32 {
	sample := float32(2.0*o.phase - 1.0)
	o.updatePhase()
	return sample
}

// Square generates a square wave sample
func (o *Oscillator) Square() float32 {
	var sample float32
	if o.phase < 0.5 {
		sample = 1.0
	} else {
		sample = -1.0
	}
	o.updatePhase()
	return sample
}

// Pulse generates a pulse wave with variable width
func (o *Oscillator) Pulse(width float64) float32 {
	var sample float32
	if o.phase < width {
		sample = 1.0
	} else {
		sample = -1.0
	}
	o.updatePhase()
	return sample
}

// Triangle generates a triangle wave sample
func (o *Oscillator) Triangle() float32 {
	var sample float32
	if o.phase < 0.5 {
		sample = float32(4.0*o.phase - 1.0)
	} else {
		sample = float32(3.0 - 4.0*o.phase)
	}
	o.updatePhase()
	return sample
}

// ProcessSine fills buffer with sine wave - no allocations
func (o *Oscillator) ProcessSine(buffer []float32) {
	for i := range buffer {
		buffer[i] = o.Sine()
	}
}

// ProcessSaw fills buffer with sawtooth wave - no allocations
func (o *Oscillator) ProcessSaw(buffer []float32) {
	for i := range buffer {
		buffer[i] = o.Saw()
	}
}

// ProcessSquare fills buffer with square wave - no allocations
func (o *Oscillator) ProcessSquare(buffer []float32) {
	for i := range buffer {
		buffer[i] = o.Square()
	}
}

// ProcessPulse fills buffer with pulse wave - no allocations
func (o *Oscillator) ProcessPulse(buffer []float32, width float64) {
	for i := range buffer {
		buffer[i] = o.Pulse(width)
	}
}

// ProcessTriangle fills buffer with triangle wave - no allocations
func (o *Oscillator) ProcessTriangle(buffer []float32) {
	for i := range buffer {
		buffer[i] = o.Triangle()
	}
}

// BLITOscillator implements band-limited impulse train synthesis
// for alias-free waveforms
type BLITOscillator struct {
	sampleRate float64
	frequency  float64
	phase      float64
	phaseInc   float64
	m          int     // number of harmonics
	a          float64 // amplitude scaling factor
}

// NewBLIT creates a new band-limited oscillator
func NewBLIT(sampleRate float64) *BLITOscillator {
	o := &BLITOscillator{
		sampleRate: sampleRate,
		frequency:  440.0,
	}
	o.SetFrequency(440.0)
	return o
}

// SetFrequency sets the oscillator frequency and updates BLIT parameters
func (b *BLITOscillator) SetFrequency(freq float64) {
	b.frequency = freq
	b.phaseInc = freq / b.sampleRate

	// Calculate number of harmonics below Nyquist
	b.m = int(math.Floor(0.5 * b.sampleRate / freq))

	// Calculate amplitude scaling
	b.a = 1.0 / (2.0*float64(b.m) + 1.0)
}

// BLIT generates a band-limited impulse train sample
func (b *BLITOscillator) BLIT() float32 {
	// Generate BLIT using closed-form solution
	phase2pi := 2.0 * math.Pi * b.phase

	var sample float64
	if b.phase < b.phaseInc || b.phase > (1.0-b.phaseInc) {
		// Near discontinuity, use limit value
		sample = b.a
	} else {
		// Normal BLIT formula
		denom := math.Sin(phase2pi)
		if math.Abs(denom) > 1e-10 {
			num := math.Sin(float64(2*b.m+1) * phase2pi)
			sample = b.a * num / denom
		} else {
			sample = b.a
		}
	}

	// Update phase
	b.phase += b.phaseInc
	if b.phase >= 1.0 {
		b.phase -= 1.0
	}

	return float32(sample)
}

// BandLimitedSaw generates an alias-free sawtooth using integrated BLIT
type BandLimitedSaw struct {
	blit    *BLITOscillator
	leaky   float64 // leaky integrator coefficient
	state   float64 // integrator state
	dcBlock float64 // DC blocking filter state
}

// NewBandLimitedSaw creates a new band-limited sawtooth oscillator
func NewBandLimitedSaw(sampleRate float64) *BandLimitedSaw {
	return &BandLimitedSaw{
		blit:  NewBLIT(sampleRate),
		leaky: 0.999, // Slight leak to prevent DC buildup
	}
}

// SetFrequency sets the sawtooth frequency
func (s *BandLimitedSaw) SetFrequency(freq float64) {
	s.blit.SetFrequency(freq)
}

// Next generates the next band-limited sawtooth sample
func (s *BandLimitedSaw) Next() float32 {
	// Get BLIT sample
	blit := float64(s.blit.BLIT())

	// Leaky integration
	s.state = s.leaky*s.state + blit

	// DC blocking
	output := s.state - s.dcBlock
	s.dcBlock = s.state * 0.999

	// Scale to [-1, 1]
	return float32(output * s.blit.phaseInc * 4.0)
}

// Process fills buffer with band-limited sawtooth - no allocations
func (s *BandLimitedSaw) Process(buffer []float32) {
	for i := range buffer {
		buffer[i] = s.Next()
	}
}
