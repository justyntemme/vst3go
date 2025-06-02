package distortion

import (
	"math"
)

// BitCrusher reduces bit depth and sample rate for lo-fi digital distortion
type BitCrusher struct {
	sampleRate      float64
	bitDepth        int
	sampleRateRatio float64
	mix             float64
	
	// Anti-aliasing filter
	antiAlias       bool
	preFilter       *SimpleLowpass
	postFilter      *SimpleLowpass
	
	// Sample rate reduction state
	sampleCounter   float64
	heldSample      float64
	
	// Dithering
	ditherAmount    float64
	noiseState      uint32
	
	// DC offset removal
	dcBlocker       *DCBlocker
}

// NewBitCrusher creates a new bit crusher effect
func NewBitCrusher(sampleRate float64) *BitCrusher {
	return &BitCrusher{
		sampleRate:      sampleRate,
		bitDepth:        16,
		sampleRateRatio: 1.0,
		mix:             1.0,
		antiAlias:       true,
		ditherAmount:    0.0,
		noiseState:      12345,
		preFilter:       NewSimpleLowpass(sampleRate, sampleRate/2),
		postFilter:      NewSimpleLowpass(sampleRate, sampleRate/2),
		dcBlocker:       NewDCBlocker(),
	}
}

// SetBitDepth sets the target bit depth (1-24 bits)
func (b *BitCrusher) SetBitDepth(bits int) {
	b.bitDepth = max(1, min(24, bits))
}

// SetSampleRateRatio sets the sample rate reduction ratio (0.01 to 1.0)
// 1.0 = no reduction, 0.5 = half sample rate, etc.
func (b *BitCrusher) SetSampleRateRatio(ratio float64) {
	b.sampleRateRatio = math.Max(0.01, math.Min(1.0, ratio))
	
	// Update anti-aliasing filter cutoff
	if b.antiAlias {
		cutoff := b.sampleRate * b.sampleRateRatio * 0.45 // Slightly below Nyquist
		b.preFilter = NewSimpleLowpass(b.sampleRate, cutoff)
		b.postFilter = NewSimpleLowpass(b.sampleRate, cutoff)
	}
}

// SetMix sets the dry/wet mix (0.0 = dry, 1.0 = wet)
func (b *BitCrusher) SetMix(mix float64) {
	b.mix = math.Max(0.0, math.Min(1.0, mix))
}

// SetAntiAlias enables/disables anti-aliasing filters
func (b *BitCrusher) SetAntiAlias(enable bool) {
	b.antiAlias = enable
}

// SetDither sets the dithering amount (0.0 to 1.0)
func (b *BitCrusher) SetDither(amount float64) {
	b.ditherAmount = math.Max(0.0, math.Min(1.0, amount))
}

// Process applies bit crushing to a single sample
func (b *BitCrusher) Process(input float64) float64 {
	// Pre-filter to prevent aliasing
	filtered := input
	if b.antiAlias && b.sampleRateRatio < 1.0 {
		filtered = b.preFilter.Process(input)
	}
	
	// Sample rate reduction
	decimated := b.decimate(filtered)
	
	// Bit depth reduction
	crushed := b.quantize(decimated)
	
	// Post-filter to smooth
	if b.antiAlias && b.sampleRateRatio < 1.0 {
		crushed = b.postFilter.Process(crushed)
	}
	
	// Remove any DC offset introduced
	crushed = b.dcBlocker.Process(crushed)
	
	// Mix with dry signal
	return input*(1.0-b.mix) + crushed*b.mix
}

// decimate performs sample rate reduction
func (b *BitCrusher) decimate(input float64) float64 {
	// Increment sample counter
	b.sampleCounter += b.sampleRateRatio
	
	// Check if we should update the held sample
	if b.sampleCounter >= 1.0 {
		b.sampleCounter -= 1.0
		b.heldSample = input
	}
	
	return b.heldSample
}

