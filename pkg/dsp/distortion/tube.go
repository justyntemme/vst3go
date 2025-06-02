package distortion

import (
	"math"
)

// TubeSaturator emulates vacuum tube saturation characteristics
type TubeSaturator struct {
	drive          float64
	bias           float64
	mix            float64
	warmth         float64
	evenHarmonics  float64
	oddHarmonics   float64
	lowFreqBoost   float64
	highFreqCut    float64
	
	// Pre-emphasis filters
	preHighpass    *SimpleHighpass
	preLowShelf    *SimpleLowShelf
	postLowpass    *SimpleLowpass
	
	// State for hysteresis
	prevInput      float64
	prevOutput     float64
	hysteresis     float64
}

// NewTubeSaturator creates a new tube saturation processor
func NewTubeSaturator(sampleRate float64) *TubeSaturator {
	return &TubeSaturator{
		drive:         1.0,
		bias:          0.0,
		mix:           1.0,
		warmth:        0.5,
		evenHarmonics: 0.3,
		oddHarmonics:  0.7,
		lowFreqBoost:  1.0,
		highFreqCut:   1.0,
		hysteresis:    0.1,
		
		// Initialize filters
		preHighpass:  NewSimpleHighpass(sampleRate, 20.0),    // Remove DC
		preLowShelf:  NewSimpleLowShelf(sampleRate, 100.0),   // Boost lows
		postLowpass:  NewSimpleLowpass(sampleRate, 15000.0),  // Smooth highs
	}
}

// SetDrive sets the tube drive amount (1.0 to 10.0)
func (t *TubeSaturator) SetDrive(drive float64) {
	t.drive = math.Max(1.0, math.Min(10.0, drive))
}

// SetBias sets the tube bias (-1.0 to 1.0)
func (t *TubeSaturator) SetBias(bias float64) {
	t.bias = math.Max(-1.0, math.Min(1.0, bias))
}

// SetMix sets the dry/wet mix (0.0 = dry, 1.0 = wet)
func (t *TubeSaturator) SetMix(mix float64) {
	t.mix = math.Max(0.0, math.Min(1.0, mix))
}

// SetWarmth controls the amount of low-frequency enhancement (0.0 to 1.0)
func (t *TubeSaturator) SetWarmth(warmth float64) {
	t.warmth = math.Max(0.0, math.Min(1.0, warmth))
	t.preLowShelf.SetGain(1.0 + warmth*0.5) // Up to +3dB boost
}

// SetHarmonicBalance adjusts the balance between even and odd harmonics
func (t *TubeSaturator) SetHarmonicBalance(evenRatio float64) {
	evenRatio = math.Max(0.0, math.Min(1.0, evenRatio))
	t.evenHarmonics = evenRatio
	t.oddHarmonics = 1.0 - evenRatio
}

// SetHysteresis sets the amount of magnetic hysteresis modeling (0.0 to 1.0)
func (t *TubeSaturator) SetHysteresis(hysteresis float64) {
	t.hysteresis = math.Max(0.0, math.Min(1.0, hysteresis))
}

// Process applies tube saturation to a single sample
func (t *TubeSaturator) Process(input float64) float64 {
	// Apply input filtering
	filtered := t.preHighpass.Process(input)
	filtered = t.preLowShelf.Process(filtered)
	
	// Apply drive and bias
	driven := filtered * t.drive
	biased := driven + t.bias
	
	// Apply hysteresis (memory effect)
	if t.hysteresis > 0 {
		diff := biased - t.prevInput
		biased = t.prevInput + diff*(1.0-t.hysteresis*0.5)
		t.prevInput = biased
	}
	
	// Generate harmonics
	saturated := t.generateHarmonics(biased)
	
	// Apply tube transfer function
	shaped := t.tubeTransfer(saturated)
	
	// Post filtering
	output := t.postLowpass.Process(shaped)
	
	// Apply output hysteresis
	if t.hysteresis > 0 {
		diff := output - t.prevOutput
		output = t.prevOutput + diff*(1.0-t.hysteresis*0.3)
		t.prevOutput = output
	}
	
	// Mix with dry signal
	return input*(1.0-t.mix) + output*t.mix
}

// generateHarmonics creates even and odd harmonics characteristic of tubes
func (t *TubeSaturator) generateHarmonics(x float64) float64 {
	// Even harmonics (2nd, 4th, etc.) - create warmth
	x2 := x * x
	x4 := x2 * x2
	even := x + t.evenHarmonics*(0.3*x2 - 0.1*x4)
	
	// Odd harmonics (3rd, 5th, etc.) - create edge
	x3 := x2 * x
	x5 := x3 * x2
	odd := x + t.oddHarmonics*(0.2*x3 - 0.05*x5)
	
	// Blend based on harmonic balance
	return even*t.evenHarmonics + odd*t.oddHarmonics
}

// tubeTransfer implements a tube-like transfer function
func (t *TubeSaturator) tubeTransfer(x float64) float64 {
	// Asymmetric clipping characteristic of tubes
	if x >= 0 {
		// Positive side: softer clipping
		return math.Tanh(x * 0.7) / 0.7
	}
	// Negative side: harder clipping
	return math.Tanh(x * 0.9) / 0.9
}

