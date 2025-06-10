package reverb

import (
	"math"
	"testing"
)

func TestCombFilterCreation(t *testing.T) {
	comb := NewCombFilter(1000)

	if comb == nil {
		t.Fatal("Failed to create comb filter")
	}

	if len(comb.buffer) != 1000 {
		t.Errorf("Buffer size mismatch: got %d, want 1000", len(comb.buffer))
	}

	if comb.feedback != 0.5 {
		t.Errorf("Default feedback incorrect: got %f, want 0.5", comb.feedback)
	}
}

func TestCombFilterProcess(t *testing.T) {
	comb := NewCombFilter(100)
	comb.SetFeedback(0.7)
	comb.SetDamping(0.3)

	// Process impulse
	output := comb.Process(1.0)

	// First output should be 0 (empty buffer)
	if output != 0.0 {
		t.Errorf("Initial output not zero: %f", output)
	}

	// Process zeros and collect output
	outputs := make([]float32, 200)
	for i := 0; i < 200; i++ {
		outputs[i] = comb.Process(0.0)
	}

	// Should see the impulse come back after delay
	if outputs[99] == 0.0 {
		t.Error("No delayed output detected")
	}

	// Should see feedback (decreasing amplitude)
	if outputs[199] == 0.0 {
		t.Error("No feedback detected")
	}

	// Feedback should cause decay
	if math.Abs(float64(outputs[199])) >= math.Abs(float64(outputs[99])) {
		t.Error("Feedback not causing decay")
	}
}

func TestAllPassFilterCreation(t *testing.T) {
	allpass := NewAllPassFilter(500)

	if allpass == nil {
		t.Fatal("Failed to create all-pass filter")
	}

	if len(allpass.buffer) != 500 {
		t.Errorf("Buffer size mismatch: got %d, want 500", len(allpass.buffer))
	}
}

func TestAllPassFilterProcess(t *testing.T) {
	allpass := NewAllPassFilter(50)
	allpass.SetFeedback(0.5)

	// Process impulse
	output := allpass.Process(1.0)

	// All-pass should pass the signal but with phase shift
	// First output = -input (for impulse)
	if output != -1.0 {
		t.Errorf("Initial output incorrect: got %f, want -1.0", output)
	}

	// Process zeros and accumulate energy
	totalEnergy := float64(output * output)
	for i := 0; i < 100; i++ {
		out := allpass.Process(0.0)
		totalEnergy += float64(out * out)
	}

	// All-pass filter should roughly preserve energy
	// With feedback, energy may be slightly higher
	inputEnergy := 1.0 // Single impulse
	// Allow for some variance due to feedback
	if totalEnergy < inputEnergy*0.8 || totalEnergy > inputEnergy*3.0 {
		t.Errorf("Energy outside expected range: input=%f, output=%f", inputEnergy, totalEnergy)
	}
}

func TestSchroederCreation(t *testing.T) {
	reverb := NewSchroeder(48000.0)

	if reverb == nil {
		t.Fatal("Failed to create Schroeder reverb")
	}

	// Check default parameters
	if reverb.roomSize != 0.5 {
		t.Errorf("Default room size incorrect: got %f, want 0.5", reverb.roomSize)
	}

	if reverb.damping != 0.5 {
		t.Errorf("Default damping incorrect: got %f, want 0.5", reverb.damping)
	}

	if reverb.wetLevel != 0.3 {
		t.Errorf("Default wet level incorrect: got %f, want 0.3", reverb.wetLevel)
	}

	// Check that filters were created
	for i := 0; i < 4; i++ {
		if reverb.combs[i] == nil {
			t.Errorf("Comb filter %d not created", i)
		}
	}

	for i := 0; i < 2; i++ {
		if reverb.allpasses[i] == nil {
			t.Errorf("All-pass filter %d not created", i)
		}
	}
}

func TestSchroederDrySignal(t *testing.T) {
	reverb := NewSchroeder(48000.0)
	reverb.SetWetLevel(0.0)
	reverb.SetDryLevel(1.0)

	input := float32(0.5)
	output := reverb.Process(input)

	// Should pass through unchanged
	if math.Abs(float64(output-input)) > 0.001 {
		t.Errorf("Dry signal altered: input=%f, output=%f", input, output)
	}
}

func TestSchroederWetSignal(t *testing.T) {
	reverb := NewSchroeder(48000.0)
	reverb.SetWetLevel(1.0)
	reverb.SetDryLevel(0.0)

	// Process impulse
	output := reverb.Process(1.0)

	// First output should be near zero (processing delay)
	if math.Abs(float64(output)) > 0.1 {
		t.Errorf("Initial wet output too high: %f", output)
	}

	// Process zeros and collect tail
	tailEnergy := float64(0.0)
	for i := 0; i < 48000; i++ { // 1 second
		out := reverb.Process(0.0)
		tailEnergy += float64(out * out)
	}

	// Should have reverb tail energy
	if tailEnergy < 0.1 {
		t.Error("No reverb tail detected")
	}
}

