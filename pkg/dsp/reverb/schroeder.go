// Package reverb provides various reverb algorithms
package reverb

import (
	"math"
)

// CombFilter implements a feedback comb filter for reverb
type CombFilter struct {
	buffer      []float32
	bufferSize  int
	bufferIdx   int
	feedback    float64
	filterstore float32
	damp1       float64
	damp2       float64
}

// NewCombFilter creates a new comb filter with the given delay in samples
func NewCombFilter(delaySamples int) *CombFilter {
	return &CombFilter{
		buffer:     make([]float32, delaySamples),
		bufferSize: delaySamples,
		bufferIdx:  0,
		feedback:   0.5,
		damp1:      0.5,
		damp2:      0.5,
	}
}

// SetFeedback sets the feedback amount (0-1)
func (c *CombFilter) SetFeedback(feedback float64) {
	c.feedback = math.Max(0.0, math.Min(1.0, feedback))
}

// SetDamping sets the damping amount (0-1)
func (c *CombFilter) SetDamping(damping float64) {
	c.damp1 = damping
	c.damp2 = 1.0 - damping
}

// Process processes a single sample through the comb filter
func (c *CombFilter) Process(input float32) float32 {
	output := c.buffer[c.bufferIdx]

	// Apply damping (simple lowpass filter)
	c.filterstore = float32(float64(output)*c.damp2 + float64(c.filterstore)*c.damp1)

	// Write to buffer with feedback
	c.buffer[c.bufferIdx] = input + float32(c.feedback)*c.filterstore

	// Advance buffer index
	c.bufferIdx++
	if c.bufferIdx >= c.bufferSize {
		c.bufferIdx = 0
	}

	return output
}

// Reset clears the comb filter state
func (c *CombFilter) Reset() {
	for i := range c.buffer {
		c.buffer[i] = 0
	}
	c.bufferIdx = 0
	c.filterstore = 0
}

// AllPassFilter implements an all-pass filter for reverb diffusion
type AllPassFilter struct {
	buffer     []float32
	bufferSize int
	bufferIdx  int
	feedback   float64
}

// NewAllPassFilter creates a new all-pass filter with the given delay in samples
func NewAllPassFilter(delaySamples int) *AllPassFilter {
	return &AllPassFilter{
		buffer:     make([]float32, delaySamples),
		bufferSize: delaySamples,
		bufferIdx:  0,
		feedback:   0.5,
	}
}

// SetFeedback sets the feedback amount (typically around 0.5)
func (a *AllPassFilter) SetFeedback(feedback float64) {
	a.feedback = feedback
}

// Process processes a single sample through the all-pass filter
func (a *AllPassFilter) Process(input float32) float32 {
	bufout := a.buffer[a.bufferIdx]

	// All-pass filter equation: y[n] = -x[n] + x[n-D] + C * y[n-D]
	// where C is the feedback coefficient
	output := -input + bufout
	a.buffer[a.bufferIdx] = input + float32(a.feedback)*bufout

	// Advance buffer index
	a.bufferIdx++
	if a.bufferIdx >= a.bufferSize {
		a.bufferIdx = 0
	}

	return output
}

// Reset clears the all-pass filter state
func (a *AllPassFilter) Reset() {
	for i := range a.buffer {
		a.buffer[i] = 0
	}
	a.bufferIdx = 0
}

// Schroeder implements the classic Schroeder reverb algorithm
type Schroeder struct {
	sampleRate float64

	// 4 parallel comb filters
	combs [4]*CombFilter

	// 2 series all-pass filters
	allpasses [2]*AllPassFilter

	// Parameters
	roomSize float64 // 0-1
	damping  float64 // 0-1
	wetLevel float64 // 0-1
	dryLevel float64 // 0-1
	width    float64 // 0-1 (stereo width)

	// Tuning parameters (in milliseconds)
	combTunings    [4]float64
	allpassTunings [2]float64
}

