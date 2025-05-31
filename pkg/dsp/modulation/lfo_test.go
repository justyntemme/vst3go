package modulation

import (
	"math"
	"testing"
)

func TestLFOCreation(t *testing.T) {
	sampleRate := 48000.0
	lfo := NewLFO(sampleRate)
	
	if lfo == nil {
		t.Fatal("Failed to create LFO")
	}
	
	if lfo.sampleRate != sampleRate {
		t.Errorf("Sample rate mismatch: got %f, want %f", lfo.sampleRate, sampleRate)
	}
	
	// Check defaults
	if lfo.frequency != 1.0 {
		t.Errorf("Default frequency incorrect: got %f, want 1.0", lfo.frequency)
	}
	
	if lfo.waveform != WaveformSine {
		t.Errorf("Default waveform incorrect: got %v, want WaveformSine", lfo.waveform)
	}
	
	if lfo.depth != 1.0 {
		t.Errorf("Default depth incorrect: got %f, want 1.0", lfo.depth)
	}
}

func TestLFOWaveforms(t *testing.T) {
	sampleRate := 48000.0
	lfo := NewLFO(sampleRate)
	lfo.SetFrequency(1.0) // 1 Hz for easy testing
	
	testCases := []struct {
		waveform Waveform
		name     string
		phase    float64
		expected float64
		tolerance float64
	}{
		// Sine wave tests
		{WaveformSine, "sine at 0", 0.0, 0.0, 0.001},
		{WaveformSine, "sine at 0.25", 0.25, 1.0, 0.001},
		{WaveformSine, "sine at 0.5", 0.5, 0.0, 0.001},
		{WaveformSine, "sine at 0.75", 0.75, -1.0, 0.001},
		
		// Triangle wave tests
		{WaveformTriangle, "triangle at 0", 0.0, -1.0, 0.001},
		{WaveformTriangle, "triangle at 0.25", 0.25, 0.0, 0.001},
		{WaveformTriangle, "triangle at 0.5", 0.5, 1.0, 0.001},
		{WaveformTriangle, "triangle at 0.75", 0.75, 0.0, 0.001},
		
		// Square wave tests
		{WaveformSquare, "square at 0", 0.0, 1.0, 0.001},
		{WaveformSquare, "square at 0.25", 0.25, 1.0, 0.001},
		{WaveformSquare, "square at 0.5", 0.5, -1.0, 0.001},
		{WaveformSquare, "square at 0.75", 0.75, -1.0, 0.001},
		
		// Sawtooth wave tests
		{WaveformSawtooth, "sawtooth at 0", 0.0, -1.0, 0.001},
		{WaveformSawtooth, "sawtooth at 0.25", 0.25, -0.5, 0.001},
		{WaveformSawtooth, "sawtooth at 0.5", 0.5, 0.0, 0.001},
		{WaveformSawtooth, "sawtooth at 0.75", 0.75, 0.5, 0.001},
	}
	
	for _, tc := range testCases {
		lfo.SetWaveform(tc.waveform)
		lfo.SetPhase(tc.phase)
		output := lfo.Process()
		
		if math.Abs(output-tc.expected) > tc.tolerance {
			t.Errorf("%s: got %f, expected %f", tc.name, output, tc.expected)
		}
	}
}

func TestLFOFrequency(t *testing.T) {
	sampleRate := 48000.0
	lfo := NewLFO(sampleRate)
	lfo.SetWaveform(WaveformSawtooth) // Easy to track phase
	lfo.SetFrequency(2.0) // 2 Hz
	
	// Process one second worth of samples
	samples := int(sampleRate)
	phaseAtStart := lfo.GetPhase()
	
	for i := 0; i < samples; i++ {
		lfo.Process()
	}
	
	// After 1 second at 2Hz, we should have completed ~2 cycles
	// Phase should be back near the start
	phaseAtEnd := lfo.GetPhase()
	
	if math.Abs(phaseAtEnd-phaseAtStart) > 0.01 {
		t.Errorf("Phase not correct after 2 cycles: start %f, end %f", 
			phaseAtStart, phaseAtEnd)
	}
}

