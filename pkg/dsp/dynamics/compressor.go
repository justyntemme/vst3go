// Package dynamics provides dynamics processing effects like compressors, limiters, and gates
package dynamics

import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/envelope"
)

// KneeType defines the compressor knee characteristic
type KneeType int

const (
	// KneeHard provides hard knee compression
	KneeHard KneeType = iota
	// KneeSoft provides soft knee compression
	KneeSoft
)

// Compressor implements a feed-forward compressor with flexible controls
type Compressor struct {
	sampleRate float64

	// Parameters
	threshold  float64  // Threshold in dB
	ratio      float64  // Compression ratio (e.g., 4.0 for 4:1)
	attack     float64  // Attack time in seconds
	release    float64  // Release time in seconds
	kneeWidth  float64  // Knee width in dB (0 for hard knee)
	makeupGain float64  // Makeup gain in dB
	kneeType   KneeType // Knee type
	lookahead  float64  // Lookahead time in seconds

	// Envelope detector
	detector *envelope.Detector

	// Lookahead delay line
	delayBuffer  []float32
	delayIndex   int
	delaySamples int

	// State
	lastGainReduction float64 // For metering
}

// NewCompressor creates a new compressor
func NewCompressor(sampleRate float64) *Compressor {
	c := &Compressor{
		sampleRate: sampleRate,
		threshold:  -20.0, // -20 dB default
		ratio:      4.0,   // 4:1 default
		attack:     0.005, // 5ms default
		release:    0.050, // 50ms default
		kneeWidth:  2.0,   // 2dB soft knee default
		makeupGain: 0.0,
		kneeType:   KneeSoft,
		detector:   envelope.NewDetector(sampleRate, envelope.ModePeak),
	}

	// Configure detector for compressor use
	c.detector.SetType(envelope.TypeLogarithmic) // More musical response
	c.detector.SetTimeConstants(c.attack, c.release)

	return c
}

// SetThreshold sets the compression threshold in dB
func (c *Compressor) SetThreshold(dB float64) {
	c.threshold = dB
}

// SetRatio sets the compression ratio (1.0 = no compression, inf = limiting)
func (c *Compressor) SetRatio(ratio float64) {
	c.ratio = math.Max(1.0, ratio)
}

// SetAttack sets the attack time in seconds
func (c *Compressor) SetAttack(seconds float64) {
	c.attack = math.Max(0.0001, seconds)
	c.detector.SetAttack(c.attack)
}

// SetRelease sets the release time in seconds
func (c *Compressor) SetRelease(seconds float64) {
	c.release = math.Max(0.001, seconds)
	c.detector.SetRelease(c.release)
}

// SetKnee sets the knee type and width
func (c *Compressor) SetKnee(kneeType KneeType, widthDB float64) {
	c.kneeType = kneeType
	c.kneeWidth = math.Max(0.0, widthDB)
}

// SetMakeupGain sets the makeup gain in dB
func (c *Compressor) SetMakeupGain(dB float64) {
	c.makeupGain = dB
}

// SetLookahead sets the lookahead time in seconds (0 to disable)
func (c *Compressor) SetLookahead(seconds float64) {
	c.lookahead = math.Max(0.0, math.Min(0.010, seconds)) // Max 10ms
	newDelaySamples := int(c.lookahead * c.sampleRate)

	// Resize delay buffer if needed
	if newDelaySamples != c.delaySamples {
		c.delaySamples = newDelaySamples
		if c.delaySamples > 0 {
			c.delayBuffer = make([]float32, c.delaySamples)
			c.delayIndex = 0
		} else {
			c.delayBuffer = nil
		}
	}
}

// GetGainReduction returns the current gain reduction in dB (for metering)
func (c *Compressor) GetGainReduction() float64 {
	return c.lastGainReduction
}