// NewSchroeder creates a new Schroeder reverb
func NewSchroeder(sampleRate float64) *Schroeder {
	s := &Schroeder{
		sampleRate: sampleRate,
		roomSize:   0.5,
		damping:    0.5,
		wetLevel:   0.3,
		dryLevel:   0.7,
		width:      1.0,

		// Tuning in milliseconds (these create a good sounding reverb)
		combTunings:    [4]float64{29.7, 37.1, 41.1, 43.7},
		allpassTunings: [2]float64{5.0, 1.7},
	}

	// Create comb filters with different delay times
	for i := 0; i < 4; i++ {
		delaySamples := int(s.combTunings[i] * sampleRate / 1000.0)
		s.combs[i] = NewCombFilter(delaySamples)
	}

	// Create all-pass filters
	for i := 0; i < 2; i++ {
		delaySamples := int(s.allpassTunings[i] * sampleRate / 1000.0)
		s.allpasses[i] = NewAllPassFilter(delaySamples)
		s.allpasses[i].SetFeedback(0.5) // Fixed feedback for all-pass filters
	}

	s.updateInternalParameters()
	return s
}

// SetRoomSize sets the room size (0-1)
func (s *Schroeder) SetRoomSize(size float64) {
	s.roomSize = math.Max(0.0, math.Min(1.0, size))
	s.updateInternalParameters()
}

// SetDamping sets the damping amount (0-1)
func (s *Schroeder) SetDamping(damping float64) {
	s.damping = math.Max(0.0, math.Min(1.0, damping))
	s.updateInternalParameters()
}

// SetWetLevel sets the wet signal level (0-1)
func (s *Schroeder) SetWetLevel(level float64) {
	s.wetLevel = math.Max(0.0, math.Min(1.0, level))
}

// SetDryLevel sets the dry signal level (0-1)
func (s *Schroeder) SetDryLevel(level float64) {
	s.dryLevel = math.Max(0.0, math.Min(1.0, level))
}

// SetWidth sets the stereo width (0-1)
func (s *Schroeder) SetWidth(width float64) {
	s.width = math.Max(0.0, math.Min(1.0, width))
}

// updateInternalParameters updates the internal filter parameters
func (s *Schroeder) updateInternalParameters() {
	// Scale feedback based on room size
	// Larger room = more feedback = longer decay
	feedback := 0.28 + s.roomSize*0.7

	// Update comb filters
	for i := 0; i < 4; i++ {
		s.combs[i].SetFeedback(feedback)
		s.combs[i].SetDamping(s.damping)
	}
}

// Process processes a mono input sample
func (s *Schroeder) Process(input float32) float32 {
	// Sum of comb filters in parallel
	output := float32(0.0)

	// Process through parallel comb filters
	for i := 0; i < 4; i++ {
		output += s.combs[i].Process(input)
	}

	// Average the comb outputs
	output *= 0.25

	// Process through series all-pass filters
	for i := 0; i < 2; i++ {
		output = s.allpasses[i].Process(output)
	}

	// Mix wet and dry signals
	return input*float32(s.dryLevel) + output*float32(s.wetLevel)
}

// ProcessStereo processes stereo input
func (s *Schroeder) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// Mix to mono for reverb processing
	mono := (inputL + inputR) * 0.5

	// Process through reverb
	wet := float32(0.0)

	// Process through parallel comb filters
	// Use different combinations for L/R to create stereo width
	for i := 0; i < 4; i++ {
		wet += s.combs[i].Process(mono)
	}

	// Average the comb outputs
	wet *= 0.25

	// Process through series all-pass filters
	for i := 0; i < 2; i++ {
		wet = s.allpasses[i].Process(wet)
	}

	// Create stereo output with width control
	wetL := wet
	wetR := wet

	// Apply width (simple stereo widening)
	if s.width < 1.0 {
		// Reduce stereo width by mixing channels
		mix := 1.0 - s.width
		wetL = wet * (float32(1.0 - mix*0.5))
		wetR = wet * (float32(1.0 - mix*0.5))
	}

	// Mix wet and dry signals
	outputL = inputL*float32(s.dryLevel) + wetL*float32(s.wetLevel)
	outputR = inputR*float32(s.dryLevel) + wetR*float32(s.wetLevel)

	return outputL, outputR
}

// Reset clears all internal state
func (s *Schroeder) Reset() {
	for i := 0; i < 4; i++ {
		s.combs[i].Reset()
	}
	for i := 0; i < 2; i++ {
		s.allpasses[i].Reset()
	}
}
