package modulation

import (
	"math"
)

// AllPassFilter implements a first-order all-pass filter for phaser stages
type AllPassFilter struct {
	a1    float64 // Coefficient
	state float32 // Filter state
}

// NewAllPassFilter creates a new all-pass filter
func NewAllPassFilter() *AllPassFilter {
	return &AllPassFilter{}
}

// SetFrequency sets the filter frequency
func (f *AllPassFilter) SetFrequency(freq, sampleRate float64) {
	// Calculate coefficient for all-pass filter
	// Using bilinear transform: a1 = (1 - tan(pi*fc/fs)) / (1 + tan(pi*fc/fs))
	tanFreq := math.Tan(math.Pi * freq / sampleRate)
	f.a1 = (1.0 - tanFreq) / (1.0 + tanFreq)
}

// Process processes a sample through the all-pass filter
func (f *AllPassFilter) Process(input float32) float32 {
	// First-order all-pass filter
	// y[n] = a1*x[n] + x[n-1] - a1*y[n-1]
	// But we need to store the right state

	// Calculate output
	output := float32(f.a1)*input + f.state

	// Update state for next sample
	// state = x[n] - a1*y[n]
	f.state = input - float32(f.a1)*output

	return output
}

// Reset resets the filter state
func (f *AllPassFilter) Reset() {
	f.state = 0
}

// Phaser implements a classic phaser effect with multiple all-pass filter stages
type Phaser struct {
	sampleRate float64

	// Parameters
	rate       float64 // LFO rate in Hz
	depth      float64 // Modulation depth (0-1)
	centerFreq float64 // Center frequency for modulation
	feedback   float64 // Feedback amount (-1 to 1)
	mix        float64 // Wet/dry mix (0-1)
	stages     int     // Number of all-pass stages (2, 4, 6, or 8)

	// All-pass filter stages
	filters []*AllPassFilter

	// LFO
	lfo *LFO

	// Feedback state
	feedbackSample float32

	// Frequency range for modulation
	minFreq float64
	maxFreq float64
}

// NewPhaser creates a new phaser effect
func NewPhaser(sampleRate float64) *Phaser {
	p := &Phaser{
		sampleRate: sampleRate,
		rate:       0.5,    // 0.5 Hz default
		depth:      0.5,    // 50% depth
		centerFreq: 1000.0, // 1kHz center
		feedback:   0.5,    // 50% feedback
		mix:        0.5,    // 50% wet
		stages:     4,      // 4 stages default
		minFreq:    200.0,  // 200Hz minimum
		maxFreq:    2000.0, // 2kHz maximum
	}

	// Create LFO
	p.lfo = NewLFO(sampleRate)
	p.lfo.SetWaveform(WaveformSine)
	p.lfo.SetFrequency(p.rate)

	// Initialize filters
	p.updateStages()

	return p
}

// SetRate sets the LFO rate in Hz
func (p *Phaser) SetRate(hz float64) {
	p.rate = math.Max(0.01, math.Min(10.0, hz))
	p.lfo.SetFrequency(p.rate)
}

// SetDepth sets the modulation depth (0-1)
func (p *Phaser) SetDepth(depth float64) {
	p.depth = math.Max(0.0, math.Min(1.0, depth))
}

// SetCenterFrequency sets the center frequency for modulation
func (p *Phaser) SetCenterFrequency(freq float64) {
	p.centerFreq = math.Max(100.0, math.Min(4000.0, freq))
	p.updateFrequencyRange()
}

// SetFrequencyRange sets the min and max frequencies for modulation
func (p *Phaser) SetFrequencyRange(minFreq, maxFreq float64) {
	p.minFreq = math.Max(20.0, math.Min(p.sampleRate/4, minFreq))
	p.maxFreq = math.Max(p.minFreq+100, math.Min(p.sampleRate/2, maxFreq))
	p.centerFreq = (p.minFreq + p.maxFreq) / 2
}

