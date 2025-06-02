package distortion

import (
	"math"
)

// TapeSaturator emulates analog tape saturation characteristics
type TapeSaturator struct {
	sampleRate    float64
	drive         float64
	saturation    float64
	bias          float64
	mix           float64
	
	// Tape characteristics
	compression   float64
	warmth        float64
	flutter       float64
	
	// Hysteresis modeling
	hysteresis    float64
	prevIn        float64
	prevOut       float64
	
	// Pre/de-emphasis filters
	preEmphasis   *PreEmphasisFilter
	deEmphasis    *DeEmphasisFilter
	
	// Flutter LFO
	flutterPhase  float64
	flutterRate   float64
	
	// Delay line for wow/flutter
	delayBuffer   []float64
	delayIndex    int
	maxDelayTime  float64
}

// NewTapeSaturator creates a new tape saturation processor
func NewTapeSaturator(sampleRate float64) *TapeSaturator {
	maxDelayMs := 5.0 // Maximum delay for wow/flutter effect
	bufferSize := int(sampleRate * maxDelayMs / 1000.0)
	
	return &TapeSaturator{
		sampleRate:    sampleRate,
		drive:         1.0,
		saturation:    0.5,
		bias:          0.15,
		mix:           1.0,
		compression:   0.3,
		warmth:        0.5,
		flutter:       0.0,
		hysteresis:    0.3,
		flutterRate:   0.5,
		maxDelayTime:  maxDelayMs,
		delayBuffer:   make([]float64, bufferSize),
		preEmphasis:   NewPreEmphasisFilter(sampleRate),
		deEmphasis:    NewDeEmphasisFilter(sampleRate),
	}
}

// SetDrive sets the input drive level (1.0 to 10.0)
func (t *TapeSaturator) SetDrive(drive float64) {
	t.drive = math.Max(1.0, math.Min(10.0, drive))
}

// SetSaturation sets the tape saturation amount (0.0 to 1.0)
func (t *TapeSaturator) SetSaturation(saturation float64) {
	t.saturation = math.Max(0.0, math.Min(1.0, saturation))
}

// SetBias sets the tape bias amount (0.0 to 1.0)
func (t *TapeSaturator) SetBias(bias float64) {
	t.bias = math.Max(0.0, math.Min(1.0, bias))
}

// SetCompression sets the tape compression amount (0.0 to 1.0)
func (t *TapeSaturator) SetCompression(compression float64) {
	t.compression = math.Max(0.0, math.Min(1.0, compression))
}

// SetWarmth sets the warmth/coloration amount (0.0 to 1.0)
func (t *TapeSaturator) SetWarmth(warmth float64) {
	t.warmth = math.Max(0.0, math.Min(1.0, warmth))
}

// SetFlutter sets the wow/flutter amount (0.0 to 1.0)
func (t *TapeSaturator) SetFlutter(flutter float64) {
	t.flutter = math.Max(0.0, math.Min(1.0, flutter))
}

// SetMix sets the dry/wet mix (0.0 = dry, 1.0 = wet)
func (t *TapeSaturator) SetMix(mix float64) {
	t.mix = math.Max(0.0, math.Min(1.0, mix))
}

// Process applies tape saturation to a single sample
func (t *TapeSaturator) Process(input float64) float64 {
	// Apply pre-emphasis (boost high frequencies before saturation)
	emphasized := t.preEmphasis.Process(input)
	
	// Apply input drive
	driven := emphasized * t.drive
	
	// Apply tape compression (soft knee)
	compressed := t.applyCompression(driven)
	
	// Apply hysteresis
	hystOut := t.applyHysteresis(compressed)
	
	// Apply tape saturation curve
	saturated := t.tapeSaturation(hystOut)
	
	// Apply bias-induced asymmetry
	biased := t.applyBias(saturated)
	
	// Apply de-emphasis (restore frequency balance)
	deemphasized := t.deEmphasis.Process(biased)
	
	// Apply wow/flutter if enabled
	output := deemphasized
	if t.flutter > 0 {
		output = t.applyFlutter(deemphasized)
	}
	
	// Apply warmth coloration
	output = t.applyWarmth(output)
	
	// Mix with dry signal
	return input*(1.0-t.mix) + output*t.mix
}

