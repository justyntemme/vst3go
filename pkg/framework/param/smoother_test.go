package param

import (
	"math"
	"testing"
)

func TestSmoother(t *testing.T) {
	t.Run("LinearSmoothing", func(t *testing.T) {
		smoother := NewSmoother(LinearSmoothing, 10) // 10 samples
		smoother.Reset(0.0)
		smoother.SetTarget(1.0)

		// Should take 10 samples to reach target
		for i := 0; i < 10; i++ {
			value := smoother.Next()
			expected := float64(i+1) * 0.1
			if math.Abs(value-expected) > 0.001 {
				t.Errorf("Sample %d: expected %f, got %f", i, expected, value)
			}
		}

		// Should stay at target
		if smoother.Next() != 1.0 {
			t.Error("Should stay at target after reaching it")
		}
		if smoother.IsSmoothing() {
			t.Error("Should not be smoothing after reaching target")
		}
	})

	t.Run("ExponentialSmoothing", func(t *testing.T) {
		smoother := NewSmoother(ExponentialSmoothing, 0.9) // High = slow
		smoother.Reset(0.0)
		smoother.SetTarget(1.0)

		// Should approach target exponentially
		prev := 0.0
		for i := 0; i < 50; i++ {
			value := smoother.Next()
			if value <= prev {
				t.Error("Value should be increasing")
			}
			if value >= 1.0 {
				t.Error("Should not exceed target")
			}
			prev = value
		}

		// Should eventually reach target (within threshold)
		for i := 0; i < 200; i++ {
			smoother.Next()
		}
		if smoother.IsSmoothing() {
			t.Error("Should have reached target by now")
		}
	})

	t.Run("LogarithmicSmoothing", func(t *testing.T) {
		smoother := NewSmoother(LogarithmicSmoothing, 10)
		smoother.Reset(100.0) // Start at 100 Hz
		smoother.SetTarget(1000.0) // Target 1000 Hz

		// Should interpolate in log space
		values := []float64{}
		for i := 0; i < 10; i++ {
			values = append(values, smoother.Next())
		}

		// Check that we're interpolating logarithmically
		// The ratio between consecutive values should be constant
		ratio := values[1] / values[0]
		for i := 2; i < len(values); i++ {
			currentRatio := values[i] / values[i-1]
			if math.Abs(currentRatio-ratio) > 0.01 {
				t.Error("Logarithmic interpolation not maintaining constant ratio")
			}
		}
	})

	t.Run("Threshold", func(t *testing.T) {
		smoother := NewSmoother(ExponentialSmoothing, 0.9)
		smoother.SetThreshold(0.1)
		smoother.Reset(0.0)
		smoother.SetTarget(0.05) // Less than threshold

		// Should not start smoothing
		if smoother.IsSmoothing() {
			t.Error("Should not smooth when change is below threshold")
		}
	})

	t.Run("Process", func(t *testing.T) {
		smoother := NewSmoother(LinearSmoothing, 5)
		smoother.Reset(0.0)
		smoother.SetTarget(1.0)

		buffer := []float32{1.0, 1.0, 1.0, 1.0, 1.0}
		smoother.Process(buffer, func(value float64, sample float32) float32 {
			return sample * float32(value)
		})

		// Check that smoothing was applied
		expected := []float32{0.2, 0.4, 0.6, 0.8, 1.0}
		for i, v := range buffer {
			if math.Abs(float64(v-expected[i])) > 0.001 {
				t.Errorf("Sample %d: expected %f, got %f", i, expected[i], v)
			}
		}
	})
}

