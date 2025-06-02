package distortion

import (
	"math"
	"testing"
)

func TestWaveshaper(t *testing.T) {
	t.Run("HardClip", func(t *testing.T) {
		ws := NewWaveshaper(CurveHardClip)
		ws.SetDrive(2.0)

		// Test clipping
		tests := []struct {
			input    float64
			expected float64
		}{
			{0.5, 1.0},      // Clipped
			{-0.5, -1.0},    // Clipped
			{0.25, 0.5},     // Not clipped
			{-0.25, -0.5},   // Not clipped
			{2.0, 1.0},      // Hard clipped
			{-2.0, -1.0},    // Hard clipped
		}

		for _, test := range tests {
			result := ws.Process(test.input)
			if math.Abs(result-test.expected) > 1e-6 {
				t.Errorf("HardClip(%f) = %f, expected %f", test.input, result, test.expected)
			}
		}
	})

	t.Run("SoftClip", func(t *testing.T) {
		ws := NewWaveshaper(CurveSoftClip)
		ws.SetDrive(1.0)

		// Test that output is bounded
		inputs := []float64{0.0, 0.5, 1.0, 2.0, 5.0, -1.0, -2.0, -5.0}
		for _, input := range inputs {
			result := ws.Process(input)
			if math.Abs(result) > 1.0 {
				t.Errorf("SoftClip output %f exceeds bounds for input %f", result, input)
			}
		}

		// Test symmetry
		for _, input := range []float64{0.5, 1.0, 2.0} {
			pos := ws.Process(input)
			neg := ws.Process(-input)
			if math.Abs(pos+neg) > 1e-6 {
				t.Errorf("SoftClip not symmetric: %f vs %f", pos, neg)
			}
		}
	})

	t.Run("Mix", func(t *testing.T) {
		ws := NewWaveshaper(CurveHardClip)
		ws.SetDrive(10.0)

		// Test dry signal (mix = 0)
		ws.SetMix(0.0)
		input := 0.5
		result := ws.Process(input)
		if math.Abs(result-input) > 1e-6 {
			t.Errorf("Mix=0 should return dry signal: got %f, expected %f", result, input)
		}

		// Test wet signal (mix = 1)
		ws.SetMix(1.0)
		result = ws.Process(input)
		if math.Abs(result-1.0) > 1e-6 {
			t.Errorf("Mix=1 with hard clip should return 1.0: got %f", result)
		}

		// Test 50% mix
		ws.SetMix(0.5)
		result = ws.Process(input)
		expected := 0.5*input + 0.5*1.0
		if math.Abs(result-expected) > 1e-6 {
			t.Errorf("Mix=0.5 incorrect: got %f, expected %f", result, expected)
		}
	})

	t.Run("Foldback", func(t *testing.T) {
		ws := NewWaveshaper(CurveFoldback)
		ws.SetDrive(4.0)

		// Test that output is bounded
		inputs := []float64{0.0, 0.5, 1.0, 2.0, 5.0, -1.0, -2.0, -5.0}
		for _, input := range inputs {
			result := ws.Process(input)
			if math.Abs(result) > 1.0 {
				t.Errorf("Foldback output %f exceeds bounds for input %f", result, input)
			}
		}
	})

	t.Run("Asymmetric", func(t *testing.T) {
		ws := NewWaveshaper(CurveAsymmetric)
		ws.SetAsymmetry(0.5)
		ws.SetDrive(2.0)

		// Positive and negative inputs should produce different magnitudes
		input := 0.5
		pos := ws.Process(input)
		neg := ws.Process(-input)

		if math.Abs(pos) == math.Abs(neg) {
			t.Errorf("Asymmetric should produce different magnitudes: %f vs %f", pos, neg)
		}
	})

	t.Run("ProcessBuffer", func(t *testing.T) {
		ws := NewWaveshaper(CurveSoftClip)
		ws.SetDrive(2.0)

		input := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
		output := make([]float64, len(input))

		ws.ProcessBuffer(input, output)

		// Verify each sample
		for i, in := range input {
			expected := ws.Process(in)
			if math.Abs(output[i]-expected) > 1e-6 {
				t.Errorf("ProcessBuffer[%d]: got %f, expected %f", i, output[i], expected)
			}
		}
	})
}

func TestWaveshaperChain(t *testing.T) {
	chain := NewWaveshaperChain()

	// Add two shapers: soft clip then hard clip
	soft := NewWaveshaper(CurveSoftClip)
	soft.SetDrive(3.0)
	chain.AddShaper(soft)

	hard := NewWaveshaper(CurveHardClip)
	hard.SetDrive(1.5)
	chain.AddShaper(hard)

	// Test that chaining works
	input := 2.0
	result := chain.Process(input)

	// Manually compute expected
	intermediate := soft.Process(input)
	expected := hard.Process(intermediate)

	if math.Abs(result-expected) > 1e-6 {
		t.Errorf("Chain process incorrect: got %f, expected %f", result, expected)
	}

	// Test buffer processing
	inputBuf := []float64{0.5, 1.0, 1.5, 2.0}
	outputBuf := make([]float64, len(inputBuf))
	chain.ProcessBuffer(inputBuf, outputBuf)

	for i := range inputBuf {
		expected := chain.Process(inputBuf[i])
		if math.Abs(outputBuf[i]-expected) > 1e-6 {
			t.Errorf("Chain buffer[%d]: got %f, expected %f", i, outputBuf[i], expected)
		}
	}
}

func BenchmarkWaveshaper(b *testing.B) {
	curves := []struct {
		name string
		curve CurveType
	}{
		{"HardClip", CurveHardClip},
		{"SoftClip", CurveSoftClip},
		{"Saturate", CurveSaturate},
		{"Foldback", CurveFoldback},
		{"Asymmetric", CurveAsymmetric},
		{"Sine", CurveSine},
		{"Exponential", CurveExponential},
	}

	for _, c := range curves {
		b.Run(c.name, func(b *testing.B) {
			ws := NewWaveshaper(c.curve)
			ws.SetDrive(2.0)
			
			// Create test buffer
			input := make([]float64, 1024)
			output := make([]float64, 1024)
			for i := range input {
				input[i] = float64(i)/512.0 - 1.0
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ws.ProcessBuffer(input, output)
			}
		})
	}
}