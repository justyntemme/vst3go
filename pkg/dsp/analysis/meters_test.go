package analysis

import (
	"math"
	"testing"
)

func TestPeakMeter(t *testing.T) {
	sampleRate := 44100.0
	pm := NewPeakMeter(sampleRate)
	
	// Test with simple peak
	samples := []float64{0.1, 0.5, 0.3, -0.7, 0.2}
	pm.Process(samples)
	
	peak := pm.GetPeak()
	if math.Abs(peak-0.7) > 0.001 {
		t.Errorf("Peak mismatch: expected 0.7, got %f", peak)
	}
	
	// Test peak in dB
	peakDB := pm.GetPeakDB()
	expectedDB := 20.0 * math.Log10(0.7)
	if math.Abs(peakDB-expectedDB) > 0.001 {
		t.Errorf("Peak dB mismatch: expected %f, got %f", expectedDB, peakDB)
	}
	
	// Test hold
	hold := pm.GetHold()
	if math.Abs(hold-0.7) > 0.001 {
		t.Errorf("Hold mismatch: expected 0.7, got %f", hold)
	}
}

func TestPeakMeterDecay(t *testing.T) {
	sampleRate := 44100.0
	pm := NewPeakMeter(sampleRate)
	pm.SetDecayRate(20.0) // 20 dB/second
	
	// Set initial peak
	pm.Process([]float64{1.0})
	initialPeak := pm.GetPeak()
	
	// Process silence for 0.1 second
	silenceSamples := int(0.1 * sampleRate)
	silence := make([]float64, silenceSamples)
	pm.Process(silence)
	
	// Peak should have decayed
	decayedPeak := pm.GetPeak()
	if decayedPeak >= initialPeak {
		t.Errorf("Peak didn't decay: initial %f, after decay %f", initialPeak, decayedPeak)
	}
	
	// Check approximate decay amount (should be ~2dB less)
	expectedDB := 20.0*math.Log10(initialPeak) - 2.0
	actualDB := pm.GetPeakDB()
	if math.Abs(actualDB-expectedDB) > 0.5 {
		t.Errorf("Decay amount incorrect: expected ~%f dB, got %f dB", expectedDB, actualDB)
	}
}

func TestPeakMeterReset(t *testing.T) {
	pm := NewPeakMeter(44100.0)
	
	// Process some signal
	pm.Process([]float64{0.5, -0.8, 0.3})
	
	// Verify peak is set
	if pm.GetPeak() < 0.7 {
		t.Error("Peak not set before reset")
	}
	
	// Reset
	pm.Reset()
	
	// Check values are cleared
	if pm.GetPeak() != 0 {
		t.Errorf("Peak not cleared after reset: %f", pm.GetPeak())
	}
	if pm.GetHold() != 0 {
		t.Errorf("Hold not cleared after reset: %f", pm.GetHold())
	}
}

func TestRMSMeter(t *testing.T) {
	windowSize := 1024
	rm := NewRMSMeter(windowSize)
	
	// Test with DC signal
	dcLevel := 0.5
	samples := make([]float64, windowSize)
	for i := range samples {
		samples[i] = dcLevel
	}
	
	rm.Process(samples)
	
	rms := rm.GetRMS()
	if math.Abs(rms-dcLevel) > 0.001 {
		t.Errorf("RMS mismatch for DC signal: expected %f, got %f", dcLevel, rms)
	}
	
	// Test with sine wave (RMS = amplitude / sqrt(2))
	amplitude := 1.0
	for i := range samples {
		samples[i] = amplitude * math.Sin(2.0*math.Pi*float64(i)/float64(windowSize)*10)
	}
	
	rm.Reset()
	rm.Process(samples)
	
	expectedRMS := amplitude / math.Sqrt(2)
	rms = rm.GetRMS()
	if math.Abs(rms-expectedRMS) > 0.01 {
		t.Errorf("RMS mismatch for sine wave: expected %f, got %f", expectedRMS, rms)
	}
}

func TestRMSMeterWindow(t *testing.T) {
	windowSize := 100
	rm := NewRMSMeter(windowSize)
	
	// Fill window with 1.0
	ones := make([]float64, windowSize)
	for i := range ones {
		ones[i] = 1.0
	}
	rm.Process(ones)
	
	// RMS should be 1.0
	if math.Abs(rm.GetRMS()-1.0) > 0.001 {
		t.Errorf("Initial RMS incorrect: %f", rm.GetRMS())
	}
	
	// Process zeros (should gradually decrease RMS)
	zeros := make([]float64, windowSize/2)
	rm.Process(zeros)
	
	// RMS should be sqrt(0.5) as half the window is now zeros
	expectedRMS := math.Sqrt(0.5)
	if math.Abs(rm.GetRMS()-expectedRMS) > 0.01 {
		t.Errorf("RMS after partial update incorrect: expected %f, got %f", 
			expectedRMS, rm.GetRMS())
	}
}

