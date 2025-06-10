package distortion

import (
	"math"
)

type CurveType int

const (
	CurveHardClip CurveType = iota
	CurveSoftClip
	CurveSaturate
	CurveFoldback
	CurveAsymmetric
	CurveSine
	CurveExponential
)

type Waveshaper struct {
	curveType CurveType
	drive     float64
	mix       float64
	output    float64
	asymmetry float64 // For asymmetric curve
}

func NewWaveshaper() *Waveshaper {
	return &Waveshaper{
		curveType: CurveSoftClip,
		drive:     1.0,
		mix:       1.0,
		output:    1.0,
		asymmetry: 0.0,
	}
}

func (w *Waveshaper) SetCurveType(curve CurveType) {
	w.curveType = curve
}

func (w *Waveshaper) SetDrive(drive float64) {
	w.drive = math.Max(1.0, math.Min(100.0, drive))
}

func (w *Waveshaper) SetMix(mix float64) {
	w.mix = math.Max(0.0, math.Min(1.0, mix))
}

func (w *Waveshaper) SetOutput(output float64) {
	w.output = math.Max(0.0, math.Min(2.0, output))
}

func (w *Waveshaper) SetAsymmetry(asymmetry float64) {
	w.asymmetry = math.Max(-1.0, math.Min(1.0, asymmetry))
}

func (w *Waveshaper) Process(input float64) float64 {
	driven := input * w.drive
	shaped := w.applyCurve(driven)
	return (shaped*w.mix + input*(1.0-w.mix)) * w.output
}

func (w *Waveshaper) ProcessBlock(input, output []float64) {
	for i := range input {
		output[i] = w.Process(input[i])
	}
}

func (w *Waveshaper) ProcessStereo(inputL, inputR, outputL, outputR []float64) {
	for i := range inputL {
		outputL[i] = w.Process(inputL[i])
		outputR[i] = w.Process(inputR[i])
	}
}

func (w *Waveshaper) applyCurve(x float64) float64 {
	switch w.curveType {
	case CurveHardClip:
		return w.hardClip(x)
	case CurveSoftClip:
		return w.softClip(x)
	case CurveSaturate:
		return w.saturate(x)
	case CurveFoldback:
		return w.foldback(x)
	case CurveAsymmetric:
		return w.asymmetric(x)
	case CurveSine:
		return w.sine(x)
	case CurveExponential:
		return w.exponential(x)
	default:
		return x
	}
}

func (w *Waveshaper) hardClip(x float64) float64 {
	if x > 1.0 {
		return 1.0
	} else if x < -1.0 {
		return -1.0
	}
	return x
}

func (w *Waveshaper) softClip(x float64) float64 {
	return math.Tanh(x)
}

func (w *Waveshaper) saturate(x float64) float64 {
	// Tube-like saturation using arctangent
	return (2.0 / math.Pi) * math.Atan(x*math.Pi/2.0)
}

func (w *Waveshaper) foldback(x float64) float64 {
	// Foldback distortion - wraps signal back when it exceeds threshold
	threshold := 1.0
	for x > threshold || x < -threshold {
		if x > threshold {
			x = 2*threshold - x
		} else if x < -threshold {
			x = -2*threshold - x
		}
	}
	return x
}

func (w *Waveshaper) asymmetric(x float64) float64 {
	// Asymmetric clipping - different curves for positive and negative
	if x >= 0 {
		// Positive side: soft saturation
		return math.Tanh(x * (1.0 + w.asymmetry))
	} else {
		// Negative side: harder clipping
		return math.Tanh(x * (1.0 - w.asymmetry))
	}
}

func (w *Waveshaper) sine(x float64) float64 {
	// Sine waveshaping - smooth harmonics
	if math.Abs(x) > 1.0 {
		return math.Copysign(1.0, x)
	}
	return math.Sin(x * math.Pi / 2.0)
}

func (w *Waveshaper) exponential(x float64) float64 {
	// Exponential curve - musical distortion
	sign := math.Copysign(1.0, x)
	absX := math.Abs(x)
	if absX > 1.0 {
		return sign
	}
	return sign * (1.0 - math.Exp(-3.0*absX))
}

// Utility functions for different curve shapes

func Sigmoid(x, k float64) float64 {
	// Ensure k is reasonable to avoid numerical issues
	k = math.Max(0.1, math.Min(10.0, k))
	result := 2.0/(1.0+math.Exp(-k*x)) - 1.0
	// Ensure output is strictly bounded
	return math.Max(-0.999999, math.Min(0.999999, result))
}

func Polynomial(x float64, coeffs []float64) float64 {
	result := 0.0
	xPower := 1.0
	for _, coeff := range coeffs {
		result += coeff * xPower
		xPower *= x
	}
	return result
}

func ChebyshevPolynomial(x float64, order int) float64 {
	if order == 0 {
		return 1.0
	}
	if order == 1 {
		return x
	}
	
	// Use recurrence relation: T_n(x) = 2xT_{n-1}(x) - T_{n-2}(x)
	tn2 := 1.0  // T_0
	tn1 := x    // T_1
	tn := 0.0
	
	for i := 2; i <= order; i++ {
		tn = 2*x*tn1 - tn2
		tn2 = tn1
		tn1 = tn
	}
	
	return tn
}