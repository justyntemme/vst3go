package dynamics

import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/envelope"
)

// Limiter implements a brick-wall limiter with optional true peak detection
type Limiter struct {
	sampleRate float64

	// Parameters
	threshold float64 // Ceiling threshold in dB
	release   float64 // Release time in seconds
	lookahead float64 // Lookahead time in seconds
	truePeak  bool    // Enable true peak detection

	// Envelope detection
	detector     *envelope.Detector
	peakDetector *envelope.Detector // For true peak detection

	// Lookahead delay
	delayBuffer  []float32
	delayIndex   int
	delaySamples int

	// True peak oversampling (simple 2x for now)
	lastSample float32

	// State
	gainReduction float64 // Current gain reduction in dB
}

// NewLimiter creates a new brick-wall limiter
func NewLimiter(sampleRate float64) *Limiter {
	l := &Limiter{
		sampleRate:   sampleRate,
		threshold:    -0.3,  // -0.3 dB default ceiling
		release:      0.050, // 50ms default release
		lookahead:    0.005, // 5ms default lookahead
		truePeak:     true,  // True peak detection enabled by default
		detector:     envelope.NewDetector(sampleRate, envelope.ModePeak),
		peakDetector: envelope.NewDetector(sampleRate, envelope.ModePeak),
	}

	// Configure main detector for limiting (very fast attack)
	l.detector.SetType(envelope.TypeLinear)
	l.detector.SetAttack(0.0001) // 0.1ms attack
	l.detector.SetRelease(l.release)

	// Configure peak detector for instant response
	l.peakDetector.SetType(envelope.TypeLinear)
	l.peakDetector.SetAttack(0.0)    // Instant
	l.peakDetector.SetRelease(0.001) // 1ms

	// Initialize lookahead
	l.updateLookahead()

	return l
}

// SetThreshold sets the limiter ceiling in dB
func (l *Limiter) SetThreshold(dB float64) {
	l.threshold = math.Min(0.0, dB) // Can't be positive
}

// SetRelease sets the release time in seconds
func (l *Limiter) SetRelease(seconds float64) {
	l.release = math.Max(0.001, seconds)
	l.detector.SetRelease(l.release)
}

// SetLookahead sets the lookahead time in seconds
func (l *Limiter) SetLookahead(seconds float64) {
	l.lookahead = math.Max(0.0, math.Min(0.010, seconds)) // Max 10ms
	l.updateLookahead()
}

// SetTruePeak enables or disables true peak detection
func (l *Limiter) SetTruePeak(enabled bool) {
	l.truePeak = enabled
}

// updateLookahead updates the lookahead buffer
func (l *Limiter) updateLookahead() {
	newDelaySamples := int(l.lookahead * l.sampleRate)

	if newDelaySamples != l.delaySamples {
		l.delaySamples = newDelaySamples
		if l.delaySamples > 0 {
			l.delayBuffer = make([]float32, l.delaySamples)
			l.delayIndex = 0
		} else {
			l.delayBuffer = nil
		}
	}
}

// GetGainReduction returns the current gain reduction in dB
func (l *Limiter) GetGainReduction() float64 {
	return l.gainReduction
}

// estimateTruePeak estimates the true peak using simple linear interpolation
func (l *Limiter) estimateTruePeak(current float32) float32 {
	// Simple 2x oversampling estimation
	// Interpolate between last and current sample
	midSample := (l.lastSample + current) * 0.5

	// Find peak among last, mid, and current
	peak := float32(math.Max(math.Abs(float64(l.lastSample)), math.Abs(float64(current))))
	peak = float32(math.Max(float64(peak), math.Abs(float64(midSample))))

	l.lastSample = current
	return peak
}

// Process processes a single sample
func (l *Limiter) Process(input float32) float32 {
	// Detection signal (with true peak if enabled)
	detectionSignal := input
	if l.truePeak {
		detectionSignal = l.estimateTruePeak(input)
	}

	// Handle lookahead
	processSignal := input
	if l.delaySamples > 0 && l.delayBuffer != nil {
		// Get delayed signal
		processSignal = l.delayBuffer[l.delayIndex]

		// Store current input
		l.delayBuffer[l.delayIndex] = input
		l.delayIndex = (l.delayIndex + 1) % l.delaySamples

		// For true peak detection in lookahead mode,
		// we need to check the peak of the delayed signal too
		if l.truePeak {
			// Use peak detector for instant response
			detectionSignal = float32(math.Max(float64(detectionSignal),
				math.Abs(float64(l.peakDetector.Detect(processSignal)))))
		}
	}

	// Get envelope
	envelope := l.detector.Detect(detectionSignal)

	// Convert to dB
	inputDB := float64(-96.0)
	if envelope > 0 {
		inputDB = 20.0 * math.Log10(float64(envelope))
	}

	// Calculate gain reduction (infinite ratio)
	gainReductionDB := 0.0
	if inputDB > l.threshold {
		gainReductionDB = inputDB - l.threshold
	}
	l.gainReduction = gainReductionDB

	// Apply gain reduction
	gain := float32(math.Pow(10.0, -gainReductionDB/20.0))

	return processSignal * gain
}

// ProcessBuffer processes a buffer of samples
func (l *Limiter) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = l.Process(input[i])
	}
}

// ProcessStereo processes stereo buffers with linked limiting
func (l *Limiter) ProcessStereo(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		// Get true peak of both channels
		peakL := inputL[i]
		peakR := inputR[i]

		if l.truePeak {
			peakL = l.estimateTruePeak(inputL[i])
			// Need separate true peak estimation for right channel
			// For simplicity, using max of both
			peakR = float32(math.Max(math.Abs(float64(inputR[i])), float64(peakR)))
		}

		// Use maximum for detection
		maxPeak := float32(math.Max(math.Abs(float64(peakL)), math.Abs(float64(peakR))))

		// Process with lookahead
		var processL, processR float32 = inputL[i], inputR[i]
		if l.delaySamples > 0 && l.delayBuffer != nil {
			// We need stereo delay buffers for proper stereo processing
			// For now, using mono detection with stereo processing
			processL = inputL[i]
			processR = inputR[i]
		}

		// Detect from combined peak
		envelope := l.detector.Detect(maxPeak)

		// Calculate limiting
		inputDB := float64(-96.0)
		if envelope > 0 {
			inputDB = 20.0 * math.Log10(float64(envelope))
		}

		gainReductionDB := 0.0
		if inputDB > l.threshold {
			gainReductionDB = inputDB - l.threshold
		}
		l.gainReduction = gainReductionDB

		// Apply same gain to both channels
		gain := float32(math.Pow(10.0, -gainReductionDB/20.0))
		outputL[i] = processL * gain
		outputR[i] = processR * gain
	}
}

// Reset resets the limiter state
func (l *Limiter) Reset() {
	l.detector.Reset()
	l.peakDetector.Reset()
	l.gainReduction = 0.0
	l.lastSample = 0.0
	l.delayIndex = 0

	// Clear delay buffer
	if l.delayBuffer != nil {
		for i := range l.delayBuffer {
			l.delayBuffer[i] = 0
		}
	}
}
