// Package filter provides digital signal processing filters
package filter

import "math"

// Biquad implements a second-order IIR filter (biquad)
// Direct Form I implementation with pre-allocated state
type Biquad struct {
	// Coefficients
	a0, a1, a2 float32 // denominator (a0 is always normalized to 1.0)
	b0, b1, b2 float32 // numerator

	// State variables (per-channel)
	x1, x2 []float32 // input delay line
	y1, y2 []float32 // output delay line
}

// NewBiquad creates a new biquad filter for the specified number of channels
func NewBiquad(channels int) *Biquad {
	return &Biquad{
		a0: 1.0,
		x1: make([]float32, channels),
		x2: make([]float32, channels),
		y1: make([]float32, channels),
		y2: make([]float32, channels),
	}
}

// Reset clears the filter state
func (b *Biquad) Reset() {
	for i := range b.x1 {
		b.x1[i] = 0
		b.x2[i] = 0
		b.y1[i] = 0
		b.y2[i] = 0
	}
}

// SetCoefficients sets the filter coefficients directly
func (b *Biquad) SetCoefficients(b0, b1, b2, a0, a1, a2 float32) {
	// Normalize by a0
	invA0 := 1.0 / a0
	b.b0 = b0 * invA0
	b.b1 = b1 * invA0
	b.b2 = b2 * invA0
	b.a0 = 1.0
	b.a1 = a1 * invA0
	b.a2 = a2 * invA0
}

// Process applies the filter to a buffer (single channel) - no allocations
func (b *Biquad) Process(buffer []float32, channel int) {
	// Get state for this channel
	x1 := b.x1[channel]
	x2 := b.x2[channel]
	y1 := b.y1[channel]
	y2 := b.y2[channel]

	// Process samples
	for i := range buffer {
		x0 := buffer[i]

		// Direct Form I
		y0 := b.b0*x0 + b.b1*x1 + b.b2*x2 - b.a1*y1 - b.a2*y2

		// Update state
		x2 = x1
		x1 = x0
		y2 = y1
		y1 = y0

		buffer[i] = y0
	}

	// Save state
	b.x1[channel] = x1
	b.x2[channel] = x2
	b.y1[channel] = y1
	b.y2[channel] = y2
}

// ProcessMulti applies the filter to multiple channels - no allocations
func (b *Biquad) ProcessMulti(buffers [][]float32) {
	for ch, buffer := range buffers {
		if ch < len(b.x1) {
			b.Process(buffer, ch)
		}
	}
}

// Design functions for common filter types

// SetLowpass configures as a lowpass filter
func (b *Biquad) SetLowpass(sampleRate, frequency, q float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	alpha := sinOmega / (2.0 * q)

	b0 := (1.0 - cosOmega) / 2.0
	b1 := 1.0 - cosOmega
	b2 := (1.0 - cosOmega) / 2.0
	a0 := 1.0 + alpha
	a1 := -2.0 * cosOmega
	a2 := 1.0 - alpha

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}

// SetHighpass configures as a highpass filter
func (b *Biquad) SetHighpass(sampleRate, frequency, q float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	alpha := sinOmega / (2.0 * q)

	b0 := (1.0 + cosOmega) / 2.0
	b1 := -(1.0 + cosOmega)
	b2 := (1.0 + cosOmega) / 2.0
	a0 := 1.0 + alpha
	a1 := -2.0 * cosOmega
	a2 := 1.0 - alpha

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}

// SetBandpass configures as a bandpass filter (constant skirt gain)
func (b *Biquad) SetBandpass(sampleRate, frequency, q float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	alpha := sinOmega / (2.0 * q)

	b0 := alpha
	b1 := 0.0
	b2 := -alpha
	a0 := 1.0 + alpha
	a1 := -2.0 * cosOmega
	a2 := 1.0 - alpha

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}

// SetNotch configures as a notch (band-reject) filter
func (b *Biquad) SetNotch(sampleRate, frequency, q float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	alpha := sinOmega / (2.0 * q)

	b0 := 1.0
	b1 := -2.0 * cosOmega
	b2 := 1.0
	a0 := 1.0 + alpha
	a1 := -2.0 * cosOmega
	a2 := 1.0 - alpha

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}

// SetAllpass configures as an allpass filter
func (b *Biquad) SetAllpass(sampleRate, frequency, q float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	alpha := sinOmega / (2.0 * q)

	b0 := 1.0 - alpha
	b1 := -2.0 * cosOmega
	b2 := 1.0 + alpha
	a0 := 1.0 + alpha
	a1 := -2.0 * cosOmega
	a2 := 1.0 - alpha

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}

// SetPeakingEQ configures as a peaking EQ filter
func (b *Biquad) SetPeakingEQ(sampleRate, frequency, q, gainDB float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	A := math.Pow(10.0, gainDB/40.0)
	alpha := sinOmega / (2.0 * q)

	b0 := 1.0 + alpha*A
	b1 := -2.0 * cosOmega
	b2 := 1.0 - alpha*A
	a0 := 1.0 + alpha/A
	a1 := -2.0 * cosOmega
	a2 := 1.0 - alpha/A

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}

// SetLowShelf configures as a low shelf filter
func (b *Biquad) SetLowShelf(sampleRate, frequency, q, gainDB float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	A := math.Pow(10.0, gainDB/40.0)
	alpha := sinOmega / (2.0 * q)

	sqrtA := math.Sqrt(A)
	sqrtAAlpha := 2.0 * sqrtA * alpha

	b0 := A * ((A + 1) - (A-1)*cosOmega + sqrtAAlpha)
	b1 := 2.0 * A * ((A - 1) - (A+1)*cosOmega)
	b2 := A * ((A + 1) - (A-1)*cosOmega - sqrtAAlpha)
	a0 := (A + 1) + (A-1)*cosOmega + sqrtAAlpha
	a1 := -2.0 * ((A - 1) + (A+1)*cosOmega)
	a2 := (A + 1) + (A-1)*cosOmega - sqrtAAlpha

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}

// SetHighShelf configures as a high shelf filter
func (b *Biquad) SetHighShelf(sampleRate, frequency, q, gainDB float64) {
	omega := 2.0 * math.Pi * frequency / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	A := math.Pow(10.0, gainDB/40.0)
	alpha := sinOmega / (2.0 * q)

	sqrtA := math.Sqrt(A)
	sqrtAAlpha := 2.0 * sqrtA * alpha

	b0 := A * ((A + 1) + (A-1)*cosOmega + sqrtAAlpha)
	b1 := -2.0 * A * ((A - 1) + (A+1)*cosOmega)
	b2 := A * ((A + 1) + (A-1)*cosOmega - sqrtAAlpha)
	a0 := (A + 1) - (A-1)*cosOmega + sqrtAAlpha
	a1 := 2.0 * ((A - 1) - (A+1)*cosOmega)
	a2 := (A + 1) - (A-1)*cosOmega - sqrtAAlpha

	b.SetCoefficients(float32(b0), float32(b1), float32(b2),
		float32(a0), float32(a1), float32(a2))
}
