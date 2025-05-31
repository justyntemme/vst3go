package modulation

import (
	"math"
)

// TremoloMode defines the tremolo operation mode
type TremoloMode int

const (
	// TremoloModeNormal applies standard amplitude modulation
	TremoloModeNormal TremoloMode = iota
	// TremoloModeHarmonic adds harmonic richness by using absolute value of LFO
	TremoloModeHarmonic
)

// Tremolo implements an amplitude modulation effect
type Tremolo struct {
	sampleRate float64

	// Parameters
	rate     float64      // LFO rate in Hz
	depth    float64      // Modulation depth (0-1)
	waveform Waveform     // LFO waveform
	mode     TremoloMode  // Tremolo mode
	stereo   bool         // Stereo operation mode
	phase    float64      // Stereo phase offset (0-1)

	// LFOs
	lfoL *LFO
	lfoR *LFO

	// Smoothing for square wave to avoid clicks
	smoothing     bool
	smoothCoeff   float64
	smoothedGainL float64
	smoothedGainR float64
}

// NewTremolo creates a new tremolo effect
func NewTremolo(sampleRate float64) *Tremolo {
	t := &Tremolo{
		sampleRate:    sampleRate,
		rate:          5.0, // 5Hz default
		depth:         0.5, // 50% depth
		waveform:      WaveformSine,
		mode:          TremoloModeNormal,
		stereo:        false,
		phase:         0.0,
		smoothing:     false,
		smoothedGainL: 1.0,
		smoothedGainR: 1.0,
	}

	// Create LFOs
	t.lfoL = NewLFO(sampleRate)
	t.lfoR = NewLFO(sampleRate)
	
	t.updateLFOs()
	t.updateSmoothing()

	return t
}

// SetRate sets the tremolo rate in Hz
func (t *Tremolo) SetRate(hz float64) {
	t.rate = math.Max(0.01, math.Min(20.0, hz))
	t.lfoL.SetFrequency(t.rate)
	t.lfoR.SetFrequency(t.rate)
}

// SetDepth sets the modulation depth (0-1)
func (t *Tremolo) SetDepth(depth float64) {
	t.depth = math.Max(0.0, math.Min(1.0, depth))
}

// SetWaveform sets the LFO waveform
func (t *Tremolo) SetWaveform(waveform Waveform) {
	t.waveform = waveform
	t.lfoL.SetWaveform(waveform)
	t.lfoR.SetWaveform(waveform)
	
	// Enable smoothing for square wave
	t.smoothing = (waveform == WaveformSquare)
	t.updateSmoothing()
}

// SetMode sets the tremolo mode
func (t *Tremolo) SetMode(mode TremoloMode) {
	t.mode = mode
}

// SetStereo enables/disables stereo operation
func (t *Tremolo) SetStereo(stereo bool) {
	t.stereo = stereo
	t.updateLFOs()
}

// SetStereoPhase sets the phase offset between L/R channels (0-1)
func (t *Tremolo) SetStereoPhase(phase float64) {
	t.phase = math.Max(0.0, math.Min(1.0, phase))
	t.updateLFOs()
}

// EnableSmoothing enables/disables smoothing for square wave
func (t *Tremolo) EnableSmoothing(enabled bool) {
	t.smoothing = enabled
}

// updateLFOs updates LFO settings
func (t *Tremolo) updateLFOs() {
	t.lfoL.SetFrequency(t.rate)
	t.lfoL.SetWaveform(t.waveform)
	
	t.lfoR.SetFrequency(t.rate)
	t.lfoR.SetWaveform(t.waveform)
	
	// Set phase offset for stereo
	if t.stereo {
		t.lfoR.SetPhase(t.phase)
	} else {
		t.lfoR.SetPhase(0.0) // Same phase as left
	}
}

// updateSmoothing updates the smoothing coefficient
func (t *Tremolo) updateSmoothing() {
	// Simple one-pole smoothing
	// Time constant of about 5ms
	smoothingTime := 0.005
	t.smoothCoeff = math.Exp(-1.0 / (smoothingTime * t.sampleRate))
}

// Process processes a mono sample
func (t *Tremolo) Process(input float32) float32 {
	// Get LFO value (-1 to 1)
	lfoValue := t.lfoL.Process()
	
	// Calculate gain based on mode
	var gain float64
	
	switch t.mode {
	case TremoloModeNormal:
		// Standard tremolo: LFO modulates between (1-depth) and 1
		// Map LFO from [-1,1] to [1-depth, 1]
		gain = 1.0 - t.depth*(1.0-lfoValue)/2.0
		
	case TremoloModeHarmonic:
		// Harmonic tremolo: use absolute value of LFO for richer harmonics
		// This creates frequency doubling effect
		// abs(LFO) goes from 0 to 1, so we modulate from (1-depth) to 1
		absLFO := math.Abs(lfoValue)
		gain = 1.0 - t.depth*absLFO
	}
	
	// Apply smoothing if enabled (mainly for square wave)
	if t.smoothing {
		t.smoothedGainL = gain + (t.smoothedGainL-gain)*t.smoothCoeff
		gain = t.smoothedGainL
	}
	
	// Apply amplitude modulation
	return input * float32(gain)
}

// ProcessStereo processes stereo input
func (t *Tremolo) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// Get LFO values
	lfoL := t.lfoL.Process()
	lfoR := lfoL // Default to same value
	
	if t.stereo {
		lfoR = t.lfoR.Process()
	}
	
	// Calculate gains based on mode
	var gainL, gainR float64
	
	switch t.mode {
	case TremoloModeNormal:
		gainL = 1.0 - t.depth*(1.0-lfoL)/2.0
		gainR = 1.0 - t.depth*(1.0-lfoR)/2.0
		
	case TremoloModeHarmonic:
		gainL = 1.0 - t.depth*math.Abs(lfoL)
		gainR = 1.0 - t.depth*math.Abs(lfoR)
	}
	
	// Apply smoothing if enabled
	if t.smoothing {
		t.smoothedGainL = gainL + (t.smoothedGainL-gainL)*t.smoothCoeff
		t.smoothedGainR = gainR + (t.smoothedGainR-gainR)*t.smoothCoeff
		gainL = t.smoothedGainL
		gainR = t.smoothedGainR
	}
	
	// Apply amplitude modulation
	outputL = inputL * float32(gainL)
	outputR = inputR * float32(gainR)
	
	return outputL, outputR
}

// ProcessBuffer processes a buffer of samples
func (t *Tremolo) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = t.Process(input[i])
	}
}

// ProcessStereoBuffer processes stereo buffers
func (t *Tremolo) ProcessStereoBuffer(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		outputL[i], outputR[i] = t.ProcessStereo(inputL[i], inputR[i])
	}
}

// GetCurrentGain returns the current gain value (for visualization)
func (t *Tremolo) GetCurrentGain() float64 {
	return t.smoothedGainL
}

// Reset resets the tremolo state
func (t *Tremolo) Reset() {
	t.lfoL.Reset()
	t.lfoR.Reset()
	t.smoothedGainL = 1.0
	t.smoothedGainR = 1.0
	
	// Restore phase offset
	if t.stereo {
		t.lfoR.SetPhase(t.phase)
	}
}