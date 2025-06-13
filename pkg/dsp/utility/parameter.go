// Package utility provides common DSP utility functions and processors.
package utility

import "math"

// ScaleParameter performs linear scaling of a normalized parameter value (0-1) to a target range.
// This is commonly used for VST parameter mapping.
func ScaleParameter(normalized, min, max float64) float64 {
	return min + normalized*(max-min)
}

// ScaleParameterExp performs exponential scaling of a normalized parameter value (0-1) to a target range.
// This is ideal for frequency, time, and other parameters where exponential scaling feels more natural.
func ScaleParameterExp(normalized, min, max float64) float64 {
	if min <= 0 || max <= 0 {
		// Fall back to linear scaling if min or max is non-positive
		return ScaleParameter(normalized, min, max)
	}
	return min * math.Pow(max/min, normalized)
}

// UnscaleParameter performs inverse linear scaling from a target range back to normalized (0-1).
func UnscaleParameter(value, min, max float64) float64 {
	if max == min {
		return 0.0
	}
	return (value - min) / (max - min)
}

// UnscaleParameterExp performs inverse exponential scaling from a target range back to normalized (0-1).
func UnscaleParameterExp(value, min, max float64) float64 {
	if min <= 0 || max <= 0 || max == min {
		// Fall back to linear unscaling
		return UnscaleParameter(value, min, max)
	}
	return math.Log(value/min) / math.Log(max/min)
}

// ClampParameter ensures a parameter value stays within the specified range.
func ClampParameter(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ScaleParameterWithCurve applies a custom curve to the normalized parameter before scaling.
// The curve parameter controls the shape: 1.0 = linear, <1.0 = logarithmic-like, >1.0 = exponential-like
func ScaleParameterWithCurve(normalized, min, max, curve float64) float64 {
	// Apply curve to normalized value
	curved := math.Pow(normalized, curve)
	// Then scale to target range
	return ScaleParameter(curved, min, max)
}

// QuantizeParameter quantizes a parameter value to discrete steps.
// Useful for stepped parameters like octave selectors or discrete modes.
func QuantizeParameter(value float64, steps int) float64 {
	if steps <= 1 {
		return value
	}
	stepSize := 1.0 / float64(steps-1)
	return math.Round(value/stepSize) * stepSize
}

// BipolarToUnipolar converts a bipolar parameter (-1 to 1) to unipolar (0 to 1).
func BipolarToUnipolar(value float64) float64 {
	return (value + 1.0) * 0.5
}

// UnipolarToBipolar converts a unipolar parameter (0 to 1) to bipolar (-1 to 1).
func UnipolarToBipolar(value float64) float64 {
	return value*2.0 - 1.0
}

// SkewParameter applies a skew factor to a normalized parameter.
// Skew > 1 pushes values towards the lower end, skew < 1 pushes towards the upper end.
func SkewParameter(normalized, skew float64) float64 {
	if skew == 1.0 {
		return normalized
	}
	return math.Pow(normalized, 1.0/skew)
}

// SmoothParameter provides parameter smoothing using a simple one-pole filter.
// This helps avoid zipper noise when parameters change.
type SmoothParameter struct {
	current   float64
	target    float64
	smoothing float64
}

// NewSmoothParameter creates a new parameter smoother.
// smoothingTime is in seconds, sampleRate in Hz.
func NewSmoothParameter(smoothingTime, sampleRate float64) *SmoothParameter {
	// Calculate smoothing coefficient
	// Smaller values = more smoothing
	smoothing := 1.0 - math.Exp(-1.0/(smoothingTime*sampleRate))
	return &SmoothParameter{
		smoothing: smoothing,
	}
}

// SetTarget sets the target value for the parameter.
func (s *SmoothParameter) SetTarget(target float64) {
	s.target = target
}

// SetImmediate sets the parameter value immediately without smoothing.
func (s *SmoothParameter) SetImmediate(value float64) {
	s.current = value
	s.target = value
}

// Process returns the next smoothed value.
func (s *SmoothParameter) Process() float64 {
	s.current += (s.target - s.current) * s.smoothing
	return s.current
}

// IsSmoothing returns true if the parameter is still smoothing towards its target.
func (s *SmoothParameter) IsSmoothing() bool {
	const epsilon = 1e-6
	return math.Abs(s.current-s.target) > epsilon
}

// GetCurrent returns the current smoothed value without processing.
func (s *SmoothParameter) GetCurrent() float64 {
	return s.current
}