func TestLFODepthAndOffset(t *testing.T) {
	lfo := NewLFO(48000.0)
	lfo.SetWaveform(WaveformSquare) // ±1 values
	
	// Test depth
	lfo.SetDepth(0.5)
	lfo.SetOffset(0.0)
	lfo.SetPhase(0.0)
	
	output := lfo.Process()
	if math.Abs(output-0.5) > 0.001 { // Should be 1.0 * 0.5 = 0.5
		t.Errorf("Depth not applied correctly: got %f, expected 0.5", output)
	}
	
	// Test offset
	lfo.SetDepth(0.5)
	lfo.SetOffset(0.25)
	lfo.SetPhase(0.0)
	
	output = lfo.Process()
	if math.Abs(output-0.75) > 0.001 { // Should be 1.0 * 0.5 + 0.25 = 0.75
		t.Errorf("Offset not applied correctly: got %f, expected 0.75", output)
	}
	
	// Test clamping
	lfo.SetDepth(1.0)
	lfo.SetOffset(0.5)
	lfo.SetPhase(0.0)
	
	output = lfo.Process()
	if output > 1.0 || output < -1.0 {
		t.Errorf("Output not clamped: %f", output)
	}
}

func TestLFOSync(t *testing.T) {
	lfo := NewLFO(48000.0)
	lfo.EnableSync(true, 0.5) // Reset to phase 0.5 on sync
	
	// Process a bit
	for i := 0; i < 1000; i++ {
		lfo.Process()
	}
	
	// Sync
	lfo.Sync()
	
	// Check phase
	if math.Abs(lfo.GetPhase()-0.5) > 0.001 {
		t.Errorf("Sync failed: phase is %f, expected 0.5", lfo.GetPhase())
	}
	
	// Test with sync disabled
	lfo.EnableSync(false, 0.5)
	lfo.SetPhase(0.75)
	lfo.Sync()
	
	// Phase should not change
	if math.Abs(lfo.GetPhase()-0.75) > 0.001 {
		t.Errorf("Sync changed phase when disabled: %f", lfo.GetPhase())
	}
}

func TestLFORandom(t *testing.T) {
	lfo := NewLFO(48000.0)
	lfo.SetWaveform(WaveformRandom)
	lfo.SetFrequency(10.0) // 10 Hz
	
	// At 10Hz with 48kHz sample rate, we should get a new random value every 4800 samples
	samplesPerPeriod := int(48000.0 / 10.0)
	
	// Collect enough samples to see multiple random values
	samples := samplesPerPeriod * 3
	values := make([]float64, samples)
	for i := 0; i < samples; i++ {
		values[i] = lfo.Process()
	}
	
	// Count unique values (should be about 3)
	uniqueValues := make(map[float64]bool)
	for _, v := range values {
		uniqueValues[v] = true
	}
	
	if len(uniqueValues) < 2 {
		t.Errorf("Random waveform not producing enough unique values: got %d unique values", 
			len(uniqueValues))
	}
	
	// Check that values are in range
	for i, v := range values {
		if v < -1.0 || v > 1.0 {
			t.Errorf("Random value out of range at sample %d: %f", i, v)
		}
	}
}

func TestLFOParameterLimits(t *testing.T) {
	lfo := NewLFO(48000.0)
	
	// Test frequency limits
	lfo.SetFrequency(0.001)
	if lfo.frequency < 0.01 {
		t.Errorf("Frequency below minimum: %f", lfo.frequency)
	}
	
	lfo.SetFrequency(100.0)
	if lfo.frequency > 20.0 {
		t.Errorf("Frequency above maximum: %f", lfo.frequency)
	}
	
	// Test depth limits
	lfo.SetDepth(-0.5)
	if lfo.depth < 0.0 {
		t.Errorf("Depth below minimum: %f", lfo.depth)
	}
	
	lfo.SetDepth(2.0)
	if lfo.depth > 1.0 {
		t.Errorf("Depth above maximum: %f", lfo.depth)
	}
	
	// Test offset limits
	lfo.SetOffset(-2.0)
	if lfo.offset < -1.0 {
		t.Errorf("Offset below minimum: %f", lfo.offset)
	}
	
	lfo.SetOffset(2.0)
	if lfo.offset > 1.0 {
		t.Errorf("Offset above maximum: %f", lfo.offset)
	}
}

func TestLFOReset(t *testing.T) {
	lfo := NewLFO(48000.0)
	
	// Process some samples
	for i := 0; i < 1000; i++ {
		lfo.Process()
	}
	
	// Reset
	lfo.Reset()
	
	// Check state
	if lfo.phase != 0.0 {
		t.Errorf("Phase not reset: %f", lfo.phase)
	}
	
	if lfo.randomCounter != 0 {
		t.Errorf("Random counter not reset: %d", lfo.randomCounter)
	}
}

// Benchmark LFO
func BenchmarkLFO(b *testing.B) {
	lfo := NewLFO(48000.0)
	lfo.SetFrequency(5.0)
	lfo.SetWaveform(WaveformSine)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lfo.Process()
	}
}

func BenchmarkLFOBuffer(b *testing.B) {
	lfo := NewLFO(48000.0)
	buffer := make([]float64, 1024)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lfo.ProcessBuffer(buffer)
	}
}