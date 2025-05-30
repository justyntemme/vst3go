// Package filter provides digital signal processing filters
package filter

import "math"

// SVF implements a state variable filter
// Provides simultaneous lowpass, highpass, bandpass, and notch outputs
// Zero-delay feedback topology for better analog modeling
type SVF struct {
	// Filter parameters
	g float32 // frequency coefficient
	k float32 // damping coefficient (1/Q)

	// State variables (per-channel)
	ic1eq []float32 // integrator 1 state
	ic2eq []float32 // integrator 2 state
}

// SVFOutputs holds all filter outputs
type SVFOutputs struct {
	Lowpass  float32
	Highpass float32
	Bandpass float32
	Notch    float32
}

// NewSVF creates a new state variable filter for the specified number of channels
func NewSVF(channels int) *SVF {
	return &SVF{
		ic1eq: make([]float32, channels),
		ic2eq: make([]float32, channels),
	}
}

// Reset clears the filter state
func (s *SVF) Reset() {
	for i := range s.ic1eq {
		s.ic1eq[i] = 0
		s.ic2eq[i] = 0
	}
}

// SetFrequency sets the filter frequency
func (s *SVF) SetFrequency(sampleRate, frequency float64) {
	// Pre-warp the frequency for the bilinear transform
	omega := math.Tan(math.Pi * frequency / sampleRate)
	s.g = float32(omega)
}

// SetQ sets the filter resonance (Q factor)
func (s *SVF) SetQ(q float64) {
	s.k = float32(1.0 / q)
}

// SetFrequencyAndQ sets both frequency and Q in one call
func (s *SVF) SetFrequencyAndQ(sampleRate, frequency, q float64) {
	s.SetFrequency(sampleRate, frequency)
	s.SetQ(q)
}

// ProcessSample processes a single sample and returns all outputs
func (s *SVF) ProcessSample(input float32, channel int) SVFOutputs {
	// Get state for this channel
	ic1eq := s.ic1eq[channel]
	ic2eq := s.ic2eq[channel]

	// Compute common terms
	g := s.g
	k := s.k
	a1 := 1.0 / (1.0 + g*(g+k))
	a2 := g * a1
	a3 := g * a2

	// Compute outputs
	v3 := input - ic2eq
	v1 := a1*ic1eq + a2*v3
	v2 := ic2eq + a2*ic1eq + a3*v3

	// Update state
	ic1eq = 2.0*v1 - ic1eq
	ic2eq = 2.0*v2 - ic2eq

	// Save state
	s.ic1eq[channel] = ic1eq
	s.ic2eq[channel] = ic2eq

	// Return all outputs
	return SVFOutputs{
		Lowpass:  v2,
		Bandpass: v1,
		Highpass: input - k*v1 - v2,
		Notch:    input - k*v1,
	}
}

// ProcessLowpass processes buffer as lowpass filter - no allocations
func (s *SVF) ProcessLowpass(buffer []float32, channel int) {
	for i := range buffer {
		outputs := s.ProcessSample(buffer[i], channel)
		buffer[i] = outputs.Lowpass
	}
}

// ProcessHighpass processes buffer as highpass filter - no allocations
func (s *SVF) ProcessHighpass(buffer []float32, channel int) {
	for i := range buffer {
		outputs := s.ProcessSample(buffer[i], channel)
		buffer[i] = outputs.Highpass
	}
}

// ProcessBandpass processes buffer as bandpass filter - no allocations
func (s *SVF) ProcessBandpass(buffer []float32, channel int) {
	for i := range buffer {
		outputs := s.ProcessSample(buffer[i], channel)
		buffer[i] = outputs.Bandpass
	}
}

// ProcessNotch processes buffer as notch filter - no allocations
func (s *SVF) ProcessNotch(buffer []float32, channel int) {
	for i := range buffer {
		outputs := s.ProcessSample(buffer[i], channel)
		buffer[i] = outputs.Notch
	}
}

// ProcessMixed processes buffer with a weighted mix of outputs - no allocations
func (s *SVF) ProcessMixed(buffer []float32, channel int, lpMix, hpMix, bpMix, notchMix float32) {
	for i := range buffer {
		outputs := s.ProcessSample(buffer[i], channel)
		buffer[i] = outputs.Lowpass*lpMix +
			outputs.Highpass*hpMix +
			outputs.Bandpass*bpMix +
			outputs.Notch*notchMix
	}
}

// MultiModeSVF implements a multi-mode state variable filter with morphing
// Allows smooth morphing between filter types
type MultiModeSVF struct {
	SVF
	mode float32 // 0=LP, 0.25=BP, 0.5=HP, 0.75=Notch, 1=LP (wraps)
}

// NewMultiModeSVF creates a new multi-mode SVF
func NewMultiModeSVF(channels int) *MultiModeSVF {
	return &MultiModeSVF{
		SVF: SVF{
			ic1eq: make([]float32, channels),
			ic2eq: make([]float32, channels),
		},
		mode: 0.0, // Default to lowpass
	}
}

// SetMode sets the filter mode (0-1, wraps around)
func (m *MultiModeSVF) SetMode(mode float64) {
	m.mode = float32(mode - math.Floor(mode)) // Wrap to 0-1
}

// Process applies the multi-mode filter with morphing - no allocations
func (m *MultiModeSVF) Process(buffer []float32, channel int) {
	mode := m.mode * 4.0 // Scale to 0-4

	// Determine which modes we're between and the mix amount
	var mix float32
	var mode1, mode2 int

	if mode < 1.0 {
		// Between LP and BP
		mode1, mode2 = 0, 1
		mix = mode
	} else if mode < 2.0 {
		// Between BP and HP
		mode1, mode2 = 1, 2
		mix = mode - 1.0
	} else if mode < 3.0 {
		// Between HP and Notch
		mode1, mode2 = 2, 3
		mix = mode - 2.0
	} else {
		// Between Notch and LP
		mode1, mode2 = 3, 0
		mix = mode - 3.0
	}

	// Process with morphing
	for i := range buffer {
		outputs := m.ProcessSample(buffer[i], channel)

		// Get the two filter outputs to morph between
		var out1, out2 float32
		switch mode1 {
		case 0:
			out1 = outputs.Lowpass
		case 1:
			out1 = outputs.Bandpass
		case 2:
			out1 = outputs.Highpass
		case 3:
			out1 = outputs.Notch
		}

		switch mode2 {
		case 0:
			out2 = outputs.Lowpass
		case 1:
			out2 = outputs.Bandpass
		case 2:
			out2 = outputs.Highpass
		case 3:
			out2 = outputs.Notch
		}

		// Linear interpolation between modes
		buffer[i] = out1*(1.0-mix) + out2*mix
	}
}
