package distortion

import (
	"math"
)

type TubeSaturation struct {
	warmth     float64
	harmonics  float64
	bias       float64
	hysteresis float64
	mix        float64
	output     float64
	
	// Internal state for hysteresis
	prevInput  float64
	prevOutput float64
	
	// Pre-emphasis/de-emphasis filters for warmth
	preEmphasisState  float64
	deEmphasisState   float64
}

func NewTubeSaturation() *TubeSaturation {
	return &TubeSaturation{
		warmth:     0.5,
		harmonics:  0.5,
		bias:       0.0,
		hysteresis: 0.1,
		mix:        1.0,
		output:     1.0,
	}
}

func (t *TubeSaturation) SetWarmth(warmth float64) {
	t.warmth = math.Max(0.0, math.Min(1.0, warmth))
}

func (t *TubeSaturation) SetHarmonics(harmonics float64) {
	t.harmonics = math.Max(0.0, math.Min(1.0, harmonics))
}

func (t *TubeSaturation) SetBias(bias float64) {
	t.bias = math.Max(-1.0, math.Min(1.0, bias))
}

func (t *TubeSaturation) SetHysteresis(hysteresis float64) {
	t.hysteresis = math.Max(0.0, math.Min(1.0, hysteresis))
}

func (t *TubeSaturation) SetMix(mix float64) {
	t.mix = math.Max(0.0, math.Min(1.0, mix))
}

func (t *TubeSaturation) SetOutput(output float64) {
	t.output = math.Max(0.0, math.Min(2.0, output))
}

func (t *TubeSaturation) Process(input float64) float64 {
	// Pre-emphasis for warmth (boost highs before saturation)
	emphasized := t.preEmphasis(input)
	
	// Apply tube bias
	biased := emphasized + t.bias*0.1
	
	// Apply hysteresis (magnetic-like behavior)
	withHysteresis := t.applyHysteresis(biased)
	
	// Tube saturation with harmonic generation
	saturated := t.tubeSaturate(withHysteresis)
	
	// De-emphasis (reduce highs after saturation for warmth)
	deEmphasized := t.deEmphasis(saturated)
	
	// Mix with dry signal
	mixed := deEmphasized*t.mix + input*(1.0-t.mix)
	
	return mixed * t.output
}

func (t *TubeSaturation) ProcessBlock(input, output []float64) {
	for i := range input {
		output[i] = t.Process(input[i])
	}
}

func (t *TubeSaturation) ProcessStereo(inputL, inputR, outputL, outputR []float64) {
	for i := range inputL {
		outputL[i] = t.Process(inputL[i])
		outputR[i] = t.Process(inputR[i])
	}
}

func (t *TubeSaturation) tubeSaturate(x float64) float64 {
	// Multiple stages of tube-like saturation
	
	// First stage: soft clipping with even harmonics
	stage1 := t.evenHarmonicSaturation(x)
	
	// Second stage: add odd harmonics based on harmonics parameter
	stage2 := stage1*(1.0-t.harmonics*0.5) + t.oddHarmonicSaturation(x)*t.harmonics*0.5
	
	// Third stage: overall tube compression curve
	compressed := t.tubeCompressionCurve(stage2)
	
	return compressed
}

func (t *TubeSaturation) evenHarmonicSaturation(x float64) float64 {
	// Generate even harmonics (2nd, 4th, etc.) - characteristic of tubes
	// Using asymmetric transfer function
	if x >= 0 {
		return x - 0.1*x*x + 0.05*x*x*x*x
	} else {
		absX := -x
		return -(absX - 0.15*absX*absX + 0.08*absX*absX*absX*absX)
	}
}

func (t *TubeSaturation) oddHarmonicSaturation(x float64) float64 {
	// Generate odd harmonics (3rd, 5th, etc.)
	// Using symmetric transfer function
	return x - 0.15*x*x*x + 0.05*x*x*x*x*x
}

func (t *TubeSaturation) tubeCompressionCurve(x float64) float64 {
	// Smooth compression curve characteristic of tubes
	// Soft knee compression with gradual onset
	threshold := 0.7
	ratio := 3.0
	
	absX := math.Abs(x)
	if absX <= threshold {
		return x
	}
	
	// Soft knee region
	knee := 0.1
	if absX < threshold+knee {
		// Quadratic interpolation in knee region
		factor := (absX - threshold) / knee
		compression := 1.0 + (1.0/ratio-1.0)*factor*factor
		return math.Copysign(threshold+(absX-threshold)*compression, x)
	}
	
	// Above knee: full compression
	compressed := threshold + (absX-threshold)/ratio
	return math.Copysign(compressed, x)
}

func (t *TubeSaturation) applyHysteresis(x float64) float64 {
	// Simple hysteresis model
	diff := x - t.prevInput
	
	// Hysteresis effect based on input change direction
	if diff > 0 {
		// Rising input
		t.prevOutput = t.prevOutput + (x-t.prevOutput)*(1.0-t.hysteresis*0.3)
	} else {
		// Falling input
		t.prevOutput = t.prevOutput + (x-t.prevOutput)*(1.0-t.hysteresis*0.5)
	}
	
	t.prevInput = x
	return t.prevOutput
}

func (t *TubeSaturation) preEmphasis(x float64) float64 {
	// Simple high-frequency boost before saturation
	// First-order high-pass filter
	cutoff := 0.1 + t.warmth*0.4 // Higher warmth = higher cutoff = more pre-emphasis
	
	output := x - t.preEmphasisState
	t.preEmphasisState += output * cutoff
	
	// Mix between filtered and original based on warmth
	return x + output*t.warmth*0.5
}

func (t *TubeSaturation) deEmphasis(x float64) float64 {
	// Simple high-frequency cut after saturation
	// First-order low-pass filter
	cutoff := 0.9 - t.warmth*0.6 // Higher warmth = lower cutoff = more de-emphasis
	
	t.deEmphasisState += (x - t.deEmphasisState) * cutoff
	return t.deEmphasisState
}

func (t *TubeSaturation) Reset() {
	t.prevInput = 0.0
	t.prevOutput = 0.0
	t.preEmphasisState = 0.0
	t.deEmphasisState = 0.0
}