// quantize reduces bit depth with optional dithering
func (b *BitCrusher) quantize(input float64) float64 {
	// Calculate quantization levels
	levels := math.Pow(2, float64(b.bitDepth))
	halfLevels := levels / 2.0
	
	// Add dither noise if enabled
	dithered := input
	if b.ditherAmount > 0 {
		noise := b.generateDither() * b.ditherAmount / halfLevels
		dithered = input + noise
	}
	
	// Scale to quantization range
	scaled := dithered * halfLevels
	
	// Quantize
	quantized := math.Round(scaled)
	
	// Clip to bit depth range
	quantized = math.Max(-halfLevels, math.Min(halfLevels-1, quantized))
	
	// Scale back to -1 to 1 range
	return quantized / halfLevels
}

// generateDither creates triangular probability distribution dither
func (b *BitCrusher) generateDither() float64 {
	// Simple linear congruential generator
	b.noiseState = (b.noiseState*1664525 + 1013904223) & 0xffffffff
	noise1 := float64(b.noiseState) / float64(0xffffffff)
	
	b.noiseState = (b.noiseState*1664525 + 1013904223) & 0xffffffff
	noise2 := float64(b.noiseState) / float64(0xffffffff)
	
	// Triangular distribution (sum of two uniform)
	return (noise1 + noise2 - 1.0)
}

// ProcessBuffer applies bit crushing to a buffer of samples
func (b *BitCrusher) ProcessBuffer(input, output []float64) {
	n := len(input)
	if len(output) < n {
		n = len(output)
	}

	for i := 0; i < n; i++ {
		output[i] = b.Process(input[i])
	}
}

// DCBlocker removes DC offset from the signal
type DCBlocker struct {
	x1, y1 float64
	r      float64
}

// NewDCBlocker creates a new DC blocking filter
func NewDCBlocker() *DCBlocker {
	return &DCBlocker{
		r: 0.995, // Filter coefficient
	}
}

// Process removes DC offset from a sample
func (dc *DCBlocker) Process(input float64) float64 {
	output := input - dc.x1 + dc.r*dc.y1
	dc.x1 = input
	dc.y1 = output
	return output
}

// BitCrusherWithModulation adds modulation capabilities to bit crushing
type BitCrusherWithModulation struct {
	*BitCrusher
	
	// Modulation sources
	bitDepthMod     float64
	sampleRateMod   float64
	
	// Base values
	baseBitDepth    float64
	baseSampleRate  float64
}

// NewBitCrusherWithModulation creates a modulatable bit crusher
func NewBitCrusherWithModulation(sampleRate float64) *BitCrusherWithModulation {
	return &BitCrusherWithModulation{
		BitCrusher:     NewBitCrusher(sampleRate),
		baseBitDepth:   16.0,
		baseSampleRate: 1.0,
	}
}

// SetBaseBitDepth sets the base bit depth before modulation
func (bcm *BitCrusherWithModulation) SetBaseBitDepth(bits float64) {
	bcm.baseBitDepth = math.Max(1.0, math.Min(24.0, bits))
}

// SetBaseSampleRateRatio sets the base sample rate ratio before modulation
func (bcm *BitCrusherWithModulation) SetBaseSampleRateRatio(ratio float64) {
	bcm.baseSampleRate = math.Max(0.01, math.Min(1.0, ratio))
}

// ModulateBitDepth applies modulation to bit depth (-1 to 1)
func (bcm *BitCrusherWithModulation) ModulateBitDepth(modulation float64) {
	bcm.bitDepthMod = math.Max(-1.0, math.Min(1.0, modulation))
	
	// Apply modulation
	modAmount := bcm.bitDepthMod * 12.0 // ±12 bits modulation range
	finalBits := bcm.baseBitDepth + modAmount
	bcm.SetBitDepth(int(finalBits))
}

// ModulateSampleRate applies modulation to sample rate (-1 to 1)
func (bcm *BitCrusherWithModulation) ModulateSampleRate(modulation float64) {
	bcm.sampleRateMod = math.Max(-1.0, math.Min(1.0, modulation))
	
	// Apply modulation (exponential for musical response)
	modAmount := math.Pow(2.0, bcm.sampleRateMod*2.0) // 0.25x to 4x range
	finalRatio := bcm.baseSampleRate * modAmount
	bcm.SetSampleRateRatio(finalRatio)
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}