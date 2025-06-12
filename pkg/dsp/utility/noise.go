// Package utility provides common DSP utility functions and processors.
package utility

import (
	"math"
	"math/rand"
)

// NoiseType represents different types of noise.
type NoiseType int

const (
	// WhiteNoise has equal energy at all frequencies
	WhiteNoise NoiseType = iota
	// PinkNoise has equal energy per octave (1/f spectrum)
	PinkNoise
	// BrownNoise has 1/f² spectrum (Brownian noise)
	BrownNoise
	// BlueNoise has increasing energy with frequency
	BlueNoise
	// VioletNoise has f² spectrum
	VioletNoise
)

// NoiseGenerator generates various types of noise.
type NoiseGenerator struct {
	noiseType NoiseType
	
	// Pink noise state (Voss-McCartney algorithm)
	pinkRows      [16]float32
	pinkRunningSum float32
	pinkIndex     int
	pinkScalar    float32
	
	// Brown noise state
	brownState float32
	
	// Blue/Violet noise state  
	blueState float32
	
	// Random source
	rand *rand.Rand
}

// NewNoiseGenerator creates a new noise generator.
func NewNoiseGenerator(noiseType NoiseType) *NoiseGenerator {
	gen := &NoiseGenerator{
		noiseType:  noiseType,
		rand:       rand.New(rand.NewSource(rand.Int63())),
		pinkScalar: 1.0 / 20.0, // Normalization for pink noise
	}
	
	// Initialize pink noise rows
	for i := range gen.pinkRows {
		gen.pinkRows[i] = gen.randomFloat()
	}
	
	return gen
}

// SetType changes the noise type.
func (n *NoiseGenerator) SetType(noiseType NoiseType) {
	n.noiseType = noiseType
}

// SetSeed sets the random seed for reproducible noise.
func (n *NoiseGenerator) SetSeed(seed int64) {
	n.rand = rand.New(rand.NewSource(seed))
}

// Next generates the next noise sample.
func (n *NoiseGenerator) Next() float32 {
	switch n.noiseType {
	case WhiteNoise:
		return n.generateWhite()
	case PinkNoise:
		return n.generatePink()
	case BrownNoise:
		return n.generateBrown()
	case BlueNoise:
		return n.generateBlue()
	case VioletNoise:
		return n.generateViolet()
	default:
		return n.generateWhite()
	}
}

// Generate fills a buffer with noise.
func (n *NoiseGenerator) Generate(buffer []float32) {
	for i := range buffer {
		buffer[i] = n.Next()
	}
}

// GenerateAdd adds noise to an existing buffer.
func (n *NoiseGenerator) GenerateAdd(buffer []float32, gain float32) {
	for i := range buffer {
		buffer[i] += n.Next() * gain
	}
}

// Reset resets the generator state.
func (n *NoiseGenerator) Reset() {
	n.brownState = 0
	n.blueState = 0
	n.pinkIndex = 0
	n.pinkRunningSum = 0
	for i := range n.pinkRows {
		n.pinkRows[i] = n.randomFloat()
	}
}

// randomFloat generates a random float32 in range [-1, 1].
func (n *NoiseGenerator) randomFloat() float32 {
	return float32(n.rand.Float64()*2.0 - 1.0)
}

// generateWhite generates white noise.
func (n *NoiseGenerator) generateWhite() float32 {
	return n.randomFloat()
}

// generatePink generates pink noise using Voss-McCartney algorithm.
func (n *NoiseGenerator) generatePink() float32 {
	// Determine how many rows to update
	n.pinkIndex++
	if n.pinkIndex > 15 {
		n.pinkIndex = 0
	}
	
	// Update rows based on binary representation of index
	if n.pinkIndex != 0 {
		numZeros := 0
		temp := n.pinkIndex
		for (temp & 1) == 0 {
			temp >>= 1
			numZeros++
		}
		
		// Update the row
		n.pinkRunningSum -= n.pinkRows[numZeros]
		n.pinkRows[numZeros] = n.randomFloat()
		n.pinkRunningSum += n.pinkRows[numZeros]
	}
	
	// Add white noise and scale
	output := (n.pinkRunningSum + n.randomFloat()) * n.pinkScalar
	
	// Clamp to [-1, 1]
	if output > 1.0 {
		output = 1.0
	} else if output < -1.0 {
		output = -1.0
	}
	
	return output
}

// generateBrown generates brown noise (integrated white noise).
func (n *NoiseGenerator) generateBrown() float32 {
	// Brown noise is the integral of white noise
	white := n.randomFloat()
	n.brownState += white * 0.0625 // Scale factor to prevent clipping
	
	// Leaky integrator to prevent DC buildup
	n.brownState *= 0.997
	
	// Clamp to [-1, 1]
	if n.brownState > 1.0 {
		n.brownState = 1.0
	} else if n.brownState < -1.0 {
		n.brownState = -1.0
	}
	
	return n.brownState
}

// generateBlue generates blue noise (differentiated white noise).
func (n *NoiseGenerator) generateBlue() float32 {
	// Blue noise is the derivative of white noise
	white := n.randomFloat()
	output := white - n.blueState
	n.blueState = white
	
	// Scale to maintain amplitude
	return output * 0.5
}

// generateViolet generates violet noise (differentiated blue noise).
func (n *NoiseGenerator) generateViolet() float32 {
	// Violet noise has even more high frequency content
	blue := n.generateBlue()
	output := blue - n.blueState
	n.blueState = blue
	
	// Scale to maintain amplitude
	return output * 0.25
}

// SimpleNoise provides quick noise generation functions without state.

// WhiteNoiseSample generates a single white noise sample.
func WhiteNoiseSample() float32 {
	return float32(rand.Float64()*2.0 - 1.0)
}

// WhiteNoiseBuffer fills a buffer with white noise.
func WhiteNoiseBuffer(buffer []float32) {
	for i := range buffer {
		buffer[i] = WhiteNoiseSample()
	}
}

// GaussianNoise generates Gaussian-distributed white noise.
type GaussianNoise struct {
	rand      *rand.Rand
	hasSpare  bool
	spare     float32
}

// NewGaussianNoise creates a Gaussian noise generator.
func NewGaussianNoise() *GaussianNoise {
	return &GaussianNoise{
		rand: rand.New(rand.NewSource(rand.Int63())),
	}
}

// Next generates the next Gaussian noise sample using Box-Muller transform.
func (g *GaussianNoise) Next() float32 {
	if g.hasSpare {
		g.hasSpare = false
		return g.spare * 0.3 // Scale to roughly [-1, 1]
	}
	
	g.hasSpare = true
	
	// Box-Muller transform
	u1 := g.rand.Float64()
	u2 := g.rand.Float64()
	
	mag := float32(math.Sqrt(-2.0 * math.Log(u1)))
	z0 := mag * float32(math.Cos(2.0*math.Pi*u2))
	z1 := mag * float32(math.Sin(2.0*math.Pi*u2))
	
	g.spare = z1
	return z0 * 0.3 // Scale to roughly [-1, 1]
}

// Generate fills a buffer with Gaussian noise.
func (g *GaussianNoise) Generate(buffer []float32) {
	for i := range buffer {
		buffer[i] = g.Next()
	}
}