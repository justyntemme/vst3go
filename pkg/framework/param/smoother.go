// Package param provides parameter management for VST3 plugins.
package param

import (
	"math"
)

// SmoothingType defines different parameter smoothing algorithms.
type SmoothingType int

const (
	// LinearSmoothing uses linear interpolation
	LinearSmoothing SmoothingType = iota
	// ExponentialSmoothing uses exponential smoothing (one-pole filter)
	ExponentialSmoothing
	// LogarithmicSmoothing uses logarithmic smoothing (better for frequency parameters)
	LogarithmicSmoothing
)

// Smoother provides parameter smoothing to prevent zipper noise.
type Smoother struct {
	smoothingType SmoothingType
	current       float64
	target        float64
	rate          float64
	threshold     float64
	isSmoothing   bool
	
	// For linear smoothing
	step float64
	
	// For logarithmic smoothing
	logCurrent float64
	logTarget  float64
	logStep    float64
}

// NewSmoother creates a new parameter smoother.
// rate: smoothing rate (0.9-0.999 for exponential, samples for linear)
func NewSmoother(smoothingType SmoothingType, rate float64) *Smoother {
	return &Smoother{
		smoothingType: smoothingType,
		rate:          rate,
		threshold:     0.0001,
	}
}

// SetTarget sets the target value for smoothing.
func (s *Smoother) SetTarget(target float64) {
	if math.Abs(target-s.target) < s.threshold {
		return // Target hasn't changed significantly
	}
	
	s.target = target
	s.isSmoothing = true
	
	switch s.smoothingType {
	case LinearSmoothing:
		// Calculate step size for linear smoothing
		if s.rate > 0 {
			s.step = (target - s.current) / s.rate
		}
		
	case LogarithmicSmoothing:
		// Convert to log space for frequency parameters
		const minVal = 0.001
		currentVal := s.current
		targetVal := target
		
		if currentVal < minVal {
			currentVal = minVal
		}
		if targetVal < minVal {
			targetVal = minVal
		}
		
		s.logCurrent = math.Log(currentVal)
		s.logTarget = math.Log(targetVal)
		
		if s.rate > 0 {
			s.logStep = (s.logTarget - s.logCurrent) / s.rate
		}
	}
}

// Next returns the next smoothed value.
func (s *Smoother) Next() float64 {
	if !s.isSmoothing {
		return s.current
	}
	
	switch s.smoothingType {
	case ExponentialSmoothing:
		// One-pole filter: y = y + a * (x - y)
		s.current += (s.target - s.current) * (1.0 - s.rate)
		
		// Check if we've reached the target
		if math.Abs(s.current-s.target) < s.threshold {
			s.current = s.target
			s.isSmoothing = false
		}
		
	case LinearSmoothing:
		// Linear interpolation
		s.current += s.step
		
		// Check if we've reached or passed the target
		if (s.step > 0 && s.current >= s.target) || (s.step < 0 && s.current <= s.target) {
			s.current = s.target
			s.isSmoothing = false
		}
		
	case LogarithmicSmoothing:
		// Interpolate in log space
		s.logCurrent += s.logStep
		
		// Check if we've reached the target
		if (s.logStep > 0 && s.logCurrent >= s.logTarget) || (s.logStep < 0 && s.logCurrent <= s.logTarget) {
			s.current = s.target
			s.isSmoothing = false
		} else {
			s.current = math.Exp(s.logCurrent)
		}
	}
	
	return s.current
}

// Process processes a buffer with the smoothed parameter.
// The callback receives the current smoothed value for each sample.
func (s *Smoother) Process(buffer []float32, callback func(value float64, sample float32) float32) {
	for i := range buffer {
		value := s.Next()
		buffer[i] = callback(value, buffer[i])
	}
}

// IsSmoothing returns true if the smoother is currently smoothing.
func (s *Smoother) IsSmoothing() bool {
	return s.isSmoothing
}

// Reset resets the smoother to a specific value.
func (s *Smoother) Reset(value float64) {
	s.current = value
	s.target = value
	s.isSmoothing = false
}

// SetRate updates the smoothing rate.
func (s *Smoother) SetRate(rate float64) {
	s.rate = rate
}

