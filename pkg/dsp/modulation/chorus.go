package modulation

import (
	"math"
)

// Chorus implements a multi-voice chorus effect
type Chorus struct {
	sampleRate float64
	
	// Parameters
	rate      float64   // LFO rate in Hz
	depth     float64   // Modulation depth in ms
	delay     float64   // Base delay time in ms
	mix       float64   // Wet/dry mix (0-1)
	feedback  float64   // Feedback amount (0-0.5)
	spread    float64   // Stereo spread (0-1)
	voices    int       // Number of chorus voices
	
	// Delay lines for each voice (stereo)
	delayLinesL [][]float32
	delayLinesR [][]float32
	delayIndex  int
	maxDelaySamples int
	
	// LFOs for each voice
	lfos []*LFO
	
	// Feedback state
	feedbackL float32
	feedbackR float32
}

// NewChorus creates a new chorus effect
func NewChorus(sampleRate float64) *Chorus {
	c := &Chorus{
		sampleRate: sampleRate,
		rate:       0.5,
		depth:      2.0,
		delay:      20.0,
		mix:        0.5,
		feedback:   0.0,
		spread:     1.0,
		voices:     2,
	}
	
	// Initialize with default voices
	c.SetVoices(2)
	c.updateDelayLines()
	
	return c
}

// SetRate sets the LFO rate in Hz
func (c *Chorus) SetRate(hz float64) {
	c.rate = math.Max(0.01, math.Min(10.0, hz))
	for _, lfo := range c.lfos {
		lfo.SetFrequency(c.rate)
	}
}

// SetDepth sets the modulation depth in milliseconds
func (c *Chorus) SetDepth(ms float64) {
	c.depth = math.Max(0.0, math.Min(10.0, ms))
}

// SetDelay sets the base delay time in milliseconds
func (c *Chorus) SetDelay(ms float64) {
	c.delay = math.Max(1.0, math.Min(50.0, ms))
	c.updateDelayLines()
}

// SetMix sets the wet/dry mix (0=dry, 1=wet)
func (c *Chorus) SetMix(mix float64) {
	c.mix = math.Max(0.0, math.Min(1.0, mix))
}

// SetFeedback sets the feedback amount
func (c *Chorus) SetFeedback(feedback float64) {
	c.feedback = math.Max(0.0, math.Min(0.5, feedback))
}

// SetSpread sets the stereo spread
func (c *Chorus) SetSpread(spread float64) {
	c.spread = math.Max(0.0, math.Min(1.0, spread))
}

// SetVoices sets the number of chorus voices (1-4)
func (c *Chorus) SetVoices(voices int) {
	c.voices = max(1, min(4, voices))
	
	// Create LFOs for each voice
	c.lfos = make([]*LFO, c.voices)
	for i := 0; i < c.voices; i++ {
		c.lfos[i] = NewLFO(c.sampleRate)
		c.lfos[i].SetFrequency(c.rate)
		c.lfos[i].SetWaveform(WaveformSine)
		
		// Offset phase for each voice
		phase := float64(i) / float64(c.voices)
		c.lfos[i].SetPhase(phase)
	}
	
	c.updateDelayLines()
}

// updateDelayLines updates the delay line buffers
func (c *Chorus) updateDelayLines() {
	// Calculate maximum delay needed (base + modulation depth)
	maxDelayMs := c.delay + c.depth
	c.maxDelaySamples = int(maxDelayMs * c.sampleRate / 1000.0)
	
	// Add some headroom
	c.maxDelaySamples = int(float64(c.maxDelaySamples) * 1.2)
	
	// Create delay lines for each voice
	c.delayLinesL = make([][]float32, c.voices)
	c.delayLinesR = make([][]float32, c.voices)
	
	for i := 0; i < c.voices; i++ {
		c.delayLinesL[i] = make([]float32, c.maxDelaySamples)
		c.delayLinesR[i] = make([]float32, c.maxDelaySamples)
	}
	
	c.delayIndex = 0
	c.feedbackL = 0
	c.feedbackR = 0
}

