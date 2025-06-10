package modulation

import (
	"math"
)

// RingModulator implements a ring modulation effect
type RingModulator struct {
	sampleRate float64

	// Parameters
	frequency float64  // Carrier frequency in Hz
	mix       float64  // Wet/dry mix (0-1)
	waveform  Waveform // Carrier waveform

	// Carrier oscillator
	phase    float64
	phaseInc float64

	// For LFO modulation of carrier frequency
	lfoEnabled bool
	lfoRate    float64
	lfoDepth   float64
	lfo        *LFO
}

// NewRingModulator creates a new ring modulator
func NewRingModulator(sampleRate float64) *RingModulator {
	rm := &RingModulator{
		sampleRate: sampleRate,
		frequency:  440.0, // A4 default
		mix:        0.5,   // 50% wet
		waveform:   WaveformSine,
		phase:      0.0,
		lfoEnabled: false,
		lfoRate:    0.5,
		lfoDepth:   0.1, // 10% frequency modulation
	}

	// Create LFO for frequency modulation
	rm.lfo = NewLFO(sampleRate)
	rm.lfo.SetWaveform(WaveformSine)
	rm.lfo.SetFrequency(rm.lfoRate)

	rm.updatePhaseIncrement()
	return rm
}

// SetFrequency sets the carrier frequency in Hz
func (rm *RingModulator) SetFrequency(hz float64) {
	rm.frequency = math.Max(0.1, math.Min(rm.sampleRate/2, hz))
	rm.updatePhaseIncrement()
}

// SetMix sets the wet/dry mix (0=dry, 1=wet)
func (rm *RingModulator) SetMix(mix float64) {
	rm.mix = math.Max(0.0, math.Min(1.0, mix))
}

// SetWaveform sets the carrier waveform
func (rm *RingModulator) SetWaveform(waveform Waveform) {
	rm.waveform = waveform
}

// EnableLFO enables/disables LFO modulation of carrier frequency
func (rm *RingModulator) EnableLFO(enabled bool) {
	rm.lfoEnabled = enabled
}

// SetLFORate sets the LFO modulation rate in Hz
func (rm *RingModulator) SetLFORate(hz float64) {
	rm.lfoRate = math.Max(0.01, math.Min(20.0, hz))
	rm.lfo.SetFrequency(rm.lfoRate)
}

// SetLFODepth sets the LFO modulation depth (0-1)
func (rm *RingModulator) SetLFODepth(depth float64) {
	rm.lfoDepth = math.Max(0.0, math.Min(1.0, depth))
}

// updatePhaseIncrement updates the phase increment based on frequency
func (rm *RingModulator) updatePhaseIncrement() {
	rm.phaseInc = rm.frequency / rm.sampleRate
}

// generateCarrier generates the carrier signal based on waveform
func (rm *RingModulator) generateCarrier() float64 {
	var carrier float64

	switch rm.waveform {
	case WaveformSine:
		carrier = math.Sin(2.0 * math.Pi * rm.phase)

	case WaveformTriangle:
		if rm.phase < 0.5 {
			carrier = 4.0*rm.phase - 1.0
		} else {
			carrier = 3.0 - 4.0*rm.phase
		}

	case WaveformSquare:
		if rm.phase < 0.5 {
			carrier = 1.0
		} else {
			carrier = -1.0
		}

	case WaveformSawtooth:
		carrier = 2.0*rm.phase - 1.0

	default:
		carrier = math.Sin(2.0 * math.Pi * rm.phase)
	}

	return carrier
}

// Process processes a mono sample
func (rm *RingModulator) Process(input float32) float32 {
	// Apply LFO modulation to frequency if enabled
	if rm.lfoEnabled {
		lfoValue := rm.lfo.Process()
		// Modulate frequency by +/- depth percentage
		modFreq := rm.frequency * (1.0 + lfoValue*rm.lfoDepth)
		rm.phaseInc = modFreq / rm.sampleRate
	}

	// Generate carrier
	carrier := rm.generateCarrier()

	// Ring modulation: multiply input by carrier
	modulated := float64(input) * carrier

	// Mix dry and wet signals
	output := float64(input)*(1-rm.mix) + modulated*rm.mix

	// Advance phase
	rm.phase += rm.phaseInc
	if rm.phase >= 1.0 {
		rm.phase -= 1.0
	}

	return float32(output)
}

// ProcessStereo processes stereo input
func (rm *RingModulator) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// For stereo, we can use the same carrier for both channels
	// This maintains the stereo image while applying the same modulation

	// Apply LFO modulation to frequency if enabled
	if rm.lfoEnabled {
		lfoValue := rm.lfo.Process()
		modFreq := rm.frequency * (1.0 + lfoValue*rm.lfoDepth)
		rm.phaseInc = modFreq / rm.sampleRate
	}

	// Generate carrier
	carrier := rm.generateCarrier()

	// Ring modulation for both channels
	modulatedL := float64(inputL) * carrier
	modulatedR := float64(inputR) * carrier

	// Mix dry and wet signals
	outputL = float32(float64(inputL)*(1-rm.mix) + modulatedL*rm.mix)
	outputR = float32(float64(inputR)*(1-rm.mix) + modulatedR*rm.mix)

	// Advance phase
	rm.phase += rm.phaseInc
	if rm.phase >= 1.0 {
		rm.phase -= 1.0
	}

	return outputL, outputR
}

// ProcessBuffer processes a buffer of samples
func (rm *RingModulator) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = rm.Process(input[i])
	}
}

// ProcessStereoBuffer processes stereo buffers
func (rm *RingModulator) ProcessStereoBuffer(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		outputL[i], outputR[i] = rm.ProcessStereo(inputL[i], inputR[i])
	}
}

// Reset resets the ring modulator state
func (rm *RingModulator) Reset() {
	rm.phase = 0.0
	rm.lfo.Reset()
}
