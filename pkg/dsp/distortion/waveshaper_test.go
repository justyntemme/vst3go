package distortion

import (
	"math"
	"testing"
)

func TestWaveshaper(t *testing.T) {
	ws := NewWaveshaper()
	
	t.Run("HardClip", func(t *testing.T) {
		ws.SetCurveType(CurveHardClip)
		ws.SetDrive(1.0)
		ws.SetMix(1.0)
		
		tests := []struct {
			input    float64
			expected float64
		}{
			{0.5, 0.5},
			{1.5, 1.0},
			{-1.5, -1.0},
			{0.0, 0.0},
		}
		
		for _, test := range tests {
			result := ws.Process(test.input)
			if math.Abs(result-test.expected) > 1e-9 {
				t.Errorf("HardClip(%f) = %f, want %f", test.input, result, test.expected)
			}
		}
	})
	
	t.Run("SoftClip", func(t *testing.T) {
		ws.SetCurveType(CurveSoftClip)
		ws.SetDrive(1.0)
		
		// Test that soft clipping is bounded
		result := ws.Process(10.0)
		if result >= 1.0 || result <= -1.0 {
			t.Errorf("SoftClip should be bounded, got %f", result)
		}
		
		// Test symmetry
		pos := ws.Process(0.5)
		neg := ws.Process(-0.5)
		if math.Abs(pos + neg) > 1e-9 {
			t.Errorf("SoftClip should be symmetric, got %f and %f", pos, neg)
		}
	})
	
	t.Run("Drive", func(t *testing.T) {
		ws.SetCurveType(CurveSoftClip)
		ws.SetDrive(2.0)
		
		withDrive := ws.Process(0.5)
		ws.SetDrive(1.0)
		withoutDrive := ws.Process(0.5)
		
		if withDrive <= withoutDrive {
			t.Errorf("Higher drive should increase distortion")
		}
	})
	
	t.Run("Mix", func(t *testing.T) {
		ws.SetCurveType(CurveHardClip)
		ws.SetDrive(2.0)
		ws.SetMix(0.5)
		
		input := 0.5
		result := ws.Process(input)
		
		// With 50% mix, result should be between clean and distorted
		ws.SetMix(1.0)
		fullDistortion := ws.Process(input)
		
		if result == input || result == fullDistortion {
			t.Errorf("Mix should blend between clean and distorted signal")
		}
	})
	
	t.Run("Foldback", func(t *testing.T) {
		ws.SetCurveType(CurveFoldback)
		ws.SetDrive(1.0)
		ws.SetMix(1.0)
		
		// Test foldback behavior
		result := ws.Process(1.5)
		if result >= 1.0 || result <= -1.0 {
			t.Errorf("Foldback should wrap signal, got %f", result)
		}
	})
	
	t.Run("Asymmetric", func(t *testing.T) {
		ws.SetCurveType(CurveAsymmetric)
		ws.SetAsymmetry(0.5)
		ws.SetDrive(1.0)
		
		pos := ws.Process(0.5)
		neg := ws.Process(-0.5)
		
		// With asymmetry, positive and negative should differ
		if math.Abs(pos + neg) < 0.01 {
			t.Errorf("Asymmetric curve should treat positive and negative differently")
		}
	})
	
	t.Run("ProcessBlock", func(t *testing.T) {
		ws.SetCurveType(CurveSoftClip)
		ws.SetDrive(1.5)
		
		input := []float64{0.0, 0.5, -0.5, 1.0, -1.0}
		output := make([]float64, len(input))
		
		ws.ProcessBlock(input, output)
		
		for i, v := range input {
			expected := ws.Process(v)
			if math.Abs(output[i]-expected) > 1e-9 {
				t.Errorf("ProcessBlock[%d] = %f, want %f", i, output[i], expected)
			}
		}
	})
}

func TestWaveshaperUtilityFunctions(t *testing.T) {
	t.Run("Sigmoid", func(t *testing.T) {
		// Test sigmoid is bounded between -1 and 1
		result := Sigmoid(100.0, 1.0)
		if result >= 1.0 || result <= -1.0 {
			t.Errorf("Sigmoid should be bounded, got %f", result)
		}
		
		// Test sigmoid(0) = 0
		result = Sigmoid(0.0, 1.0)
		if math.Abs(result) > 1e-9 {
			t.Errorf("Sigmoid(0) should be 0, got %f", result)
		}
	})
	
	t.Run("Polynomial", func(t *testing.T) {
		// Test simple polynomial: 1 + 2x + 3x^2
		coeffs := []float64{1.0, 2.0, 3.0}
		result := Polynomial(2.0, coeffs)
		expected := 1.0 + 2.0*2.0 + 3.0*4.0
		
		if math.Abs(result-expected) > 1e-9 {
			t.Errorf("Polynomial(2.0) = %f, want %f", result, expected)
		}
	})
	
	t.Run("ChebyshevPolynomial", func(t *testing.T) {
		// Test known Chebyshev values
		tests := []struct {
			x        float64
			order    int
			expected float64
		}{
			{0.5, 0, 1.0},      // T_0(x) = 1
			{0.5, 1, 0.5},      // T_1(x) = x
			{0.5, 2, -0.5},     // T_2(0.5) = 2(0.5)^2 - 1 = -0.5
			{1.0, 3, 1.0},      // T_3(1) = 1
		}
		
		for _, test := range tests {
			result := ChebyshevPolynomial(test.x, test.order)
			if math.Abs(result-test.expected) > 1e-9 {
				t.Errorf("T_%d(%f) = %f, want %f", test.order, test.x, result, test.expected)
			}
		}
	})
}

func BenchmarkWaveshaper(b *testing.B) {
	ws := NewWaveshaper()
	ws.SetCurveType(CurveSoftClip)
	ws.SetDrive(2.0)
	
	input := make([]float64, 512)
	output := make([]float64, 512)
	
	for i := range input {
		input[i] = float64(i) / 512.0 * 2.0 - 1.0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ws.ProcessBlock(input, output)
	}
}