// applyCompression implements tape-style compression
func (t *TapeSaturator) applyCompression(x float64) float64 {
	threshold := 0.5
	ratio := 1.0 + t.compression*3.0 // 1:1 to 4:1
	
	absX := math.Abs(x)
	if absX <= threshold {
		return x
	}
	
	// Soft knee compression
	overThreshold := absX - threshold
	compressedOver := overThreshold / ratio
	compressed := threshold + compressedOver
	
	if x < 0 {
		return -compressed
	}
	return compressed
}

// applyHysteresis models magnetic hysteresis
func (t *TapeSaturator) applyHysteresis(x float64) float64 {
	// Simple hysteresis model
	diff := x - t.prevIn
	
	// Apply hysteresis curve
	hystCurve := math.Tanh(diff * 2.0) * t.hysteresis
	output := t.prevOut + diff*(1.0-t.hysteresis) + hystCurve
	
	// Update state
	t.prevIn = x
	t.prevOut = output
	
	return output
}

// tapeSaturation applies the characteristic tape saturation curve
func (t *TapeSaturator) tapeSaturation(x float64) float64 {
	// Tape saturation is softer than tube saturation
	// Uses a combination of tanh and polynomial shaping
	
	// Scale by saturation amount
	x *= 1.0 + t.saturation*2.0
	
	// Apply soft saturation curve
	if math.Abs(x) < 0.5 {
		// Linear region with slight compression
		return x * (1.0 - 0.1*t.saturation)
	}
	
	// Saturation region
	sign := 1.0
	if x < 0 {
		sign = -1.0
		x = -x
	}
	
	// Soft polynomial saturation
	sat := x - (x*x*x)/3.0
	if sat > 1.0 {
		sat = 1.0 - math.Exp(-(x-1.0))
	}
	
	return sign * sat * (1.0 - 0.1*t.saturation)
}

// applyBias adds tape bias coloration
func (t *TapeSaturator) applyBias(x float64) float64 {
	// Bias creates even harmonics and slight asymmetry
	biasAmount := t.bias * 0.3
	
	// Add even harmonics
	x2 := x * x
	harmonics := x + biasAmount*x2*0.5
	
	// Add slight DC offset then remove it
	// This creates asymmetric distortion
	offset := biasAmount * 0.1
	biased := harmonics + offset
	
	// Soft clip to prevent excessive output
	if biased > 1.5 {
		biased = 1.5 - 0.5*math.Exp(-(biased-1.5))
	} else if biased < -1.5 {
		biased = -1.5 + 0.5*math.Exp(-(-biased-1.5))
	}
	
	return biased - offset
}

// applyFlutter simulates wow and flutter
func (t *TapeSaturator) applyFlutter(x float64) float64 {
	// Update flutter LFO
	t.flutterPhase += 2.0 * math.Pi * t.flutterRate / t.sampleRate
	if t.flutterPhase > 2.0*math.Pi {
		t.flutterPhase -= 2.0 * math.Pi
	}
	
	// Complex flutter pattern (combination of frequencies)
	flutter1 := math.Sin(t.flutterPhase)
	flutter2 := math.Sin(t.flutterPhase*3.3) * 0.3
	flutter3 := math.Sin(t.flutterPhase*7.7) * 0.1
	
	totalFlutter := (flutter1 + flutter2 + flutter3) * t.flutter
	
	// Calculate delay time in samples
	delayMs := 0.5 + totalFlutter*2.0 // 0.5-2.5ms variation
	delaySamples := delayMs * t.sampleRate / 1000.0
	
	// Write to delay buffer
	t.delayBuffer[t.delayIndex] = x
	
	// Read with interpolation
	readIndex := float64(t.delayIndex) - delaySamples
	if readIndex < 0 {
		readIndex += float64(len(t.delayBuffer))
	}
	
	// Linear interpolation
	idx1 := int(readIndex)
	idx2 := (idx1 + 1) % len(t.delayBuffer)
	frac := readIndex - float64(idx1)
	
	delayed := t.delayBuffer[idx1]*(1.0-frac) + t.delayBuffer[idx2]*frac
	
	// Update write index
	t.delayIndex = (t.delayIndex + 1) % len(t.delayBuffer)
	
	return delayed
}