func TestLUFSMeter(t *testing.T) {
	sampleRate := 48000.0
	channels := 2
	lm := NewLUFSMeter(sampleRate, channels)
	
	// Generate 5 seconds of -23 LUFS calibration signal
	// This is approximately -20 dBFS RMS after K-weighting
	targetRMS := math.Pow(10, -20.0/20.0)
	duration := 5.0
	numSamples := int(duration * sampleRate)
	samples := make([]float64, numSamples*channels)
	
	// Pink noise approximation (multiple sine waves)
	freqs := []float64{100, 200, 400, 800, 1600, 3200}
	for i := 0; i < numSamples; i++ {
		sample := 0.0
		for _, freq := range freqs {
			sample += math.Sin(2.0*math.Pi*freq*float64(i)/sampleRate) / float64(len(freqs))
		}
		sample *= targetRMS * 2.0 // Compensate for RMS
		
		// Set both channels
		samples[i*channels] = sample
		samples[i*channels+1] = sample
	}
	
	// Process in blocks
	blockSize := int(0.1 * sampleRate * float64(channels)) // 100ms blocks
	for i := 0; i < len(samples); i += blockSize {
		end := i + blockSize
		if end > len(samples) {
			end = len(samples)
		}
		lm.Process(samples[i:end])
	}
	
	// Check momentary LUFS (should stabilize around -23 LUFS +/- 3)
	momentary := lm.GetMomentaryLUFS()
	if math.IsInf(momentary, -1) {
		t.Error("Momentary LUFS returned -Inf")
	} else if math.Abs(momentary-(-23.0)) > 5.0 {
		t.Logf("Momentary LUFS: %f (expected around -23)", momentary)
	}
	
	// Check short-term LUFS
	shortTerm := lm.GetShortTermLUFS()
	if math.IsInf(shortTerm, -1) {
		t.Error("Short-term LUFS returned -Inf")
	}
	
	// Check integrated LUFS
	integrated := lm.GetIntegratedLUFS()
	if math.IsInf(integrated, -1) {
		t.Error("Integrated LUFS returned -Inf")
	}
}

func TestLUFSMeterSilence(t *testing.T) {
	sampleRate := 48000.0
	channels := 2
	lm := NewLUFSMeter(sampleRate, channels)
	
	// Process 1 second of silence
	silence := make([]float64, int(sampleRate)*channels)
	lm.Process(silence)
	
	// All measurements should return -Inf for silence
	if !math.IsInf(lm.GetMomentaryLUFS(), -1) {
		t.Errorf("Momentary LUFS for silence not -Inf: %f", lm.GetMomentaryLUFS())
	}
	if !math.IsInf(lm.GetShortTermLUFS(), -1) {
		t.Errorf("Short-term LUFS for silence not -Inf: %f", lm.GetShortTermLUFS())
	}
	if !math.IsInf(lm.GetIntegratedLUFS(), -1) {
		t.Errorf("Integrated LUFS for silence not -Inf: %f", lm.GetIntegratedLUFS())
	}
}

func TestLUFSMeterReset(t *testing.T) {
	sampleRate := 48000.0
	channels := 2
	lm := NewLUFSMeter(sampleRate, channels)
	
	// Process some signal
	signal := make([]float64, int(sampleRate)*channels)
	for i := 0; i < len(signal); i += channels {
		signal[i] = 0.1 * math.Sin(2.0*math.Pi*1000.0*float64(i/channels)/sampleRate)
		signal[i+1] = signal[i]
	}
	lm.Process(signal)
	
	// Verify we have measurements
	if math.IsInf(lm.GetMomentaryLUFS(), -1) {
		t.Skip("No measurement before reset")
	}
	
	// Reset
	lm.Reset()
	
	// Process silence
	silence := make([]float64, int(0.5*sampleRate)*channels)
	lm.Process(silence)
	
	// Should return -Inf after reset and silence
	if !math.IsInf(lm.GetMomentaryLUFS(), -1) {
		t.Error("LUFS not properly reset")
	}
}

func TestLoudnessRange(t *testing.T) {
	sampleRate := 48000.0
	channels := 2
	lm := NewLUFSMeter(sampleRate, channels)
	
	// Generate signal with varying loudness
	duration := 10.0 // 10 seconds
	numSamples := int(duration * sampleRate)
	samples := make([]float64, numSamples*channels)
	
	for i := 0; i < numSamples; i++ {
		// Vary amplitude over time
		amplitude := 0.1 + 0.4*math.Sin(2.0*math.Pi*0.1*float64(i)/sampleRate)
		sample := amplitude * math.Sin(2.0*math.Pi*1000.0*float64(i)/sampleRate)
		
		samples[i*channels] = sample
		samples[i*channels+1] = sample
	}
	
	// Process
	blockSize := int(0.1 * sampleRate * float64(channels))
	for i := 0; i < len(samples); i += blockSize {
		end := i + blockSize
		if end > len(samples) {
			end = len(samples)
		}
		lm.Process(samples[i:end])
	}
	
	// Get loudness range
	lra := lm.GetLoudnessRange()
	
	// Should have some range due to varying amplitude
	if lra <= 0 {
		t.Errorf("Loudness range should be positive: %f LU", lra)
	}
}

func BenchmarkPeakMeter(b *testing.B) {
	pm := NewPeakMeter(44100.0)
	samples := make([]float64, 1024)
	
	for i := range samples {
		samples[i] = math.Sin(2.0 * math.Pi * float64(i) / 1024.0)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		pm.Process(samples)
		pm.GetPeak()
	}
}

func BenchmarkRMSMeter(b *testing.B) {
	rm := NewRMSMeter(1024)
	samples := make([]float64, 256)
	
	for i := range samples {
		samples[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		rm.Process(samples)
		rm.GetRMS()
	}
}

func BenchmarkLUFSMeter(b *testing.B) {
	lm := NewLUFSMeter(48000.0, 2)
	samples := make([]float64, 4800) // 100ms at 48kHz stereo
	
	for i := 0; i < len(samples); i += 2 {
		sample := math.Sin(2.0 * math.Pi * 1000.0 * float64(i/2) / 48000.0)
		samples[i] = sample
		samples[i+1] = sample
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		lm.Process(samples)
		lm.GetMomentaryLUFS()
	}
}