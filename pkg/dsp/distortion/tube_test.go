package distortion

import (
	"math"
	"testing"
)

func TestTubeSaturator(t *testing.T) {
	sampleRate := 48000.0

	t.Run("Basic Operation", func(t *testing.T) {
		tube := NewTubeSaturator(sampleRate)
		
		// Test that clean signal passes through with mix=0
		tube.SetMix(0.0)
		input := 0.5
		output := tube.Process(input)
		if math.Abs(output-input) > 1e-6 {
			t.Errorf("Mix=0 should pass dry signal: got %f, expected %f", output, input)
		}
		
		// Test that saturation is applied with mix=1
		tube.SetMix(1.0)
		tube.SetDrive(3.0)
		output = tube.Process(input)
		if math.Abs(output-input) < 1e-6 {
			t.Errorf("Drive should modify signal: got %f, input was %f", output, input)
		}
	})

	t.Run("Drive Limits", func(t *testing.T) {
		tube := NewTubeSaturator(sampleRate)
		tube.SetMix(1.0)
		
		// Test minimum drive
		tube.SetDrive(0.5) // Should clamp to 1.0
		input := 0.5
		output1 := tube.Process(input)
		
		tube.SetDrive(1.0)
		output2 := tube.Process(input)
		
		if math.Abs(output1-output2) > 1e-6 {
			t.Errorf("Drive should clamp to minimum 1.0")
		}
		
		// Test maximum drive
		tube.SetDrive(15.0) // Should clamp to 10.0
		output1 = tube.Process(input)
		
		tube.SetDrive(10.0)
		output2 = tube.Process(input)
		
		if math.Abs(output1-output2) > 1e-6 {
			t.Errorf("Drive should clamp to maximum 10.0")
		}
	})

	t.Run("Harmonic Balance", func(t *testing.T) {
		tube := NewTubeSaturator(sampleRate)
		tube.SetMix(1.0)
		tube.SetDrive(3.0)
		
		input := 0.3
		
		// Test all even harmonics
		tube.SetHarmonicBalance(1.0)
		evenOutput := tube.Process(input)
		
		// Test all odd harmonics
		tube.SetHarmonicBalance(0.0)
		oddOutput := tube.Process(input)
		
		// Outputs should be different
		if math.Abs(evenOutput-oddOutput) < 1e-6 {
			t.Errorf("Different harmonic balances should produce different outputs")
		}
	})

	t.Run("Warmth Control", func(t *testing.T) {
		tube := NewTubeSaturator(sampleRate)
		tube.SetMix(1.0)
		tube.SetDrive(2.0)
		
		// Create a low frequency test signal
		testFreq := 100.0
		omega := 2.0 * math.Pi * testFreq / sampleRate
		
		// Process with no warmth
		tube.SetWarmth(0.0)
		var sum1 float64
		for i := 0; i < 100; i++ {
			input := 0.5 * math.Sin(omega*float64(i))
			output := tube.Process(input)
			sum1 += math.Abs(output)
		}
		
		// Process with full warmth
		tube.SetWarmth(1.0)
		var sum2 float64
		for i := 0; i < 100; i++ {
			input := 0.5 * math.Sin(omega*float64(i))
			output := tube.Process(input)
			sum2 += math.Abs(output)
		}
		
		// Warmth should increase low frequency content
		if sum2 <= sum1 {
			t.Errorf("Warmth should boost low frequencies: %f vs %f", sum2, sum1)
		}
	})

	t.Run("Asymmetric Clipping", func(t *testing.T) {
		tube := NewTubeSaturator(sampleRate)
		tube.SetMix(1.0)
		tube.SetDrive(5.0)
		
		// Test positive and negative clipping
		input := 0.8
		posOutput := tube.Process(input)
		negOutput := tube.Process(-input)
		
		// Due to asymmetric transfer function, absolute values should differ
		if math.Abs(math.Abs(posOutput)-math.Abs(negOutput)) < 1e-6 {
			t.Errorf("Tube should have asymmetric clipping: %f vs %f", posOutput, negOutput)
		}
	})

	t.Run("ProcessBuffer", func(t *testing.T) {
		tube := NewTubeSaturator(sampleRate)
		tube.SetDrive(2.0)
		tube.SetMix(1.0)
		
		input := []float64{-0.8, -0.4, 0.0, 0.4, 0.8}
		output := make([]float64, len(input))
		
		tube.ProcessBuffer(input, output)
		
		// Verify processing was applied
		for i := range input {
			if math.Abs(output[i]) > 1.0 {
				t.Errorf("Output should be bounded: got %f at index %d", output[i], i)
			}
		}
		
		// Verify asymmetry
		if math.Abs(math.Abs(output[0])-math.Abs(output[4])) < 1e-6 {
			t.Errorf("Asymmetric processing expected")
		}
	})
}

func TestSimpleFilters(t *testing.T) {
	sampleRate := 48000.0

	t.Run("Highpass DC Removal", func(t *testing.T) {
		hp := NewSimpleHighpass(sampleRate, 20.0)
		
		// Feed DC offset
		dc := 0.5
		var lastOutput float64
		
		// Process enough samples for filter to settle
		for i := 0; i < 1000; i++ {
			lastOutput = hp.Process(dc)
		}
		
		// Output should be near zero (DC removed)
		if math.Abs(lastOutput) > 0.01 {
			t.Errorf("Highpass should remove DC: got %f", lastOutput)
		}
	})

	t.Run("Lowpass Smoothing", func(t *testing.T) {
		lp := NewSimpleLowpass(sampleRate, 1000.0)
		
		// Feed alternating signal (high frequency)
		var sum float64
		for i := 0; i < 100; i++ {
			input := 1.0
			if i%2 == 0 {
				input = -1.0
			}
			output := lp.Process(input)
			sum += math.Abs(output)
		}
		
		// Average output should be much less than input
		avg := sum / 100.0
		if avg > 0.5 {
			t.Errorf("Lowpass should attenuate high frequencies: got avg %f", avg)
		}
	})

	t.Run("LowShelf Gain", func(t *testing.T) {
		ls := NewSimpleLowShelf(sampleRate, 200.0)
		ls.SetGain(2.0) // +6dB
		
		// Test with low frequency
		testFreq := 100.0
		omega := 2.0 * math.Pi * testFreq / sampleRate
		
		var inputSum, outputSum float64
		for i := 0; i < 100; i++ {
			input := math.Sin(omega * float64(i))
			output := ls.Process(input)
			inputSum += math.Abs(input)
			outputSum += math.Abs(output)
		}
		
		// Output should be boosted
		if outputSum < inputSum*1.5 {
			t.Errorf("Low shelf should boost low frequencies: %f vs %f", outputSum, inputSum)
		}
	})
}

func BenchmarkTubeSaturator(b *testing.B) {
	sampleRate := 48000.0
	tube := NewTubeSaturator(sampleRate)
	tube.SetDrive(3.0)
	tube.SetMix(1.0)
	tube.SetWarmth(0.5)
	
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
		tube.ProcessBuffer(input, output)
	}
}