// SetFeedback sets the feedback amount (-1 to 1)
func (p *Phaser) SetFeedback(feedback float64) {
	p.feedback = math.Max(-0.99, math.Min(0.99, feedback))
}

// SetMix sets the wet/dry mix (0=dry, 1=wet)
func (p *Phaser) SetMix(mix float64) {
	p.mix = math.Max(0.0, math.Min(1.0, mix))
}

// SetStages sets the number of all-pass stages (2, 4, 6, or 8)
func (p *Phaser) SetStages(stages int) {
	// Limit to even numbers between 2 and 8
	if stages < 2 {
		stages = 2
	} else if stages > 8 {
		stages = 8
	} else if stages%2 != 0 {
		stages = stages - 1 // Make even
	}

	p.stages = stages
	p.updateStages()
}

// updateStages creates the all-pass filter stages
func (p *Phaser) updateStages() {
	p.filters = make([]*AllPassFilter, p.stages)
	for i := 0; i < p.stages; i++ {
		p.filters[i] = NewAllPassFilter()
	}
	p.updateFrequencyRange()
}

// updateFrequencyRange updates the frequency range based on center and depth
func (p *Phaser) updateFrequencyRange() {
	// Calculate frequency range based on center frequency and depth
	freqRange := p.centerFreq * p.depth
	p.minFreq = p.centerFreq - freqRange/2
	p.maxFreq = p.centerFreq + freqRange/2

	// Ensure valid range
	p.minFreq = math.Max(20.0, p.minFreq)
	p.maxFreq = math.Min(p.sampleRate/4, p.maxFreq)
}

// Process processes a mono sample
func (p *Phaser) Process(input float32) float32 {
	// Get LFO modulation (-1 to 1)
	lfoValue := p.lfo.Process()

	// Map LFO to frequency range
	// Use exponential scaling for more musical response
	normalizedLFO := (lfoValue + 1.0) / 2.0 // 0 to 1
	logMin := math.Log(p.minFreq)
	logMax := math.Log(p.maxFreq)
	logFreq := logMin + (logMax-logMin)*normalizedLFO
	freq := math.Exp(logFreq)

	// Update all-pass filter frequencies
	for _, filter := range p.filters {
		filter.SetFrequency(freq, p.sampleRate)
	}

	// Process through all-pass cascade with feedback
	wetSignal := input + p.feedbackSample*float32(p.feedback)

	// Limit to prevent runaway feedback
	if wetSignal > 1.0 {
		wetSignal = 1.0
	} else if wetSignal < -1.0 {
		wetSignal = -1.0
	}

	// Process through all-pass stages
	for _, filter := range p.filters {
		wetSignal = filter.Process(wetSignal)
	}

	// Store for feedback
	p.feedbackSample = wetSignal

	// Mix dry and wet signals
	output := input*float32(1-p.mix) + wetSignal*float32(p.mix)

	return output
}

// ProcessStereo processes stereo input with phase-shifted modulation
func (p *Phaser) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// For stereo, we could use two separate phasers with phase-shifted LFOs
	// For now, use the same processing but with inverted wet signal on right
	outputL = p.Process(inputL)

	// Get the wet component
	wetL := outputL - inputL*float32(1-p.mix)

	// Create right channel with slightly different phasing
	// Process right input normally but invert the wet signal
	dryR := inputR * float32(1-p.mix)
	outputR = dryR - wetL // Inverted wet creates stereo width

	return outputL, outputR
}

// ProcessBuffer processes a buffer of samples
func (p *Phaser) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = p.Process(input[i])
	}
}

// ProcessStereoBuffer processes stereo buffers
func (p *Phaser) ProcessStereoBuffer(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		outputL[i], outputR[i] = p.ProcessStereo(inputL[i], inputR[i])
	}
}

// Reset resets the phaser state
func (p *Phaser) Reset() {
	// Reset all filters
	for _, filter := range p.filters {
		filter.Reset()
	}

	// Reset state
	p.feedbackSample = 0

	// Reset LFO
	p.lfo.Reset()
}
