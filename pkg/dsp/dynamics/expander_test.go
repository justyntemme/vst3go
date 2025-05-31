package dynamics

import (
	"math"
	"testing"
)

func TestExpanderCreation(t *testing.T) {
	sampleRate := 48000.0
	e := NewExpander(sampleRate)

	if e == nil {
		t.Fatal("Failed to create expander")
	}

	if e.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", e.sampleRate, sampleRate)
	}

	// Check defaults
	if e.threshold != -40.0 {
		t.Errorf("Default threshold incorrect: got %f, want -40.0", e.threshold)
	}

	if e.ratio != 2.0 {
		t.Errorf("Default ratio incorrect: got %f, want 2.0", e.ratio)
	}

	if e.range_ != -40.0 {
		t.Errorf("Default range incorrect: got %f, want -40.0", e.range_)
	}
}

func TestExpanderGainComputation(t *testing.T) {
	e := NewExpander(48000.0)
	e.SetThreshold(-30.0)
	e.SetRatio(3.0) // 3:1 expansion
	e.SetKnee(0.0)   // Hard knee

	testCases := []struct {
		inputDB      float64
		expectedGain float64
		tolerance    float64
		description  string
	}{
		{-20.0, 0.0, 0.001, "Above threshold - no expansion"},
		{-30.0, 0.0, 0.001, "At threshold - no expansion"},
		{-40.0, -20.0, 0.001, "10dB below threshold - 20dB expansion (3:1)"},
		{-50.0, -40.0, 0.001, "20dB below threshold - limited by range"},
	}

	for _, tc := range testCases {
		gain := e.computeGain(tc.inputDB)
		if math.Abs(gain-tc.expectedGain) > tc.tolerance {
			t.Errorf("%s: input %f dB, got %f dB gain, expected %f dB",
				tc.description, tc.inputDB, gain, tc.expectedGain)
		}
	}
}

func TestExpanderSoftKnee(t *testing.T) {
	e := NewExpander(48000.0)
	e.SetThreshold(-30.0)
	e.SetRatio(2.0)
	e.SetKnee(6.0) // 6dB soft knee

	// Test in knee region
	inputDB := -32.0 // 2dB below threshold, in knee
	gain := e.computeGain(inputDB)

	// Should have some expansion but less than hard knee
	// Hard knee would give: -2 * (2-1) = -2dB
	if gain >= 0 || gain <= -2.0 {
		t.Errorf("Soft knee not working: got %f dB gain at %f dB input", gain, inputDB)
	}
}

func TestExpanderProcessing(t *testing.T) {
	sampleRate := 48000.0
	e := NewExpander(sampleRate)
	e.SetThreshold(-30.0)
	e.SetRatio(2.0)
	e.SetAttack(0.0)   // Instant attack
	e.SetRelease(0.0)  // Instant release

	// Test signal below threshold
	quietSignal := float32(0.01) // ~-40 dB
	output := e.Process(quietSignal)

	// Should be expanded (reduced)
	if output >= quietSignal {
		t.Errorf("Expander not reducing quiet signal: input %f, output %f", quietSignal, output)
	}

	// Test signal above threshold
	loudSignal := float32(0.1) // -20 dB
	output = e.Process(loudSignal)

	// Should pass through unchanged
	if math.Abs(float64(output-loudSignal)) > 0.001 {
		t.Errorf("Expander affecting signal above threshold: input %f, output %f", loudSignal, output)
	}
}

func TestExpanderRange(t *testing.T) {
	e := NewExpander(48000.0)
	e.SetThreshold(-20.0)
	e.SetRatio(4.0)      // High ratio
	e.SetRange(-20.0)    // Limit expansion to 20dB
	e.SetAttack(0.0)
	e.SetRelease(0.0)

	// Very quiet signal that would normally get more than 20dB expansion
	veryQuiet := float32(0.001) // ~-60 dB
	
	// Process multiple times to ensure steady state
	var output float32
	for i := 0; i < 10; i++ {
		output = e.Process(veryQuiet)
	}

	// Calculate actual expansion
	expansionDB := 20.0 * math.Log10(float64(output/veryQuiet))
	
	// Should be limited to -20dB
	if expansionDB < -21.0 || expansionDB > -19.0 {
		t.Errorf("Range limiting not working: got %f dB expansion, expected ~-20 dB", expansionDB)
	}
}

