package modulation

import (
	"math"
)

// Flanger implements a classic flanger effect with feedback
type Flanger struct {
	sampleRate float64

	// Parameters
	rate     float64 // LFO rate in Hz
	depth    float64 // Modulation depth in ms
	delay    float64 // Base delay time in ms (center delay)
	feedback float64 // Feedback amount (-1 to 1, negative for phase inversion)
	mix      float64 // Wet/dry mix (0-1)
	manual   float64 // Manual control for static flanging (0-1)

	// Delay line
	delayLine       []float32
	delayIndex      int
	maxDelaySamples int

	// LFO
	lfo *LFO

	// Feedback state
	feedbackSample float32

	// Manual mode
	manualMode bool
}

// NewFlanger creates a new flanger effect
func NewFlanger(sampleRate float64) *Flanger {
	f := &Flanger{
		sampleRate: sampleRate,
		rate:       0.5, // 0.5 Hz default
		depth:      2.0, // 2ms modulation depth
		delay:      5.0, // 5ms center delay
		feedback:   0.5, // 50% feedback
		mix:        0.5, // 50% wet
		manual:     0.5, // Center position
		manualMode: false,
	}

	// Create LFO
	f.lfo = NewLFO(sampleRate)
	f.lfo.SetWaveform(WaveformTriangle) // Triangle is classic for flangers
	f.lfo.SetFrequency(f.rate)

	// Initialize delay line
	f.updateDelayLine()

	return f
}

// SetRate sets the LFO rate in Hz
func (f *Flanger) SetRate(hz float64) {
	f.rate = math.Max(0.01, math.Min(20.0, hz))
	f.lfo.SetFrequency(f.rate)
}

// SetDepth sets the modulation depth in milliseconds
func (f *Flanger) SetDepth(ms float64) {
	f.depth = math.Max(0.0, math.Min(10.0, ms))
}

// SetDelay sets the center delay time in milliseconds
func (f *Flanger) SetDelay(ms float64) {
	f.delay = math.Max(0.1, math.Min(10.0, ms))
	f.updateDelayLine()
}

// SetFeedback sets the feedback amount (-1 to 1)
func (f *Flanger) SetFeedback(feedback float64) {
	f.feedback = math.Max(-0.99, math.Min(0.99, feedback))
}

// SetMix sets the wet/dry mix (0=dry, 1=wet)
func (f *Flanger) SetMix(mix float64) {
	f.mix = math.Max(0.0, math.Min(1.0, mix))
}

// SetManual sets the manual control position (0-1)
func (f *Flanger) SetManual(position float64) {
	f.manual = math.Max(0.0, math.Min(1.0, position))
}

// SetManualMode enables/disables manual mode
func (f *Flanger) SetManualMode(enabled bool) {
	f.manualMode = enabled
}

// updateDelayLine updates the delay line buffer size
func (f *Flanger) updateDelayLine() {
	// Calculate maximum delay needed
	maxDelayMs := f.delay + f.depth
	f.maxDelaySamples = int(maxDelayMs * f.sampleRate / 1000.0)

	// Add some headroom
	f.maxDelaySamples = int(float64(f.maxDelaySamples) * 1.2)

	// Create new delay line
	f.delayLine = make([]float32, f.maxDelaySamples)
	f.delayIndex = 0
	f.feedbackSample = 0
}

// Process processes a mono sample
func (f *Flanger) Process(input float32) float32 {
	// Mix input with feedback
	delayInput := input + f.feedbackSample*float32(f.feedback)

	// Limit to prevent runaway feedback
	if delayInput > 1.0 {
		delayInput = 1.0
	} else if delayInput < -1.0 {
		delayInput = -1.0
	}

	// Write to delay line
	f.delayLine[f.delayIndex] = delayInput

	// Calculate modulated delay time
	var modulation float64
	if f.manualMode {
		// Use manual control (0-1 mapped to -1 to 1)
		modulation = 2.0*f.manual - 1.0
	} else {
		// Use LFO
		modulation = f.lfo.Process()
	}

	// Calculate delay time in samples
	delayMs := f.delay + f.depth*modulation
	delaySamples := delayMs * f.sampleRate / 1000.0

	// Ensure delay is within bounds
	delaySamples = math.Max(0.1, math.Min(float64(f.maxDelaySamples-1), delaySamples))

	// Calculate read position with linear interpolation
	readPos := float64(f.delayIndex) - delaySamples
	if readPos < 0 {
		readPos += float64(f.maxDelaySamples)
	}

	// Get integer and fractional parts
	readIdx := int(readPos)
	frac := float32(readPos - float64(readIdx))

	// Linear interpolation
	idx1 := readIdx % f.maxDelaySamples
	idx2 := (readIdx + 1) % f.maxDelaySamples
	delayedSample := f.delayLine[idx1]*(1-frac) + f.delayLine[idx2]*frac

	// Store for feedback
	f.feedbackSample = delayedSample

	// Mix dry and wet signals
	output := input*(1-float32(f.mix)) + delayedSample*float32(f.mix)

	// Advance delay index
	f.delayIndex = (f.delayIndex + 1) % f.maxDelaySamples

	return output
}

// ProcessStereo processes stereo input with inverted phase on right channel
func (f *Flanger) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// Process left channel normally
	flangedL := f.Process(inputL)

	// For stereo effect, invert the wet signal on right channel
	// This creates the classic stereo flanger sound

	// Get the wet component from left channel processing
	wetL := flangedL - inputL*(1-float32(f.mix))

	// Create right channel with inverted wet signal
	outputL = flangedL
	outputR = inputR*(1-float32(f.mix)) - wetL*float32(f.mix)

	return outputL, outputR
}

// ProcessBuffer processes a buffer of samples
func (f *Flanger) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = f.Process(input[i])
	}
}

// ProcessStereoBuffer processes stereo buffers
func (f *Flanger) ProcessStereoBuffer(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		outputL[i], outputR[i] = f.ProcessStereo(inputL[i], inputR[i])
	}
}

// Reset resets the flanger state
func (f *Flanger) Reset() {
	// Clear delay line
	for i := range f.delayLine {
		f.delayLine[i] = 0
	}

	// Reset state
	f.delayIndex = 0
	f.feedbackSample = 0

	// Reset LFO
	f.lfo.Reset()
}
