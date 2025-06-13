package dsp

import (
	"math"
	"testing"
)

func TestConstants(t *testing.T) {
	// Test that min/max values make sense
	tests := []struct {
		name string
		min  float64
		max  float64
	}{
		{"Threshold", DefaultMinThresholdDB, DefaultMaxThresholdDB},
		{"Ratio", DefaultMinRatio, DefaultMaxRatio},
		{"Attack", DefaultMinAttack, DefaultMaxAttack},
		{"Release", DefaultMinRelease, DefaultMaxRelease},
		{"Frequency", MinFrequency, MaxFrequency},
		{"Q", MinQ, MaxQ},
		{"Mix", MinMix, MaxMix},
		{"Depth", DefaultMinDepth, DefaultMaxDepth},
		{"Rate", DefaultMinRate, DefaultMaxRate},
		{"Gate Threshold", GateMinThreshold, GateMaxThreshold},
		{"Comp Threshold", CompMinThreshold, CompMaxThreshold},
		{"EQ Gain", EQMinGain, EQMaxGain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.min >= tt.max {
				t.Errorf("%s: min (%f) >= max (%f)", tt.name, tt.min, tt.max)
			}
		})
	}
}

func TestMathConstants(t *testing.T) {
	// Test mathematical constants
	if math.Abs(Pi-math.Pi) > 1e-10 {
		t.Errorf("Pi constant incorrect: %f vs %f", Pi, math.Pi)
	}

	if math.Abs(TwoPi-2*math.Pi) > 1e-10 {
		t.Errorf("TwoPi constant incorrect: %f vs %f", TwoPi, 2*math.Pi)
	}

	if math.Abs(HalfPi-math.Pi/2) > 1e-10 {
		t.Errorf("HalfPi constant incorrect: %f vs %f", HalfPi, math.Pi/2)
	}

	// Test conversion factors
	degrees := 180.0
	radians := degrees * DegreesToRadians
	if math.Abs(radians-math.Pi) > 1e-10 {
		t.Errorf("DegreesToRadians conversion incorrect: %f", radians)
	}

	backToDegrees := radians * RadiansToDegrees
	if math.Abs(backToDegrees-degrees) > 1e-10 {
		t.Errorf("RadiansToDegrees conversion incorrect: %f", backToDegrees)
	}
}

func TestChannelConstants(t *testing.T) {
	if Mono != 1 {
		t.Errorf("Mono should be 1, got %d", Mono)
	}
	if Stereo != 2 {
		t.Errorf("Stereo should be 2, got %d", Stereo)
	}
}

func TestSampleRates(t *testing.T) {
	rates := []float64{
		SampleRate32k,
		SampleRate44k1,
		SampleRate48k,
		SampleRate88k2,
		SampleRate96k,
		SampleRate192k,
	}

	expectedRates := []float64{
		32000.0,
		44100.0,
		48000.0,
		88200.0,
		96000.0,
		192000.0,
	}

	for i, rate := range rates {
		if rate != expectedRates[i] {
			t.Errorf("Sample rate %d: expected %f, got %f", i, expectedRates[i], rate)
		}
	}
}

func TestProcessorTypeString(t *testing.T) {
	tests := []struct {
		pt       ProcessorType
		expected string
	}{
		{ProcessorTypeGain, "Gain"},
		{ProcessorTypeFilter, "Filter"},
		{ProcessorTypeDynamics, "Dynamics"},
		{ProcessorTypeModulation, "Modulation"},
		{ProcessorTypeDelay, "Delay"},
		{ProcessorTypeReverb, "Reverb"},
		{ProcessorTypeDistortion, "Distortion"},
		{ProcessorTypeAnalysis, "Analysis"},
		{ProcessorTypeUtility, "Utility"},
		{ProcessorTypeUnknown, "Unknown"},
		{ProcessorType(999), "Unknown"}, // Invalid type
	}

	for _, tt := range tests {
		result := tt.pt.String()
		if result != tt.expected {
			t.Errorf("ProcessorType.String() for %v: expected %q, got %q",
				tt.pt, tt.expected, result)
		}
	}
}