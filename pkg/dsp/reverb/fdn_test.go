package reverb

import (
	"math"
	"testing"
)

func TestFDNCreation(t *testing.T) {
	// Test with different numbers of delay lines
	configs := []int{4, 8, 16}
	
	for _, numDelays := range configs {
		fdn := NewFDN(numDelays, 44100)
		
		if fdn == nil {
			t.Fatalf("Failed to create FDN with %d delays", numDelays)
		}
		
		if fdn.numDelays != numDelays {
			t.Errorf("Expected %d delays, got %d", numDelays, fdn.numDelays)
		}
		
		// Check that delay lines are properly initialized
		if len(fdn.delayLines) != numDelays {
			t.Errorf("Expected %d delay lines, got %d", numDelays, len(fdn.delayLines))
		}
		
		// Check feedback matrix dimensions
		if len(fdn.feedbackMatrix) != numDelays {
			t.Errorf("Expected %dx%d feedback matrix, got %d rows", numDelays, numDelays, len(fdn.feedbackMatrix))
		}
	}
}

func TestFDNParameterRanges(t *testing.T) {
	fdn := NewFDN(4, 44100)
	
	// Test decay clamping
	fdn.SetDecay(2.0)
	if fdn.decay != 1.0 {
		t.Errorf("Decay should be clamped to 1.0, got %f", fdn.decay)
	}
	
	fdn.SetDecay(-1.0)
	if fdn.decay != 0.0 {
		t.Errorf("Decay should be clamped to 0.0, got %f", fdn.decay)
	}
	
	// Test damping clamping
	fdn.SetDamping(2.0)
	if fdn.damping != 1.0 {
		t.Errorf("Damping should be clamped to 1.0, got %f", fdn.damping)
	}
	
	// Test diffusion clamping
	fdn.SetDiffusion(1.5)
	if fdn.diffusion != 1.0 {
		t.Errorf("Diffusion should be clamped to 1.0, got %f", fdn.diffusion)
	}
}

func TestFDNProcessing(t *testing.T) {
	fdn := NewFDN(4, 44100)
	
	// Test with silence
	output := fdn.Process(0.0)
	if output != 0.0 {
		t.Error("FDN should output silence for silent input initially")
	}
	
	// Test with impulse
	output = fdn.Process(1.0)
	if math.IsNaN(float64(output)) {
		t.Error("FDN output should not be NaN")
	}
	
	// Process more samples and check for reverb tail
	hasReverb := false
	for i := 0; i < 1000; i++ {
		output = fdn.Process(0.0)
		if output != 0.0 {
			hasReverb = true
			break
		}
	}
	
	if !hasReverb {
		t.Error("FDN should produce a reverb tail after impulse")
	}
}

func TestFDNStereoProcessing(t *testing.T) {
	fdn := NewFDN(4, 44100)
	
	// Process stereo impulse
	outL, outR := fdn.ProcessStereo(1.0, 1.0)
	
	if math.IsNaN(float64(outL)) || math.IsNaN(float64(outR)) {
		t.Error("FDN stereo output should not contain NaN")
	}
	
	// Process more samples
	for i := 0; i < 100; i++ {
		outL, outR = fdn.ProcessStereo(0.0, 0.0)
	}
	
	// Outputs should have some decorrelation
	if outL == outR {
		t.Log("Warning: Stereo outputs are identical, may lack decorrelation")
	}
}

func TestFDNReset(t *testing.T) {
	fdn := NewFDN(4, 44100)
	
	// Process an impulse
	fdn.Process(1.0)
	
	// Process a few more samples to build up reverb
	for i := 0; i < 100; i++ {
		fdn.Process(0.0)
	}
	
	// Reset
	fdn.Reset()
	
	// Check that output is zero after reset
	output := fdn.Process(0.0)
	if output != 0.0 {
		t.Error("FDN should output silence after reset")
	}
}

func TestFDNModulation(t *testing.T) {
	fdn := NewFDN(4, 44100)
	
	// Enable modulation
	fdn.SetModulation(1.0)
	
	// Process an impulse
	fdn.Process(1.0)
	
	// Collect samples over time
	samples := make([]float32, 1000)
	for i := 0; i < 1000; i++ {
		samples[i] = fdn.Process(0.0)
	}
	
	// With modulation, the signal should vary more over time
	// Calculate variance
	var sum, sumSq float64
	for _, s := range samples {
		sum += float64(s)
		sumSq += float64(s) * float64(s)
	}
	mean := sum / float64(len(samples))
	variance := sumSq/float64(len(samples)) - mean*mean
	
	if variance == 0 {
		t.Error("With modulation enabled, output should vary over time")
	}
}