// ProcessBuffer applies tube saturation to a buffer of samples
func (t *TubeSaturator) ProcessBuffer(input, output []float64) {
	n := len(input)
	if len(output) < n {
		n = len(output)
	}

	for i := 0; i < n; i++ {
		output[i] = t.Process(input[i])
	}
}

// Simple filter implementations for tube saturation

// SimpleHighpass is a basic first-order highpass filter
type SimpleHighpass struct {
	cutoff     float64
	sampleRate float64
	a0, a1     float64
	b1         float64
	x1, y1     float64
}

// NewSimpleHighpass creates a new highpass filter
func NewSimpleHighpass(sampleRate, cutoff float64) *SimpleHighpass {
	hp := &SimpleHighpass{
		cutoff:     cutoff,
		sampleRate: sampleRate,
	}
	hp.updateCoefficients()
	return hp
}

func (hp *SimpleHighpass) updateCoefficients() {
	omega := 2.0 * math.Pi * hp.cutoff / hp.sampleRate
	alpha := math.Sin(omega) / (2.0 * 0.707) // Q = 0.707
	cosw := math.Cos(omega)
	
	norm := 1.0 / (1.0 + alpha)
	hp.a0 = (1.0 + cosw) / 2.0 * norm
	hp.a1 = -(1.0 + cosw) / 2.0 * norm
	hp.b1 = (1.0 - alpha) * norm
}

// Process applies the highpass filter to a sample
func (hp *SimpleHighpass) Process(input float64) float64 {
	output := hp.a0*input + hp.a1*hp.x1 - hp.b1*hp.y1
	hp.x1 = input
	hp.y1 = output
	return output
}

// SimpleLowShelf is a basic low shelf filter
type SimpleLowShelf struct {
	cutoff     float64
	gain       float64
	sampleRate float64
	a0, a1, a2 float64
	b1, b2     float64
	x1, x2     float64
	y1, y2     float64
}

// NewSimpleLowShelf creates a new low shelf filter
func NewSimpleLowShelf(sampleRate, cutoff float64) *SimpleLowShelf {
	ls := &SimpleLowShelf{
		cutoff:     cutoff,
		gain:       1.0,
		sampleRate: sampleRate,
	}
	ls.updateCoefficients()
	return ls
}

// SetGain sets the shelf gain (linear)
func (ls *SimpleLowShelf) SetGain(gain float64) {
	ls.gain = gain
	ls.updateCoefficients()
}

func (ls *SimpleLowShelf) updateCoefficients() {
	A := math.Sqrt(ls.gain)
	omega := 2.0 * math.Pi * ls.cutoff / ls.sampleRate
	sinw := math.Sin(omega)
	cosw := math.Cos(omega)
	
	alpha := sinw / 2.0 * math.Sqrt((A+1.0/A)*(1.0/0.707-1.0)+2.0)
	
	norm := 1.0 / ((A+1.0) + (A-1.0)*cosw + alpha)
	
	ls.a0 = A * ((A+1.0) - (A-1.0)*cosw + alpha) * norm
	ls.a1 = 2.0 * A * ((A-1.0) - (A+1.0)*cosw) * norm
	ls.a2 = A * ((A+1.0) - (A-1.0)*cosw - alpha) * norm
	ls.b1 = -2.0 * ((A-1.0) + (A+1.0)*cosw) * norm
	ls.b2 = ((A+1.0) + (A-1.0)*cosw - alpha) * norm
}

// Process applies the low shelf filter to a sample
func (ls *SimpleLowShelf) Process(input float64) float64 {
	output := ls.a0*input + ls.a1*ls.x1 + ls.a2*ls.x2 - ls.b1*ls.y1 - ls.b2*ls.y2
	ls.x2 = ls.x1
	ls.x1 = input
	ls.y2 = ls.y1
	ls.y1 = output
	return output
}

// SimpleLowpass is a basic first-order lowpass filter
type SimpleLowpass struct {
	cutoff     float64
	sampleRate float64
	a0, a1     float64
	b1         float64
	x1, y1     float64
}

// NewSimpleLowpass creates a new lowpass filter
func NewSimpleLowpass(sampleRate, cutoff float64) *SimpleLowpass {
	lp := &SimpleLowpass{
		cutoff:     cutoff,
		sampleRate: sampleRate,
	}
	lp.updateCoefficients()
	return lp
}

func (lp *SimpleLowpass) updateCoefficients() {
	omega := 2.0 * math.Pi * lp.cutoff / lp.sampleRate
	alpha := math.Sin(omega) / (2.0 * 0.707)
	cosw := math.Cos(omega)
	
	norm := 1.0 / (1.0 + alpha)
	lp.a0 = (1.0 - cosw) / 2.0 * norm
	lp.a1 = (1.0 - cosw) / 2.0 * norm
	lp.b1 = (1.0 - alpha) * norm
}

// Process applies the lowpass filter to a sample
func (lp *SimpleLowpass) Process(input float64) float64 {
	output := lp.a0*input + lp.a1*lp.x1 - lp.b1*lp.y1
	lp.x1 = input
	lp.y1 = output
	return output
}