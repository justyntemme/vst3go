package distortion

import (
	"math"
)

type DitherType int

const (
	DitherNone DitherType = iota
	DitherWhite
	DitherTriangular
)

type Bitcrusher struct {
	// User parameters
	bitDepth         float64
	sampleRateReduce float64
	antiAlias        bool
	dither           DitherType
	mix              float64
	output           float64

	// Internal state
	originalSampleRate float64

	// Sample rate reduction state
	sampleHoldCounter float64
	lastSample        float64

	// Anti-aliasing filter state (simple one-pole)
	filterState [2]float64 // Stereo

	// Dithering state
	ditherState float64
}

func NewBitcrusher(sampleRate float64) *Bitcrusher {
	return &Bitcrusher{
		bitDepth:           16.0,
		sampleRateReduce:   1.0,
		antiAlias:          true,
		dither:             DitherNone,
		mix:                1.0,
		output:             1.0,
		originalSampleRate: sampleRate,
	}
}

func (b *Bitcrusher) SetBitDepth(bits float64) {
	b.bitDepth = math.Max(1.0, math.Min(32.0, bits))
}

func (b *Bitcrusher) SetSampleRateReduction(factor float64) {
	b.sampleRateReduce = math.Max(1.0, math.Min(100.0, factor))
}

func (b *Bitcrusher) SetAntiAlias(enable bool) {
	b.antiAlias = enable
}

func (b *Bitcrusher) SetDither(dither DitherType) {
	b.dither = dither
}

func (b *Bitcrusher) SetMix(mix float64) {
	b.mix = math.Max(0.0, math.Min(1.0, mix))
}

func (b *Bitcrusher) SetOutput(output float64) {
	b.output = math.Max(0.0, math.Min(2.0, output))
}

func (b *Bitcrusher) Process(input float64) float64 {
	return b.processChannel(input, 0)
}

func (b *Bitcrusher) processChannel(input float64, channel int) float64 {
	processed := input

	// Apply anti-aliasing filter before sample rate reduction if enabled
	if b.antiAlias && b.sampleRateReduce > 1.0 {
		processed = b.applyAntiAliasFilter(processed, channel)
	}

	// Sample rate reduction
	processed = b.applySampleRateReduction(processed)

	// Bit depth reduction
	processed = b.applyBitReduction(processed)

	// Mix with dry signal
	mixed := processed*b.mix + input*(1.0-b.mix)

	return mixed * b.output
}

func (b *Bitcrusher) ProcessBlock(input, output []float64) {
	for i := range input {
		output[i] = b.Process(input[i])
	}
}

func (b *Bitcrusher) ProcessStereo(inputL, inputR, outputL, outputR []float64) {
	for i := range inputL {
		outputL[i] = b.processChannel(inputL[i], 0)
		outputR[i] = b.processChannel(inputR[i], 1)
	}
}

func (b *Bitcrusher) applyBitReduction(x float64) float64 {
	if b.bitDepth >= 32.0 {
		return x
	}

	// Calculate quantization levels
	levels := math.Pow(2.0, b.bitDepth)

	// Add dither before quantization
	dithered := x + b.generateDither()

	// Scale to quantization range
	scaled := dithered * 0.5 * levels

	// Quantize
	quantized := math.Round(scaled) / levels * 2.0

	// Ensure output stays in bounds
	return math.Max(-1.0, math.Min(1.0, quantized))
}

func (b *Bitcrusher) applySampleRateReduction(x float64) float64 {
	if b.sampleRateReduce <= 1.0 {
		return x
	}

	// Check if we should update the held sample
	if b.sampleHoldCounter == 0.0 {
		b.lastSample = x
	}

	// Increment counter
	b.sampleHoldCounter += 1.0

	// Reset counter when we reach the reduction factor
	if b.sampleHoldCounter >= b.sampleRateReduce {
		b.sampleHoldCounter = 0.0
	}

	return b.lastSample
}

func (b *Bitcrusher) applyAntiAliasFilter(x float64, channel int) float64 {
	// Simple one-pole low-pass filter
	// Cutoff frequency based on new sample rate
	newSampleRate := b.originalSampleRate / b.sampleRateReduce
	cutoffFreq := newSampleRate * 0.45 // Just below Nyquist

	// Calculate filter coefficient
	omega := 2.0 * math.Pi * cutoffFreq / b.originalSampleRate
	alpha := math.Sin(omega) / (math.Sin(omega) + math.Cos(omega))

	// Apply filter
	b.filterState[channel] = b.filterState[channel] + alpha*(x-b.filterState[channel])
	return b.filterState[channel]
}

func (b *Bitcrusher) generateDither() float64 {
	switch b.dither {
	case DitherWhite:
		// White noise dither
		return (math.Float64frombits(randomBits()) - 0.5) * 2.0 / math.Pow(2.0, b.bitDepth)

	case DitherTriangular:
		// Triangular dither (sum of two uniform random values)
		r1 := math.Float64frombits(randomBits()) - 0.5
		r2 := math.Float64frombits(randomBits()) - 0.5
		return (r1 + r2) / math.Pow(2.0, b.bitDepth)

	default:
		return 0.0
	}
}

func (b *Bitcrusher) Reset() {
	b.sampleHoldCounter = 0.0
	b.lastSample = 0.0
	b.filterState[0] = 0.0
	b.filterState[1] = 0.0
	b.ditherState = 0.0
}

// Simple random number generator for dithering
var randomState uint64 = 1

func randomBits() uint64 {
	// Simple linear congruential generator
	randomState = randomState*6364136223846793005 + 1442695040888963407
	return randomState
}

// Utility functions for specific bit-crushing effects

func QuantizeToSteps(x float64, steps int) float64 {
	if steps <= 1 {
		return 0.0
	}

	// Map to steps
	scaled := (x + 1.0) * 0.5 * float64(steps-1)
	quantized := math.Round(scaled)

	// Map back to [-1, 1]
	return quantized/float64(steps-1)*2.0 - 1.0
}

func FoldbackDistortion(x float64, threshold float64) float64 {
	// Similar to waveshaper foldback but specific to bit-crushing context
	for x > threshold || x < -threshold {
		if x > threshold {
			x = 2*threshold - x
		} else if x < -threshold {
			x = -2*threshold - x
		}
	}
	return x
}

func GateEffect(x, threshold float64) float64 {
	// Hard gate effect often used with bit crushing
	if math.Abs(x) < threshold {
		return 0.0
	}
	return x
}
