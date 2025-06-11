// Package pan provides stereo panning operations.
package pan

import (
	"math"
)

// Law represents different panning laws
type Law int

const (
	// Linear uses linear panning (constant power not maintained)
	Linear Law = iota
	// ConstantPower uses sine/cosine panning (maintains constant power)
	ConstantPower
	// Balanced uses -4.5dB center compensation
	Balanced
)

// MonoToStereo pans a mono signal to stereo.
// pan: -1.0 = hard left, 0.0 = center, 1.0 = hard right
// Returns left and right gains.
func MonoToStereo(pan float32, law Law) (left, right float32) {
	switch law {
	case Linear:
		return linearPan(pan)
	case ConstantPower:
		return constantPowerPan(pan)
	case Balanced:
		return balancedPan(pan)
	default:
		return constantPowerPan(pan)
	}
}

// Process applies panning to a mono buffer, creating stereo output.
func Process(mono []float32, pan float32, law Law, leftOut, rightOut []float32) {
	leftGain, rightGain := MonoToStereo(pan, law)
	
	length := len(mono)
	if len(leftOut) < length {
		length = len(leftOut)
	}
	if len(rightOut) < length {
		length = len(rightOut)
	}
	
	for i := 0; i < length; i++ {
		sample := mono[i]
		leftOut[i] = sample * leftGain
		rightOut[i] = sample * rightGain
	}
}

// ProcessStereo adjusts the panning of an existing stereo signal.
func ProcessStereo(leftIn, rightIn []float32, pan float32, law Law, leftOut, rightOut []float32) {
	leftGain, rightGain := MonoToStereo(pan, law)
	
	length := len(leftIn)
	if len(rightIn) < length {
		length = len(rightIn)
	}
	if len(leftOut) < length {
		length = len(leftOut)
	}
	if len(rightOut) < length {
		length = len(rightOut)
	}
	
	// For stereo panning, we need to cross-mix
	for i := 0; i < length; i++ {
		l := leftIn[i]
		r := rightIn[i]
		
		if pan < 0 {
			// Panning left: reduce right channel
			leftOut[i] = l
			rightOut[i] = r * rightGain
		} else if pan > 0 {
			// Panning right: reduce left channel
			leftOut[i] = l * leftGain
			rightOut[i] = r
		} else {
			// Center: pass through
			leftOut[i] = l
			rightOut[i] = r
		}
	}
}

// linearPan implements simple linear panning.
func linearPan(pan float32) (left, right float32) {
	left = 1.0 - pan
	right = 1.0 + pan
	
	// Normalize to 0-1 range
	left *= 0.5
	right *= 0.5
	
	return
}

// constantPowerPan implements equal power panning using sine/cosine.
func constantPowerPan(pan float32) (left, right float32) {
	// Convert pan from [-1, 1] to [0, π/2]
	angle := (pan + 1.0) * math.Pi / 4.0
	left = float32(math.Cos(float64(angle)))
	right = float32(math.Sin(float64(angle)))
	return
}

// balancedPan implements panning with -4.5dB center compensation.
func balancedPan(pan float32) (left, right float32) {
	// Start with constant power
	left, right = constantPowerPan(pan)
	
	// Apply center compensation
	// At center, both channels are at 0.707 (-3dB)
	// We want -4.5dB (0.595) instead
	centerGain := float32(0.595 / 0.707)
	
	// Interpolate compensation based on pan position
	compensation := 1.0 - (pan*pan)*0.159 // Approximation for smooth transition
	
	left *= compensation
	right *= compensation
	
	return
}

// Width adjusts the stereo width of a signal.
// width: 0.0 = mono, 1.0 = normal stereo, 2.0 = extra wide
func Width(leftIn, rightIn []float32, width float32, leftOut, rightOut []float32) {
	length := len(leftIn)
	if len(rightIn) < length {
		length = len(rightIn)
	}
	if len(leftOut) < length {
		length = len(leftOut)
	}
	if len(rightOut) < length {
		length = len(rightOut)
	}
	
	// Calculate mid/side
	for i := 0; i < length; i++ {
		mid := (leftIn[i] + rightIn[i]) * 0.5
		side := (leftIn[i] - rightIn[i]) * 0.5
		
		// Apply width
		side *= width
		
		// Convert back to L/R
		leftOut[i] = mid + side
		rightOut[i] = mid - side
	}
}

// Balance adjusts the balance between left and right channels.
// balance: -1.0 = left only, 0.0 = centered, 1.0 = right only
func Balance(leftIn, rightIn []float32, balance float32, leftOut, rightOut []float32) {
	// Calculate channel gains
	leftGain := float32(1.0)
	rightGain := float32(1.0)
	
	if balance < 0 {
		// Attenuate right channel
		rightGain = 1.0 + balance
	} else if balance > 0 {
		// Attenuate left channel
		leftGain = 1.0 - balance
	}
	
	length := len(leftIn)
	if len(rightIn) < length {
		length = len(rightIn)
	}
	if len(leftOut) < length {
		length = len(leftOut)
	}
	if len(rightOut) < length {
		length = len(rightOut)
	}
	
	for i := 0; i < length; i++ {
		leftOut[i] = leftIn[i] * leftGain
		rightOut[i] = rightIn[i] * rightGain
	}
}

// AutoPan represents an automatic panner
type AutoPan struct {
	phase    float32
	rate     float32 // Hz
	depth    float32 // 0-1
	law      Law
}

// NewAutoPan creates a new automatic panner
func NewAutoPan(rate, depth float32, law Law) *AutoPan {
	return &AutoPan{
		rate:  rate,
		depth: depth,
		law:   law,
	}
}

// Process applies automatic panning to a mono signal
func (ap *AutoPan) Process(mono []float32, sampleRate float32, leftOut, rightOut []float32) {
	phaseInc := 2.0 * math.Pi * ap.rate / sampleRate
	
	length := len(mono)
	if len(leftOut) < length {
		length = len(leftOut)
	}
	if len(rightOut) < length {
		length = len(rightOut)
	}
	
	for i := 0; i < length; i++ {
		// Calculate pan position from LFO
		pan := float32(math.Sin(float64(ap.phase))) * ap.depth
		
		// Apply panning
		leftGain, rightGain := MonoToStereo(pan, ap.law)
		sample := mono[i]
		leftOut[i] = sample * leftGain
		rightOut[i] = sample * rightGain
		
		// Update phase
		ap.phase += float32(phaseInc)
		if ap.phase > 2*math.Pi {
			ap.phase -= 2 * math.Pi
		}
	}
}

// SetRate updates the auto-pan rate
func (ap *AutoPan) SetRate(rate float32) {
	ap.rate = rate
}

// SetDepth updates the auto-pan depth
func (ap *AutoPan) SetDepth(depth float32) {
	ap.depth = depth
}

// Reset resets the auto-pan phase
func (ap *AutoPan) Reset() {
	ap.phase = 0
}