// Process processes mono input
func (c *Chorus) Process(input float32) (outputL, outputR float32) {
	// Process as mono -> stereo
	return c.ProcessStereo(input, input)
}

// ProcessStereo processes stereo input
func (c *Chorus) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// Start with dry signal
	outputL = inputL * float32(1.0-c.mix)
	outputR = inputR * float32(1.0-c.mix)
	
	// Mix of input and feedback for delay lines
	delayInputL := inputL + c.feedbackL*float32(c.feedback)
	delayInputR := inputR + c.feedbackR*float32(c.feedback)
	
	// Write to delay lines
	for v := 0; v < c.voices; v++ {
		c.delayLinesL[v][c.delayIndex] = delayInputL
		c.delayLinesR[v][c.delayIndex] = delayInputR
	}
	
	// Process each voice
	wetL := float32(0)
	wetR := float32(0)
	
	for v := 0; v < c.voices; v++ {
		// Get modulation from LFO (±1)
		modulation := c.lfos[v].Process()
		
		// Calculate delay time in samples
		delayMs := c.delay + c.depth*modulation
		delaySamples := delayMs * c.sampleRate / 1000.0
		
		// Ensure delay is within bounds
		delaySamples = math.Max(1.0, math.Min(float64(c.maxDelaySamples-1), delaySamples))
		
		// Calculate read position with linear interpolation
		readPos := float64(c.delayIndex) - delaySamples
		if readPos < 0 {
			readPos += float64(c.maxDelaySamples)
		}
		
		// Get integer and fractional parts
		readIdx := int(readPos)
		frac := float32(readPos - float64(readIdx))
		
		// Linear interpolation for left channel
		idx1 := readIdx % c.maxDelaySamples
		idx2 := (readIdx + 1) % c.maxDelaySamples
		sampleL := c.delayLinesL[v][idx1]*(1-frac) + c.delayLinesL[v][idx2]*frac
		
		// Linear interpolation for right channel
		sampleR := c.delayLinesR[v][idx1]*(1-frac) + c.delayLinesR[v][idx2]*frac
		
		// Apply stereo spread
		// For multiple voices, pan them across the stereo field
		if c.voices > 1 {
			pan := (float64(v)/float64(c.voices-1) - 0.5) * c.spread
			panL := float32(math.Cos((pan + 0.5) * math.Pi / 2))
			panR := float32(math.Sin((pan + 0.5) * math.Pi / 2))
			
			wetL += sampleL * panL / float32(c.voices)
			wetR += sampleR * panR / float32(c.voices)
		} else {
			wetL += sampleL
			wetR += sampleR
		}
	}
	
	// Store feedback
	c.feedbackL = wetL
	c.feedbackR = wetR
	
	// Add wet signal
	outputL += wetL * float32(c.mix)
	outputR += wetR * float32(c.mix)
	
	// Advance delay index
	c.delayIndex = (c.delayIndex + 1) % c.maxDelaySamples
	
	return outputL, outputR
}

// ProcessBuffer processes a mono buffer
func (c *Chorus) ProcessBuffer(input, outputL, outputR []float32) {
	for i := range input {
		outputL[i], outputR[i] = c.Process(input[i])
	}
}

// ProcessStereoBuffer processes stereo buffers
func (c *Chorus) ProcessStereoBuffer(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		outputL[i], outputR[i] = c.ProcessStereo(inputL[i], inputR[i])
	}
}

// Reset resets the chorus state
func (c *Chorus) Reset() {
	// Clear delay lines
	for v := 0; v < c.voices; v++ {
		for i := range c.delayLinesL[v] {
			c.delayLinesL[v][i] = 0
			c.delayLinesR[v][i] = 0
		}
	}
	
	// Reset LFOs
	for _, lfo := range c.lfos {
		lfo.Reset()
	}
	
	c.delayIndex = 0
	c.feedbackL = 0
	c.feedbackR = 0
}

// Helper functions for min/max
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