package modulation

import (
	"math"
	"testing"
)

func TestTremoloCreation(t *testing.T) {
	sampleRate := 48000.0
	trem := NewTremolo(sampleRate)

	if trem == nil {
		t.Fatal("Failed to create tremolo")
	}

	if trem.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", trem.sampleRate, sampleRate)
	}

	// Check defaults
	if trem.rate != 5.0 {
		t.Errorf("Default rate incorrect: got %f, want 5.0", trem.rate)
	}

	if trem.depth != 0.5 {
		t.Errorf("Default depth incorrect: got %f, want 0.5", trem.depth)
	}

	if trem.waveform != WaveformSine {
		t.Errorf("Default waveform incorrect: got %v, want WaveformSine", trem.waveform)
	}
}

func TestTremoloNoModulation(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetDepth(0.0) // No modulation

	input := float32(0.8)
	output := trem.Process(input)

	// Should be unchanged
	if math.Abs(float64(output-input)) > 0.001 {
		t.Errorf("Signal altered with zero depth: input %f, output %f", input, output)
	}
}

func TestTremoloFullDepth(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetDepth(1.0) // Full depth
	trem.SetRate(2.0)  // 2Hz

	// Process for one full cycle
	samplesPerCycle := int(48000.0 / 2.0)
	input := float32(1.0)

	minOutput := float32(1.0)
	maxOutput := float32(0.0)

	for i := 0; i < samplesPerCycle; i++ {
		output := trem.Process(input)
		if output < minOutput {
			minOutput = output
		}
		if output > maxOutput {
			maxOutput = output
		}
	}

	// With full depth, output should vary from 0 to 1
	if minOutput > 0.1 {
		t.Errorf("Minimum output too high with full depth: %f", minOutput)
	}

	if maxOutput < 0.9 {
		t.Errorf("Maximum output too low with full depth: %f", maxOutput)
	}
}

func TestTremoloWaveforms(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetDepth(0.5)
	trem.SetRate(10.0) // 10Hz for clear waveform

	waveforms := []Waveform{
		WaveformSine,
		WaveformTriangle,
		WaveformSquare,
		WaveformSawtooth,
	}

	for _, wf := range waveforms {
		trem.SetWaveform(wf)
		trem.Reset()

		// Process constant input
		input := float32(1.0)
		samples := 4800 // 100ms

		outputs := make([]float32, samples)
		for i := 0; i < samples; i++ {
			outputs[i] = trem.Process(input)
		}

		// Check that output is modulated
		var minVal, maxVal float32 = 1.0, 0.0
		for _, out := range outputs {
			if out < minVal {
				minVal = out
			}
			if out > maxVal {
				maxVal = out
			}
		}

		// Should see modulation
		if maxVal-minVal < 0.4 { // With 50% depth, expect ~0.5 variation
			t.Errorf("Insufficient modulation with waveform %v: range=%f", wf, maxVal-minVal)
		}
	}
}

