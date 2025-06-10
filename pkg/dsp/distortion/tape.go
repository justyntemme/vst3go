package distortion

import (
	"math"
	"math/rand"
)

type TapeSaturation struct {
	// User parameters
	saturation float64
	compression float64
	flutter     float64
	warmth      float64
	mix         float64
	output      float64
	
	// Internal state
	sampleRate float64
	
	// Pre/de-emphasis filters
	preEmphasisState  [2]float64 // Stereo state
	deEmphasisState   [2]float64
	
	// Flutter LFO
	flutterPhase float64
	flutterRate  float64
	
	// Delay buffer for flutter effect
	delayBuffer     []float64
	delayBufferSize int
	delayWritePos   int
	
	// Compression state
	envelope float64
	
	// Noise generator for tape hiss
	noiseLevel float64
}

func NewTapeSaturation(sampleRate float64) *TapeSaturation {
	bufferSize := int(sampleRate * 0.01) // 10ms max delay for flutter
	
	return &TapeSaturation{
		saturation:      0.5,
		compression:     0.5,
		flutter:         0.0,
		warmth:          0.5,
		mix:             1.0,
		output:          1.0,
		sampleRate:      sampleRate,
		delayBuffer:     make([]float64, bufferSize),
		delayBufferSize: bufferSize,
		flutterRate:     0.3 + rand.Float64()*0.2, // 0.3-0.5 Hz
		noiseLevel:      0.0001,
	}
}

func (t *TapeSaturation) SetSaturation(saturation float64) {
	t.saturation = math.Max(0.0, math.Min(1.0, saturation))
}

func (t *TapeSaturation) SetCompression(compression float64) {
	t.compression = math.Max(0.0, math.Min(1.0, compression))
}

func (t *TapeSaturation) SetFlutter(flutter float64) {
	t.flutter = math.Max(0.0, math.Min(1.0, flutter))
}

func (t *TapeSaturation) SetWarmth(warmth float64) {
	t.warmth = math.Max(0.0, math.Min(1.0, warmth))
}

func (t *TapeSaturation) SetMix(mix float64) {
	t.mix = math.Max(0.0, math.Min(1.0, mix))
}

func (t *TapeSaturation) SetOutput(output float64) {
	t.output = math.Max(0.0, math.Min(2.0, output))
}

func (t *TapeSaturation) Process(input float64) float64 {
	return t.processChannel(input, 0)
}

func (t *TapeSaturation) processChannel(input float64, channel int) float64 {
	// Pre-emphasis (boost highs before saturation)
	emphasized := t.preEmphasis(input, channel)
	
	// Apply tape compression
	compressed := t.tapeCompress(emphasized)
	
	// Apply tape saturation
	saturated := t.tapeSaturate(compressed)
	
	// Apply flutter (pitch modulation)
	fluttered := t.applyFlutter(saturated)
	
	// Add subtle tape noise
	withNoise := fluttered + (rand.Float64()*2.0-1.0)*t.noiseLevel*t.saturation
	
	// De-emphasis (cut highs after saturation)
	deEmphasized := t.deEmphasis(withNoise, channel)
	
	// Mix with dry signal
	mixed := deEmphasized*t.mix + input*(1.0-t.mix)
	
	return mixed * t.output
}

func (t *TapeSaturation) ProcessBlock(input, output []float64) {
	for i := range input {
		output[i] = t.Process(input[i])
	}
}

func (t *TapeSaturation) ProcessStereo(inputL, inputR, outputL, outputR []float64) {
	for i := range inputL {
		outputL[i] = t.processChannel(inputL[i], 0)
		outputR[i] = t.processChannel(inputR[i], 1)
	}
}

func (t *TapeSaturation) tapeSaturate(x float64) float64 {
	// Tape saturation characteristics
	// Soft saturation with 3rd harmonic emphasis
	
	drive := 1.0 + t.saturation*4.0
	driven := x * drive
	
	// Soft clipping with tape-like curve
	saturated := math.Tanh(driven * 0.7)
	
	// Add subtle 3rd harmonic
	third := driven - 0.1*driven*driven*driven
	
	// Mix original and harmonics
	return saturated*0.8 + third*0.2*t.saturation
}

