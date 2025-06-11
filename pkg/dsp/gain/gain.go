// Package gain provides amplitude and gain-related DSP operations.
package gain

import (
	"math"
)

// Constants for dB conversion
const (
	// MinDB is the minimum dB value (effectively -infinity)
	MinDB = -200.0
	
	// Reference amplitude for dB calculations
	RefAmplitude = 1.0
)

// LinearToDb converts a linear amplitude value to decibels.
// Returns MinDB for values <= 0.
func LinearToDb(linear float64) float64 {
	if linear <= 0 {
		return MinDB
	}
	return 20.0 * math.Log10(linear)
}

// DbToLinear converts a decibel value to linear amplitude.
// Values <= MinDB return 0.
func DbToLinear(db float64) float64 {
	if db <= MinDB {
		return 0
	}
	return math.Pow(10.0, db/20.0)
}

// LinearToDb32 is the float32 version of LinearToDb.
func LinearToDb32(linear float32) float32 {
	if linear <= 0 {
		return MinDB
	}
	return 20.0 * float32(math.Log10(float64(linear)))
}

// DbToLinear32 is the float32 version of DbToLinear.
func DbToLinear32(db float32) float32 {
	if db <= MinDB {
		return 0
	}
	return float32(math.Pow(10.0, float64(db)/20.0))
}

// Apply applies a gain factor to a sample.
func Apply(sample, gain float32) float32 {
	return sample * gain
}

// ApplyDb applies a dB gain to a sample.
func ApplyDb(sample float32, db float32) float32 {
	return sample * DbToLinear32(db)
}

// ApplyBuffer applies gain to an entire buffer in-place.
func ApplyBuffer(buffer []float32, gain float32) {
	for i := range buffer {
		buffer[i] *= gain
	}
}

// ApplyDbBuffer applies dB gain to an entire buffer in-place.
func ApplyDbBuffer(buffer []float32, db float32) {
	gain := DbToLinear32(db)
	ApplyBuffer(buffer, gain)
}

// ApplyBufferTo applies gain to a buffer and stores in destination.
func ApplyBufferTo(src []float32, gain float32, dst []float32) {
	length := len(src)
	if len(dst) < length {
		length = len(dst)
	}
	
	for i := 0; i < length; i++ {
		dst[i] = src[i] * gain
	}
}

// Fade applies a linear fade between two gain values.
func Fade(buffer []float32, startGain, endGain float32) {
	if len(buffer) == 0 {
		return
	}
	
	samples := float32(len(buffer) - 1)
	if samples <= 0 {
		buffer[0] *= startGain
		return
	}
	
	gainDelta := (endGain - startGain) / samples
	gain := startGain
	
	for i := range buffer {
		buffer[i] *= gain
		gain += gainDelta
	}
}

// FadeDb applies a linear fade between two dB values.
func FadeDb(buffer []float32, startDb, endDb float32) {
	startGain := DbToLinear32(startDb)
	endGain := DbToLinear32(endDb)
	Fade(buffer, startGain, endGain)
}

// SoftClip applies soft clipping to limit signal amplitude.
func SoftClip(input, threshold float32) float32 {
	absInput := input
	if absInput < 0 {
		absInput = -absInput
	}
	if absInput <= threshold {
		return input
	}
	return threshold * fastTanh32(input/threshold)
}

// SoftClipBuffer applies soft clipping to an entire buffer.
func SoftClipBuffer(buffer []float32, threshold float32) {
	for i := range buffer {
		buffer[i] = SoftClip(buffer[i], threshold)
	}
}

// HardClip applies hard clipping to limit signal amplitude.
func HardClip(input, threshold float32) float32 {
	if input > threshold {
		return threshold
	}
	if input < -threshold {
		return -threshold
	}
	return input
}

// HardClipBuffer applies hard clipping to an entire buffer.
func HardClipBuffer(buffer []float32, threshold float32) {
	for i := range buffer {
		buffer[i] = HardClip(buffer[i], threshold)
	}
}

// fastTanh32 approximates tanh for soft clipping.
func fastTanh32(x float32) float32 {
	if x < -3 {
		return -1
	}
	if x > 3 {
		return 1
	}
	x2 := x * x
	return x * (27 + x2) / (27 + 9*x2)
}