// applyWarmth adds tape-style warmth
func (t *TapeSaturator) applyWarmth(x float64) float64 {
	// Simple second-order harmonic generation for warmth
	warmthAmount := t.warmth * 0.1
	return x + warmthAmount*x*x*x
}

// ProcessBuffer applies tape saturation to a buffer of samples
func (t *TapeSaturator) ProcessBuffer(input, output []float64) {
	n := len(input)
	if len(output) < n {
		n = len(output)
	}

	for i := 0; i < n; i++ {
		output[i] = t.Process(input[i])
	}
}

// PreEmphasisFilter boosts high frequencies before saturation
type PreEmphasisFilter struct {
	sampleRate float64
	fc         float64 // Corner frequency
	a0, a1     float64
	b0, b1     float64
	x1, y1     float64
}

// NewPreEmphasisFilter creates a pre-emphasis filter
func NewPreEmphasisFilter(sampleRate float64) *PreEmphasisFilter {
	f := &PreEmphasisFilter{
		sampleRate: sampleRate,
		fc:         3183.0, // Standard 50μs time constant
	}
	f.updateCoefficients()
	return f
}

func (f *PreEmphasisFilter) updateCoefficients() {
	// First-order high shelf filter
	omega := 2.0 * math.Pi * f.fc / f.sampleRate
	K := math.Tan(omega / 2.0)
	V := math.Sqrt(2.0) // ~3dB boost
	
	norm := 1.0 / (1.0 + K)
	
	f.b0 = (V + K) * norm
	f.b1 = (K - V) * norm
	f.a0 = 1.0
	f.a1 = (K - 1.0) * norm
}

// Process applies pre-emphasis
func (f *PreEmphasisFilter) Process(input float64) float64 {
	output := f.b0*input + f.b1*f.x1 - f.a1*f.y1
	f.x1 = input
	f.y1 = output
	return output
}

// DeEmphasisFilter restores frequency balance after saturation
type DeEmphasisFilter struct {
	sampleRate float64
	fc         float64
	a0, a1     float64
	b0, b1     float64
	x1, y1     float64
}

// NewDeEmphasisFilter creates a de-emphasis filter
func NewDeEmphasisFilter(sampleRate float64) *DeEmphasisFilter {
	f := &DeEmphasisFilter{
		sampleRate: sampleRate,
		fc:         3183.0, // Match pre-emphasis
	}
	f.updateCoefficients()
	return f
}

func (f *DeEmphasisFilter) updateCoefficients() {
	// First-order low shelf filter (inverse of pre-emphasis)
	omega := 2.0 * math.Pi * f.fc / f.sampleRate
	K := math.Tan(omega / 2.0)
	V := math.Sqrt(2.0)
	
	norm := 1.0 / (V + K)
	
	f.b0 = (1.0 + K) * norm
	f.b1 = (K - 1.0) * norm
	f.a0 = 1.0
	f.a1 = (K - V) * norm
}

// Process applies de-emphasis
func (f *DeEmphasisFilter) Process(input float64) float64 {
	output := f.b0*input + f.b1*f.x1 - f.a1*f.y1
	f.x1 = input
	f.y1 = output
	return output
}