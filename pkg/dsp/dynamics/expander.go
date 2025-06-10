package dynamics

import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/envelope"
)

// Expander implements a downward expander for reducing low-level signals
type Expander struct {
	sampleRate float64

	// Parameters
	threshold float64 // Threshold in dB
	ratio     float64 // Expansion ratio (e.g., 2.0 for 2:1)
	attack    float64 // Attack time in seconds
	release   float64 // Release time in seconds
	knee      float64 // Knee width in dB
	range_    float64 // Maximum expansion range in dB

	// Envelope detection
	detector *envelope.Detector

	// Smoothing
	currentGain  float64
	attackCoeff  float64
	releaseCoeff float64

	// State
	gainReduction float64 // Current gain reduction in dB (negative for expansion)
}

// NewExpander creates a new downward expander
func NewExpander(sampleRate float64) *Expander {
	e := &Expander{
		sampleRate:  sampleRate,
		threshold:   -40.0, // -40 dB default
		ratio:       2.0,   // 2:1 default
		attack:      0.001, // 1ms default
		release:     0.100, // 100ms default
		knee:        2.0,   // 2dB soft knee
		range_:      -40.0, // Max 40dB expansion
		currentGain: 1.0,
		detector:    envelope.NewDetector(sampleRate, envelope.ModePeak),
	}

	// Configure detector
	e.detector.SetType(envelope.TypeLogarithmic)
	e.updateTimeConstants()

	return e
}

// SetThreshold sets the expansion threshold in dB
func (e *Expander) SetThreshold(dB float64) {
	e.threshold = dB
}

// SetRatio sets the expansion ratio (1.0 = no expansion)
func (e *Expander) SetRatio(ratio float64) {
	e.ratio = math.Max(1.0, ratio)
}

// SetAttack sets the attack time in seconds
func (e *Expander) SetAttack(seconds float64) {
	e.attack = math.Max(0.0, seconds)
	e.updateTimeConstants()
}

// SetRelease sets the release time in seconds
func (e *Expander) SetRelease(seconds float64) {
	e.release = math.Max(0.0, seconds)
	e.updateTimeConstants()
}

// SetKnee sets the knee width in dB
func (e *Expander) SetKnee(dB float64) {
	e.knee = math.Max(0.0, dB)
}

// SetRange sets the maximum expansion range in dB
func (e *Expander) SetRange(dB float64) {
	e.range_ = math.Min(0.0, dB)
}

// GetGainReduction returns the current gain reduction in dB
func (e *Expander) GetGainReduction() float64 {
	return e.gainReduction
}

// updateTimeConstants updates the attack and release coefficients
func (e *Expander) updateTimeConstants() {
	e.detector.SetAttack(e.attack)
	e.detector.SetRelease(e.release)

	// Smoothing coefficients
	if e.attack > 0 {
		e.attackCoeff = math.Exp(-1.0 / (e.attack * e.sampleRate))
	} else {
		e.attackCoeff = 0.0
	}

	if e.release > 0 {
		e.releaseCoeff = math.Exp(-1.0 / (e.release * e.sampleRate))
	} else {
		e.releaseCoeff = 0.0
	}
}

// computeGain calculates the gain for a given input level
func (e *Expander) computeGain(inputDB float64) float64 {
	// Above threshold: no expansion
	if inputDB > e.threshold+e.knee/2 {
		return 0.0
	}

	// Below threshold - knee: full expansion
	if inputDB < e.threshold-e.knee/2 {
		// Expansion formula: gain = (input - threshold) * (ratio - 1)
		// This gives negative gain (reduction) for signals below threshold
		gain := (inputDB - e.threshold) * (e.ratio - 1.0)

		// Limit to range
		if gain < e.range_ {
			gain = e.range_
		}

		return gain
	}

	// In knee region: interpolate
	if e.knee > 0 {
		// Calculate position in knee (0 to 1)
		// 0 = top of knee (threshold + knee/2), 1 = bottom of knee
		kneePos := ((e.threshold + e.knee/2) - inputDB) / e.knee

		// Calculate full expansion at this level
		fullGain := (inputDB - e.threshold) * (e.ratio - 1.0)

		// Quadratic interpolation
		return kneePos * kneePos * fullGain
	}

	return 0.0
}

// Process processes a single sample
func (e *Expander) Process(input float32) float32 {
	// Get envelope
	envelope := e.detector.Detect(input)

	// Convert to dB
	inputDB := float64(-96.0)
	if envelope > 0 {
		inputDB = 20.0 * math.Log10(float64(envelope))
	}

	// Calculate target gain
	targetGainDB := e.computeGain(inputDB)
	targetGain := math.Pow(10.0, targetGainDB/20.0)

	// Smooth gain changes
	if e.currentGain > targetGain {
		// Decreasing gain (attack - expanding)
		if e.attackCoeff == 0 {
			e.currentGain = targetGain
		} else {
			e.currentGain = targetGain + (e.currentGain-targetGain)*e.attackCoeff
		}
	} else {
		// Increasing gain (release - returning to unity)
		if e.releaseCoeff == 0 {
			e.currentGain = targetGain
		} else {
			e.currentGain = targetGain + (e.currentGain-targetGain)*e.releaseCoeff
		}
	}

	// Update gain reduction for metering
	if e.currentGain < 1.0 {
		e.gainReduction = 20.0 * math.Log10(e.currentGain)
	} else {
		e.gainReduction = 0.0
	}

	// Apply gain
	return input * float32(e.currentGain)
}

// ProcessBuffer processes a buffer of samples
func (e *Expander) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = e.Process(input[i])
	}
}

// ProcessStereo processes stereo buffers with linked expansion
func (e *Expander) ProcessStereo(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		// Use maximum of both channels for detection
		maxInput := float32(math.Max(math.Abs(float64(inputL[i])), math.Abs(float64(inputR[i]))))

		// Get envelope
		envelope := e.detector.Detect(maxInput)

		// Convert to dB
		inputDB := float64(-96.0)
		if envelope > 0 {
			inputDB = 20.0 * math.Log10(float64(envelope))
		}

		// Calculate target gain
		targetGainDB := e.computeGain(inputDB)
		targetGain := math.Pow(10.0, targetGainDB/20.0)

		// Smooth gain changes
		if e.currentGain > targetGain {
			if e.attackCoeff == 0 {
				e.currentGain = targetGain
			} else {
				e.currentGain = targetGain + (e.currentGain-targetGain)*e.attackCoeff
			}
		} else {
			if e.releaseCoeff == 0 {
				e.currentGain = targetGain
			} else {
				e.currentGain = targetGain + (e.currentGain-targetGain)*e.releaseCoeff
			}
		}

		// Update gain reduction
		if e.currentGain < 1.0 {
			e.gainReduction = 20.0 * math.Log10(e.currentGain)
		} else {
			e.gainReduction = 0.0
		}

		// Apply same gain to both channels
		gain := float32(e.currentGain)
		outputL[i] = inputL[i] * gain
		outputR[i] = inputR[i] * gain
	}
}

// Reset resets the expander state
func (e *Expander) Reset() {
	e.detector.Reset()
	e.currentGain = 1.0
	e.gainReduction = 0.0
}
