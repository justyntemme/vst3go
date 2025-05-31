package dynamics

import (
	"math"
	"testing"
)

func TestLimiterCreation(t *testing.T) {
	sampleRate := 48000.0
	l := NewLimiter(sampleRate)
	
	if l == nil {
		t.Fatal("Failed to create limiter")
	}
	
	if l.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", l.sampleRate, sampleRate)
	}
	
	// Check defaults
	if l.threshold != -0.3 {
		t.Errorf("Default threshold incorrect: got %f, want -0.3", l.threshold)
	}
	
	if !l.truePeak {
		t.Error("True peak detection should be enabled by default")
	}
}

func TestLimiterBrickWall(t *testing.T) {
	sampleRate := 48000.0
	l := NewLimiter(sampleRate)
	l.SetThreshold(-3.0) // -3dB ceiling
	l.SetLookahead(0.0)  // No lookahead for simple test
	l.SetTruePeak(false) // Disable true peak for predictable behavior
	
	// Test signals at various levels
	testCases := []struct {
		inputDB    float64
		expectedDB float64
		tolerance  float64
	}{
		{-10.0, -10.0, 0.1}, // Below threshold, no limiting
		{-3.0, -3.0, 0.1},   // At threshold
		{0.0, -3.0, 0.5},    // Above threshold, should be limited to -3dB
		{6.0, -3.0, 0.5},    // Well above threshold
	}
	
	for _, tc := range testCases {
		// Convert dB to linear
		input := float32(math.Pow(10.0, tc.inputDB/20.0))
		
		// Process several samples to let envelope settle
		var output float32
		for i := 0; i < 100; i++ {
			output = l.Process(input)
		}
		
		// Convert output to dB
		outputDB := 20.0 * math.Log10(math.Abs(float64(output)))
		
		if math.Abs(outputDB-tc.expectedDB) > tc.tolerance {
			t.Errorf("Limiter failed at %f dB input: got %f dB, expected %f dB",
				tc.inputDB, outputDB, tc.expectedDB)
		}
	}
}

func TestLimiterTruePeak(t *testing.T) {
	sampleRate := 48000.0
	l := NewLimiter(sampleRate)
	l.SetThreshold(-1.0)
	l.SetTruePeak(true)
	
	// Create a signal that has inter-sample peaks
	// Two samples at 0.8 can have a peak of ~1.0 between them
	signal := []float32{0.8, 0.8, -0.8, -0.8}
	output := make([]float32, len(signal))
	
	l.ProcessBuffer(signal, output)
	
	// Check that output doesn't exceed threshold
	for i, out := range output {
		outDB := 20.0 * math.Log10(math.Abs(float64(out)) + 1e-10)
		if outDB > -0.5 { // Allow small margin
			t.Errorf("True peak limiting failed at sample %d: %f dB", i, outDB)
		}
	}
}

func TestLimiterLookahead(t *testing.T) {
	sampleRate := 48000.0
	l := NewLimiter(sampleRate)
	l.SetThreshold(-6.0)
	l.SetLookahead(0.005) // 5ms lookahead
	l.SetTruePeak(false)
	l.SetRelease(0.010) // Fast release
	
	// Create a signal with a transient
	numSamples := 1000
	input := make([]float32, numSamples)
	output := make([]float32, numSamples)
	
	// Create a moderate level signal with a spike
	for i := 0; i < numSamples; i++ {
		input[i] = 0.1 // -20 dB baseline
	}
	
	// Add a loud transient
	transientStart := 500
	transientLength := 10
	for i := transientStart; i < transientStart+transientLength; i++ {
		input[i] = 1.0 // 0 dB spike
	}
	
	// Process
	l.ProcessBuffer(input, output)
	
	// With lookahead, the output at the spike should be limited
	// Check that the spike is properly limited
	for i := transientStart; i < transientStart+transientLength; i++ {
		outputDB := 20.0 * math.Log10(math.Abs(float64(output[i])) + 1e-10)
		if outputDB > -5.5 { // Allow 0.5dB margin
			t.Errorf("Lookahead limiting failed at sample %d: %f dB", i, outputDB)
		}
	}
	
	// Also verify that lookahead doesn't affect signal too much before transient
	preTransientIndex := transientStart - int(0.020 * sampleRate) // Well before lookahead window
	if preTransientIndex >= 0 {
		// Should be close to input level (allow for some processing variance)
		ratio := output[preTransientIndex] / input[preTransientIndex]
		if ratio < 0.95 || ratio > 1.05 {
			t.Errorf("Lookahead affecting signal too early: ratio %f", ratio)
		}
	}
}

func TestLimiterStereo(t *testing.T) {
	sampleRate := 48000.0
	l := NewLimiter(sampleRate)
	l.SetThreshold(-6.0)
	l.SetTruePeak(false)
	
	// Create test signals
	inputL := []float32{0.5, 0.1, 0.1}
	inputR := []float32{0.1, 0.5, 0.1}
	outputL := make([]float32, 3)
	outputR := make([]float32, 3)
	
	l.ProcessStereo(inputL, inputR, outputL, outputR)
	
	// Both channels should be limited when either exceeds
	for i := range outputL {
		if outputL[i] > 0.5 || outputR[i] > 0.5 {
			t.Errorf("Stereo limiting failed at sample %d", i)
		}
	}
}

func TestLimiterGainReduction(t *testing.T) {
	l := NewLimiter(48000.0)
	l.SetThreshold(-6.0)
	l.SetTruePeak(false)
	
	// Process a loud signal
	input := float32(1.0) // 0 dB
	for i := 0; i < 100; i++ {
		l.Process(input)
	}
	
	// Should report ~6dB gain reduction
	gr := l.GetGainReduction()
	if gr < 5.5 || gr > 6.5 {
		t.Errorf("Incorrect gain reduction: got %f dB, expected ~6 dB", gr)
	}
}

func TestLimiterReset(t *testing.T) {
	l := NewLimiter(48000.0)
	
	// Process some signal
	l.Process(1.0)
	
	// Reset
	l.Reset()
	
	// Check state
	if l.GetGainReduction() != 0 {
		t.Errorf("Gain reduction not reset: %f", l.GetGainReduction())
	}
	
	if l.lastSample != 0 {
		t.Error("True peak state not reset")
	}
}

// Benchmark limiter
func BenchmarkLimiter(b *testing.B) {
	l := NewLimiter(48000.0)
	input := float32(0.8)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Process(input)
	}
}

func BenchmarkLimiterTruePeak(b *testing.B) {
	l := NewLimiter(48000.0)
	l.SetTruePeak(true)
	input := float32(0.8)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Process(input)
	}
}