func TestFDNPresets(t *testing.T) {
	fdn := NewFDN(4, 44100)
	
	// Test small room preset
	fdn.SetPresetSmallRoom()
	if fdn.decay != 0.2 {
		t.Errorf("Small room preset: expected decay 0.2, got %f", fdn.decay)
	}
	if fdn.damping != 0.8 {
		t.Errorf("Small room preset: expected damping 0.8, got %f", fdn.damping)
	}
	
	// Test cathedral preset
	fdn.SetPresetCathedral()
	if fdn.decay != 0.95 {
		t.Errorf("Cathedral preset: expected decay 0.95, got %f", fdn.decay)
	}
	if fdn.damping != 0.1 {
		t.Errorf("Cathedral preset: expected damping 0.1, got %f", fdn.damping)
	}
}

func TestDampingFilter(t *testing.T) {
	df := NewDampingFilter()
	
	// Test with no damping
	df.SetDamping(0.0)
	output := df.Process(1.0)
	if output != 1.0 {
		t.Error("With no damping, filter should pass signal unchanged")
	}
	
	// Test with maximum damping
	df.Reset()
	df.SetDamping(1.0)
	
	// Process several samples
	for i := 0; i < 10; i++ {
		output = df.Process(1.0)
	}
	
	// Output should be attenuated
	if output >= 1.0 {
		t.Error("With maximum damping, filter should attenuate signal")
	}
}

func TestFDNFeedbackMatrix(t *testing.T) {
	// Test 4x4 Hadamard matrix
	fdn4 := NewFDN(4, 44100)
	
	// Check matrix is orthogonal (simplified check)
	// Sum of squares in each row should be 1
	for i := 0; i < 4; i++ {
		sum := 0.0
		for j := 0; j < 4; j++ {
			sum += fdn4.feedbackMatrix[i][j] * fdn4.feedbackMatrix[i][j]
		}
		if math.Abs(sum-1.0) > 0.1 {
			t.Errorf("Row %d: sum of squares = %f, expected ~1.0", i, sum)
		}
	}
	
	// Test other sizes
	fdn8 := NewFDN(8, 44100)
	if len(fdn8.feedbackMatrix) != 8 {
		t.Error("8x8 feedback matrix not created properly")
	}
}

func TestFDNDelayTimes(t *testing.T) {
	fdn := NewFDN(4, 44100)
	
	// Check that delay times are different (for good diffusion)
	times := make(map[int]bool)
	for i := 0; i < fdn.numDelays; i++ {
		if times[fdn.delayTimes[i]] {
			t.Error("Delay times should be unique for good diffusion")
		}
		times[fdn.delayTimes[i]] = true
		
		// Check delay times are reasonable (10-100ms range)
		delayMs := float64(fdn.delayTimes[i]) / 44.1
		if delayMs < 5 || delayMs > 200 {
			t.Errorf("Delay time %f ms is outside expected range", delayMs)
		}
	}
}

func BenchmarkFDNMono(b *testing.B) {
	fdn := NewFDN(8, 44100)
	fdn.SetPresetMediumHall()
	
	// Create test buffer
	input := make([]float32, 512)
	output := make([]float32, 512)
	
	// Fill with test signal
	for i := range input {
		input[i] = float32(i%100) / 100.0
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		for j := 0; j < 512; j++ {
			output[j] = fdn.Process(input[j])
		}
	}
}

func BenchmarkFDNStereo(b *testing.B) {
	fdn := NewFDN(8, 44100)
	fdn.SetPresetMediumHall()
	
	// Create test buffers
	inputL := make([]float32, 512)
	inputR := make([]float32, 512)
	outputL := make([]float32, 512)
	outputR := make([]float32, 512)
	
	// Fill with test signal
	for i := range inputL {
		inputL[i] = float32(i%100) / 100.0
		inputR[i] = float32(i%100) / 100.0
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		for j := 0; j < 512; j++ {
			outputL[j], outputR[j] = fdn.ProcessStereo(inputL[j], inputR[j])
		}
	}
}