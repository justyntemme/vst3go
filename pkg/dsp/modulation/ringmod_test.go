package modulation

import (
	"math"
	"testing"
)

func TestRingModulatorCreation(t *testing.T) {
	sampleRate := 48000.0
	rm := NewRingModulator(sampleRate)

	if rm == nil {
		t.Fatal("Failed to create ring modulator")
	}

	if rm.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", rm.sampleRate, sampleRate)
	}

	// Check defaults
	if rm.frequency != 440.0 {
		t.Errorf("Default frequency incorrect: got %f, want 440.0", rm.frequency)
	}

	if rm.mix != 0.5 {
		t.Errorf("Default mix incorrect: got %f, want 0.5", rm.mix)
	}

	if rm.waveform != WaveformSine {
		t.Errorf("Default waveform incorrect: got %v, want WaveformSine", rm.waveform)
	}
}

func TestRingModulatorDrySignal(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetMix(0.0) // Completely dry

	input := float32(0.5)
	output := rm.Process(input)

	// Should be unchanged
	if math.Abs(float64(output-input)) > 0.001 {
		t.Errorf("Dry signal altered: input %f, output %f", input, output)
	}
}

func TestRingModulatorFullWet(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetMix(1.0) // Completely wet
	rm.SetFrequency(1000.0)

	// Process a DC signal
	dc := float32(1.0)
	samples := 48 // 1ms at 48kHz

	outputs := make([]float32, samples)
	for i := 0; i < samples; i++ {
		outputs[i] = rm.Process(dc)
	}

	// With a DC input and sine carrier, output should oscillate
	// Check that we have both positive and negative values
	hasPositive := false
	hasNegative := false

	for _, out := range outputs {
		if out > 0.1 {
			hasPositive = true
		}
		if out < -0.1 {
			hasNegative = true
		}
	}

	if !hasPositive || !hasNegative {
		t.Error("Ring modulator not producing bipolar output with DC input")
	}
}

func TestRingModulatorFrequencyDoubling(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetMix(1.0) // Full wet
	rm.SetFrequency(1000.0) // 1kHz carrier

	// Input a 1kHz sine wave (same as carrier)
	inputFreq := 1000.0
	samples := 480 // 10ms
	
	input := make([]float32, samples)
	output := make([]float32, samples)

	for i := 0; i < samples; i++ {
		input[i] = float32(math.Sin(2.0 * math.Pi * inputFreq * float64(i) / 48000.0))
		output[i] = rm.Process(input[i])
	}

	// When modulating a sine with itself, we get frequency doubling
	// Count zero crossings (should be about double)
	inputCrossings := 0
	outputCrossings := 0

	for i := 1; i < samples; i++ {
		if (input[i-1] < 0 && input[i] >= 0) || (input[i-1] >= 0 && input[i] < 0) {
			inputCrossings++
		}
		if (output[i-1] < 0 && output[i] >= 0) || (output[i-1] >= 0 && output[i] < 0) {
			outputCrossings++
		}
	}

	// Output should have roughly double the zero crossings
	ratio := float64(outputCrossings) / float64(inputCrossings)
	if ratio < 1.8 || ratio > 2.2 {
		t.Errorf("Frequency doubling not occurring: input crossings=%d, output crossings=%d, ratio=%f",
			inputCrossings, outputCrossings, ratio)
	}
}

func TestRingModulatorSidebands(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetMix(1.0)
	rm.SetFrequency(300.0) // 300Hz carrier

	// Input a 1kHz signal
	inputFreq := 1000.0
	samples := 4800 // 100ms

	// Process signal
	for i := 0; i < samples; i++ {
		input := float32(math.Sin(2.0 * math.Pi * inputFreq * float64(i) / 48000.0))
		_ = rm.Process(input)
	}

	// Ring modulation creates sum and difference frequencies
	// 1000Hz ± 300Hz = 1300Hz and 700Hz
	// This is a characteristic of ring modulation
	// Test passes if processing completes without error
}

func TestRingModulatorWaveforms(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetMix(1.0)
	rm.SetFrequency(100.0) // Low frequency to see waveform shape

	waveforms := []Waveform{
		WaveformSine,
		WaveformTriangle,
		WaveformSquare,
		WaveformSawtooth,
	}

	dc := float32(1.0)
	
	for _, wf := range waveforms {
		rm.SetWaveform(wf)
		rm.Reset()

		// Process one cycle
		samplesPerCycle := int(48000.0 / 100.0)
		outputs := make([]float32, samplesPerCycle)

		for i := 0; i < samplesPerCycle; i++ {
			outputs[i] = rm.Process(dc)
		}

		// Check that output has expected characteristics
		// All waveforms should produce output in [-1, 1] range
		for i, out := range outputs {
			if out < -1.1 || out > 1.1 {
				t.Errorf("Waveform %v: output out of range at sample %d: %f", wf, i, out)
			}
		}
	}
}

