package modulation

import (
	"math"
	"testing"
)

func TestChorusCreation(t *testing.T) {
	sampleRate := 48000.0
	chorus := NewChorus(sampleRate)

	if chorus == nil {
		t.Fatal("Failed to create chorus")
	}

	if chorus.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", chorus.sampleRate, sampleRate)
	}

	// Check defaults
	if chorus.voices != 2 {
		t.Errorf("Default voices incorrect: got %d, want 2", chorus.voices)
	}

	if chorus.mix != 0.5 {
		t.Errorf("Default mix incorrect: got %f, want 0.5", chorus.mix)
	}
}

func TestChorusDrySignal(t *testing.T) {
	chorus := NewChorus(48000.0)
	chorus.SetMix(0.0) // Completely dry

	input := float32(0.5)
	outputL, outputR := chorus.Process(input)

	// Should be unchanged
	if math.Abs(float64(outputL-input)) > 0.001 {
		t.Errorf("Dry signal altered in left channel: input %f, output %f", input, outputL)
	}

	if math.Abs(float64(outputR-input)) > 0.001 {
		t.Errorf("Dry signal altered in right channel: input %f, output %f", input, outputR)
	}
}

func TestChorusWetSignal(t *testing.T) {
	chorus := NewChorus(48000.0)
	chorus.SetMix(1.0)      // Completely wet
	chorus.SetDelay(10.0)   // 10ms delay
	chorus.SetDepth(0.0)    // No modulation
	chorus.SetFeedback(0.0) // No feedback

	// Send impulse
	impulse := float32(1.0)
	outputL1, _ := chorus.Process(impulse)

	// First output should be near zero (signal is delayed)
	if math.Abs(float64(outputL1)) > 0.1 {
		t.Errorf("Wet signal not delayed: %f", outputL1)
	}

	// Process silence for delay time
	delaySamples := int(10.0 * 48000.0 / 1000.0)
	var outputL float32

	for i := 0; i < delaySamples; i++ {
		outputL, _ = chorus.Process(0.0)
	}

	// Should now hear the delayed impulse
	if outputL < 0.1 {
		t.Errorf("Delayed signal not appearing: %f", outputL)
	}
}

func TestChorusVoices(t *testing.T) {
	chorus := NewChorus(48000.0)

	// Test voice limits
	chorus.SetVoices(0)
	if chorus.voices != 1 {
		t.Errorf("Voices below minimum: %d", chorus.voices)
	}

	chorus.SetVoices(10)
	if chorus.voices != 4 {
		t.Errorf("Voices above maximum: %d", chorus.voices)
	}

	// Test that different voice counts work
	for v := 1; v <= 4; v++ {
		chorus.SetVoices(v)
		chorus.SetMix(1.0)

		// Process some samples
		for i := 0; i < 100; i++ {
			chorus.Process(0.5)
		}

		// Should not crash or produce NaN
		outputL, outputR := chorus.Process(0.5)
		if math.IsNaN(float64(outputL)) || math.IsNaN(float64(outputR)) {
			t.Errorf("NaN output with %d voices", v)
		}
	}
}

func TestChorusModulation(t *testing.T) {
	chorus := NewChorus(48000.0)
	chorus.SetMix(1.0)
	chorus.SetRate(5.0)   // 5Hz modulation
	chorus.SetDepth(5.0)  // 5ms depth
	chorus.SetDelay(20.0) // 20ms base delay

	// Process a constant signal
	samples := 48000 // 1 second
	outputs := make([]float32, samples)

	for i := 0; i < samples; i++ {
		outputs[i], _ = chorus.Process(0.5)
	}

	// Find variations in output (should see modulation effects)
	minVal := float32(1.0)
	maxVal := float32(-1.0)

	// Skip initial delay
	for i := 1000; i < samples; i++ {
		if outputs[i] < minVal {
			minVal = outputs[i]
		}
		if outputs[i] > maxVal {
			maxVal = outputs[i]
		}
	}

	// Should see some variation due to modulation
	variation := maxVal - minVal
	if variation < 0.01 {
		t.Errorf("No modulation detected: variation = %f", variation)
	}
}

