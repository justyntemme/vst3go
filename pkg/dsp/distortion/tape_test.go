package distortion

import (
	"math"
	"testing"
)

func TestTapeSaturator(t *testing.T) {
	sampleRate := 48000.0

	t.Run("Basic Operation", func(t *testing.T) {
		tape := NewTapeSaturator(sampleRate)
		
		// Test dry signal passes through with mix=0
		tape.SetMix(0.0)
		input := 0.5
		output := tape.Process(input)
		if math.Abs(output-input) > 1e-6 {
			t.Errorf("Mix=0 should pass dry signal: got %f, expected %f", output, input)
		}
		
		// Test that saturation is applied with mix=1
		tape.SetMix(1.0)
		tape.SetDrive(3.0)
		tape.SetSaturation(0.8)
		output = tape.Process(input)
		if math.Abs(output-input) < 1e-6 {
			t.Errorf("Saturation should modify signal: got %f, input was %f", output, input)
		}
	})

	t.Run("Parameter Limits", func(t *testing.T) {
		tape := NewTapeSaturator(sampleRate)
		
		// Test drive limits
		tape.SetDrive(0.5) // Should clamp to 1.0
		tape.SetDrive(15.0) // Should clamp to 10.0
		
		// Test saturation limits
		tape.SetSaturation(-0.5) // Should clamp to 0.0
		tape.SetSaturation(1.5) // Should clamp to 1.0
		
		// Test other parameter limits
		tape.SetBias(-0.1) // Should clamp to 0.0
		tape.SetBias(1.1) // Should clamp to 1.0
		
		tape.SetCompression(-0.1) // Should clamp to 0.0
		tape.SetCompression(1.1) // Should clamp to 1.0
		
		// Process should not crash
		output := tape.Process(0.5)
		if math.IsNaN(output) || math.IsInf(output, 0) {
			t.Errorf("Output should be finite: got %f", output)
		}
	})

	t.Run("Compression", func(t *testing.T) {
		tape := NewTapeSaturator(sampleRate)
		tape.SetMix(1.0)
		tape.SetDrive(1.0)
		tape.SetSaturation(0.0) // Disable saturation to test compression alone
		
		// Test with no compression
		tape.SetCompression(0.0)
		loudInput := 0.9
		quietInput := 0.1
		loudOut1 := tape.Process(loudInput)
		quietOut1 := tape.Process(quietInput)
		ratio1 := math.Abs(loudOut1) / math.Abs(quietOut1)
		
		// Test with full compression
		tape.SetCompression(1.0)
		loudOut2 := tape.Process(loudInput)
		quietOut2 := tape.Process(quietInput)
		ratio2 := math.Abs(loudOut2) / math.Abs(quietOut2)
		
		// Compression should reduce the ratio between loud and quiet
		if ratio2 >= ratio1 {
			t.Errorf("Compression should reduce dynamic range: %f vs %f", ratio2, ratio1)
		}
	})

	t.Run("Hysteresis", func(t *testing.T) {
		tape := NewTapeSaturator(sampleRate)
		tape.SetMix(1.0)
		tape.SetDrive(2.0)
		tape.SetSaturation(0.5)
		
		// Process the same value multiple times
		// With hysteresis, outputs should converge but not be identical initially
		input := 0.3
		output1 := tape.Process(input)
		output2 := tape.Process(input)
		output3 := tape.Process(input)
		
		// First two outputs should differ due to hysteresis
		if math.Abs(output1-output2) < 1e-9 {
			t.Errorf("Hysteresis should cause different outputs: %f vs %f", output1, output2)
		}
		
		// Should converge
		if math.Abs(output2-output3) > math.Abs(output1-output2) {
			t.Errorf("Hysteresis should converge over time")
		}
	})

	t.Run("Flutter", func(t *testing.T) {
		tape := NewTapeSaturator(sampleRate)
		tape.SetMix(1.0)
		tape.SetFlutter(0.5)
		
		// Process a constant signal
		input := 0.5
		outputs := make([]float64, 100)
		
		for i := range outputs {
			outputs[i] = tape.Process(input)
		}
		
		// With flutter, outputs should vary
		var variance float64
		for i := 1; i < len(outputs); i++ {
			variance += math.Abs(outputs[i] - outputs[i-1])
		}
		
		if variance < 1e-6 {
			t.Errorf("Flutter should cause output variation: variance = %f", variance)
		}
	})

	t.Run("Warmth", func(t *testing.T) {
		tape := NewTapeSaturator(sampleRate)
		tape.SetMix(1.0)
		tape.SetDrive(1.5)
		
		input := 0.4
		
		// Test with no warmth
		tape.SetWarmth(0.0)
		output1 := tape.Process(input)
		
		// Test with full warmth
		tape.SetWarmth(1.0)
		output2 := tape.Process(input)
		
		// Warmth should add harmonics, changing the output
		if math.Abs(output1-output2) < 1e-6 {
			t.Errorf("Warmth should affect output: %f vs %f", output1, output2)
		}
	})

	t.Run("ProcessBuffer", func(t *testing.T) {
		tape := NewTapeSaturator(sampleRate)
		tape.SetDrive(2.5)
		tape.SetSaturation(0.7)
		tape.SetMix(1.0)
		
		input := []float64{-0.8, -0.4, 0.0, 0.4, 0.8}
		output := make([]float64, len(input))
		
		tape.ProcessBuffer(input, output)
		
		// Verify processing was applied
		for i := range input {
			if math.IsNaN(output[i]) || math.IsInf(output[i], 0) {
				t.Errorf("Output should be finite: got %f at index %d", output[i], i)
			}
			
			// Output should be bounded
			if math.Abs(output[i]) > 2.0 {
				t.Errorf("Output seems too large: %f at index %d", output[i], i)
			}
		}
	})
}

