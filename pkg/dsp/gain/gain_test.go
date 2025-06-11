package gain

import (
	"math"
	"testing"
)

func TestDbConversion(t *testing.T) {
	tests := []struct {
		name     string
		linear   float64
		db       float64
		epsilon  float64
	}{
		{"Unity gain", 1.0, 0.0, 0.001},
		{"Half amplitude", 0.5, -6.02, 0.01},
		{"Double amplitude", 2.0, 6.02, 0.01},
		{"Quarter amplitude", 0.25, -12.04, 0.01},
		{"Zero amplitude", 0.0, MinDB, 0.001},
		{"Negative amplitude", -1.0, MinDB, 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test LinearToDb
			gotDb := LinearToDb(tt.linear)
			if math.Abs(gotDb-tt.db) > tt.epsilon {
				t.Errorf("LinearToDb(%f) = %f, want %f", tt.linear, gotDb, tt.db)
			}

			// Test DbToLinear (skip for MinDB cases)
			if tt.db != MinDB {
				gotLinear := DbToLinear(tt.db)
				if math.Abs(gotLinear-math.Abs(tt.linear)) > tt.epsilon {
					t.Errorf("DbToLinear(%f) = %f, want %f", tt.db, gotLinear, math.Abs(tt.linear))
				}
			}
		})
	}
}

func TestDb32Conversion(t *testing.T) {
	// Test float32 versions
	linear := float32(0.5)
	expectedDb := float32(-6.02)
	
	gotDb := LinearToDb32(linear)
	if math.Abs(float64(gotDb-expectedDb)) > 0.1 {
		t.Errorf("LinearToDb32(%f) = %f, want %f", linear, gotDb, expectedDb)
	}
	
	gotLinear := DbToLinear32(expectedDb)
	if math.Abs(float64(gotLinear-linear)) > 0.01 {
		t.Errorf("DbToLinear32(%f) = %f, want %f", expectedDb, gotLinear, linear)
	}
}

func TestApplyGain(t *testing.T) {
	sample := float32(0.5)
	gain := float32(2.0)
	expected := float32(1.0)
	
	result := Apply(sample, gain)
	if result != expected {
		t.Errorf("Apply(%f, %f) = %f, want %f", sample, gain, result, expected)
	}
}

func TestApplyDb(t *testing.T) {
	sample := float32(0.5)
	db := float32(6.0) // ~2x gain
	
	result := ApplyDb(sample, db)
	expected := sample * DbToLinear32(db)
	
	if math.Abs(float64(result-expected)) > 0.001 {
		t.Errorf("ApplyDb(%f, %f) = %f, want %f", sample, db, result, expected)
	}
}

func TestApplyBuffer(t *testing.T) {
	buffer := []float32{1.0, 0.5, -0.5, -1.0}
	gain := float32(0.5)
	expected := []float32{0.5, 0.25, -0.25, -0.5}
	
	ApplyBuffer(buffer, gain)
	
	for i, v := range buffer {
		if v != expected[i] {
			t.Errorf("ApplyBuffer: buffer[%d] = %f, want %f", i, v, expected[i])
		}
	}
}

func TestFade(t *testing.T) {
	buffer := []float32{1.0, 1.0, 1.0, 1.0}
	startGain := float32(0.0)
	endGain := float32(1.0)
	
	Fade(buffer, startGain, endGain)
	
	// Check that gain increases linearly
	for i := 0; i < len(buffer); i++ {
		expectedGain := float32(i) / float32(len(buffer)-1)
		if math.Abs(float64(buffer[i]-expectedGain)) > 0.001 {
			t.Errorf("Fade: buffer[%d] = %f, want ~%f", i, buffer[i], expectedGain)
		}
	}
}

func TestSoftClip(t *testing.T) {
	tests := []struct {
		input     float32
		threshold float32
		name      string
	}{
		{0.5, 1.0, "Below threshold"},
		{1.5, 1.0, "Above threshold"},
		{-1.5, 1.0, "Negative above threshold"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SoftClip(tt.input, tt.threshold)
			
			// Check that result is within threshold
			if math.Abs(float64(result)) > float64(tt.threshold)*1.1 { // Allow small overshoot
				t.Errorf("SoftClip(%f, %f) = %f, exceeds threshold", tt.input, tt.threshold, result)
			}
			
			// Check that signal below threshold is unchanged
			if math.Abs(float64(tt.input)) <= float64(tt.threshold) && result != tt.input {
				t.Errorf("SoftClip(%f, %f) = %f, should be unchanged", tt.input, tt.threshold, result)
			}
		})
	}
}

func TestHardClip(t *testing.T) {
	tests := []struct {
		input     float32
		threshold float32
		expected  float32
	}{
		{0.5, 1.0, 0.5},
		{1.5, 1.0, 1.0},
		{-1.5, 1.0, -1.0},
		{0.0, 1.0, 0.0},
	}
	
	for _, tt := range tests {
		result := HardClip(tt.input, tt.threshold)
		if result != tt.expected {
			t.Errorf("HardClip(%f, %f) = %f, want %f", tt.input, tt.threshold, result, tt.expected)
		}
	}
}

func BenchmarkDbToLinear32(b *testing.B) {
	db := float32(-6.0)
	for i := 0; i < b.N; i++ {
		_ = DbToLinear32(db)
	}
}

func BenchmarkSoftClip(b *testing.B) {
	input := float32(1.5)
	threshold := float32(1.0)
	for i := 0; i < b.N; i++ {
		_ = SoftClip(input, threshold)
	}
}