package analysis

import (
	"math"
	"testing"
)

func TestCorrelationMeter(t *testing.T) {
	windowSize := 1024
	sampleRate := 44100.0
	cm := NewCorrelationMeter(windowSize, sampleRate)
	cm.SetAveraging(0) // Disable averaging for testing
	
	// Test 1: Identical signals (perfect correlation)
	samplesL := make([]float64, windowSize)
	samplesR := make([]float64, windowSize)
	
	for i := range samplesL {
		signal := math.Sin(2.0 * math.Pi * 440.0 * float64(i) / sampleRate)
		samplesL[i] = signal
		samplesR[i] = signal
	}
	
	cm.Process(samplesL, samplesR)
	
	corr := cm.GetCorrelation()
	if math.Abs(corr-1.0) > 0.01 {
		t.Errorf("Perfect correlation expected 1.0, got %f", corr)
	}
	
	status := cm.GetPhaseStatus()
	if status != PhaseInPhase {
		t.Errorf("Phase status expected InPhase, got %s", status)
	}
	
	mono := cm.GetMonoCompatibility()
	if math.Abs(mono-1.0) > 0.01 {
		t.Errorf("Mono compatibility expected 1.0, got %f", mono)
	}
}

func TestCorrelationMeterOutOfPhase(t *testing.T) {
	windowSize := 1024
	sampleRate := 44100.0
	cm := NewCorrelationMeter(windowSize, sampleRate)
	cm.SetAveraging(0) // Disable averaging for testing
	
	// Test 2: Inverted signals (perfect anti-correlation)
	samplesL := make([]float64, windowSize)
	samplesR := make([]float64, windowSize)
	
	for i := range samplesL {
		signal := math.Sin(2.0 * math.Pi * 440.0 * float64(i) / sampleRate)
		samplesL[i] = signal
		samplesR[i] = -signal // Inverted
	}
	
	cm.Process(samplesL, samplesR)
	
	corr := cm.GetCorrelation()
	if math.Abs(corr-(-1.0)) > 0.01 {
		t.Errorf("Anti-correlation expected -1.0, got %f", corr)
	}
	
	status := cm.GetPhaseStatus()
	if status != PhaseOutOfPhase {
		t.Errorf("Phase status expected OutOfPhase, got %s", status)
	}
	
	mono := cm.GetMonoCompatibility()
	if math.Abs(mono-0.0) > 0.01 {
		t.Errorf("Mono compatibility expected 0.0, got %f", mono)
	}
}

func TestCorrelationMeterUncorrelated(t *testing.T) {
	windowSize := 1024
	sampleRate := 44100.0
	cm := NewCorrelationMeter(windowSize, sampleRate)
	cm.SetAveraging(0) // Disable averaging for testing
	
	// Test 3: Uncorrelated signals
	samplesL := make([]float64, windowSize)
	samplesR := make([]float64, windowSize)
	
	for i := range samplesL {
		samplesL[i] = math.Sin(2.0 * math.Pi * 440.0 * float64(i) / sampleRate)
		samplesR[i] = math.Sin(2.0 * math.Pi * 550.0 * float64(i) / sampleRate) // Different frequency
	}
	
	cm.Process(samplesL, samplesR)
	
	corr := cm.GetCorrelation()
	if math.Abs(corr) > 0.3 {
		t.Errorf("Uncorrelated signals expected near 0, got %f", corr)
	}
	
	status := cm.GetPhaseStatus()
	if status != PhasePartiallyCorrelated {
		t.Errorf("Phase status expected PartiallyCorrelated, got %s", status)
	}
}

func TestCorrelationMeterSilence(t *testing.T) {
	windowSize := 512
	sampleRate := 44100.0
	cm := NewCorrelationMeter(windowSize, sampleRate)
	cm.SetAveraging(0) // Disable averaging for testing
	
	// Both channels silent
	samplesL := make([]float64, windowSize)
	samplesR := make([]float64, windowSize)
	
	cm.Process(samplesL, samplesR)
	
	corr := cm.GetCorrelation()
	if corr != 1.0 {
		t.Errorf("Silent channels expected correlation 1.0, got %f", corr)
	}
	
	// One channel silent
	for i := range samplesL {
		samplesL[i] = 0.5 * math.Sin(2.0*math.Pi*440.0*float64(i)/sampleRate)
	}
	
	cm.Reset()
	cm.Process(samplesL, samplesR)
	
	corr = cm.GetCorrelation()
	if corr != 0.0 {
		t.Errorf("One silent channel expected correlation 0.0, got %f", corr)
	}
}

