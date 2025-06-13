package utility

import (
	"math"
	"testing"
)

func TestScaleParameter(t *testing.T) {
	tests := []struct {
		name       string
		normalized float64
		min        float64
		max        float64
		expected   float64
	}{
		{"Zero to min", 0.0, -60.0, 0.0, -60.0},
		{"One to max", 1.0, -60.0, 0.0, 0.0},
		{"Half", 0.5, -60.0, 0.0, -30.0},
		{"Quarter", 0.25, 0.0, 100.0, 25.0},
		{"Three quarters", 0.75, 0.0, 100.0, 75.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScaleParameter(tt.normalized, tt.min, tt.max)
			if math.Abs(result-tt.expected) > 1e-6 {
				t.Errorf("ScaleParameter(%f, %f, %f) = %f, want %f",
					tt.normalized, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

func TestScaleParameterExp(t *testing.T) {
	tests := []struct {
		name       string
		normalized float64
		min        float64
		max        float64
		expected   float64
	}{
		{"Zero to min", 0.0, 20.0, 20000.0, 20.0},
		{"One to max", 1.0, 20.0, 20000.0, 20000.0},
		{"Half", 0.5, 20.0, 20000.0, 632.455532}, // sqrt(20*20000)
		{"Negative min fallback", 0.5, -10.0, 10.0, 0.0}, // Falls back to linear
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScaleParameterExp(tt.normalized, tt.min, tt.max)
			if math.Abs(result-tt.expected) > 1e-6 {
				t.Errorf("ScaleParameterExp(%f, %f, %f) = %f, want %f",
					tt.normalized, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

func TestUnscaleParameter(t *testing.T) {
	// Test that unscaling is the inverse of scaling
	testValues := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
	min, max := -60.0, 0.0

	for _, normalized := range testValues {
		scaled := ScaleParameter(normalized, min, max)
		unscaled := UnscaleParameter(scaled, min, max)
		if math.Abs(unscaled-normalized) > 1e-6 {
			t.Errorf("UnscaleParameter inverse failed: %f -> %f -> %f",
				normalized, scaled, unscaled)
		}
	}
}

func TestClampParameter(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{"Within range", 5.0, 0.0, 10.0, 5.0},
		{"Below min", -5.0, 0.0, 10.0, 0.0},
		{"Above max", 15.0, 0.0, 10.0, 10.0},
		{"At min", 0.0, 0.0, 10.0, 0.0},
		{"At max", 10.0, 0.0, 10.0, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClampParameter(tt.value, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("ClampParameter(%f, %f, %f) = %f, want %f",
					tt.value, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

func TestQuantizeParameter(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		steps    int
		expected float64
	}{
		{"5 steps at 0", 0.0, 5, 0.0},
		{"5 steps at 0.5", 0.5, 5, 0.5},
		{"5 steps at 0.4", 0.4, 5, 0.5},
		{"5 steps at 0.1", 0.1, 5, 0.0},
		{"5 steps at 0.9", 0.9, 5, 1.0},
		{"1 step", 0.7, 1, 0.7}, // No quantization
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuantizeParameter(tt.value, tt.steps)
			if math.Abs(result-tt.expected) > 1e-6 {
				t.Errorf("QuantizeParameter(%f, %d) = %f, want %f",
					tt.value, tt.steps, result, tt.expected)
			}
		})
	}
}

func TestBipolarConversion(t *testing.T) {
	// Test conversion and back
	testValues := []float64{-1.0, -0.5, 0.0, 0.5, 1.0}

	for _, bipolar := range testValues {
		unipolar := BipolarToUnipolar(bipolar)
		backToBipolar := UnipolarToBipolar(unipolar)
		
		if math.Abs(backToBipolar-bipolar) > 1e-6 {
			t.Errorf("Bipolar conversion roundtrip failed: %f -> %f -> %f",
				bipolar, unipolar, backToBipolar)
		}
	}
}

func TestSmoothParameter(t *testing.T) {
	// Create a smoother with 10ms smoothing time at 48kHz
	smoother := NewSmoothParameter(0.01, 48000)
	
	// Set initial value
	smoother.SetImmediate(0.0)
	
	// Set target
	smoother.SetTarget(1.0)
	
	// Should be smoothing
	if !smoother.IsSmoothing() {
		t.Error("Expected parameter to be smoothing")
	}
	
	// Process some samples
	prev := smoother.GetCurrent()
	for i := 0; i < 100; i++ {
		current := smoother.Process()
		// Value should be increasing
		if current <= prev {
			t.Errorf("Expected smoothed value to increase: %f -> %f", prev, current)
		}
		prev = current
	}
	
	// After many samples, should be close to target
	for i := 0; i < 10000; i++ {
		smoother.Process()
	}
	
	final := smoother.GetCurrent()
	if math.Abs(final-1.0) > 0.01 {
		t.Errorf("Expected smoothed value to reach near target: %f", final)
	}
}