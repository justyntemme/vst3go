package debug

import (
	"math"
	"strings"
	"testing"
)

func TestAudioAnalyzer(t *testing.T) {
	t.Run("BasicAnalysis", func(t *testing.T) {
		analyzer := NewAudioAnalyzer()
		
		// Create test buffer with known properties
		buffer := make([]float32, 1000)
		for i := range buffer {
			// Sine wave at 440Hz, 48kHz sample rate
			buffer[i] = 0.5 * float32(math.Sin(2*math.Pi*440*float64(i)/48000))
		}
		
		result := analyzer.Analyze(buffer)
		
		// Check peak (should be around 0.5)
		if result.Peak < 0.49 || result.Peak > 0.51 {
			t.Errorf("Peak incorrect: %f", result.Peak)
		}
		
		// Check RMS (sine wave RMS = peak / sqrt(2))
		expectedRMS := 0.5 / math.Sqrt(2)
		if math.Abs(float64(result.RMS)-expectedRMS) > 0.01 {
			t.Errorf("RMS incorrect: %f, expected ~%f", result.RMS, expectedRMS)
		}
		
		// Should have zero crossings
		if result.ZeroCrossings == 0 {
			t.Error("No zero crossings detected")
		}
		
		// Should not be silent
		if result.Silent {
			t.Error("Should not be silent")
		}
	})
	
	t.Run("Clipping", func(t *testing.T) {
		analyzer := NewAudioAnalyzer()
		
		buffer := []float32{0.5, 0.99, 1.0, -0.99, -1.0, 0.5}
		result := analyzer.Analyze(buffer)
		
		if !result.Clipping {
			t.Error("Should detect clipping")
		}
		
		if result.ClippedSamples != 4 { // ±0.99 and ±1.0
			t.Errorf("Wrong clipped sample count: %d", result.ClippedSamples)
		}
	})
	
	t.Run("DCOffset", func(t *testing.T) {
		analyzer := NewAudioAnalyzer()
		
		// Buffer with DC offset
		buffer := make([]float32, 100)
		for i := range buffer {
			buffer[i] = 0.3 // DC offset
		}
		
		result := analyzer.Analyze(buffer)
		
		if math.Abs(float64(result.DC)-0.3) > 0.001 {
			t.Errorf("DC offset incorrect: %f", result.DC)
		}
	})
	
	t.Run("Silence", func(t *testing.T) {
		analyzer := NewAudioAnalyzer()
		
		buffer := make([]float32, 100)
		// All zeros
		
		result := analyzer.Analyze(buffer)
		
		if !result.Silent {
			t.Error("Should detect silence")
		}
		
		if result.Peak != 0 {
			t.Error("Peak should be 0")
		}
	})
	
	t.Run("NaN", func(t *testing.T) {
		analyzer := NewAudioAnalyzer()
		
		buffer := []float32{1.0, float32(math.NaN()), 0.5, float32(math.NaN())}
		result := analyzer.Analyze(buffer)
		
		if !result.HasNaN {
			t.Error("Should detect NaN")
		}
		
		if result.NaNCount != 2 {
			t.Errorf("Wrong NaN count: %d", result.NaNCount)
		}
	})
}

func TestPrintBuffer(t *testing.T) {
	t.Run("EmptyBuffer", func(t *testing.T) {
		result := PrintBuffer([]float32{}, 80)
		if result != "Empty buffer" {
			t.Error("Wrong empty buffer message")
		}
	})
	
	t.Run("SilentBuffer", func(t *testing.T) {
		buffer := make([]float32, 100)
		result := PrintBuffer(buffer, 80)
		if !strings.Contains(result, "Silent buffer") {
			t.Error("Should indicate silent buffer")
		}
	})
	
	t.Run("Visualization", func(t *testing.T) {
		buffer := []float32{0.5, 1.0, -0.5, -1.0, 0}
		result := PrintBuffer(buffer, 5)
		
		if !strings.Contains(result, "peak: 1.000") {
			t.Error("Should show peak")
		}
		
		// Should contain visualization characters
		if !strings.Contains(result, "█") {
			t.Error("Should contain visualization")
		}
	})
}

