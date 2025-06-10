package distortion

import (
	"math"
	"testing"
)

func TestTubeSaturation(t *testing.T) {
	tube := NewTubeSaturation()
	
	t.Run("Basic", func(t *testing.T) {
		tube.Reset()
		tube.SetMix(1.0)
		tube.SetOutput(1.0)
		
		// Test that small signals pass through relatively unchanged
		small := 0.1
		result := tube.Process(small)
		if math.Abs(result-small) > 0.05 {
			t.Errorf("Small signals should pass through with minimal change, got %f from %f", result, small)
		}
		
		// Test that large signals are compressed
		large := 0.9
		result = tube.Process(large)
		if result >= large {
			t.Errorf("Large signals should be compressed, got %f from %f", result, large)
		}
	})
	
	t.Run("Warmth", func(t *testing.T) {
		tube.Reset()
		
		// Process with no warmth
		tube.SetWarmth(0.0)
		noWarmth := tube.Process(0.5)
		
		// Process with full warmth
		tube.Reset()
		tube.SetWarmth(1.0)
		fullWarmth := tube.Process(0.5)
		
		// Warmth should affect the output
		if math.Abs(noWarmth-fullWarmth) < 0.01 {
			t.Errorf("Warmth parameter should affect output, got %f and %f", noWarmth, fullWarmth)
		}
	})
	
	t.Run("Harmonics", func(t *testing.T) {
		tube.Reset()
		
		// Test with different harmonic settings
		tube.SetHarmonics(0.0)
		noHarmonics := tube.Process(0.5)
		
		tube.Reset()
		tube.SetHarmonics(1.0)
		fullHarmonics := tube.Process(0.5)
		
		if math.Abs(noHarmonics-fullHarmonics) < 0.001 {
			t.Errorf("Harmonics parameter should affect output, got %f and %f", noHarmonics, fullHarmonics)
		}
	})
	
	t.Run("Bias", func(t *testing.T) {
		tube.Reset()
		tube.SetBias(0.5)
		
		// Bias should create asymmetry
		pos := tube.Process(0.3)
		tube.Reset()
		tube.SetBias(0.5)
		neg := tube.Process(-0.3)
		
		// With bias, positive and negative inputs should produce different magnitudes
		if math.Abs(math.Abs(pos)-math.Abs(neg)) < 0.01 {
			t.Errorf("Bias should create asymmetry, got %f and %f", pos, neg)
		}
	})
	
	t.Run("Hysteresis", func(t *testing.T) {
		tube.Reset()
		tube.SetHysteresis(0.8)
		
		// Process same value multiple times
		first := tube.Process(0.5)
		second := tube.Process(0.5)
		
		// With hysteresis, output should depend on history
		if math.Abs(first-second) < 0.001 {
			t.Errorf("Hysteresis should cause output to depend on history")
		}
	})
	
	t.Run("Mix", func(t *testing.T) {
		tube.Reset()
		tube.SetHarmonics(1.0)
		tube.SetWarmth(1.0)
		
		input := 0.5
		
		// Full wet
		tube.SetMix(1.0)
		wet := tube.Process(input)
		
		// Full dry
		tube.Reset()
		tube.SetMix(0.0)
		dry := tube.Process(input)
		
		if math.Abs(dry-input) > 1e-9 {
			t.Errorf("With mix=0, output should equal input, got %f from %f", dry, input)
		}
		
		// 50% mix
		tube.Reset()
		tube.SetMix(0.5)
		mixed := tube.Process(input)
		
		// Should be roughly between dry and wet
		if mixed <= math.Min(dry, wet) || mixed >= math.Max(dry, wet) {
			t.Errorf("50%% mix should be between dry and wet signals")
		}
	})
	
	t.Run("ProcessBlock", func(t *testing.T) {
		tube.Reset()
		tube.SetHarmonics(0.7)
		tube.SetWarmth(0.5)
		
		input := []float64{-0.8, -0.4, 0.0, 0.4, 0.8}
		output := make([]float64, len(input))
		
		tube.ProcessBlock(input, output)
		
		// Verify each sample was processed
		tube.Reset()
		for i, v := range input {
			expected := tube.Process(v)
			if math.Abs(output[i]-expected) > 0.1 { // Higher tolerance due to hysteresis
				t.Errorf("ProcessBlock[%d] = %f, want approximately %f", i, output[i], expected)
			}
		}
	})
	
	t.Run("CompressionCurve", func(t *testing.T) {
		tube.Reset()
		tube.SetWarmth(0.0) // Disable warmth to focus on compression
		tube.SetHarmonics(0.0) // Disable harmonics
		
		// Test that saturation/compression increases with signal level
		low := 0.1
		high := 0.9
		
		outputLow := tube.Process(low)
		tube.Reset()
		tube.SetWarmth(0.0)
		tube.SetHarmonics(0.0)
		outputHigh := tube.Process(high)
		
		// The ratio output/input should be lower for high signals (more compression)
		ratioLow := outputLow / low
		ratioHigh := outputHigh / high
		
		if ratioHigh >= ratioLow {
			t.Errorf("High level signals should show more compression: low ratio=%f, high ratio=%f", ratioLow, ratioHigh)
		}
	})
}

func BenchmarkTubeSaturation(b *testing.B) {
	tube := NewTubeSaturation()
	tube.SetWarmth(0.7)
	tube.SetHarmonics(0.5)
	tube.SetHysteresis(0.3)
	
	input := make([]float64, 512)
	output := make([]float64, 512)
	
	for i := range input {
		input[i] = math.Sin(2.0 * math.Pi * float64(i) / 512.0) * 0.8
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tube.ProcessBlock(input, output)
	}
}