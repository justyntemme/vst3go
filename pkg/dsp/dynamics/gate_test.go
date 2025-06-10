package dynamics

import (
	"math"
	"testing"
)

func TestGateCreation(t *testing.T) {
	sampleRate := 48000.0
	g := NewGate(sampleRate)

	if g == nil {
		t.Fatal("Failed to create gate")
	}

	if g.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", g.sampleRate, sampleRate)
	}

	// Check defaults
	if g.threshold != -40.0 {
		t.Errorf("Default threshold incorrect: got %f, want -40.0", g.threshold)
	}

	if g.hysteresis != 5.0 {
		t.Errorf("Default hysteresis incorrect: got %f, want 5.0", g.hysteresis)
	}

	if g.range_ != -80.0 {
		t.Errorf("Default range incorrect: got %f, want -80.0", g.range_)
	}
}

func TestGateBasicOperation(t *testing.T) {
	sampleRate := 48000.0
	g := NewGate(sampleRate)
	g.SetThreshold(-20.0)
	g.SetHysteresis(5.0)
	g.SetAttack(0.001)
	g.SetRelease(0.010)
	g.SetHold(0.0) // No hold for simple test

	// Test with signal below threshold
	quietSignal := float32(0.01) // ~-40 dB
	output := g.Process(quietSignal)

	// Should be attenuated
	if output >= quietSignal {
		t.Errorf("Gate not attenuating quiet signal: input %f, output %f", quietSignal, output)
	}

	// Test with signal above threshold
	loudSignal := float32(0.2) // ~-14 dB
	// Process for longer than attack time (1ms = 48 samples at 48kHz)
	for i := 0; i < 200; i++ {
		output = g.Process(loudSignal)
	}

	// After attack time, should be close to input
	if math.Abs(float64(output-loudSignal)) > 0.01 {
		t.Errorf("Gate not passing loud signal: input %f, output %f", loudSignal, output)
	}
}

func TestGateHysteresis(t *testing.T) {
	sampleRate := 48000.0
	g := NewGate(sampleRate)
	g.SetThreshold(-20.0)
	g.SetHysteresis(6.0) // 6dB hysteresis
	g.SetAttack(0.0)     // Instant attack for testing
	g.SetRelease(0.0)    // Instant release for testing
	g.SetHold(0.0)       // No hold

	// Signal at threshold should open gate
	thresholdSignal := float32(0.1) // -20 dB

	// Process twice to ensure state transitions
	output1 := g.Process(thresholdSignal)
	output2 := g.Process(thresholdSignal)

	t.Logf("After 1st process: Output: %f, State: %s, IsOpen: %v, GainReduction: %f dB",
		output1, g.GetState(), g.IsOpen(), g.GetGainReduction())
	t.Logf("After 2nd process: Output: %f, State: %s, IsOpen: %v, GainReduction: %f dB",
		output2, g.GetState(), g.IsOpen(), g.GetGainReduction())

	if !g.IsOpen() {
		t.Error("Gate should open at threshold")
	}

	// Signal 3dB below threshold (within hysteresis) should keep gate open
	slightlyQuieter := float32(0.0707) // ~-23 dB
	g.Process(slightlyQuieter)

	if !g.IsOpen() {
		t.Error("Gate should stay open within hysteresis band")
	}

	// Signal below hysteresis band should close gate
	belowHysteresis := float32(0.04) // ~-28 dB (below -26 dB close threshold)
	g.Process(belowHysteresis)
	g.Process(belowHysteresis) // Process twice for state transition

	if g.IsOpen() {
		t.Error("Gate should close below hysteresis band")
	}
}

func TestGateHoldTime(t *testing.T) {
	sampleRate := 48000.0
	g := NewGate(sampleRate)
	g.SetThreshold(-20.0)
	g.SetHold(0.010) // 10ms hold time
	g.SetAttack(0.0)
	g.SetRelease(0.0)

	// Open the gate
	loudSignal := float32(0.2)
	g.Process(loudSignal)

	if !g.IsOpen() {
		t.Fatal("Gate should be open")
	}

	// Signal drops below threshold
	quietSignal := float32(0.01)
	holdSamples := int(0.010 * sampleRate)

	// Process for half the hold time
	for i := 0; i < holdSamples/2; i++ {
		g.Process(quietSignal)
	}

	// Should still be open (in hold state)
	if !g.IsOpen() {
		t.Error("Gate should remain open during hold time")
	}

	if g.GetState() != "hold" {
		t.Errorf("Gate should be in hold state, got %s", g.GetState())
	}

	// Process for rest of hold time
	for i := 0; i < holdSamples; i++ {
		g.Process(quietSignal)
	}

	// Now should be closing
	if g.GetState() != "release" && g.GetState() != "closed" {
		t.Errorf("Gate should be in release or closed state after hold time, got %s", g.GetState())
	}
}

func TestGateRange(t *testing.T) {
	g := NewGate(48000.0)
	g.SetThreshold(-20.0)
	g.SetRange(-40.0) // 40dB attenuation when closed

	// Process quiet signal
	quietSignal := float32(0.01)
	for i := 0; i < 100; i++ {
		g.Process(quietSignal)
	}

	// Check gain reduction
	gr := g.GetGainReduction()
	if gr > -39.0 || gr < -41.0 {
		t.Errorf("Incorrect gain reduction: got %f dB, expected ~-40 dB", gr)
	}
}