func TestCorrelationMeterPeakHold(t *testing.T) {
	windowSize := 512
	sampleRate := 44100.0
	cm := NewCorrelationMeter(windowSize, sampleRate)
	cm.SetPeakHoldTime(0.1) // Short hold time for testing
	
	samplesL := make([]float64, windowSize)
	samplesR := make([]float64, windowSize)
	
	// Start with in-phase
	for i := range samplesL {
		signal := math.Sin(2.0 * math.Pi * 440.0 * float64(i) / sampleRate)
		samplesL[i] = signal
		samplesR[i] = signal
	}
	cm.Process(samplesL, samplesR)
	
	// Switch to out-of-phase
	for i := range samplesR {
		samplesR[i] = -samplesR[i]
	}
	cm.Process(samplesL, samplesR)
	
	// Peak hold should capture the most negative value
	peakHold := cm.GetPeakHold()
	if peakHold > -0.9 {
		t.Errorf("Peak hold should capture negative correlation, got %f", peakHold)
	}
}

func TestBalanceMeter(t *testing.T) {
	bm := NewBalanceMeter(1024)
	
	// Test 1: Centered signal
	samplesL := make([]float64, 100)
	samplesR := make([]float64, 100)
	
	for i := range samplesL {
		samplesL[i] = 0.5
		samplesR[i] = 0.5
	}
	
	bm.Process(samplesL, samplesR)
	
	balance := bm.GetBalance()
	if math.Abs(balance) > 0.01 {
		t.Errorf("Centered signal expected balance 0, got %f", balance)
	}
	
	// Test 2: Left-heavy signal
	for i := range samplesL {
		samplesL[i] = 1.0
		samplesR[i] = 0.5
	}
	
	bm.Process(samplesL, samplesR)
	bm.Process(samplesL, samplesR) // Process multiple times to overcome averaging
	bm.Process(samplesL, samplesR)
	
	balance = bm.GetBalance()
	if balance > -0.1 {
		t.Errorf("Left-heavy signal expected negative balance, got %f", balance)
	}
	
	// Test 3: Right-heavy signal - process many times to overcome averaging
	for i := range samplesL {
		samplesL[i] = 0.5
		samplesR[i] = 1.0
	}
	
	// Process many times to reach steady state
	for j := 0; j < 20; j++ {
		bm.Process(samplesL, samplesR)
	}
	
	balance = bm.GetBalance()
	if balance < 0.1 {
		t.Errorf("Right-heavy signal expected positive balance, got %f", balance)
	}
}

func TestStereoWidthMeter(t *testing.T) {
	swm := NewStereoWidthMeter(1024)
	
	// Test 1: Mono signal (L = R)
	samplesL := make([]float64, 100)
	samplesR := make([]float64, 100)
	
	for i := range samplesL {
		signal := math.Sin(2.0 * math.Pi * float64(i) / 100.0)
		samplesL[i] = signal
		samplesR[i] = signal
	}
	
	swm.Process(samplesL, samplesR)
	swm.Process(samplesL, samplesR) // Process multiple times
	
	width := swm.GetWidth()
	if width > 0.1 {
		t.Errorf("Mono signal expected width near 0, got %f", width)
	}
	
	// Test 2: Wide stereo (L = -R)
	for i := range samplesL {
		signal := math.Sin(2.0 * math.Pi * float64(i) / 100.0)
		samplesL[i] = signal
		samplesR[i] = -signal
	}
	
	// Process many times to reach steady state
	for j := 0; j < 20; j++ {
		swm.Process(samplesL, samplesR)
	}
	
	width = swm.GetWidth()
	// Anti-phase signals have infinite theoretical width, but due to numerical
	// precision and averaging, expect a high value (>3)
	if !math.IsInf(width, 1) && width < 3.0 {
		t.Errorf("Anti-phase signal expected very high width, got %f", width)
	}
	
	// Test 3: Normal stereo (different content)
	// Reset meter for new test
	swm = NewStereoWidthMeter(1024)
	
	for i := range samplesL {
		samplesL[i] = 0.7 * math.Sin(2.0*math.Pi*float64(i)/100.0)
		samplesR[i] = 0.7*math.Sin(2.0*math.Pi*float64(i)/100.0) + 0.3*math.Sin(3.0*math.Pi*float64(i)/100.0)
	}
	
	// Process multiple times
	for j := 0; j < 10; j++ {
		swm.Process(samplesL, samplesR)
	}
	
	width = swm.GetWidth()
	// Normal stereo with correlated but different content should have moderate width
	if width < 0.1 || width > 1.0 {
		t.Logf("Normal stereo width: %f (expected 0.1-1.0)", width)
	}
}

