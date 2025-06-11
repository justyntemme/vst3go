package analysis

import (
	"math"
	"testing"
)

func TestPhaseScope(t *testing.T) {
	bufferSize := 256
	ps := NewPhaseScope(bufferSize)
	
	// Test 1: Mono signal (L = R)
	samplesL := make([]float64, 100)
	samplesR := make([]float64, 100)
	
	for i := range samplesL {
		signal := math.Sin(2.0 * math.Pi * float64(i) / 50.0)
		samplesL[i] = signal
		samplesR[i] = signal
	}
	
	ps.Process(samplesL, samplesR)
	
	points, brightness := ps.GetPoints()
	
	// Check that points exist
	if len(points) != 100 {
		t.Errorf("Expected 100 points, got %d", len(points))
	}
	
	// For mono signal in Lissajous mode, X=L and Y=R, so they should be equal
	for i, pt := range points {
		// Since L=R for mono, X should equal Y
		if math.Abs(pt.X-pt.Y) > 0.001 {
			t.Errorf("Mono signal point %d not on diagonal: X=%f, Y=%f", i, pt.X, pt.Y)
			break
		}
	}
	
	// Check brightness
	if len(brightness) != len(points) {
		t.Errorf("Brightness array size mismatch: %d vs %d", len(brightness), len(points))
	}
	
	// Recent points should be bright
	if brightness[len(brightness)-1] < 0.9 {
		t.Errorf("Recent point not bright enough: %f", brightness[len(brightness)-1])
	}
}

func TestPhaseScopeGoniometer(t *testing.T) {
	bufferSize := 256
	ps := NewPhaseScope(bufferSize)
	ps.SetMode(ModeGoniometer)
	
	// Test with pure side signal (L = -R)
	samplesL := make([]float64, 100)
	samplesR := make([]float64, 100)
	
	for i := range samplesL {
		signal := 0.5 * math.Sin(2.0*math.Pi*float64(i)/50.0)
		samplesL[i] = signal
		samplesR[i] = -signal
	}
	
	ps.Process(samplesL, samplesR)
	
	points, _ := ps.GetPoints()
	
	// In goniometer mode with L=-R (pure side), points should be horizontal
	for i, pt := range points {
		// After 45° rotation, pure side signal should have Y ≈ 0
		if math.Abs(pt.Y) > 0.1 {
			t.Errorf("Goniometer mode side signal point %d has non-zero Y: %f", i, pt.Y)
			break
		}
	}
}

func TestPhaseScopePolar(t *testing.T) {
	bufferSize := 256
	ps := NewPhaseScope(bufferSize)
	
	// Test with circular motion (quadrature signals)
	samplesL := make([]float64, 100)
	samplesR := make([]float64, 100)
	
	for i := range samplesL {
		phase := 2.0 * math.Pi * float64(i) / 100.0
		samplesL[i] = 0.5 * math.Cos(phase)
		samplesR[i] = 0.5 * math.Sin(phase)
	}
	
	ps.Process(samplesL, samplesR)
	
	radius, angle, bright := ps.GetPolarData()
	
	// Check array sizes
	if len(radius) != 100 || len(angle) != 100 || len(bright) != 100 {
		t.Errorf("Polar data size mismatch: r=%d, a=%d, b=%d", 
			len(radius), len(angle), len(bright))
	}
	
	// Radius should be constant for circular motion
	for i, r := range radius {
		if math.Abs(r-0.5) > 0.01 {
			t.Errorf("Radius %d not constant: %f", i, r)
			break
		}
	}
	
	// Angle should increase linearly
	for i := 1; i < len(angle); i++ {
		diff := angle[i] - angle[i-1]
		// Handle wrap around
		if diff < -math.Pi {
			diff += 2 * math.Pi
		} else if diff > math.Pi {
			diff -= 2 * math.Pi
		}
		
		expectedDiff := 2.0 * math.Pi / 100.0
		if math.Abs(diff-expectedDiff) > 0.1 {
			t.Errorf("Angle increment %d incorrect: %f (expected %f)", i, diff, expectedDiff)
			break
		}
	}
}

func TestPhaseScopeStatistics(t *testing.T) {
	bufferSize := 512
	ps := NewPhaseScope(bufferSize)
	
	// Generate test signal
	samplesL := make([]float64, bufferSize)
	samplesR := make([]float64, bufferSize)
	
	// Mixed mono and stereo content
	for i := range samplesL {
		if i < bufferSize/2 {
			// First half: mono
			signal := 0.5 * math.Sin(2.0*math.Pi*float64(i)/100.0)
			samplesL[i] = signal
			samplesR[i] = signal
		} else {
			// Second half: wide stereo
			samplesL[i] = 0.5 * math.Sin(2.0*math.Pi*float64(i)/100.0)
			samplesR[i] = 0.5 * math.Sin(2.0*math.Pi*float64(i)/100.0 + math.Pi/2)
		}
	}
	
	ps.Process(samplesL, samplesR)
	
	stats := ps.GetStatistics()
	
	// Check statistics are reasonable
	if stats.AverageMid <= 0 {
		t.Errorf("Average mid should be positive: %f", stats.AverageMid)
	}
	
	if stats.AverageSide < 0 {
		t.Errorf("Average side should be non-negative: %f", stats.AverageSide)
	}
	
	if stats.MaxRadius <= 0 || stats.MaxRadius > math.Sqrt(2) {
		t.Errorf("Max radius out of range: %f", stats.MaxRadius)
	}
	
	if stats.Width < 0 {
		t.Errorf("Width should be non-negative: %f", stats.Width)
	}
	
	if stats.PhaseConcentration < 0 || stats.PhaseConcentration > 1 {
		t.Errorf("Phase concentration out of range: %f", stats.PhaseConcentration)
	}
}