func TestGateSidechainFilter(t *testing.T) {
	sampleRate := 48000.0
	g := NewGate(sampleRate)
	g.SetThreshold(-20.0)
	g.SetSidechainFilter(true, 200.0) // 200Hz high-pass for more attenuation

	// Low frequency signal (should be filtered out)
	samples := 1000
	maxFiltered := float32(0.0)
	for i := 0; i < samples; i++ {
		// 50Hz sine wave at -15dB (closer to threshold)
		phase := 2.0 * math.Pi * 50.0 * float64(i) / sampleRate
		signal := float32(0.178 * math.Sin(phase))
		output := g.Process(signal)
		if math.Abs(float64(output)) > float64(maxFiltered) {
			maxFiltered = float32(math.Abs(float64(output)))
		}
	}

	// Debug: check max filtered level
	maxFilteredDB := 20.0 * math.Log10(float64(maxFiltered)+1e-10)
	t.Logf("Max filtered output: %f (%f dB), State: %s", maxFiltered, maxFilteredDB, g.GetState())

	// Gate should remain closed due to HPF filtering out 50Hz
	if g.IsOpen() {
		t.Error("Gate opened on low-frequency signal that should be filtered")
	}

	// Reset gate
	g.Reset()

	// High frequency signal (should pass through filter)
	for i := 0; i < samples; i++ {
		// 1kHz sine wave at -10dB
		phase := 2.0 * math.Pi * 1000.0 * float64(i) / sampleRate
		signal := float32(0.316 * math.Sin(phase))
		g.Process(signal)
	}

	// Gate should open for high frequency
	if !g.IsOpen() {
		t.Error("Gate didn't open on high-frequency signal")
	}
}

func TestGateStereoLinking(t *testing.T) {
	g := NewGate(48000.0)
	g.SetThreshold(-20.0)
	g.SetAttack(0.0) // Instant for testing

	// Left channel loud, right channel quiet
	inputL := []float32{0.2, 0.2, 0.2}    // Above threshold
	inputR := []float32{0.01, 0.01, 0.01} // Below threshold
	outputL := make([]float32, 3)
	outputR := make([]float32, 3)

	g.ProcessStereo(inputL, inputR, outputL, outputR)

	// Both channels should pass through (linked detection)
	for i := range outputL {
		if outputL[i] < inputL[i]*0.9 {
			t.Errorf("Left channel incorrectly gated at sample %d", i)
		}
		if outputR[i] < inputR[i]*0.9 {
			t.Errorf("Right channel incorrectly gated at sample %d", i)
		}
	}
}

func TestGateAttackRelease(t *testing.T) {
	sampleRate := 48000.0
	g := NewGate(sampleRate)
	g.SetThreshold(-20.0)
	g.SetAttack(0.010)  // 10ms attack
	g.SetRelease(0.020) // 20ms release
	g.SetHold(0.0)

	// Test attack
	loudSignal := float32(0.2)
	attackSamples := int(0.010 * sampleRate)

	var lastOutput float32
	for i := 0; i < attackSamples; i++ {
		output := g.Process(loudSignal)
		if i > 0 && output <= lastOutput {
			t.Error("Output should increase during attack")
		}
		lastOutput = output
	}

	// Process more to ensure fully open
	for i := 0; i < 100; i++ {
		g.Process(loudSignal)
	}

	// Test release
	quietSignal := float32(0.01)
	releaseSamples := int(0.020 * sampleRate)

	lastOutput = loudSignal // Reset to loud level
	for i := 0; i < releaseSamples; i++ {
		output := g.Process(quietSignal)
		normalizedOutput := output / quietSignal // Normalize to see gain change
		if i > 0 && normalizedOutput >= lastOutput/quietSignal {
			t.Error("Gain should decrease during release")
		}
		lastOutput = output
	}
}

func TestGateReset(t *testing.T) {
	g := NewGate(48000.0)

	// Open the gate
	g.Process(1.0)

	// Reset
	g.Reset()

	// Check state
	if g.IsOpen() {
		t.Error("Gate should be closed after reset")
	}

	if g.GetGainReduction() != g.range_ {
		t.Errorf("Gain reduction not reset correctly: got %f, expected %f",
			g.GetGainReduction(), g.range_)
	}

	if g.GetState() != "closed" {
		t.Errorf("Gate state incorrect after reset: %s", g.GetState())
	}
}

func TestGateStates(t *testing.T) {
	g := NewGate(48000.0)
	g.SetThreshold(-20.0) // Set explicit threshold
	g.SetHysteresis(5.0)
	g.SetAttack(0.0)
	g.SetRelease(0.0)
	g.SetHold(0.001) // 1ms hold

	// Initial state
	if g.GetState() != "closed" {
		t.Errorf("Initial state should be closed, got %s", g.GetState())
	}

	// Open gate
	g.Process(0.2) // Well above -20dB threshold
	g.Process(0.2) // Process again to ensure state transition completes
	if g.GetState() != "open" {
		t.Errorf("State should be open, got %s", g.GetState())
	}

	// Trigger hold by going below close threshold (-25dB)
	g.Process(0.005) // ~-46dB, well below close threshold
	if g.GetState() != "hold" {
		t.Errorf("State should be hold, got %s", g.GetState())
	}
}

// Benchmark gate processing
func BenchmarkGate(b *testing.B) {
	g := NewGate(48000.0)
	input := float32(0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Process(input)
	}
}

func BenchmarkGateWithSidechain(b *testing.B) {
	g := NewGate(48000.0)
	g.SetSidechainFilter(true, 100.0)
	input := float32(0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Process(input)
	}
}