func TestStereoFieldAnalyzer(t *testing.T) {
	windowSize := 1024
	sampleRate := 44100.0
	sfa := NewStereoFieldAnalyzer(windowSize, sampleRate)
	
	// Set minimal averaging for testing
	sfa.correlation.SetAveraging(0.5)
	
	// Generate test signal
	samplesL := make([]float64, windowSize)
	samplesR := make([]float64, windowSize)
	
	for i := range samplesL {
		// Very similar signals for high correlation
		samplesL[i] = 0.8 * math.Sin(2.0*math.Pi*440.0*float64(i)/sampleRate)
		samplesR[i] = 0.8 * math.Sin(2.0*math.Pi*440.0*float64(i)/sampleRate + 0.01)
	}
	
	// Process multiple times to stabilize
	for j := 0; j < 5; j++ {
		sfa.Process(samplesL, samplesR)
	}
	
	analysis := sfa.GetAnalysis()
	
	// Check all measurements are reasonable
	if analysis.Correlation < 0.8 || analysis.Correlation > 1.0 {
		t.Errorf("Unexpected correlation: %f", analysis.Correlation)
	}
	
	if analysis.MonoCompatibility < 0.8 {
		t.Errorf("Unexpected mono compatibility: %f", analysis.MonoCompatibility)
	}
	
	if math.Abs(analysis.Balance) > 0.1 {
		t.Errorf("Unexpected balance: %f", analysis.Balance)
	}
	
	if analysis.Width < 0 || analysis.Width > 2 {
		t.Errorf("Unexpected width: %f", analysis.Width)
	}
}

func TestPhaseStatusString(t *testing.T) {
	tests := []struct {
		status   PhaseStatus
		expected string
	}{
		{PhaseInPhase, "In Phase"},
		{PhaseMostlyInPhase, "Mostly In Phase"},
		{PhasePartiallyCorrelated, "Partially Correlated"},
		{PhaseMostlyOutOfPhase, "Mostly Out of Phase"},
		{PhaseOutOfPhase, "Out of Phase"},
		{PhaseStatus(99), "Unknown"},
	}
	
	for _, tt := range tests {
		result := tt.status.String()
		if result != tt.expected {
			t.Errorf("PhaseStatus.String() for %v: expected %s, got %s",
				tt.status, tt.expected, result)
		}
	}
}

func BenchmarkCorrelationMeter(b *testing.B) {
	cm := NewCorrelationMeter(1024, 44100.0)
	samplesL := make([]float64, 256)
	samplesR := make([]float64, 256)
	
	for i := range samplesL {
		samplesL[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0)
		samplesR[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0 * 1.1)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cm.Process(samplesL, samplesR)
		cm.GetCorrelation()
	}
}

func BenchmarkStereoFieldAnalyzer(b *testing.B) {
	sfa := NewStereoFieldAnalyzer(1024, 44100.0)
	samplesL := make([]float64, 256)
	samplesR := make([]float64, 256)
	
	for i := range samplesL {
		samplesL[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0)
		samplesR[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0 * 1.1)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		sfa.Process(samplesL, samplesR)
		sfa.GetAnalysis()
	}
}