func TestPhaseScopeDecay(t *testing.T) {
	bufferSize := 128
	ps := NewPhaseScope(bufferSize)
	ps.SetDecay(0.5) // Fast decay for testing
	
	// Process initial samples
	samplesL := []float64{1.0, 0, 0, 0}
	samplesR := []float64{1.0, 0, 0, 0}
	
	ps.Process(samplesL, samplesR)
	
	// Get initial brightness
	_, brightness1 := ps.GetPoints()
	initialBright := brightness1[0]
	
	// Process more samples (zeros)
	zeros := make([]float64, 10)
	ps.Process(zeros, zeros)
	
	// Check brightness has decayed
	_, brightness2 := ps.GetPoints()
	decayedBright := brightness2[0]
	
	if decayedBright >= initialBright {
		t.Errorf("Brightness didn't decay: initial %f, after %f", initialBright, decayedBright)
	}
	
	// The decay happens each time Process is called, not per sample
	// We called Process twice (once with 4 samples, once with 10 zeros)
	// So brightness should have decayed by 0.5 once
	expectedBright := 0.5
	if math.Abs(decayedBright-expectedBright) > 0.01 {
		t.Errorf("Unexpected decay: expected ~%f, got %f", expectedBright, decayedBright)
	}
}

func TestPhaseScopeReset(t *testing.T) {
	ps := NewPhaseScope(256)
	
	// Process some data
	samplesL := make([]float64, 100)
	samplesR := make([]float64, 100)
	for i := range samplesL {
		samplesL[i] = math.Sin(float64(i) * 0.1)
		samplesR[i] = math.Cos(float64(i) * 0.1)
	}
	
	ps.Process(samplesL, samplesR)
	
	// Verify data exists
	points, _ := ps.GetPoints()
	if len(points) == 0 {
		t.Error("No points before reset")
	}
	
	// Reset
	ps.Reset()
	
	// Check everything is cleared
	points, brightness := ps.GetPoints()
	if len(points) != 0 {
		t.Errorf("Points not cleared after reset: %d points remain", len(points))
	}
	if len(brightness) != 0 {
		t.Errorf("Brightness not cleared after reset: %d values remain", len(brightness))
	}
	
	stats := ps.GetStatistics()
	if stats.MaxRadius != 0 {
		t.Errorf("Statistics not cleared after reset: MaxRadius=%f", stats.MaxRadius)
	}
}

func TestVectorScope(t *testing.T) {
	vs := NewVectorScope(256)
	
	// Process some stereo data
	samplesL := make([]float64, 100)
	samplesR := make([]float64, 100)
	
	for i := range samplesL {
		samplesL[i] = 0.7 * math.Sin(2.0*math.Pi*float64(i)/50.0)
		samplesR[i] = 0.7 * math.Sin(2.0*math.Pi*float64(i)/50.0 + 0.1)
	}
	
	vs.Process(samplesL, samplesR)
	
	points, brightness, grid, labels := vs.GetDisplay()
	
	// Check all components are present
	if len(points) != 100 {
		t.Errorf("Expected 100 points, got %d", len(points))
	}
	
	if len(brightness) != len(points) {
		t.Errorf("Brightness size mismatch: %d vs %d", len(brightness), len(points))
	}
	
	if len(grid) == 0 {
		t.Error("No grid points generated")
	}
	
	if len(labels) != 4 {
		t.Errorf("Expected 4 labels (M,L,R,S), got %d", len(labels))
	}
	
	// Check labels are correct
	expectedLabels := map[string]bool{"M": true, "L": true, "R": true, "S": true}
	for _, label := range labels {
		if !expectedLabels[label.Text] {
			t.Errorf("Unexpected label: %s", label.Text)
		}
		delete(expectedLabels, label.Text)
	}
	
	if len(expectedLabels) > 0 {
		t.Error("Missing expected labels")
	}
}

func TestPhaseScopeScale(t *testing.T) {
	ps := NewPhaseScope(128)
	ps.SetScale(2.0)
	
	// Process unit amplitude signal
	samplesL := []float64{1.0, -1.0, 1.0, -1.0}
	samplesR := []float64{1.0, -1.0, 1.0, -1.0}
	
	ps.Process(samplesL, samplesR)
	
	points, _ := ps.GetPoints()
	
	// With scale=2, points should be doubled
	for i, pt := range points {
		expectedX := samplesL[i] * 2.0
		expectedY := samplesR[i] * 2.0
		
		if math.Abs(pt.X-expectedX) > 0.001 || math.Abs(pt.Y-expectedY) > 0.001 {
			t.Errorf("Scale not applied correctly at point %d: expected (%.1f,%.1f), got (%.1f,%.1f)",
				i, expectedX, expectedY, pt.X, pt.Y)
		}
	}
}

func BenchmarkPhaseScope(b *testing.B) {
	ps := NewPhaseScope(1024)
	samplesL := make([]float64, 256)
	samplesR := make([]float64, 256)
	
	for i := range samplesL {
		samplesL[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0)
		samplesR[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0 * 1.1)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ps.Process(samplesL, samplesR)
		ps.GetPoints()
	}
}

func BenchmarkVectorScope(b *testing.B) {
	vs := NewVectorScope(1024)
	samplesL := make([]float64, 256)
	samplesR := make([]float64, 256)
	
	for i := range samplesL {
		samplesL[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0)
		samplesR[i] = math.Sin(2.0 * math.Pi * float64(i) / 256.0 * 1.1)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		vs.Process(samplesL, samplesR)
		vs.GetDisplay()
	}
}