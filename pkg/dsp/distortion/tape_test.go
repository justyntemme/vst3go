package distortion

import (
	"math"
	"testing"
)

func TestTapeSaturation(t *testing.T) {
	sampleRate := 44100.0
	tape := NewTapeSaturation(sampleRate)
	
	t.Run("Basic", func(t *testing.T) {
		tape.Reset()
		tape.SetMix(1.0)
		tape.SetOutput(1.0)
		tape.SetFlutter(0.0) // Disable flutter for basic tests
		
		// Small signals should pass through with minimal change
		small := 0.1
		result := tape.Process(small)
		if math.Abs(result-small) > 0.1 {
			t.Errorf("Small signals should pass through with minimal change, got %f from %f", result, small)
		}
		
		// Large signals should be compressed/saturated
		large := 0.9
		result = tape.Process(large)
		if result >= large {
			t.Errorf("Large signals should be compressed, got %f from %f", result, large)
		}
	})
	
	t.Run("Saturation", func(t *testing.T) {
		tape.Reset()
		tape.SetFlutter(0.0)
		
		// Test with no saturation
		tape.SetSaturation(0.0)
		noSat := tape.Process(0.5)
		
		// Test with full saturation
		tape.Reset()
		tape.SetSaturation(1.0)
		fullSat := tape.Process(0.5)
		
		if math.Abs(noSat-fullSat) < 0.05 {
			t.Errorf("Saturation should affect output, got %f and %f", noSat, fullSat)
		}
	})
	
	t.Run("Compression", func(t *testing.T) {
		tape.Reset()
		tape.SetFlutter(0.0)
		tape.SetSaturation(0.0) // Disable saturation to test compression alone
		
		// Feed a loud signal to engage compression
		for i := 0; i < 100; i++ {
			tape.Process(0.8)
		}
		
		// Test with different compression settings
		tape.SetCompression(0.0)
		noComp := tape.Process(0.8)
		
		tape.SetCompression(1.0)
		fullComp := tape.Process(0.8)
		
		if fullComp >= noComp {
			t.Errorf("Higher compression should reduce output, got %f vs %f", fullComp, noComp)
		}
	})
	
	t.Run("Flutter", func(t *testing.T) {
		tape.Reset()
		tape.SetFlutter(0.5)
		tape.SetSaturation(0.0)
		tape.SetCompression(0.0)
		
		// Process enough samples to see flutter effect
		const numSamples = 1000
		results := make([]float64, numSamples)
		
		for i := 0; i < numSamples; i++ {
			results[i] = tape.Process(0.5)
		}
		
		// Check for variation due to flutter
		minVal, maxVal := results[0], results[0]
		for _, v := range results {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		
		variation := maxVal - minVal
		if variation < 0.01 {
			t.Errorf("Flutter should cause output variation, got range %f", variation)
		}
	})
	
	t.Run("Warmth", func(t *testing.T) {
		tape.Reset()
		tape.SetFlutter(0.0)
		
		// Process with no warmth
		tape.SetWarmth(0.0)
		noWarmth := tape.Process(0.5)
		
		// Process with full warmth
		tape.Reset()
		tape.SetWarmth(1.0)
		fullWarmth := tape.Process(0.5)
		
		if math.Abs(noWarmth-fullWarmth) < 0.01 {
			t.Errorf("Warmth should affect output, got %f and %f", noWarmth, fullWarmth)
		}
	})
	
	t.Run("Mix", func(t *testing.T) {
		tape.Reset()
		tape.SetSaturation(1.0)
		tape.SetFlutter(0.0)
		
		input := 0.5
		
		// Full dry
		tape.SetMix(0.0)
		dry := tape.Process(input)
		
		if math.Abs(dry-input) > 1e-9 {
			t.Errorf("With mix=0, output should equal input, got %f from %f", dry, input)
		}
		
		// Full wet
		tape.Reset()
		tape.SetMix(1.0)
		wet := tape.Process(input)
		
		// 50% mix
		tape.Reset()
		tape.SetMix(0.5)
		mixed := tape.Process(input)
		
		// Should be between dry and wet
		if mixed <= math.Min(dry, wet) || mixed >= math.Max(dry, wet) {
			t.Errorf("50%% mix should be between dry and wet signals")
		}
	})
	
	t.Run("ProcessBlock", func(t *testing.T) {
		tape.Reset()
		tape.SetSaturation(0.7)
		tape.SetCompression(0.5)
		tape.SetFlutter(0.0)
		
		input := []float64{-0.8, -0.4, 0.0, 0.4, 0.8}
		output := make([]float64, len(input))
		
		tape.ProcessBlock(input, output)
		
		// Verify processing occurred
		allZero := true
		for _, v := range output {
			if v != 0.0 {
				allZero = false
				break
			}
		}
		
		if allZero {
			t.Errorf("ProcessBlock should produce non-zero output")
		}
	})
	
	t.Run("Stereo", func(t *testing.T) {
		tape.Reset()
		tape.SetSaturation(0.5)
		tape.SetFlutter(0.3)
		
		// Create different input for L/R
		inputL := []float64{0.5, 0.3, 0.1, -0.1, -0.3}
		inputR := []float64{0.4, 0.2, 0.0, -0.2, -0.4}
		outputL := make([]float64, len(inputL))
		outputR := make([]float64, len(inputR))
		
		tape.ProcessStereo(inputL, inputR, outputL, outputR)
		
		// Verify stereo processing maintains channel separation
		different := false
		for i := range outputL {
			if math.Abs(outputL[i]-outputR[i]) > 0.001 {
				different = true
				break
			}
		}
		
		if !different {
			t.Errorf("Stereo processing should maintain channel differences")
		}
	})
}

func BenchmarkTapeSaturation(b *testing.B) {
	tape := NewTapeSaturation(44100.0)
	tape.SetSaturation(0.7)
	tape.SetCompression(0.5)
	tape.SetFlutter(0.2)
	tape.SetWarmth(0.6)
	
	input := make([]float64, 512)
	output := make([]float64, 512)
	
	for i := range input {
		input[i] = math.Sin(2.0 * math.Pi * float64(i) / 512.0) * 0.8
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tape.ProcessBlock(input, output)
	}
}