func TestSmoothedParameter(t *testing.T) {
	t.Run("BasicOperation", func(t *testing.T) {
		// Create a test parameter
		param := &Parameter{
			ID:           1,
			Name:         "Test",
			ShortName:    "Test", 
			Unit:         "",
			Min:          0.0,
			Max:          1.0,
			DefaultValue: 0.5,
			StepCount:    0,
			Flags:        CanAutomate,
			UnitID:       0,
		}
		param.SetValue(0.5)

		// Wrap with smoothing
		smoothed := NewSmoothedParameter(param, ExponentialSmoothing, 0.9)

		// Set a new value
		smoothed.SetValue(1.0)

		// Should smooth towards the new value
		prev := 0.5
		for i := 0; i < 10; i++ {
			value := smoothed.GetSmoothedValue()
			if value <= prev {
				t.Error("Value should be increasing")
			}
			prev = value
		}
	})

	t.Run("DisableSmoothing", func(t *testing.T) {
		param := &Parameter{
			ID:           1,
			Name:         "Test",
			ShortName:    "Test",
			Unit:         "",
			Min:          0.0,
			Max:          1.0,
			DefaultValue: 0.5,
			StepCount:    0,
			Flags:        CanAutomate,
			UnitID:       0,
		}
		param.SetValue(0.5)

		smoothed := NewSmoothedParameter(param, LinearSmoothing, 10)
		smoothed.SetSmoothing(false)
		smoothed.SetValue(1.0)

		// Should jump immediately
		if smoothed.GetSmoothedValue() != 1.0 {
			t.Error("Should not smooth when disabled")
		}
	})

	t.Run("UpdateSampleRate", func(t *testing.T) {
		param := &Parameter{
			ID:           1,
			Name:         "Test",
			ShortName:    "Test",
			Unit:         "",
			Min:          0.0,
			Max:          1.0,
			DefaultValue: 0.5,
			StepCount:    0,
			Flags:        CanAutomate,
			UnitID:       0,
		}
		param.SetValue(0.5)

		// Linear smoothing with time-based rate
		smoothed := NewSmoothedParameter(param, LinearSmoothing, 10)
		smoothed.UpdateSampleRate(48000, 20) // 20ms at 48kHz

		// Should have updated the rate
		if smoothed.smoother.rate != 960 { // 48000 * 20 / 1000
			t.Errorf("Expected rate 960, got %f", smoothed.smoother.rate)
		}
	})
}

func TestParameterSmoother(t *testing.T) {
	t.Run("ManageMultiple", func(t *testing.T) {
		ps := NewParameterSmoother()

		// Create test parameters
		param1 := &Parameter{
			ID:           1,
			Name:         "Param1",
			Min:          0.0,
			Max:          1.0,
			DefaultValue: 0.0,
		}
		param1.SetValue(0.0)
		
		param2 := &Parameter{
			ID:           2,
			Name:         "Param2",
			Min:          0.0,
			Max:          1.0,
			DefaultValue: 0.5,
		}
		param2.SetValue(0.5)

		// Add with different smoothing types
		ps.Add(1, param1, LinearSmoothing, 5)
		ps.Add(2, param2, ExponentialSmoothing, 0.9)

		// Set new values through the parameter smoother
		ps.SetValue(1, 1.0)
		ps.SetValue(2, 1.0)

		// Get smoothed values
		val1 := ps.GetSmoothed(1)
		val2 := ps.GetSmoothed(2)

		// For linear smoothing with 5 steps, first value should be 0.2
		if val1 <= 0.0 || val1 >= 0.3 {
			t.Errorf("Parameter 1 should be smoothing (expected ~0.2), got %f", val1)
		}
		// For exponential smoothing, should be between old and new value
		if val2 <= 0.5 || val2 >= 1.0 {
			t.Errorf("Parameter 2 should be smoothing (between 0.5 and 1.0), got %f", val2)
		}
	})

	t.Run("DisableSpecific", func(t *testing.T) {
		ps := NewParameterSmoother()

		param := &Parameter{
			ID:           1,
			Name:         "Test",
			Min:          0.0,
			Max:          1.0,
			DefaultValue: 0.0,
		}
		param.SetValue(0.0)

		ps.Add(1, param, LinearSmoothing, 10)
		ps.SetSmoothing(1, false)

		ps.SetValue(1, 1.0)
		if ps.GetSmoothed(1) != 1.0 {
			t.Error("Should not smooth when disabled")
		}
	})
}

func BenchmarkSmoother(b *testing.B) {
	b.Run("LinearNext", func(b *testing.B) {
		smoother := NewSmoother(LinearSmoothing, 100)
		smoother.SetTarget(1.0)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = smoother.Next()
		}
	})

	b.Run("ExponentialNext", func(b *testing.B) {
		smoother := NewSmoother(ExponentialSmoothing, 0.99)
		smoother.SetTarget(1.0)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = smoother.Next()
		}
	})

	b.Run("LogarithmicNext", func(b *testing.B) {
		smoother := NewSmoother(LogarithmicSmoothing, 100)
		smoother.SetTarget(1000.0)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = smoother.Next()
		}
	})

	b.Run("ProcessBuffer", func(b *testing.B) {
		smoother := NewSmoother(ExponentialSmoothing, 0.99)
		smoother.SetTarget(1.0)
		buffer := make([]float32, 512)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			smoother.Process(buffer, func(value float64, sample float32) float32 {
				return sample * float32(value)
			})
		}
	})
}