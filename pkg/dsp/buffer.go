// Package dsp provides digital signal processing utilities for audio
package dsp

import "math"

// Buffer utilities for common audio operations

// Clear zeroes a buffer - no allocations
func Clear(buffer []float32) {
	for i := range buffer {
		buffer[i] = 0
	}
}

// Copy copies from source to destination - no allocations
func Copy(dst, src []float32) {
	copy(dst, src)
}

// Add adds source to destination - no allocations
func Add(dst, src []float32) {
	n := len(dst)
	if len(src) < n {
		n = len(src)
	}
	for i := 0; i < n; i++ {
		dst[i] += src[i]
	}
}

// AddScaled adds scaled source to destination - no allocations
func AddScaled(dst, src []float32, scale float32) {
	n := len(dst)
	if len(src) < n {
		n = len(src)
	}
	for i := 0; i < n; i++ {
		dst[i] += src[i] * scale
	}
}

// Scale multiplies buffer by a constant - no allocations
func Scale(buffer []float32, scale float32) {
	for i := range buffer {
		buffer[i] *= scale
	}
}

// Mix blends two buffers with a mix factor (0=all src1, 1=all src2)
func Mix(dst, src1, src2 []float32, mix float32) {
	n := len(dst)
	if len(src1) < n {
		n = len(src1)
	}
	if len(src2) < n {
		n = len(src2)
	}

	invMix := 1.0 - mix
	for i := 0; i < n; i++ {
		dst[i] = src1[i]*invMix + src2[i]*mix
	}
}

// Peak finds the maximum absolute value in a buffer
func Peak(buffer []float32) float32 {
	peak := float32(0)
	for _, sample := range buffer {
		abs := float32(math.Abs(float64(sample)))
		if abs > peak {
			peak = abs
		}
	}
	return peak
}

// RMS calculates the root mean square of a buffer
func RMS(buffer []float32) float32 {
	if len(buffer) == 0 {
		return 0
	}

	sum := float32(0)
	for _, sample := range buffer {
		sum += sample * sample
	}

	return float32(math.Sqrt(float64(sum / float32(len(buffer)))))
}

// Clip limits samples to [-limit, limit]
func Clip(buffer []float32, limit float32) {
	for i := range buffer {
		if buffer[i] > limit {
			buffer[i] = limit
		} else if buffer[i] < -limit {
			buffer[i] = -limit
		}
	}
}

// SoftClip applies soft saturation to limit peaks
func SoftClip(buffer []float32, threshold float32) {
	for i := range buffer {
		sample := buffer[i]
		if sample > threshold {
			buffer[i] = threshold + (1.0-threshold)*float32(math.Tanh(float64(sample-threshold)))
		} else if sample < -threshold {
			buffer[i] = -threshold + (-1.0+threshold)*float32(math.Tanh(float64(sample+threshold)))
		}
	}
}