func TestTremoloHarmonicMode(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetDepth(1.0)
	trem.SetRate(5.0)
	trem.SetMode(TremoloModeHarmonic)

	// Process for one LFO cycle
	samplesPerCycle := int(48000.0 / 5.0)
	input := float32(1.0)

	// Collect outputs
	outputs := make([]float32, samplesPerCycle)
	for i := 0; i < samplesPerCycle; i++ {
		outputs[i] = trem.Process(input)
	}

	// Find min/max values
	minVal := outputs[0]
	maxVal := outputs[0]
	for _, v := range outputs {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Count how many times the output goes below a threshold
	// In harmonic mode with full depth, output should hit near-zero twice per cycle
	lowCount := 0
	for _, output := range outputs {
		if output < 0.1 {
			lowCount++
		}
	}

	// Also count local minima to verify frequency doubling
	minimaCount := 0
	for i := 1; i < len(outputs)-1; i++ {
		if outputs[i] < outputs[i-1] && outputs[i] < outputs[i+1] {
			minimaCount++
		}
	}

	// Log some outputs for debugging
	t.Logf("First 10 outputs: %v", outputs[:10])
	t.Logf("Min: %f, Max: %f", minVal, maxVal)
	t.Logf("Samples: %d, Low count: %d, Minima count: %d", samplesPerCycle, lowCount, minimaCount)

	// Harmonic mode should create two minima per LFO cycle (frequency doubling)
	// We're processing one LFO cycle, so we expect 2 minima
	if minimaCount < 1 || minimaCount > 3 {
		t.Errorf("Harmonic mode not creating frequency doubling: %d minima found (expected ~2)", minimaCount)
	}

	// Should reach near-zero at the minima with full depth
	if minVal > 0.05 {
		t.Errorf("Harmonic mode not reaching near-zero: min=%f", minVal)
	}
}

func TestTremoloStereo(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetDepth(0.8) // Higher depth for clearer difference
	trem.SetRate(5.0)
	trem.SetStereo(true)
	trem.SetStereoPhase(0.5) // 180 degrees out of phase

	// Process identical inputs
	inputL := float32(1.0)
	inputR := float32(1.0)

	// Collect samples over a full LFO cycle
	samplesPerCycle := int(48000.0 / 5.0)
	maxDiff := float32(0.0)

	for i := 0; i < samplesPerCycle; i++ {
		outputL, outputR := trem.ProcessStereo(inputL, inputR)
		diff := float32(math.Abs(float64(outputL - outputR)))
		if diff > maxDiff {
			maxDiff = diff
		}
	}

	// With 180 degree phase difference and 0.8 depth, max difference should be significant
	if maxDiff < 0.5 {
		t.Errorf("Stereo phase not creating enough difference: max diff=%f", maxDiff)
	}

	// Also test that without stereo mode, outputs are identical
	trem.SetStereo(false)
	outputL, outputR := trem.ProcessStereo(inputL, inputR)
	if math.Abs(float64(outputL-outputR)) > 0.001 {
		t.Errorf("Non-stereo mode producing different outputs: L=%f, R=%f", outputL, outputR)
	}
}

func TestTremoloSmoothing(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetWaveform(WaveformSquare)
	trem.SetDepth(0.5)
	trem.SetRate(10.0)
	trem.EnableSmoothing(true)

	// Process and check for smooth transitions
	input := float32(1.0)
	samples := 4800 // 100ms to see full cycle

	outputs := make([]float32, samples)
	for i := 0; i < samples; i++ {
		outputs[i] = trem.Process(input)
	}

	// Check for large jumps (should be smoothed)
	maxJump := float32(0.0)
	for i := 1; i < samples; i++ {
		jump := float32(math.Abs(float64(outputs[i] - outputs[i-1])))
		if jump > maxJump {
			maxJump = jump
		}
	}

	// With smoothing, jumps should be small
	if maxJump > 0.1 {
		t.Errorf("Square wave not properly smoothed: max jump=%f", maxJump)
	}

	// Test without smoothing
	trem2 := NewTremolo(48000.0)
	trem2.SetWaveform(WaveformSquare)
	trem2.SetDepth(0.5)
	trem2.SetRate(10.0)
	trem2.EnableSmoothing(false)

	outputs2 := make([]float32, samples)
	for i := 0; i < samples; i++ {
		outputs2[i] = trem2.Process(input)
	}

	// Without smoothing, should see larger jumps
	maxJumpUnsmoothed := float32(0.0)
	for i := 1; i < samples; i++ {
		jump := float32(math.Abs(float64(outputs2[i] - outputs2[i-1])))
		if jump > maxJumpUnsmoothed {
			maxJumpUnsmoothed = jump
		}
	}

	t.Logf("Max jump with smoothing: %f, without: %f", maxJump, maxJumpUnsmoothed)
	t.Logf("First 5 smoothed outputs: %v", outputs[:5])
	t.Logf("First 5 unsmoothed outputs: %v", outputs2[:5])

	// Unsmoothed should have larger jumps for square wave
	// If both are 0, the square wave isn't working
	if maxJumpUnsmoothed == 0 && maxJump == 0 {
		t.Error("Square wave not producing any output variation")
	} else if maxJumpUnsmoothed <= maxJump {
		t.Error("Smoothing not making expected difference")
	}
}

func TestTremoloRate(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetDepth(1.0)
	trem.SetRate(10.0) // 10Hz

	// Process and count cycles
	samples := 48000 // 1 second
	input := float32(1.0)

	cycleCount := 0
	wasLow := false

	for i := 0; i < samples; i++ {
		output := trem.Process(input)

		// Count transitions from low to high
		if output < 0.1 {
			wasLow = true
		} else if wasLow && output > 0.9 {
			cycleCount++
			wasLow = false
		}
	}

	// Should see about 10 cycles in 1 second
	if cycleCount < 8 || cycleCount > 12 {
		t.Errorf("Rate not accurate: expected ~10 cycles, got %d", cycleCount)
	}
}

func TestTremoloReset(t *testing.T) {
	trem := NewTremolo(48000.0)
	trem.SetStereo(true)
	trem.SetStereoPhase(0.25)

	// Process some samples
	for i := 0; i < 1000; i++ {
		trem.Process(0.5)
	}

	// Reset
	trem.Reset()

	// Gain should be reset
	if trem.GetCurrentGain() != 1.0 {
		t.Errorf("Gain not reset: %f", trem.GetCurrentGain())
	}

	// Process again and compare with fresh instance
	output1L, output1R := trem.ProcessStereo(1.0, 1.0)

	trem2 := NewTremolo(48000.0)
	trem2.SetStereo(true)
	trem2.SetStereoPhase(0.25)
	output2L, output2R := trem2.ProcessStereo(1.0, 1.0)

	if math.Abs(float64(output1L-output2L)) > 0.001 {
		t.Errorf("Left channel not reset properly: %f vs %f", output1L, output2L)
	}

	if math.Abs(float64(output1R-output2R)) > 0.001 {
		t.Errorf("Right channel not reset properly: %f vs %f", output1R, output2R)
	}
}

func TestTremoloParameterLimits(t *testing.T) {
	trem := NewTremolo(48000.0)

	// Test rate limits
	trem.SetRate(-1.0)
	if trem.rate < 0.01 {
		t.Errorf("Rate below minimum: %f", trem.rate)
	}

	trem.SetRate(50.0)
	if trem.rate > 20.0 {
		t.Errorf("Rate above maximum: %f", trem.rate)
	}

	// Test depth limits
	trem.SetDepth(-0.5)
	if trem.depth < 0.0 {
		t.Errorf("Depth below minimum: %f", trem.depth)
	}

	trem.SetDepth(2.0)
	if trem.depth > 1.0 {
		t.Errorf("Depth above maximum: %f", trem.depth)
	}

	// Test phase limits
	trem.SetStereoPhase(-0.5)
	if trem.phase < 0.0 {
		t.Errorf("Phase below minimum: %f", trem.phase)
	}

	trem.SetStereoPhase(1.5)
	if trem.phase > 1.0 {
		t.Errorf("Phase above maximum: %f", trem.phase)
	}
}

// Benchmark tremolo
func BenchmarkTremolo(b *testing.B) {
	trem := NewTremolo(48000.0)
	trem.SetRate(5.0)
	trem.SetDepth(0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trem.Process(0.5)
	}
}

func BenchmarkTremoloStereo(b *testing.B) {
	trem := NewTremolo(48000.0)
	trem.SetStereo(true)
	trem.SetStereoPhase(0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trem.ProcessStereo(0.5, 0.5)
	}
}