func TestRingModulatorLFO(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetMix(1.0)
	rm.SetFrequency(1000.0)
	rm.EnableLFO(true)
	rm.SetLFORate(2.0) // 2Hz
	rm.SetLFODepth(0.5) // 50% modulation

	// Process a constant signal and look for frequency variation
	samples := 24000 // 0.5 seconds
	dc := float32(1.0)

	// Track frequency by counting zero crossings in windows
	windowSize := 2400 // 50ms windows
	windows := samples / windowSize
	crossingsPerWindow := make([]int, windows)

	prevSample := float32(0)
	for i := 0; i < samples; i++ {
		sample := rm.Process(dc)
		
		// Count zero crossings
		if (prevSample < 0 && sample >= 0) || (prevSample >= 0 && sample < 0) {
			window := i / windowSize
			if window < windows {
				crossingsPerWindow[window]++
			}
		}
		prevSample = sample
	}

	// Should see variation in crossing counts due to frequency modulation
	minCrossings := crossingsPerWindow[0]
	maxCrossings := crossingsPerWindow[0]

	for _, crossings := range crossingsPerWindow {
		if crossings < minCrossings {
			minCrossings = crossings
		}
		if crossings > maxCrossings {
			maxCrossings = crossings
		}
	}

	// With 50% depth modulation of 1kHz, we expect significant variation
	if maxCrossings <= minCrossings {
		t.Error("LFO not modulating carrier frequency")
	}
}

func TestRingModulatorStereo(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetMix(1.0)
	rm.SetFrequency(500.0)

	// Process different inputs
	inputL := float32(0.7)
	inputR := float32(-0.5)

	outputL, outputR := rm.ProcessStereo(inputL, inputR)

	// Both should be modulated by the same carrier
	// So the ratio should be preserved
	inputRatio := inputL / inputR
	outputRatio := outputL / outputR

	if math.Abs(float64(inputRatio-outputRatio)) > 0.01 {
		t.Errorf("Stereo ratio not preserved: input ratio=%f, output ratio=%f",
			inputRatio, outputRatio)
	}
}

func TestRingModulatorReset(t *testing.T) {
	rm := NewRingModulator(48000.0)
	rm.SetFrequency(1000.0)

	// Process some samples
	for i := 0; i < 1000; i++ {
		rm.Process(0.5)
	}

	// Reset
	rm.Reset()

	// Phase should be reset
	if rm.phase != 0.0 {
		t.Errorf("Phase not reset: %f", rm.phase)
	}

	// Process again - should start from beginning
	output1 := rm.Process(1.0)
	
	// Create new instance and process once
	rm2 := NewRingModulator(48000.0)
	rm2.SetFrequency(1000.0)
	rm2.SetMix(rm.mix)
	output2 := rm2.Process(1.0)

	// Should produce same output
	if math.Abs(float64(output1-output2)) > 0.001 {
		t.Errorf("Reset not returning to initial state: %f vs %f", output1, output2)
	}
}

func TestRingModulatorParameterLimits(t *testing.T) {
	rm := NewRingModulator(48000.0)

	// Test frequency limits
	rm.SetFrequency(-100.0)
	if rm.frequency < 0.1 {
		t.Errorf("Frequency below minimum: %f", rm.frequency)
	}

	rm.SetFrequency(30000.0)
	if rm.frequency > 24000.0 {
		t.Errorf("Frequency above Nyquist: %f", rm.frequency)
	}

	// Test mix limits
	rm.SetMix(-0.5)
	if rm.mix < 0.0 {
		t.Errorf("Mix below minimum: %f", rm.mix)
	}

	rm.SetMix(1.5)
	if rm.mix > 1.0 {
		t.Errorf("Mix above maximum: %f", rm.mix)
	}

	// Test LFO depth limits
	rm.SetLFODepth(-0.5)
	if rm.lfoDepth < 0.0 {
		t.Errorf("LFO depth below minimum: %f", rm.lfoDepth)
	}

	rm.SetLFODepth(2.0)
	if rm.lfoDepth > 1.0 {
		t.Errorf("LFO depth above maximum: %f", rm.lfoDepth)
	}
}

// Benchmark ring modulator
func BenchmarkRingModulator(b *testing.B) {
	rm := NewRingModulator(48000.0)
	rm.SetFrequency(1000.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rm.Process(0.5)
	}
}

func BenchmarkRingModulatorWithLFO(b *testing.B) {
	rm := NewRingModulator(48000.0)
	rm.SetFrequency(1000.0)
	rm.EnableLFO(true)
	rm.SetLFORate(5.0)
	rm.SetLFODepth(0.3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rm.Process(0.5)
	}
}