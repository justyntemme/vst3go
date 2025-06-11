package pan

import (
	"math"
	"testing"
)

func TestMonoToStereo(t *testing.T) {
	tests := []struct {
		name string
		pan  float32
		law  Law
	}{
		{"Center Linear", 0.0, Linear},
		{"Left Linear", -1.0, Linear},
		{"Right Linear", 1.0, Linear},
		{"Center ConstantPower", 0.0, ConstantPower},
		{"Left ConstantPower", -1.0, ConstantPower},
		{"Right ConstantPower", 1.0, ConstantPower},
		{"Center Balanced", 0.0, Balanced},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, right := MonoToStereo(tt.pan, tt.law)
			
			// Check basic constraints
			if left < 0 || left > 1 || right < 0 || right > 1 {
				t.Errorf("Gains out of range: left=%f, right=%f", left, right)
			}
			
			// Check specific positions
			switch tt.pan {
			case -1.0: // Hard left
				if left < 0.9 || right > 0.1 {
					t.Errorf("Hard left incorrect: left=%f, right=%f", left, right)
				}
			case 0.0: // Center
				if math.Abs(float64(left-right)) > 0.001 {
					t.Errorf("Center not balanced: left=%f, right=%f", left, right)
				}
				// Check constant power at center
				if tt.law == ConstantPower {
					power := left*left + right*right
					if math.Abs(float64(power-1.0)) > 0.01 {
						t.Errorf("Constant power violation at center: %f", power)
					}
				}
			case 1.0: // Hard right
				if right < 0.9 || left > 0.1 {
					t.Errorf("Hard right incorrect: left=%f, right=%f", left, right)
				}
			}
		})
	}
}

func TestProcess(t *testing.T) {
	mono := []float32{1.0, 0.5, -0.5, -1.0}
	leftOut := make([]float32, 4)
	rightOut := make([]float32, 4)
	
	// Test center pan
	Process(mono, 0.0, ConstantPower, leftOut, rightOut)
	
	for i := range mono {
		if math.Abs(float64(leftOut[i]-rightOut[i])) > 0.001 {
			t.Errorf("Center pan not balanced at sample %d", i)
		}
	}
	
	// Test hard left
	Process(mono, -1.0, Linear, leftOut, rightOut)
	
	for i, v := range leftOut {
		if math.Abs(float64(v-mono[i])) > 0.001 {
			t.Errorf("Hard left: left[%d] should equal mono", i)
		}
		if rightOut[i] > 0.001 {
			t.Errorf("Hard left: right[%d] should be ~0", i)
		}
	}
}

func TestProcessStereo(t *testing.T) {
	leftIn := []float32{1.0, 1.0, 1.0, 1.0}
	rightIn := []float32{0.5, 0.5, 0.5, 0.5}
	leftOut := make([]float32, 4)
	rightOut := make([]float32, 4)
	
	// Test center - should pass through
	ProcessStereo(leftIn, rightIn, 0.0, Linear, leftOut, rightOut)
	
	for i := range leftIn {
		if leftOut[i] != leftIn[i] || rightOut[i] != rightIn[i] {
			t.Errorf("Center pan should pass through at sample %d", i)
		}
	}
	
	// Test pan left
	ProcessStereo(leftIn, rightIn, -0.5, Linear, leftOut, rightOut)
	
	for i := range leftIn {
		if leftOut[i] != leftIn[i] {
			t.Errorf("Pan left: left channel should be unchanged at sample %d", i)
		}
		if rightOut[i] >= rightIn[i] {
			t.Errorf("Pan left: right channel should be attenuated at sample %d", i)
		}
	}
}