func TestPreDeEmphasisFilters(t *testing.T) {
	sampleRate := 48000.0

	t.Run("PreEmphasis Boosts Highs", func(t *testing.T) {
		pre := NewPreEmphasisFilter(sampleRate)
		
		// Test with high frequency
		highFreq := 5000.0
		omega := 2.0 * math.Pi * highFreq / sampleRate
		
		var inputSum, outputSum float64
		for i := 0; i < 100; i++ {
			input := math.Sin(omega * float64(i))
			output := pre.Process(input)
			inputSum += math.Abs(input)
			outputSum += math.Abs(output)
		}
		
		// Pre-emphasis should boost high frequencies
		if outputSum <= inputSum {
			t.Errorf("Pre-emphasis should boost highs: %f vs %f", outputSum, inputSum)
		}
	})

	t.Run("DeEmphasis Compensates", func(t *testing.T) {
		pre := NewPreEmphasisFilter(sampleRate)
		de := NewDeEmphasisFilter(sampleRate)
		
		// Process through both filters
		testFreqs := []float64{100.0, 1000.0, 5000.0, 10000.0}
		
		for _, freq := range testFreqs {
			omega := 2.0 * math.Pi * freq / sampleRate
			
			var inputSum, outputSum float64
			for i := 0; i < 100; i++ {
				input := math.Sin(omega * float64(i))
				emphasized := pre.Process(input)
				output := de.Process(emphasized)
				
				inputSum += math.Abs(input)
				outputSum += math.Abs(output)
			}
			
			// Pre+De emphasis should roughly preserve amplitude
			ratio := outputSum / inputSum
			if ratio < 0.8 || ratio > 1.2 {
				t.Errorf("Pre+De emphasis should preserve amplitude at %fHz: ratio=%f", freq, ratio)
			}
		}
	})
}

func BenchmarkTapeSaturator(b *testing.B) {
	sampleRate := 48000.0
	tape := NewTapeSaturator(sampleRate)
	tape.SetDrive(2.5)
	tape.SetSaturation(0.7)
	tape.SetCompression(0.3)
	tape.SetFlutter(0.1)
	tape.SetMix(1.0)
	
	// Create test buffer
	bufferSize := 512
	input := make([]float64, bufferSize)
	output := make([]float64, bufferSize)
	
	// Fill with test signal
	for i := range input {
		input[i] = math.Sin(2.0*math.Pi*1000.0*float64(i)/sampleRate) * 0.5
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tape.ProcessBuffer(input, output)
	}
}