// Package interpolation provides audio interpolation and resampling utilities.
package interpolation

import (
	"math"
)

// Linear performs linear interpolation between two samples.
// frac is the fractional position between y0 and y1 (0.0 to 1.0).
func Linear(y0, y1, frac float32) float32 {
	return y0 + (y1-y0)*frac
}

// Cubic performs 4-point cubic interpolation.
// frac is the fractional position between y1 and y2 (0.0 to 1.0).
func Cubic(y0, y1, y2, y3, frac float32) float32 {
	// Catmull-Rom cubic spline
	c0 := y1
	c1 := 0.5 * (y2 - y0)
	c2 := y0 - 2.5*y1 + 2*y2 - 0.5*y3
	c3 := 0.5 * (y3 - y0 + 3*(y1-y2))
	
	return ((c3*frac+c2)*frac+c1)*frac + c0
}

// Hermite performs 4-point Hermite interpolation.
// frac is the fractional position between y1 and y2 (0.0 to 1.0).
func Hermite(y0, y1, y2, y3, frac float32) float32 {
	// 4-point, 3rd-order Hermite
	c0 := y1
	c1 := 0.5 * (y2 - y0)
	c2 := y0 - 2.5*y1 + 2*y2 - 0.5*y3
	c3 := 0.5 * (y3 - y0 + 3*(y1-y2))
	
	return ((c3*frac+c2)*frac+c1)*frac + c0
}

// Sinc performs windowed sinc interpolation (high quality but expensive).
// buffer contains the samples, index is the integer position,
// frac is the fractional position (0.0 to 1.0).
func Sinc(buffer []float32, index int, frac float32, windowSize int) float32 {
	if windowSize < 2 {
		windowSize = 2
	}
	
	halfWindow := windowSize / 2
	result := float32(0.0)
	
	for i := -halfWindow; i <= halfWindow; i++ {
		idx := index + i
		if idx < 0 || idx >= len(buffer) {
			continue
		}
		
		x := float32(i) - frac
		if x == 0 {
			result += buffer[idx]
		} else {
			// Sinc function with Blackman window
			pi_x := math.Pi * float64(x)
			sinc := float32(math.Sin(pi_x) / pi_x)
			
			// Blackman window
			w := 0.42 - 0.5*math.Cos(2*math.Pi*(float64(i+halfWindow))/float64(windowSize)) +
				0.08*math.Cos(4*math.Pi*(float64(i+halfWindow))/float64(windowSize))
			
			result += buffer[idx] * sinc * float32(w)
		}
	}
	
	return result
}

// Lanczos performs Lanczos interpolation.
// a is the filter size parameter (typically 2 or 3).
func Lanczos(buffer []float32, index int, frac float32, a int) float32 {
	if a < 1 {
		a = 2
	}
	
	result := float32(0.0)
	
	for i := -a + 1; i <= a; i++ {
		idx := index + i
		if idx < 0 || idx >= len(buffer) {
			continue
		}
		
		x := float32(i) - frac
		if x == 0 {
			result += buffer[idx]
		} else if x > -float32(a) && x < float32(a) {
			pi_x := math.Pi * float64(x)
			pi_x_a := pi_x / float64(a)
			L := float32(math.Sin(pi_x) / pi_x * math.Sin(pi_x_a) / pi_x_a)
			result += buffer[idx] * L
		}
	}
	
	return result
}

// AllPass implements an all-pass interpolator for fractional delays.
type AllPass struct {
	x1, y1 float32 // Previous input and output
}

// NewAllPass creates a new all-pass interpolator.
func NewAllPass() *AllPass {
	return &AllPass{}
}

// Process interpolates using all-pass filtering.
// Suitable for fractional delay lines.
func (ap *AllPass) Process(input, frac float32) float32 {
	// All-pass coefficient
	a := (1 - frac) / (1 + frac)
	
	// All-pass filter
	output := input + a*ap.x1 - a*ap.y1
	
	// Update state
	ap.x1 = input
	ap.y1 = output
	
	return output
}

// Reset clears the all-pass interpolator state.
func (ap *AllPass) Reset() {
	ap.x1 = 0
	ap.y1 = 0
}

// Resample performs basic resampling of a buffer.
// ratio > 1 upsamples, ratio < 1 downsamples.
func Resample(input []float32, ratio float32, output []float32) int {
	if len(input) == 0 || len(output) == 0 || ratio <= 0 {
		return 0
	}
	
	inputLen := float32(len(input))
	outputLen := len(output)
	
	samplesWritten := 0
	
	for i := 0; i < outputLen; i++ {
		// Calculate source position
		srcPos := float32(i) / ratio
		if srcPos >= inputLen-1 {
			break
		}
		
		// Get integer and fractional parts
		srcIdx := int(srcPos)
		srcFrac := srcPos - float32(srcIdx)
		
		// Linear interpolation (use Cubic for better quality)
		if srcIdx < len(input)-1 {
			output[i] = Linear(input[srcIdx], input[srcIdx+1], srcFrac)
			samplesWritten++
		}
	}
	
	return samplesWritten
}

// ResampleCubic performs cubic interpolation resampling.
func ResampleCubic(input []float32, ratio float32, output []float32) int {
	if len(input) < 4 || len(output) == 0 || ratio <= 0 {
		return 0
	}
	
	inputLen := float32(len(input))
	outputLen := len(output)
	
	samplesWritten := 0
	
	for i := 0; i < outputLen; i++ {
		// Calculate source position
		srcPos := float32(i) / ratio
		if srcPos >= inputLen-2 {
			break
		}
		
		// Get integer and fractional parts
		srcIdx := int(srcPos)
		srcFrac := srcPos - float32(srcIdx)
		
		// Ensure we have enough samples for cubic interpolation
		if srcIdx >= 1 && srcIdx < len(input)-2 {
			output[i] = Cubic(
				input[srcIdx-1],
				input[srcIdx],
				input[srcIdx+1],
				input[srcIdx+2],
				srcFrac,
			)
			samplesWritten++
		} else if srcIdx < len(input)-1 {
			// Fall back to linear at boundaries
			output[i] = Linear(input[srcIdx], input[srcIdx+1], srcFrac)
			samplesWritten++
		}
	}
	
	return samplesWritten
}

// Smooth performs exponential smoothing on a value.
// Useful for parameter smoothing.
func Smooth(current, target, smoothingFactor float32) float32 {
	return current + (target-current)*smoothingFactor
}

// SmoothBuffer applies exponential smoothing to a buffer of values.
func SmoothBuffer(buffer []float32, target, smoothingFactor float32) {
	for i := range buffer {
		buffer[i] = Smooth(buffer[i], target, smoothingFactor)
	}
}