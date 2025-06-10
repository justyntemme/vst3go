// Package envelope provides envelope generators and detectors for audio synthesis and dynamics processing
package envelope

import (
	"math"
)

// DetectorMode defines the envelope detection mode
type DetectorMode int

const (
	// ModePeak detects the peak level
	ModePeak DetectorMode = iota
	// ModeRMS detects the RMS (Root Mean Square) level
	ModeRMS
	// ModePeakHold detects peak with hold time
	ModePeakHold
)

// DetectorType defines the envelope detector response type
type DetectorType int

const (
	// TypeLinear uses linear envelope detection
	TypeLinear DetectorType = iota
	// TypeLogarithmic uses logarithmic envelope detection (better for audio perception)
	TypeLogarithmic
	// TypeAnalog simulates analog envelope behavior
	TypeAnalog
)

// Detector implements an advanced envelope detector for dynamics processing
type Detector struct {
	sampleRate float64
	mode       DetectorMode
	detType    DetectorType

	// Time constants
	attack  float64 // Attack time in seconds
	release float64 // Release time in seconds
	hold    float64 // Hold time in seconds (for peak hold mode)

	// Coefficients (pre-calculated)
	attackCoef  float64
	releaseCoef float64

	// State
	envelope    float64
	holdCounter int

	// RMS window
	rmsWindow    []float64
	rmsIndex     int
	rmsSum       float64
	rmsWindowLen int
}

// NewDetector creates a new envelope detector
func NewDetector(sampleRate float64, mode DetectorMode) *Detector {
	d := &Detector{
		sampleRate:   sampleRate,
		mode:         mode,
		detType:      TypeLinear,
		attack:       0.001,                   // 1ms default
		release:      0.100,                   // 100ms default
		hold:         0.010,                   // 10ms default
		rmsWindowLen: int(sampleRate * 0.003), // 3ms RMS window
		envelope:     0.0,
		holdCounter:  0,
	}

	// Initialize RMS window if needed
	if mode == ModeRMS {
		d.rmsWindow = make([]float64, d.rmsWindowLen)
	}

	d.updateCoefficients()
	return d
}

// SetMode sets the detection mode
func (d *Detector) SetMode(mode DetectorMode) {
	d.mode = mode

	// Initialize RMS window if switching to RMS mode
	if mode == ModeRMS && d.rmsWindow == nil {
		d.rmsWindow = make([]float64, d.rmsWindowLen)
		d.rmsIndex = 0
		d.rmsSum = 0
	}
}

// SetType sets the detector response type
func (d *Detector) SetType(detType DetectorType) {
	d.detType = detType
	d.updateCoefficients()
}

// SetAttack sets the attack time in seconds
func (d *Detector) SetAttack(seconds float64) {
	d.attack = math.Max(0.0001, seconds)
	d.updateCoefficients()
}

// SetRelease sets the release time in seconds
func (d *Detector) SetRelease(seconds float64) {
	d.release = math.Max(0.0001, seconds)
	d.updateCoefficients()
}

// SetHold sets the hold time in seconds (for peak hold mode)
func (d *Detector) SetHold(seconds float64) {
	d.hold = math.Max(0.0, seconds)
}

// SetTimeConstants sets attack and release times together
func (d *Detector) SetTimeConstants(attack, release float64) {
	d.attack = math.Max(0.0001, attack)
	d.release = math.Max(0.0001, release)
	d.updateCoefficients()
}

// SetRMSWindow sets the RMS window length in milliseconds
func (d *Detector) SetRMSWindow(ms float64) {
	newLen := int(d.sampleRate * ms / 1000.0)
	if newLen < 1 {
		newLen = 1
	}

	if newLen != d.rmsWindowLen {
		d.rmsWindowLen = newLen
		d.rmsWindow = make([]float64, d.rmsWindowLen)
		d.rmsIndex = 0
		d.rmsSum = 0
	}
}

