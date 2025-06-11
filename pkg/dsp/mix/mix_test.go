package mix

import (
	"math"
	"testing"
)

func TestDryWet(t *testing.T) {
	tests := []struct {
		name     string
		dry      float32
		wet      float32
		amount   float32
		expected float32
	}{
		{"100% dry", 1.0, 0.5, 0.0, 1.0},
		{"100% wet", 1.0, 0.5, 1.0, 0.5},
		{"50/50 mix", 1.0, 0.5, 0.5, 0.75},
		{"25% wet", 1.0, 0.0, 0.25, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DryWet(tt.dry, tt.wet, tt.amount)
			if math.Abs(float64(result-tt.expected)) > 0.001 {
				t.Errorf("DryWet(%f, %f, %f) = %f, want %f", 
					tt.dry, tt.wet, tt.amount, result, tt.expected)
			}
		})
	}
}

func TestDryWetBuffer(t *testing.T) {
	dry := []float32{1.0, 1.0, 1.0, 1.0}
	wet := []float32{0.0, 0.0, 0.0, 0.0}
	amount := float32(0.5)
	
	DryWetBuffer(dry, wet, amount)
	
	for i, v := range dry {
		expected := float32(0.5) // 50% of 1.0 + 50% of 0.0
		if math.Abs(float64(v-expected)) > 0.001 {
			t.Errorf("DryWetBuffer: dry[%d] = %f, want %f", i, v, expected)
		}
	}
}

func TestCrossfadeCosine(t *testing.T) {
	a := float32(1.0)
	b := float32(0.0)
	
	tests := []struct {
		position float32
		name     string
	}{
		{0.0, "100% A"},
		{0.5, "50/50"},
		{1.0, "100% B"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CrossfadeCosine(a, b, tt.position)
			
			// At position 0.5, should maintain constant power
			if tt.position == 0.5 {
				// Check that it's approximately 0.707 (equal power)
				if math.Abs(float64(result-0.707)) > 0.01 {
					t.Errorf("CrossfadeCosine at 50%% = %f, want ~0.707", result)
				}
			}
		})
	}
}

func TestCrossfadeLinear(t *testing.T) {
	a := float32(1.0)
	b := float32(0.0)
	
	tests := []struct {
		position float32
		expected float32
	}{
		{0.0, 1.0},
		{0.5, 0.5},
		{1.0, 0.0},
		{0.25, 0.75},
	}
	
	for _, tt := range tests {
		result := CrossfadeLinear(a, b, tt.position)
		if math.Abs(float64(result-tt.expected)) > 0.001 {
			t.Errorf("CrossfadeLinear(%f, %f, %f) = %f, want %f",
				a, b, tt.position, result, tt.expected)
		}
	}
}

func TestCrossfadeBuffer(t *testing.T) {
	a := []float32{1.0, 1.0, 1.0, 1.0}
	b := []float32{0.0, 0.0, 0.0, 0.0}
	dst := make([]float32, 4)
	
	// Test linear crossfade at 50%
	CrossfadeBuffer(a, b, 0.5, false, dst)
	
	for i, v := range dst {
		if math.Abs(float64(v-0.5)) > 0.001 {
			t.Errorf("Linear crossfade: dst[%d] = %f, want 0.5", i, v)
		}
	}
	
	// Test equal power crossfade at 50%
	CrossfadeBuffer(a, b, 0.5, true, dst)
	
	for i, v := range dst {
		// Should be approximately 0.707 for equal power
		if math.Abs(float64(v-0.707)) > 0.01 {
			t.Errorf("Equal power crossfade: dst[%d] = %f, want ~0.707", i, v)
		}
	}
}

func TestSum(t *testing.T) {
	buffers := [][]float32{
		{1.0, 2.0, 3.0, 4.0},
		{0.5, 0.5, 0.5, 0.5},
		{-0.5, -0.5, -0.5, -0.5},
	}
	dst := make([]float32, 4)
	expected := []float32{1.0, 2.0, 3.0, 4.0}
	
	Sum(buffers, dst)
	
	for i, v := range dst {
		if math.Abs(float64(v-expected[i])) > 0.001 {
			t.Errorf("Sum: dst[%d] = %f, want %f", i, v, expected[i])
		}
	}
}

func TestSumWeighted(t *testing.T) {
	buffers := [][]float32{
		{1.0, 1.0, 1.0, 1.0},
		{1.0, 1.0, 1.0, 1.0},
	}
	gains := []float32{0.5, 0.25}
	dst := make([]float32, 4)
	expected := float32(0.75) // 1.0*0.5 + 1.0*0.25
	
	SumWeighted(buffers, gains, dst)
	
	for i, v := range dst {
		if math.Abs(float64(v-expected)) > 0.001 {
			t.Errorf("SumWeighted: dst[%d] = %f, want %f", i, v, expected)
		}
	}
}

func TestBlend(t *testing.T) {
	a := []float32{1.0, 1.0, 1.0, 1.0}
	b := []float32{0.0, 0.0, 0.0, 0.0}
	dst := make([]float32, 4)
	
	// Test center balance
	Blend(a, b, 0.0, dst)
	
	for i, v := range dst {
		// At center, should be equal power mix
		if math.Abs(float64(v-0.707)) > 0.01 {
			t.Errorf("Blend center: dst[%d] = %f, want ~0.707", i, v)
		}
	}
	
	// Test hard left
	Blend(a, b, -1.0, dst)
	
	for i, v := range dst {
		if math.Abs(float64(v-1.0)) > 0.001 {
			t.Errorf("Blend left: dst[%d] = %f, want 1.0", i, v)
		}
	}
}

func BenchmarkDryWetBuffer(b *testing.B) {
	dry := make([]float32, 512)
	wet := make([]float32, 512)
	
	for i := range dry {
		dry[i] = 0.5
		wet[i] = 0.25
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DryWetBuffer(dry, wet, 0.5)
	}
}

func BenchmarkCrossfadeBuffer(b *testing.B) {
	a := make([]float32, 512)
	bb := make([]float32, 512)
	dst := make([]float32, 512)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CrossfadeBuffer(a, bb, 0.5, true, dst)
	}
}