// SetThreshold sets the threshold for considering smoothing complete.
func (s *Smoother) SetThreshold(threshold float64) {
	s.threshold = threshold
}

// SmoothedParameter wraps a Parameter with smoothing capability.
type SmoothedParameter struct {
	*Parameter
	smoother      *Smoother
	smoothingRate float64
	enabled       bool
}

// NewSmoothedParameter creates a parameter with built-in smoothing.
func NewSmoothedParameter(param *Parameter, smoothingType SmoothingType, rate float64) *SmoothedParameter {
	sp := &SmoothedParameter{
		Parameter:     param,
		smoother:      NewSmoother(smoothingType, rate),
		smoothingRate: rate,
		enabled:       true,
	}
	
	// Initialize smoother with current parameter value
	sp.smoother.Reset(param.GetPlainValue())
	
	return sp
}

// SetValue sets the parameter value and updates the smoother target.
func (sp *SmoothedParameter) SetValue(value float64) {
	sp.Parameter.SetValue(value)
	if sp.enabled {
		sp.smoother.SetTarget(sp.GetPlainValue())
	}
}

// GetSmoothedValue returns the current smoothed value.
func (sp *SmoothedParameter) GetSmoothedValue() float64 {
	if sp.enabled {
		return sp.smoother.Next()
	}
	return sp.GetPlainValue()
}

// SetSmoothing enables or disables smoothing.
func (sp *SmoothedParameter) SetSmoothing(enabled bool) {
	sp.enabled = enabled
	if !enabled {
		sp.smoother.Reset(sp.GetPlainValue())
	}
}

// SetSmoothingRate updates the smoothing rate.
func (sp *SmoothedParameter) SetSmoothingRate(rate float64) {
	sp.smoothingRate = rate
	sp.smoother.SetRate(rate)
}

// Update smoothing rate based on sample rate for time-based smoothing.
func (sp *SmoothedParameter) UpdateSampleRate(sampleRate float64, targetTimeMs float64) {
	if sp.smoother.smoothingType == LinearSmoothing {
		// Convert time to samples
		samples := sampleRate * targetTimeMs / 1000.0
		sp.SetSmoothingRate(samples)
	} else if sp.smoother.smoothingType == ExponentialSmoothing {
		// Calculate coefficient for target time
		// -60dB in targetTimeMs
		sp.SetSmoothingRate(math.Exp(-6.908 / (sampleRate * targetTimeMs / 1000.0)))
	}
}

// ParameterSmoother manages smoothing for multiple parameters.
type ParameterSmoother struct {
	smoothers map[uint32]*SmoothedParameter
}

// NewParameterSmoother creates a new parameter smoother manager.
func NewParameterSmoother() *ParameterSmoother {
	return &ParameterSmoother{
		smoothers: make(map[uint32]*SmoothedParameter),
	}
}

// Add adds a parameter with smoothing.
func (ps *ParameterSmoother) Add(id uint32, param *Parameter, smoothingType SmoothingType, rate float64) {
	ps.smoothers[id] = NewSmoothedParameter(param, smoothingType, rate)
}

// GetSmoothed returns the smoothed value for a parameter.
func (ps *ParameterSmoother) GetSmoothed(id uint32) float64 {
	if sp, ok := ps.smoothers[id]; ok {
		return sp.GetSmoothedValue()
	}
	return 0
}

// UpdateAll updates all smoothers (call once per sample).
func (ps *ParameterSmoother) UpdateAll() {
	for _, sp := range ps.smoothers {
		sp.GetSmoothedValue() // This advances the smoother
	}
}

// SetSmoothing enables/disables smoothing for a specific parameter.
func (ps *ParameterSmoother) SetSmoothing(id uint32, enabled bool) {
	if sp, ok := ps.smoothers[id]; ok {
		sp.SetSmoothing(enabled)
	}
}

// SetValue sets the value for a parameter (normalized 0-1).
func (ps *ParameterSmoother) SetValue(id uint32, value float64) {
	if sp, ok := ps.smoothers[id]; ok {
		sp.SetValue(value)
	}
}

// Get returns the SmoothedParameter for direct access.
func (ps *ParameterSmoother) Get(id uint32) (*SmoothedParameter, bool) {
	sp, ok := ps.smoothers[id]
	return sp, ok
}