// updateCoefficients recalculates the envelope coefficients
func (d *Detector) updateCoefficients() {
	switch d.detType {
	case TypeLinear:
		// Linear coefficients - for one-pole filter approach
		d.attackCoef = 1.0 - math.Exp(-1.0/(d.attack*d.sampleRate))
		d.releaseCoef = 1.0 - math.Exp(-1.0/(d.release*d.sampleRate))

	case TypeLogarithmic:
		// Logarithmic coefficients (more musical) - faster attack
		d.attackCoef = 1.0 - math.Exp(-2.2/(d.attack*d.sampleRate))
		d.releaseCoef = 1.0 - math.Exp(-2.2/(d.release*d.sampleRate))

	case TypeAnalog:
		// Analog-style coefficients - exponential decay
		d.attackCoef = math.Exp(-1.0 / (d.attack * d.sampleRate))
		d.releaseCoef = math.Exp(-1.0 / (d.release * d.sampleRate))
	}
}

// Detect processes a single sample and returns the envelope value
func (d *Detector) Detect(input float32) float32 {
	var inputLevel float64

	// Get input level based on mode
	switch d.mode {
	case ModePeak, ModePeakHold:
		// Peak detection - just absolute value
		inputLevel = math.Abs(float64(input))

	case ModeRMS:
		// RMS detection - square the input
		squared := float64(input) * float64(input)

		// Update RMS window
		oldValue := d.rmsWindow[d.rmsIndex]
		d.rmsWindow[d.rmsIndex] = squared
		d.rmsSum += squared - oldValue
		d.rmsIndex = (d.rmsIndex + 1) % d.rmsWindowLen

		// Calculate RMS
		meanSquare := d.rmsSum / float64(d.rmsWindowLen)
		inputLevel = math.Sqrt(meanSquare)
	}

	// Apply envelope detection based on type
	switch d.detType {
	case TypeLinear, TypeLogarithmic:
		if inputLevel > d.envelope {
			// Attack - rise towards input level
			d.envelope += (inputLevel - d.envelope) * d.attackCoef
			// For peak mode with instantaneous peaks, capture them immediately
			if d.mode == ModePeak || d.mode == ModePeakHold {
				// If attack time is very short or input is significantly higher, jump to peak
				if d.attackCoef > 0.5 || inputLevel > d.envelope*2.0 {
					d.envelope = inputLevel
				}
			}
			d.holdCounter = int(d.hold * d.sampleRate) // Reset hold counter
		} else {
			// Release (with hold for peak hold mode)
			if d.mode == ModePeakHold && d.holdCounter > 0 {
				d.holdCounter--
			} else {
				d.envelope += (inputLevel - d.envelope) * d.releaseCoef
			}
		}

	case TypeAnalog:
		// Analog-style envelope (using coefficients differently)
		if inputLevel > d.envelope {
			d.envelope = inputLevel + (d.envelope-inputLevel)*d.attackCoef
			d.holdCounter = int(d.hold * d.sampleRate)
		} else {
			if d.mode == ModePeakHold && d.holdCounter > 0 {
				d.holdCounter--
			} else {
				d.envelope = inputLevel + (d.envelope-inputLevel)*d.releaseCoef
			}
		}
	}

	return float32(d.envelope)
}

// Process processes a buffer of samples and fills output with envelope values
func (d *Detector) Process(input, output []float32) {
	for i := range input {
		output[i] = d.Detect(input[i])
	}
}

// ProcessSidechain processes input using sidechain signal for detection
func (d *Detector) ProcessSidechain(input, sidechain, output []float32) {
	for i := range input {
		// Detect envelope from sidechain
		envelope := d.Detect(sidechain[i])
		// Apply to input (could be used for compression ratio later)
		output[i] = envelope
	}
}

// GetEnvelope returns the current envelope value
func (d *Detector) GetEnvelope() float32 {
	return float32(d.envelope)
}

// GetEnvelopeDB returns the current envelope value in decibels
func (d *Detector) GetEnvelopeDB() float32 {
	if d.envelope <= 0 {
		return -96.0 // Minimum dB
	}
	return float32(20.0 * math.Log10(d.envelope))
}

// Reset resets the detector state
func (d *Detector) Reset() {
	d.envelope = 0
	d.holdCounter = 0
	if d.rmsWindow != nil {
		for i := range d.rmsWindow {
			d.rmsWindow[i] = 0
		}
		d.rmsSum = 0
		d.rmsIndex = 0
	}
}
