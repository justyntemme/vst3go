package distortion

import (
	"math"
)

// CurveType represents different waveshaping transfer functions
type CurveType int

const (
	// CurveHardClip clips the signal at the threshold
	CurveHardClip CurveType = iota
	// CurveSoftClip applies soft clipping using tanh
	CurveSoftClip
	// CurveSaturate applies exponential saturation
	CurveSaturate
	// CurveFoldback creates wave folding distortion
	CurveFoldback
	// CurveAsymmetric applies different curves to positive and negative values
	CurveAsymmetric
	// CurveSine applies sine waveshaping
	CurveSine
	// CurveExponential applies exponential curve
	CurveExponential
)

// Waveshaper applies waveshaping distortion to audio signals
type Waveshaper struct {
	curveType CurveType
	drive     float64
	mix       float64
	dcOffset  float64
	asymmetry float64 // For asymmetric clipping
}

// NewWaveshaper creates a new waveshaper with the specified curve type
func NewWaveshaper(curveType CurveType) *Waveshaper {
	return &Waveshaper{
		curveType: curveType,
		drive:     1.0,
		mix:       1.0,
		dcOffset:  0.0,
		asymmetry: 0.0,
	}
}

// SetCurveType changes the waveshaping curve
func (w *Waveshaper) SetCurveType(curveType CurveType) {
	w.curveType = curveType
}

// SetDrive sets the distortion amount (typically 1.0 to 20.0)
func (w *Waveshaper) SetDrive(drive float64) {
	w.drive = math.Max(1.0, drive)
}

// SetMix sets the dry/wet mix (0.0 = dry, 1.0 = wet)
func (w *Waveshaper) SetMix(mix float64) {
	w.mix = math.Max(0.0, math.Min(1.0, mix))
}

// SetDCOffset adds a DC offset before waveshaping (for asymmetric distortion)
func (w *Waveshaper) SetDCOffset(offset float64) {
	w.dcOffset = math.Max(-1.0, math.Min(1.0, offset))
}

// SetAsymmetry sets the asymmetry factor for asymmetric clipping
func (w *Waveshaper) SetAsymmetry(asymmetry float64) {
	w.asymmetry = math.Max(-1.0, math.Min(1.0, asymmetry))
}

// Process applies waveshaping to a single sample
func (w *Waveshaper) Process(input float64) float64 {
	// Apply DC offset
	driven := input*w.drive + w.dcOffset

	// Apply waveshaping curve
	var shaped float64
	switch w.curveType {
	case CurveHardClip:
		shaped = w.hardClip(driven)
	case CurveSoftClip:
		shaped = w.softClip(driven)
	case CurveSaturate:
		shaped = w.saturate(driven)
	case CurveFoldback:
		shaped = w.foldback(driven)
	case CurveAsymmetric:
		shaped = w.asymmetric(driven)
	case CurveSine:
		shaped = w.sineShape(driven)
	case CurveExponential:
		shaped = w.exponential(driven)
	default:
		shaped = driven
	}

	// Remove DC offset from output
	shaped -= w.dcOffset

	// Mix with dry signal
	return input*(1.0-w.mix) + shaped*w.mix
}

// ProcessBuffer applies waveshaping to a buffer of samples
func (w *Waveshaper) ProcessBuffer(input, output []float64) {
	n := len(input)
	if len(output) < n {
		n = len(output)
	}

	for i := 0; i < n; i++ {
		output[i] = w.Process(input[i])
	}
}

// hardClip implements hard clipping at ±1.0
func (w *Waveshaper) hardClip(x float64) float64 {
	if x > 1.0 {
		return 1.0
	} else if x < -1.0 {
		return -1.0
	}
	return x
}

// softClip implements soft clipping using tanh
func (w *Waveshaper) softClip(x float64) float64 {
	return math.Tanh(x)
}

// saturate implements exponential saturation
func (w *Waveshaper) saturate(x float64) float64 {
	if x >= 0 {
		return 1.0 - math.Exp(-x)
	}
	return -1.0 + math.Exp(x)
}

// foldback implements wave folding distortion
func (w *Waveshaper) foldback(x float64) float64 {
	// Normalize to 0-4 range for folding
	normalized := (x + 2.0) / 4.0
	
	// Apply folding
	folded := normalized - math.Floor(normalized)
	if int(math.Floor(normalized))%2 == 1 {
		folded = 1.0 - folded
	}
	
	// Scale back to -1 to 1
	return folded*2.0 - 1.0
}

// asymmetric applies different curves to positive and negative values
func (w *Waveshaper) asymmetric(x float64) float64 {
	if x >= 0 {
		// Positive side: soft clipping
		return math.Tanh(x * (1.0 + w.asymmetry))
	}
	// Negative side: harder clipping
	return math.Tanh(x * (1.0 - w.asymmetry))
}

// sineShape applies sine waveshaping
func (w *Waveshaper) sineShape(x float64) float64 {
	// Limit input to prevent aliasing
	x = math.Max(-math.Pi/2, math.Min(math.Pi/2, x))
	return math.Sin(x)
}

// exponential applies exponential curve
func (w *Waveshaper) exponential(x float64) float64 {
	sign := 1.0
	if x < 0 {
		sign = -1.0
		x = -x
	}
	return sign * (1.0 - math.Exp(-x*2.0))
}

// WaveshaperChain allows chaining multiple waveshapers
type WaveshaperChain struct {
	shapers []*Waveshaper
}

// NewWaveshaperChain creates a new chain of waveshapers
func NewWaveshaperChain() *WaveshaperChain {
	return &WaveshaperChain{
		shapers: make([]*Waveshaper, 0),
	}
}

// AddShaper adds a waveshaper to the chain
func (wc *WaveshaperChain) AddShaper(shaper *Waveshaper) {
	wc.shapers = append(wc.shapers, shaper)
}

// Process applies all waveshapers in sequence
func (wc *WaveshaperChain) Process(input float64) float64 {
	output := input
	for _, shaper := range wc.shapers {
		output = shaper.Process(output)
	}
	return output
}

// ProcessBuffer applies the chain to a buffer
func (wc *WaveshaperChain) ProcessBuffer(input, output []float64) {
	n := len(input)
	if len(output) < n {
		n = len(output)
	}

	for i := 0; i < n; i++ {
		output[i] = wc.Process(input[i])
	}
}