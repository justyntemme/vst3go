package distortion

import (
	"math"
	"testing"
)

func TestBitCrusher(t *testing.T) {
	sampleRate := 48000.0

	t.Run("Basic Operation", func(t *testing.T) {
		bc := NewBitCrusher(sampleRate)
		
		// Test dry signal
		bc.SetMix(0.0)
		input := 0.5
		output := bc.Process(input)
		if math.Abs(output-input) > 1e-6 {
			t.Errorf("Mix=0 should pass dry signal: got %f, expected %f", output, input)
		}
		
		// Test wet signal
		bc.SetMix(1.0)
		bc.SetBitDepth(4) // Low bit depth for obvious effect
		output = bc.Process(input)
		if output == input {
			t.Errorf("Bit crushing should modify signal: got %f, input was %f", output, input)
		}
	})

	t.Run("Bit Depth Reduction", func(t *testing.T) {
		bc := NewBitCrusher(sampleRate)
		bc.SetMix(1.0)
		bc.SetSampleRateRatio(1.0) // No sample rate reduction
		
		// Test different bit depths
		bitDepths := []int{1, 2, 4, 8, 16}
		
		previousLevels := 0
		for _, bits := range bitDepths {
			bc.SetBitDepth(bits)
			
			// Count unique output levels
			levels := make(map[float64]bool)
			testInputs := []float64{-1.0, -0.5, 0.0, 0.5, 1.0}
			
			for _, in := range testInputs {
				out := bc.Process(in)
				// Round to avoid floating point comparison issues
				rounded := math.Round(out * 1000) / 1000
				levels[rounded] = true
			}
			
			// More bits should allow more levels
			if bits > 1 && len(levels) <= previousLevels {
				t.Errorf("Bit depth %d should have more levels than %d", bits, previousLevels)
			}
			previousLevels = len(levels)
		}
	})

	t.Run("Sample Rate Reduction", func(t *testing.T) {
		bc := NewBitCrusher(sampleRate)
		bc.SetMix(1.0)
		bc.SetBitDepth(24) // High bit depth to isolate sample rate effect
		bc.SetSampleRateRatio(0.25) // 1/4 sample rate
		bc.SetAntiAlias(false) // Disable for clearer test
		
		// Process several samples
		inputs := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
		outputs := make([]float64, len(inputs))
		
		for i, in := range inputs {
			outputs[i] = bc.Process(in)
		}
		
		// Should see held samples (stair-stepping)
		changesCount := 0
		for i := 1; i < len(outputs); i++ {
			if outputs[i] != outputs[i-1] {
				changesCount++
			}
		}
		
		// With 0.25 ratio, expect about 1/4 as many changes
		expectedChanges := len(inputs) / 4
		if changesCount > expectedChanges+1 {
			t.Errorf("Too many sample changes for 0.25 ratio: %d (expected ~%d)", changesCount, expectedChanges)
		}
	})

	t.Run("Anti-Aliasing", func(t *testing.T) {
		bc := NewBitCrusher(sampleRate)
		bc.SetMix(1.0)
		bc.SetBitDepth(16)
		bc.SetSampleRateRatio(0.1) // Heavy decimation
		
		// Create high frequency input
		nyquist := sampleRate / 2
		testFreq := nyquist * 0.8 // High frequency
		omega := 2.0 * math.Pi * testFreq / sampleRate
		
		// Test with anti-aliasing off
		bc.SetAntiAlias(false)
		var energyNoAA float64
		for i := 0; i < 100; i++ {
			input := math.Sin(omega * float64(i))
			output := bc.Process(input)
			energyNoAA += output * output
		}
		
		// Test with anti-aliasing on
		bc.SetAntiAlias(true)
		var energyWithAA float64
		for i := 0; i < 100; i++ {
			input := math.Sin(omega * float64(i))
			output := bc.Process(input)
			energyWithAA += output * output
		}
		
		// Anti-aliasing should reduce high frequency energy
		if energyWithAA >= energyNoAA {
			t.Errorf("Anti-aliasing should reduce high frequency energy: %f vs %f", energyWithAA, energyNoAA)
		}
	})

	t.Run("Dithering", func(t *testing.T) {
		bc := NewBitCrusher(sampleRate)
		bc.SetMix(1.0)
		bc.SetBitDepth(2) // Very low bit depth to make dither obvious
		bc.SetSampleRateRatio(1.0)
		
		// Test quantization without dither
		bc.SetDither(0.0)
		input := 0.1 // Small signal that might get quantized to 0
		
		noDitherOutputs := make([]float64, 10)
		for i := range noDitherOutputs {
			noDitherOutputs[i] = bc.Process(input)
		}
		
		// All outputs should be the same without dither
		for i := 1; i < len(noDitherOutputs); i++ {
			if noDitherOutputs[i] != noDitherOutputs[0] {
				t.Errorf("Without dither, outputs should be identical")
			}
		}
		
		// Test with dither
		bc.SetDither(0.5)
		ditherOutputs := make([]float64, 10)
		for i := range ditherOutputs {
			ditherOutputs[i] = bc.Process(input)
		}
		
		// With dither, should see some variation
		hasVariation := false
		for i := 1; i < len(ditherOutputs); i++ {
			if ditherOutputs[i] != ditherOutputs[0] {
				hasVariation = true
				break
			}
		}
		
		if !hasVariation {
			t.Errorf("With dither, outputs should vary")
		}
	})

	t.Run("DC Blocking", func(t *testing.T) {
		bc := NewBitCrusher(sampleRate)
		bc.SetMix(1.0)
		bc.SetBitDepth(4)
		
		// Process a signal with DC offset
		dcOffset := 0.5
		var avgOutput float64
		samples := 1000
		
		for i := 0; i < samples; i++ {
			output := bc.Process(dcOffset)
			if i > 100 { // Let filter settle
				avgOutput += output
			}
		}
		
		avgOutput /= float64(samples - 100)
		
		// Average should be near zero (DC removed)
		if math.Abs(avgOutput) > 0.1 {
			t.Errorf("DC blocker should remove offset: avg = %f", avgOutput)
		}
	})

	t.Run("ProcessBuffer", func(t *testing.T) {
		bc := NewBitCrusher(sampleRate)
		bc.SetBitDepth(8)
		bc.SetSampleRateRatio(0.5)
		bc.SetMix(1.0)
		
		input := []float64{-0.8, -0.4, 0.0, 0.4, 0.8}
		output := make([]float64, len(input))
		
		bc.ProcessBuffer(input, output)
		
		// Verify all outputs are valid
		for i, out := range output {
			if math.IsNaN(out) || math.IsInf(out, 0) {
				t.Errorf("Output should be finite: got %f at index %d", out, i)
			}
			if math.Abs(out) > 1.0 {
				t.Errorf("Output should be bounded: got %f at index %d", out, i)
			}
		}
	})
}

