package reverb

import (
	"math"
	"testing"
)

func TestFreeverbCreation(t *testing.T) {
	reverb := NewFreeverb(44100)

	if reverb == nil {
		t.Fatal("Failed to create Freeverb instance")
	}

	// Check initial parameters
	if reverb.roomSize != initialRoom {
		t.Errorf("Expected initial room size %f, got %f", initialRoom, reverb.roomSize)
	}

	if reverb.damping != initialDamp {
		t.Errorf("Expected initial damping %f, got %f", initialDamp, reverb.damping)
	}
}

func TestFreeverbParameterRanges(t *testing.T) {
	reverb := NewFreeverb(44100)

	// Test room size clamping
	reverb.SetRoomSize(2.0)
	if reverb.roomSize != 1.0 {
		t.Errorf("Room size should be clamped to 1.0, got %f", reverb.roomSize)
	}

	reverb.SetRoomSize(-1.0)
	if reverb.roomSize != 0.0 {
		t.Errorf("Room size should be clamped to 0.0, got %f", reverb.roomSize)
	}

	// Test damping clamping
	reverb.SetDamping(2.0)
	if reverb.damping != 1.0 {
		t.Errorf("Damping should be clamped to 1.0, got %f", reverb.damping)
	}

	reverb.SetDamping(-1.0)
	if reverb.damping != 0.0 {
		t.Errorf("Damping should be clamped to 0.0, got %f", reverb.damping)
	}
}

func TestFreeverbProcessing(t *testing.T) {
	reverb := NewFreeverb(44100)

	// Test with silence
	outL, outR := reverb.ProcessStereo(0.0, 0.0)
	if outL != 0.0 || outR != 0.0 {
		t.Error("Reverb should output silence for silent input initially")
	}

	// Test with impulse
	outL, outR = reverb.ProcessStereo(1.0, 1.0)
	if math.IsNaN(float64(outL)) || math.IsNaN(float64(outR)) {
		t.Error("Reverb output should not be NaN")
	}

	// Process more samples and check for reverb tail
	hasReverb := false
	for i := 0; i < 1000; i++ {
		outL, outR = reverb.ProcessStereo(0.0, 0.0)
		if outL != 0.0 || outR != 0.0 {
			hasReverb = true
			break
		}
	}

	if !hasReverb {
		t.Error("Reverb should produce a tail after impulse")
	}
}

func TestFreeverbReset(t *testing.T) {
	reverb := NewFreeverb(44100)

	// Process an impulse
	reverb.ProcessStereo(1.0, 1.0)

	// Process a few more samples to build up reverb
	for i := 0; i < 100; i++ {
		reverb.ProcessStereo(0.0, 0.0)
	}

	// Reset
	reverb.Reset()

	// Check that output is zero after reset
	outL, outR := reverb.ProcessStereo(0.0, 0.0)
	if outL != 0.0 || outR != 0.0 {
		t.Error("Reverb should output silence after reset")
	}
}

func TestFreeverbFreezeMode(t *testing.T) {
	reverb := NewFreeverb(44100)

	// Process an impulse
	reverb.ProcessStereo(1.0, 1.0)

	// Enable freeze mode
	reverb.SetMode(1.0)

	// In freeze mode, the reverb should sustain indefinitely
	// Check that we still have output after many samples
	var lastOut float32
	for i := 0; i < 10000; i++ {
		outL, _ := reverb.ProcessStereo(0.0, 0.0)
		if i == 9999 {
			lastOut = outL
		}
	}

	if lastOut == 0.0 {
		t.Error("In freeze mode, reverb should sustain indefinitely")
	}
}

func TestFreeverbPresets(t *testing.T) {
	reverb := NewFreeverb(44100)

	// Test small room preset
	reverb.SetPresetSmallRoom()
	if reverb.roomSize != 0.3 {
		t.Errorf("Small room preset: expected room size 0.3, got %f", reverb.roomSize)
	}

	// Test medium hall preset
	reverb.SetPresetMediumHall()
	if reverb.roomSize != 0.6 {
		t.Errorf("Medium hall preset: expected room size 0.6, got %f", reverb.roomSize)
	}

	// Test large hall preset
	reverb.SetPresetLargeHall()
	if reverb.roomSize != 0.85 {
		t.Errorf("Large hall preset: expected room size 0.85, got %f", reverb.roomSize)
	}

	// Test cathedral preset
	reverb.SetPresetCathedral()
	if reverb.roomSize != 0.95 {
		t.Errorf("Cathedral preset: expected room size 0.95, got %f", reverb.roomSize)
	}
}

func TestFreeverbStereoWidth(t *testing.T) {
	reverb := NewFreeverb(44100)

	// Set to mono (width = 0)
	reverb.SetWidth(0.0)
	reverb.update()

	// Process a stereo signal
	reverb.ProcessStereo(1.0, -1.0)

	// Continue processing to let reverb build up
	var outL, outR float32
	for i := 0; i < 1000; i++ {
		outL, outR = reverb.ProcessStereo(0.0, 0.0)
	}

	// With width=0, outputs should be nearly identical
	diff := math.Abs(float64(outL - outR))
	if diff > 0.001 {
		t.Errorf("With width=0, outputs should be nearly identical, got difference: %f", diff)
	}

	// Set to full stereo (width = 1)
	reverb.SetWidth(1.0)
	reverb.Reset()

	// Process again
	reverb.ProcessStereo(1.0, -1.0)
	for i := 0; i < 1000; i++ {
		outL, outR = reverb.ProcessStereo(0.0, 0.0)
	}

	// With width=1, outputs should be different
	diff = math.Abs(float64(outL - outR))
	if diff < 0.001 {
		t.Error("With width=1, outputs should be different for stereo input")
	}
}

func TestFreeverbDifferentSampleRates(t *testing.T) {
	// Test at different sample rates
	sampleRates := []float64{44100, 48000, 88200, 96000}

	for _, sr := range sampleRates {
		reverb := NewFreeverb(sr)

		// Process an impulse
		outL, outR := reverb.ProcessStereo(1.0, 1.0)

		// Should not crash or produce NaN
		if math.IsNaN(float64(outL)) || math.IsNaN(float64(outR)) {
			t.Errorf("Reverb at %fHz produced NaN output", sr)
		}

		// Check that delay times are scaled properly
		// Higher sample rates should have proportionally longer delay buffers
		expectedScaling := sr / 44100.0
		actualDelay := len(reverb.combL[0].buffer)
		expectedDelay := int(float64(combTuning[0]) * expectedScaling)

		// Allow some rounding error
		if math.Abs(float64(actualDelay-expectedDelay)) > 1.0 {
			t.Errorf("At %fHz, expected delay ~%d samples, got %d", sr, expectedDelay, actualDelay)
		}
	}
}

func BenchmarkFreeverbStereo(b *testing.B) {
	reverb := NewFreeverb(44100)
	reverb.SetPresetMediumHall()

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
			outputL[j], outputR[j] = reverb.ProcessStereo(inputL[j], inputR[j])
		}
	}
}

func BenchmarkFreeverbMono(b *testing.B) {
	reverb := NewFreeverb(44100)
	reverb.SetPresetMediumHall()

	// Create test buffers
	input := make([]float32, 512)
	output := make([]float32, 512)

	// Fill with test signal
	for i := range input {
		input[i] = float32(i%100) / 100.0
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 512; j++ {
			output[j] = reverb.Process(input[j])
		}
	}
}