// computeGain calculates the gain reduction for a given input level
func (c *Compressor) computeGain(inputDB float64) float64 {
	// Below threshold - knee: no compression
	if inputDB < c.threshold-c.kneeWidth/2 {
		return 0.0
	}

	// Above threshold + knee: full compression
	if inputDB > c.threshold+c.kneeWidth/2 {
		// Gain reduction formula: reduction = (input - threshold) * (1 - 1/ratio)
		return (inputDB - c.threshold) * (1.0 - 1.0/c.ratio)
	}

	// In knee region: interpolate
	if c.kneeType == KneeSoft && c.kneeWidth > 0 {
		// Calculate position in knee (0 to 1)
		kneePos := (inputDB - (c.threshold - c.kneeWidth/2)) / c.kneeWidth

		// Quadratic interpolation for smooth transition
		// At kneePos=0: no compression
		// At kneePos=1: full compression at this level
		compressionRatio := 1.0 - 1.0/c.ratio
		overshoot := inputDB - c.threshold

		// Smooth transition using squared interpolation
		kneeGain := kneePos * kneePos * overshoot * compressionRatio
		return kneeGain
	}

	// Hard knee (shouldn't reach here if knee width is 0)
	return 0.0
}

// Process processes a single sample
func (c *Compressor) Process(input float32) float32 {
	// For lookahead: detect from current input, but apply to delayed signal
	detectionSignal := input
	processSignal := input

	// Handle lookahead delay
	if c.delaySamples > 0 && c.delayBuffer != nil {
		// Get delayed signal for processing
		processSignal = c.delayBuffer[c.delayIndex]

		// Store current input in delay buffer
		c.delayBuffer[c.delayIndex] = input
		c.delayIndex = (c.delayIndex + 1) % c.delaySamples
	}

	// Get envelope of detection signal
	envelope := c.detector.Detect(detectionSignal)

	// Convert to dB
	inputDB := float64(-96.0)
	if envelope > 0 {
		inputDB = 20.0 * math.Log10(float64(envelope))
	}

	// Calculate gain reduction
	gainReductionDB := c.computeGain(inputDB)
	c.lastGainReduction = gainReductionDB

	// Convert gain reduction to linear and apply with makeup gain
	totalGainDB := -gainReductionDB + c.makeupGain
	gain := math.Pow(10.0, totalGainDB/20.0)

	// Apply gain to delayed signal
	return processSignal * float32(gain)
}

// ProcessBuffer processes a buffer of samples
func (c *Compressor) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = c.Process(input[i])
	}
}

// ProcessStereo processes stereo buffers with linked compression
func (c *Compressor) ProcessStereo(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		// Get max of both channels for linked compression
		maxInput := float32(math.Max(math.Abs(float64(inputL[i])), math.Abs(float64(inputR[i]))))

		// Get envelope from combined signal
		envelope := c.detector.Detect(maxInput)

		// Convert to dB
		inputDB := float64(-96.0)
		if envelope > 0 {
			inputDB = 20.0 * math.Log10(float64(envelope))
		}

		// Calculate gain reduction
		gainReductionDB := c.computeGain(inputDB)
		c.lastGainReduction = gainReductionDB

		// Convert to linear gain
		totalGainDB := -gainReductionDB + c.makeupGain
		gain := float32(math.Pow(10.0, totalGainDB/20.0))

		// Apply same gain to both channels
		outputL[i] = inputL[i] * gain
		outputR[i] = inputR[i] * gain
	}
}

// ProcessSidechain processes input using a sidechain signal for detection
func (c *Compressor) ProcessSidechain(input, sidechain, output []float32) {
	for i := range input {
		// Detect from sidechain
		envelope := c.detector.Detect(sidechain[i])

		// Convert to dB
		inputDB := float64(-96.0)
		if envelope > 0 {
			inputDB = 20.0 * math.Log10(float64(envelope))
		}

		// Calculate gain reduction
		gainReductionDB := c.computeGain(inputDB)
		c.lastGainReduction = gainReductionDB

		// Apply to input signal
		totalGainDB := -gainReductionDB + c.makeupGain
		gain := float32(math.Pow(10.0, totalGainDB/20.0))
		output[i] = input[i] * gain
	}
}

// Reset resets the compressor state
func (c *Compressor) Reset() {
	c.detector.Reset()
	c.lastGainReduction = 0.0
	c.delayIndex = 0

	// Clear delay buffer
	if c.delayBuffer != nil {
		for i := range c.delayBuffer {
			c.delayBuffer[i] = 0
		}
	}
}