func TestWidth(t *testing.T) {
	leftIn := []float32{1.0, 1.0, 1.0, 1.0}
	rightIn := []float32{-1.0, -1.0, -1.0, -1.0}
	leftOut := make([]float32, 4)
	rightOut := make([]float32, 4)
	
	// Test mono (width = 0)
	Width(leftIn, rightIn, 0.0, leftOut, rightOut)
	
	for i := range leftIn {
		// Should collapse to mono (mid only)
		if leftOut[i] != 0.0 || rightOut[i] != 0.0 {
			t.Errorf("Width 0: should be mono at sample %d", i)
		}
	}
	
	// Test normal stereo (width = 1)
	Width(leftIn, rightIn, 1.0, leftOut, rightOut)
	
	for i := range leftIn {
		if leftOut[i] != leftIn[i] || rightOut[i] != rightIn[i] {
			t.Errorf("Width 1: should be unchanged at sample %d", i)
		}
	}
	
	// Test extra wide (width = 2)
	Width(leftIn, rightIn, 2.0, leftOut, rightOut)
	
	for i := range leftIn {
		// Should be wider than original
		if math.Abs(float64(leftOut[i])) <= math.Abs(float64(leftIn[i])) {
			t.Errorf("Width 2: should be wider at sample %d", i)
		}
	}
}

func TestBalance(t *testing.T) {
	leftIn := []float32{1.0, 1.0, 1.0, 1.0}
	rightIn := []float32{1.0, 1.0, 1.0, 1.0}
	leftOut := make([]float32, 4)
	rightOut := make([]float32, 4)
	
	// Test center balance
	Balance(leftIn, rightIn, 0.0, leftOut, rightOut)
	
	for i := range leftIn {
		if leftOut[i] != leftIn[i] || rightOut[i] != rightIn[i] {
			t.Errorf("Balance 0: should be unchanged at sample %d", i)
		}
	}
	
	// Test balance left
	Balance(leftIn, rightIn, -0.5, leftOut, rightOut)
	
	for i := range leftIn {
		if leftOut[i] != leftIn[i] {
			t.Errorf("Balance left: left channel should be unchanged at sample %d", i)
		}
		if rightOut[i] != 0.5 {
			t.Errorf("Balance left: right channel should be 0.5 at sample %d", i)
		}
	}
	
	// Test balance right
	Balance(leftIn, rightIn, 0.5, leftOut, rightOut)
	
	for i := range leftIn {
		if leftOut[i] != 0.5 {
			t.Errorf("Balance right: left channel should be 0.5 at sample %d", i)
		}
		if rightOut[i] != rightIn[i] {
			t.Errorf("Balance right: right channel should be unchanged at sample %d", i)
		}
	}
}

func TestAutoPan(t *testing.T) {
	sampleRate := float32(44100)
	ap := NewAutoPan(1.0, 1.0, ConstantPower) // 1Hz, full depth
	
	mono := make([]float32, int(sampleRate)) // 1 second
	leftOut := make([]float32, len(mono))
	rightOut := make([]float32, len(mono))
	
	// Fill with constant signal
	for i := range mono {
		mono[i] = 1.0
	}
	
	ap.Process(mono, sampleRate, leftOut, rightOut)
	
	// Check that panning occurs
	// At 0.25 seconds, should be panned right
	quarterSecond := int(sampleRate / 4)
	if leftOut[quarterSecond] >= rightOut[quarterSecond] {
		t.Error("AutoPan should be panned right at 0.25s")
	}
	
	// At 0.75 seconds, should be panned left
	threeQuarterSecond := int(3 * sampleRate / 4)
	if leftOut[threeQuarterSecond] <= rightOut[threeQuarterSecond] {
		t.Error("AutoPan should be panned left at 0.75s")
	}
	
	// Test reset
	ap.Reset()
	if ap.phase != 0 {
		t.Error("Reset should clear phase")
	}
}

func BenchmarkMonoToStereo(b *testing.B) {
	pan := float32(0.5)
	for i := 0; i < b.N; i++ {
		_, _ = MonoToStereo(pan, ConstantPower)
	}
}

func BenchmarkProcess(b *testing.B) {
	mono := make([]float32, 512)
	leftOut := make([]float32, 512)
	rightOut := make([]float32, 512)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Process(mono, 0.5, ConstantPower, leftOut, rightOut)
	}
}