func TestCompareBuffers(t *testing.T) {
	t.Run("IdenticalBuffers", func(t *testing.T) {
		a := []float32{1.0, 2.0, 3.0}
		b := []float32{1.0, 2.0, 3.0}
		
		result := CompareBuffers(a, b, 0.001)
		if !strings.Contains(result, "identical") {
			t.Error("Should be identical")
		}
	})
	
	t.Run("LengthMismatch", func(t *testing.T) {
		a := []float32{1.0, 2.0}
		b := []float32{1.0, 2.0, 3.0}
		
		result := CompareBuffers(a, b, 0.001)
		if !strings.Contains(result, "length mismatch") {
			t.Error("Should detect length mismatch")
		}
	})
	
	t.Run("Differences", func(t *testing.T) {
		a := []float32{1.0, 2.0, 3.0}
		b := []float32{1.0, 2.1, 3.0}
		
		result := CompareBuffers(a, b, 0.05)
		if !strings.Contains(result, "1 / 3") {
			t.Error("Should report 1 difference")
		}
		if !strings.Contains(result, "0.100000") {
			t.Error("Should report difference magnitude")
		}
	})
}

func TestCheckBuffer(t *testing.T) {
	t.Run("NoIssues", func(t *testing.T) {
		buffer := []float32{0.1, 0.2, -0.1, -0.2}
		issues := CheckBuffer(buffer, "test")
		
		if len(issues) != 0 {
			t.Errorf("Should have no issues, got: %v", issues)
		}
	})
	
	t.Run("MultipleIssues", func(t *testing.T) {
		buffer := []float32{
			float32(math.NaN()), // NaN
			1.5,                 // Over 1.0
			0.3, 0.3, 0.3,       // DC offset
		}
		
		issues := CheckBuffer(buffer, "test")
		
		hasNaN := false
		hasPeak := false
		hasDC := false
		
		for _, issue := range issues {
			if strings.Contains(issue, "NaN") {
				hasNaN = true
			}
			if strings.Contains(issue, "Peak exceeds") {
				hasPeak = true
			}
			if strings.Contains(issue, "DC offset") {
				hasDC = true
			}
		}
		
		if !hasNaN || !hasPeak || !hasDC {
			t.Error("Missing expected issues")
		}
	})
}

func TestDumpBuffer(t *testing.T) {
	t.Run("EmptyBuffer", func(t *testing.T) {
		result := DumpBuffer([]float32{}, 10)
		if result != "Empty buffer" {
			t.Error("Wrong empty buffer message")
		}
	})
	
	t.Run("FullDump", func(t *testing.T) {
		buffer := []float32{0.5, -0.5, 1.0}
		result := DumpBuffer(buffer, 10)
		
		if !strings.Contains(result, "3 samples") {
			t.Error("Should show sample count")
		}
		
		if !strings.Contains(result, "+0.500000") {
			t.Error("Should show positive value")
		}
		
		if !strings.Contains(result, "-0.500000") {
			t.Error("Should show negative value")
		}
		
		// Should have hex representation
		if !strings.Contains(result, "0x") {
			t.Error("Should show hex values")
		}
	})
	
	t.Run("LimitedDump", func(t *testing.T) {
		buffer := make([]float32, 100)
		result := DumpBuffer(buffer, 5)
		
		if !strings.Contains(result, "showing first 5") {
			t.Error("Should indicate limited dump")
		}
		
		if !strings.Contains(result, "95 more samples") {
			t.Error("Should show remaining count")
		}
	})
}

func BenchmarkAnalyzer(b *testing.B) {
	analyzer := NewAudioAnalyzer()
	buffer := make([]float32, 512)
	
	// Fill with test data
	for i := range buffer {
		buffer[i] = float32(math.Sin(2 * math.Pi * float64(i) / 100))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.Analyze(buffer)
	}
}