func TestBitCrusherWithModulation(t *testing.T) {
	sampleRate := 48000.0

	t.Run("Bit Depth Modulation", func(t *testing.T) {
		bcm := NewBitCrusherWithModulation(sampleRate)
		bcm.SetMix(1.0)
		bcm.SetBaseBitDepth(8.0)
		
		input := 0.5
		
		// Test no modulation
		bcm.ModulateBitDepth(0.0)
		output1 := bcm.Process(input)
		
		// Test positive modulation
		bcm.ModulateBitDepth(0.5)
		output2 := bcm.Process(input)
		
		// Test negative modulation
		bcm.ModulateBitDepth(-0.5)
		output3 := bcm.Process(input)
		
		// Different modulations should produce different outputs
		if output1 == output2 || output1 == output3 || output2 == output3 {
			t.Errorf("Different bit depth modulations should produce different outputs")
		}
	})

	t.Run("Sample Rate Modulation", func(t *testing.T) {
		bcm := NewBitCrusherWithModulation(sampleRate)
		bcm.SetMix(1.0)
		bcm.SetBaseSampleRateRatio(0.5)
		bcm.SetBitDepth(16) // Fixed bit depth
		
		// Process with different modulations
		testSignal := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
		
		// No modulation
		bcm.ModulateSampleRate(0.0)
		outputs1 := make([]float64, len(testSignal))
		for i, in := range testSignal {
			outputs1[i] = bcm.Process(in)
		}
		
		// Positive modulation (higher sample rate)
		bcm.ModulateSampleRate(0.5)
		outputs2 := make([]float64, len(testSignal))
		for i, in := range testSignal {
			outputs2[i] = bcm.Process(in)
		}
		
		// Results should differ
		different := false
		for i := range outputs1 {
			if outputs1[i] != outputs2[i] {
				different = true
				break
			}
		}
		
		if !different {
			t.Errorf("Sample rate modulation should affect output")
		}
	})
}

func TestDCBlocker(t *testing.T) {
	dc := NewDCBlocker()
	
	// Feed constant DC signal
	dcLevel := 0.7
	var lastOutput float64
	
	// Process enough samples for settling
	for i := 0; i < 1000; i++ {
		lastOutput = dc.Process(dcLevel)
	}
	
	// Output should converge to near zero
	if math.Abs(lastOutput) > 0.01 {
		t.Errorf("DC blocker should remove DC offset: got %f", lastOutput)
	}
	
	// Test that AC signals pass through
	// Reset the filter
	dc = NewDCBlocker()
	
	// Process sine wave
	freq := 1000.0
	sampleRate := 48000.0
	omega := 2.0 * math.Pi * freq / sampleRate
	
	var inputEnergy, outputEnergy float64
	for i := 0; i < 100; i++ {
		input := math.Sin(omega * float64(i))
		output := dc.Process(input)
		
		inputEnergy += input * input
		outputEnergy += output * output
	}
	
	// AC signal should mostly pass through
	ratio := outputEnergy / inputEnergy
	if ratio < 0.9 {
		t.Errorf("DC blocker should pass AC signals: energy ratio = %f", ratio)
	}
}

func BenchmarkBitCrusher(b *testing.B) {
	sampleRate := 48000.0
	bc := NewBitCrusher(sampleRate)
	bc.SetBitDepth(8)
	bc.SetSampleRateRatio(0.25)
	bc.SetMix(1.0)
	bc.SetAntiAlias(true)
	
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
		bc.ProcessBuffer(input, output)
	}
}