func (t *TapeSaturation) tapeCompress(x float64) float64 {
	// Tape-style compression (program-dependent)
	
	// Update envelope follower
	absX := math.Abs(x)
	attack := 0.01
	release := 0.1
	
	if absX > t.envelope {
		t.envelope += (absX - t.envelope) * attack
	} else {
		t.envelope += (absX - t.envelope) * release
	}
	
	// Calculate compression
	threshold := 0.5
	ratio := 2.0 + t.compression*3.0 // 2:1 to 5:1
	
	if t.envelope > threshold {
		// Calculate gain reduction
		excess := t.envelope - threshold
		compressedExcess := excess / ratio
		gainReduction := (threshold + compressedExcess) / t.envelope
		
		// Apply compression
		return x * gainReduction
	}
	
	return x
}

func (t *TapeSaturation) applyFlutter(x float64) float64 {
	if t.flutter < 0.01 {
		return x
	}
	
	// Write to delay buffer
	t.delayBuffer[t.delayWritePos] = x
	t.delayWritePos = (t.delayWritePos + 1) % t.delayBufferSize
	
	// Calculate flutter modulation
	t.flutterPhase += 2.0 * math.Pi * t.flutterRate / t.sampleRate
	if t.flutterPhase > 2.0*math.Pi {
		t.flutterPhase -= 2.0 * math.Pi
		// Occasionally change flutter rate slightly
		if rand.Float64() < 0.1 {
			t.flutterRate = 0.3 + rand.Float64()*0.2
		}
	}
	
	// Flutter modulation depth in samples
	modDepth := t.flutter * 3.0 // Max 3 samples
	modulation := math.Sin(t.flutterPhase) * modDepth
	
	// Add some randomness for more realistic flutter
	modulation += (rand.Float64()*2.0 - 1.0) * modDepth * 0.3
	
	// Calculate delayed position
	delaySamples := 5.0 + modulation // Base delay + modulation
	
	// Linear interpolation for fractional delay
	delayInt := int(delaySamples)
	delayFrac := delaySamples - float64(delayInt)
	
	// Read from delay buffer with interpolation
	readPos1 := (t.delayWritePos - delayInt + t.delayBufferSize) % t.delayBufferSize
	readPos2 := (readPos1 - 1 + t.delayBufferSize) % t.delayBufferSize
	
	sample1 := t.delayBuffer[readPos1]
	sample2 := t.delayBuffer[readPos2]
	
	return sample1*(1.0-delayFrac) + sample2*delayFrac
}

func (t *TapeSaturation) preEmphasis(x float64, channel int) float64 {
	// CCIR pre-emphasis curve approximation
	// Boost high frequencies before recording
	
	// High-pass filter component
	cutoff := 0.15 + t.warmth*0.1
	highpass := x - t.preEmphasisState[channel]
	t.preEmphasisState[channel] += highpass * cutoff
	
	// Mix based on warmth (more warmth = more pre-emphasis)
	return x + highpass*t.warmth*0.3
}

func (t *TapeSaturation) deEmphasis(x float64, channel int) float64 {
	// CCIR de-emphasis curve approximation
	// Cut high frequencies after playback
	
	// Low-pass filter
	cutoff := 0.8 - t.warmth*0.5
	t.deEmphasisState[channel] += (x - t.deEmphasisState[channel]) * cutoff
	
	return t.deEmphasisState[channel]
}

func (t *TapeSaturation) Reset() {
	t.preEmphasisState[0] = 0.0
	t.preEmphasisState[1] = 0.0
	t.deEmphasisState[0] = 0.0
	t.deEmphasisState[1] = 0.0
	t.envelope = 0.0
	t.flutterPhase = 0.0
	t.delayWritePos = 0
	
	// Clear delay buffer
	for i := range t.delayBuffer {
		t.delayBuffer[i] = 0.0
	}
}