func TestSchroederRoomSize(t *testing.T) {
	reverb := NewSchroeder(48000.0)
	reverb.SetWetLevel(1.0)
	reverb.SetDryLevel(0.0)

	// Test small room
	reverb.SetRoomSize(0.1)
	reverb.Process(1.0) // Impulse

	smallRoomEnergy := float64(0.0)
	for i := 0; i < 24000; i++ { // 0.5 seconds
		out := reverb.Process(0.0)
		smallRoomEnergy += float64(out * out)
	}

	// Reset and test large room
	reverb.Reset()
	reverb.SetRoomSize(0.9)
	reverb.Process(1.0) // Impulse

	largeRoomEnergy := float64(0.0)
	for i := 0; i < 24000; i++ { // 0.5 seconds
		out := reverb.Process(0.0)
		largeRoomEnergy += float64(out * out)
	}

	// Large room should have more energy (longer decay)
	if largeRoomEnergy <= smallRoomEnergy {
		t.Errorf("Large room not producing longer decay: small=%f, large=%f",
			smallRoomEnergy, largeRoomEnergy)
	}
}

func TestSchroederDamping(t *testing.T) {
	reverb := NewSchroeder(48000.0)
	reverb.SetWetLevel(1.0)
	reverb.SetDryLevel(0.0)
	reverb.SetRoomSize(0.8)

	// Test low damping (bright)
	reverb.SetDamping(0.1)
	reverb.Process(1.0) // Impulse

	brightTail := make([]float32, 4800) // 100ms
	for i := range brightTail {
		brightTail[i] = reverb.Process(0.0)
	}

	// Reset and test high damping (dark)
	reverb.Reset()
	reverb.SetDamping(0.9)
	reverb.Process(1.0) // Impulse

	darkTail := make([]float32, 4800) // 100ms
	for i := range darkTail {
		darkTail[i] = reverb.Process(0.0)
	}

	// High damping should reduce high frequency content
	// We can approximate this by checking if the dark tail is "smoother"
	brightVariance := calculateVariance(brightTail)
	darkVariance := calculateVariance(darkTail)

	if darkVariance >= brightVariance {
		t.Error("High damping not reducing high frequency content")
	}
}

func TestSchroederStereo(t *testing.T) {
	reverb := NewSchroeder(48000.0)
	reverb.SetWetLevel(1.0)
	reverb.SetDryLevel(0.0)
	reverb.SetWidth(1.0)

	// Process centered impulse
	outputL, outputR := reverb.ProcessStereo(1.0, 1.0)

	// Initial outputs should be similar
	if math.Abs(float64(outputL-outputR)) > 0.1 {
		t.Errorf("Initial stereo outputs differ too much: L=%f, R=%f", outputL, outputR)
	}

	// Process and check for stereo decorrelation
	correlation := float64(0.0)
	energyL := float64(0.0)
	energyR := float64(0.0)

	for i := 0; i < 4800; i++ { // 100ms
		outL, outR := reverb.ProcessStereo(0.0, 0.0)
		correlation += float64(outL * outR)
		energyL += float64(outL * outL)
		energyR += float64(outR * outR)
	}

	// Channels should have similar energy
	energyRatio := energyL / energyR
	if energyRatio < 0.8 || energyRatio > 1.2 {
		t.Errorf("Channel energy imbalance: ratio=%f", energyRatio)
	}
}

func TestSchroederReset(t *testing.T) {
	reverb := NewSchroeder(48000.0)

	// Process some signal
	for i := 0; i < 1000; i++ {
		reverb.Process(0.5)
	}

	// Reset
	reverb.Reset()

	// Output should be zero
	output := reverb.Process(0.0)
	if output != 0.0 {
		t.Errorf("Output not zero after reset: %f", output)
	}
}

func TestSchroederParameterLimits(t *testing.T) {
	reverb := NewSchroeder(48000.0)

	// Test room size limits
	reverb.SetRoomSize(-0.5)
	if reverb.roomSize < 0.0 {
		t.Errorf("Room size below minimum: %f", reverb.roomSize)
	}

	reverb.SetRoomSize(1.5)
	if reverb.roomSize > 1.0 {
		t.Errorf("Room size above maximum: %f", reverb.roomSize)
	}

	// Test damping limits
	reverb.SetDamping(-0.5)
	if reverb.damping < 0.0 {
		t.Errorf("Damping below minimum: %f", reverb.damping)
	}

	reverb.SetDamping(1.5)
	if reverb.damping > 1.0 {
		t.Errorf("Damping above maximum: %f", reverb.damping)
	}

	// Test level limits
	reverb.SetWetLevel(2.0)
	if reverb.wetLevel > 1.0 {
		t.Errorf("Wet level above maximum: %f", reverb.wetLevel)
	}

	reverb.SetDryLevel(-1.0)
	if reverb.dryLevel < 0.0 {
		t.Errorf("Dry level below minimum: %f", reverb.dryLevel)
	}
}

// Helper function to calculate variance
func calculateVariance(samples []float32) float64 {
	if len(samples) == 0 {
		return 0
	}

	// Calculate mean
	sum := float64(0.0)
	for _, s := range samples {
		sum += float64(s)
	}
	mean := sum / float64(len(samples))

	// Calculate variance
	variance := float64(0.0)
	for _, s := range samples {
		diff := float64(s) - mean
		variance += diff * diff
	}

	return variance / float64(len(samples))
}

// Benchmarks
func BenchmarkSchroederMono(b *testing.B) {
	reverb := NewSchroeder(48000.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reverb.Process(0.5)
	}
}

func BenchmarkSchroederStereo(b *testing.B) {
	reverb := NewSchroeder(48000.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reverb.ProcessStereo(0.5, 0.5)
	}
}