func TestExpanderStereoLinking(t *testing.T) {
	e := NewExpander(48000.0)
	e.SetThreshold(-30.0)
	e.SetRatio(2.0)
	e.SetAttack(0.0)

	// Left quiet, right loud
	inputL := []float32{0.01, 0.01, 0.01}  // Below threshold
	inputR := []float32{0.1, 0.1, 0.1}     // Above threshold
	outputL := make([]float32, 3)
	outputR := make([]float32, 3)

	e.ProcessStereo(inputL, inputR, outputL, outputR)

	// Both channels should NOT be expanded (linked detection uses max)
	for i := range outputL {
		if outputL[i] < inputL[i]*0.99 {
			t.Errorf("Left channel incorrectly expanded at sample %d", i)
		}
		if outputR[i] < inputR[i]*0.99 {
			t.Errorf("Right channel incorrectly expanded at sample %d", i)
		}
	}
}

func TestExpanderAttackRelease(t *testing.T) {
	sampleRate := 48000.0
	e := NewExpander(sampleRate)
	e.SetThreshold(-30.0)
	e.SetRatio(2.0)
	e.SetAttack(0.010)  // 10ms attack
	e.SetRelease(0.020) // 20ms release

	// Start with quiet signal (will be expanded)
	quietSignal := float32(0.01)
	attackSamples := int(0.010 * sampleRate)
	
	// Process through attack (expansion increasing)
	var lastGR float64 = 0
	for i := 0; i < attackSamples; i++ {
		_ = e.Process(quietSignal)
		gr := e.GetGainReduction()
		if i > 0 && gr >= lastGR {
			t.Error("Gain reduction should increase during attack (more negative)")
		}
		lastGR = gr
	}

	// Switch to loud signal
	loudSignal := float32(0.2)
	releaseSamples := int(0.020 * sampleRate)
	
	// Process through release (returning to unity gain)
	for i := 0; i < releaseSamples; i++ {
		_ = e.Process(loudSignal)
		gr := e.GetGainReduction()
		if i > 0 && gr <= lastGR {
			t.Error("Gain reduction should decrease during release (towards 0)")
		}
		lastGR = gr
	}

	// Should be close to 0 dB after release
	if math.Abs(e.GetGainReduction()) > 1.0 {
		t.Errorf("Gain reduction not returned to unity: %f dB", e.GetGainReduction())
	}
}

func TestExpanderReset(t *testing.T) {
	e := NewExpander(48000.0)
	
	// Process some signal
	e.Process(0.001) // Quiet signal to trigger expansion
	
	// Reset
	e.Reset()
	
	// Check state
	if e.GetGainReduction() != 0 {
		t.Errorf("Gain reduction not reset: %f", e.GetGainReduction())
	}
	
	if e.currentGain != 1.0 {
		t.Errorf("Current gain not reset: %f", e.currentGain)
	}
}

// Benchmark expander
func BenchmarkExpander(b *testing.B) {
	e := NewExpander(48000.0)
	input := float32(0.05)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Process(input)
	}
}

func BenchmarkExpanderStereo(b *testing.B) {
	e := NewExpander(48000.0)
	inputL := make([]float32, 1024)
	inputR := make([]float32, 1024)
	outputL := make([]float32, 1024)
	outputR := make([]float32, 1024)
	
	// Fill with test signal
	for i := range inputL {
		inputL[i] = float32(0.1 * math.Sin(float64(i)*0.1))
		inputR[i] = float32(0.1 * math.Cos(float64(i)*0.1))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ProcessStereo(inputL, inputR, outputL, outputR)
	}
}