func TestChorusStereoSpread(t *testing.T) {
	chorus := NewChorus(48000.0)
	chorus.SetVoices(3) // Use 3 voices for clearer stereo difference
	chorus.SetMix(1.0)
	chorus.SetSpread(1.0) // Full spread
	chorus.SetDepth(3.0)  // More modulation depth
	chorus.SetRate(2.0)   // Faster rate

	// Send an impulse to create clear signal
	chorus.Process(1.0)

	// Process some samples
	for i := 0; i < 2000; i++ {
		chorus.Process(0.0)
	}

	// Now send another impulse and check stereo field
	outputL, outputR := chorus.Process(1.0)

	// Process more samples and accumulate difference
	totalDiff := float64(0)
	for i := 0; i < 1000; i++ {
		outputL, outputR = chorus.Process(0.0)
		totalDiff += math.Abs(float64(outputL - outputR))
	}

	// With 3 voices and spread, there should be stereo difference
	if totalDiff < 0.1 {
		t.Errorf("No stereo spread detected, total difference: %f", totalDiff)
	}

	// Test with no spread
	chorus.SetSpread(0.0)

	// Process some more samples
	for i := 0; i < 1000; i++ {
		chorus.Process(0.5)
	}

	outL, outR := chorus.Process(0.5)

	// Without spread, should be more similar
	if math.Abs(float64(outL-outR)) > 0.1 {
		t.Error("Stereo spread still active when set to 0")
	}
}

func TestChorusFeedback(t *testing.T) {
	chorus := NewChorus(48000.0)
	chorus.SetMix(1.0)
	chorus.SetFeedback(0.5) // Maximum feedback
	chorus.SetDelay(5.0)    // Short delay
	chorus.SetDepth(0.0)    // No modulation

	// Send impulse
	chorus.Process(1.0)

	// Process and accumulate output
	delaySamples := int(5.0 * 48000.0 / 1000.0)
	totalOutput := float32(0)

	for i := 0; i < delaySamples*10; i++ {
		output, _ := chorus.Process(0.0)
		totalOutput += float32(math.Abs(float64(output)))
	}

	// With feedback, total output should be more than without
	// Reset and test without feedback
	chorus.Reset()
	chorus.SetFeedback(0.0)

	chorus.Process(1.0)
	totalOutputNoFeedback := float32(0)

	for i := 0; i < delaySamples*10; i++ {
		output, _ := chorus.Process(0.0)
		totalOutputNoFeedback += float32(math.Abs(float64(output)))
	}

	if totalOutput <= totalOutputNoFeedback {
		t.Error("Feedback not increasing output energy")
	}
}

func TestChorusReset(t *testing.T) {
	chorus := NewChorus(48000.0)

	// Process some signal
	for i := 0; i < 1000; i++ {
		chorus.Process(0.5)
	}

	// Reset
	chorus.Reset()

	// Output should be silent (only dry signal)
	chorus.SetMix(1.0) // Full wet
	outputL, outputR := chorus.Process(0.0)

	if math.Abs(float64(outputL)) > 0.001 || math.Abs(float64(outputR)) > 0.001 {
		t.Errorf("Chorus not silent after reset: L=%f, R=%f", outputL, outputR)
	}
}

func TestChorusParameterLimits(t *testing.T) {
	chorus := NewChorus(48000.0)

	// Test rate limits
	chorus.SetRate(-1.0)
	if chorus.rate < 0.01 {
		t.Errorf("Rate below minimum: %f", chorus.rate)
	}

	chorus.SetRate(100.0)
	if chorus.rate > 10.0 {
		t.Errorf("Rate above maximum: %f", chorus.rate)
	}

	// Test depth limits
	chorus.SetDepth(-5.0)
	if chorus.depth < 0.0 {
		t.Errorf("Depth below minimum: %f", chorus.depth)
	}

	chorus.SetDepth(50.0)
	if chorus.depth > 10.0 {
		t.Errorf("Depth above maximum: %f", chorus.depth)
	}

	// Test delay limits
	chorus.SetDelay(0.1)
	if chorus.delay < 1.0 {
		t.Errorf("Delay below minimum: %f", chorus.delay)
	}

	chorus.SetDelay(100.0)
	if chorus.delay > 50.0 {
		t.Errorf("Delay above maximum: %f", chorus.delay)
	}
}

// Benchmark chorus
func BenchmarkChorus(b *testing.B) {
	chorus := NewChorus(48000.0)
	chorus.SetVoices(2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chorus.Process(0.5)
	}
}

func BenchmarkChorusStereo(b *testing.B) {
	chorus := NewChorus(48000.0)
	chorus.SetVoices(4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chorus.ProcessStereo(0.5, 0.5)
	}
}
