package dynamics

import (
	"math"
	"testing"
)

func TestCompressorCreation(t *testing.T) {
	sampleRate := 48000.0
	c := NewCompressor(sampleRate)

	if c == nil {
		t.Fatal("Failed to create compressor")
	}

	if c.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", c.sampleRate, sampleRate)
	}

	// Check default values
	if c.threshold != -20.0 {
		t.Errorf("Default threshold incorrect: got %f, want -20.0", c.threshold)
	}

	if c.ratio != 4.0 {
		t.Errorf("Default ratio incorrect: got %f, want 4.0", c.ratio)
	}
}

func TestCompressorGainComputation(t *testing.T) {
	c := NewCompressor(48000.0)
	c.SetThreshold(-20.0)
	c.SetRatio(4.0)
	c.SetKnee(KneeHard, 0.0)

	testCases := []struct {
		inputDB    float64
		expectedGR float64 // Expected gain reduction
		tolerance  float64
	}{
		{-30.0, 0.0, 0.001}, // Below threshold
		{-20.0, 0.0, 0.001}, // At threshold
		{-10.0, 7.5, 0.001}, // 10dB over threshold, 4:1 ratio -> 7.5dB reduction
		{0.0, 15.0, 0.001},  // 20dB over threshold -> 15dB reduction
	}

	for _, tc := range testCases {
		gr := c.computeGain(tc.inputDB)
		if math.Abs(gr-tc.expectedGR) > tc.tolerance {
			t.Errorf("Gain computation error at %f dB: got %f dB reduction, expected %f dB",
				tc.inputDB, gr, tc.expectedGR)
		}
	}
}

func TestCompressorSoftKnee(t *testing.T) {
	c := NewCompressor(48000.0)
	c.SetThreshold(-20.0)
	c.SetRatio(4.0)
	c.SetKnee(KneeSoft, 6.0) // 6dB soft knee

	// Test in knee region - slightly above threshold
	inputDB := -18.0 // 2dB above threshold, in the middle of 6dB knee
	gr := c.computeGain(inputDB)

	// With soft knee, compression should be less than hard knee would give
	// Hard knee would give: 2dB * (1 - 1/4) = 1.5dB reduction
	// Soft knee should give less
	if gr <= 0.0 || gr >= 1.5 {
		t.Errorf("Soft knee not working correctly: got %f dB reduction at %f dB input", gr, inputDB)
	}

	// Test at knee boundary
	inputDB = -17.0 // At top of knee region (threshold + knee/2)
	gr = c.computeGain(inputDB)
	expectedGR := 3.0 * (1.0 - 1.0/4.0) // Should match hard knee at boundary
	if math.Abs(gr-expectedGR) > 0.1 {
		t.Errorf("Soft knee boundary incorrect: got %f dB reduction, expected %f", gr, expectedGR)
	}
}

func TestCompressorProcessing(t *testing.T) {
	sampleRate := 48000.0
	c := NewCompressor(sampleRate)
	c.SetThreshold(-20.0)
	c.SetRatio(4.0)
	c.SetAttack(0.001)  // 1ms
	c.SetRelease(0.010) // 10ms

	// Generate test signal: sine wave that exceeds threshold
	duration := 0.1 // 100ms
	numSamples := int(sampleRate * duration)
	input := make([]float32, numSamples)
	output := make([]float32, numSamples)

	// Generate loud sine wave (0 dB peak)
	freq := 1000.0
	for i := 0; i < numSamples; i++ {
		input[i] = float32(math.Sin(2.0 * math.Pi * freq * float64(i) / sampleRate))
	}

	// Process
	c.ProcessBuffer(input, output)

	// Check that output is compressed (lower than input)
	// After attack time, should see compression
	attackSamples := int(0.002 * sampleRate) // Check after 2ms

	inputRMS := float32(0.0)
	outputRMS := float32(0.0)
	count := 0

	// Calculate RMS after attack time
	for i := attackSamples; i < numSamples/2; i++ {
		inputRMS += input[i] * input[i]
		outputRMS += output[i] * output[i]
		count++
	}

	inputRMS = float32(math.Sqrt(float64(inputRMS / float32(count))))
	outputRMS = float32(math.Sqrt(float64(outputRMS / float32(count))))

	// Output should be compressed
	if outputRMS >= inputRMS {
		t.Errorf("Compression not applied: input RMS %f, output RMS %f", inputRMS, outputRMS)
	}

	// Check gain reduction is being calculated
	if c.GetGainReduction() <= 0 {
		t.Error("No gain reduction reported")
	}
}

func TestCompressorStereoLinking(t *testing.T) {
	c := NewCompressor(48000.0)
	c.SetThreshold(-20.0)
	c.SetRatio(4.0)

	// Create test buffers
	inputL := []float32{0.5, 0.0, 0.0}
	inputR := []float32{0.0, 0.5, 0.0}
	outputL := make([]float32, 3)
	outputR := make([]float32, 3)

	c.ProcessStereo(inputL, inputR, outputL, outputR)

	// Both channels should be compressed when either exceeds threshold
	// The gain reduction should be the same for both channels
	if outputL[0] >= inputL[0] || outputR[1] >= inputR[1] {
		t.Error("Stereo linking not working properly")
	}
}

func TestCompressorSidechain(t *testing.T) {
	c := NewCompressor(48000.0)
	c.SetThreshold(-20.0)
	c.SetRatio(10.0) // High ratio for clear effect

	// Input is quiet, sidechain is loud
	input := []float32{0.1, 0.1, 0.1}
	sidechain := []float32{1.0, 1.0, 1.0}
	output := make([]float32, 3)

	c.ProcessSidechain(input, sidechain, output)

	// Output should be compressed based on sidechain
	for i := range output {
		if output[i] >= input[i] {
			t.Errorf("Sidechain compression not working at sample %d", i)
		}
	}
}

func TestCompressorReset(t *testing.T) {
	c := NewCompressor(48000.0)

	// Process some loud signal
	c.Process(1.0)

	// Reset
	c.Reset()

	// Check state is cleared
	if c.GetGainReduction() != 0 {
		t.Errorf("Gain reduction not reset: %f", c.GetGainReduction())
	}
}

// Benchmark single sample processing
func BenchmarkCompressor(b *testing.B) {
	c := NewCompressor(48000.0)
	input := float32(0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Process(input)
	}
}

// Benchmark buffer processing
func BenchmarkCompressorBuffer(b *testing.B) {
	c := NewCompressor(48000.0)
	input := make([]float32, 1024)
	output := make([]float32, 1024)

	// Fill with test signal
	for i := range input {
		input[i] = float32(math.Sin(float64(i) * 0.1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.ProcessBuffer(input, output)
	}
}
