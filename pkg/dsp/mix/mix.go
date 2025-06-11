// Package mix provides audio mixing and crossfading operations.
package mix

import (
	"math"
)

// DryWet performs a dry/wet mix between two signals.
// amount parameter: 0.0 = 100% dry, 1.0 = 100% wet
func DryWet(dry, wet, amount float32) float32 {
	return dry*(1.0-amount) + wet*amount
}

// DryWetBuffer performs in-place dry/wet mixing on audio buffers.
// amount parameter: 0.0 = 100% dry, 1.0 = 100% wet
func DryWetBuffer(dry, wet []float32, amount float32) {
	dryGain := 1.0 - amount
	wetGain := amount
	
	length := len(dry)
	if len(wet) < length {
		length = len(wet)
	}
	
	for i := 0; i < length; i++ {
		dry[i] = dry[i]*dryGain + wet[i]*wetGain
	}
}

// DryWetBufferTo performs dry/wet mixing into a destination buffer.
// amount parameter: 0.0 = 100% dry, 1.0 = 100% wet
func DryWetBufferTo(dry, wet []float32, amount float32, dst []float32) {
	dryGain := 1.0 - amount
	wetGain := amount
	
	length := len(dry)
	if len(wet) < length {
		length = len(wet)
	}
	if len(dst) < length {
		length = len(dst)
	}
	
	for i := 0; i < length; i++ {
		dst[i] = dry[i]*dryGain + wet[i]*wetGain
	}
}

// CrossfadeCosine performs an equal-power cosine crossfade.
// position: 0.0 = 100% a, 1.0 = 100% b
func CrossfadeCosine(a, b, position float32) float32 {
	// Equal power cosine crossfade
	angle := position * math.Pi / 2.0
	gainA := float32(math.Cos(float64(angle)))
	gainB := float32(math.Sin(float64(angle)))
	return a*gainA + b*gainB
}

// CrossfadeLinear performs a linear crossfade.
// position: 0.0 = 100% a, 1.0 = 100% b
func CrossfadeLinear(a, b, position float32) float32 {
	return a*(1.0-position) + b*position
}

// CrossfadeBuffer performs crossfading between two buffers.
// position: 0.0 = 100% a, 1.0 = 100% b
// useEqualPower: true for cosine crossfade, false for linear
func CrossfadeBuffer(a, b []float32, position float32, useEqualPower bool, dst []float32) {
	length := len(a)
	if len(b) < length {
		length = len(b)
	}
	if len(dst) < length {
		length = len(dst)
	}
	
	if useEqualPower {
		angle := position * math.Pi / 2.0
		gainA := float32(math.Cos(float64(angle)))
		gainB := float32(math.Sin(float64(angle)))
		
		for i := 0; i < length; i++ {
			dst[i] = a[i]*gainA + b[i]*gainB
		}
	} else {
		gainA := 1.0 - position
		gainB := position
		
		for i := 0; i < length; i++ {
			dst[i] = a[i]*gainA + b[i]*gainB
		}
	}
}

// Sum adds multiple buffers together.
func Sum(buffers [][]float32, dst []float32) {
	if len(buffers) == 0 {
		return
	}
	
	length := len(dst)
	
	// Clear destination
	for i := 0; i < length; i++ {
		dst[i] = 0
	}
	
	// Sum all buffers
	for _, buffer := range buffers {
		bufLen := len(buffer)
		if bufLen < length {
			bufLen = length
		}
		
		for i := 0; i < bufLen; i++ {
			dst[i] += buffer[i]
		}
	}
}

// SumWeighted adds multiple buffers with individual gains.
func SumWeighted(buffers [][]float32, gains []float32, dst []float32) {
	if len(buffers) == 0 {
		return
	}
	
	length := len(dst)
	
	// Clear destination
	for i := 0; i < length; i++ {
		dst[i] = 0
	}
	
	// Sum all buffers with gains
	for j, buffer := range buffers {
		gain := float32(1.0)
		if j < len(gains) {
			gain = gains[j]
		}
		
		bufLen := len(buffer)
		if bufLen < length {
			bufLen = length
		}
		
		for i := 0; i < bufLen; i++ {
			dst[i] += buffer[i] * gain
		}
	}
}

// Blend mixes two buffers with automatic gain compensation.
// balance: -1.0 = 100% a, 0.0 = 50/50, 1.0 = 100% b
func Blend(a, b []float32, balance float32, dst []float32) {
	// Convert balance to 0-1 range
	position := (balance + 1.0) * 0.5
	
	// Use equal power crossfade
	CrossfadeBuffer(a, b, position, true, dst)
}