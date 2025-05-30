// Package delay provides delay line implementations for audio effects
package delay

import "math"

// Line implements a basic delay line with linear interpolation
type Line struct {
	buffer     []float32
	bufferSize int
	writePos   int
	sampleRate float64
}

// New creates a new delay line with the specified maximum delay time
func New(maxDelaySeconds, sampleRate float64) *Line {
	bufferSize := int(maxDelaySeconds*sampleRate) + 1
	return &Line{
		buffer:     make([]float32, bufferSize),
		bufferSize: bufferSize,
		writePos:   0,
		sampleRate: sampleRate,
	}
}

// Reset clears the delay buffer
func (d *Line) Reset() {
	for i := range d.buffer {
		d.buffer[i] = 0
	}
	d.writePos = 0
}

// Write adds a sample to the delay line
func (d *Line) Write(sample float32) {
	d.buffer[d.writePos] = sample
	d.writePos++
	if d.writePos >= d.bufferSize {
		d.writePos = 0
	}
}

// Read gets a delayed sample (delay in samples)
func (d *Line) Read(delaySamples float64) float32 {
	// Calculate read position
	readPos := float64(d.writePos) - delaySamples
	if readPos < 0 {
		readPos += float64(d.bufferSize)
	}
	
	// Linear interpolation
	readPosInt := int(readPos)
	frac := float32(readPos - float64(readPosInt))
	
	// Get two samples for interpolation
	s1 := d.buffer[readPosInt]
	s2 := d.buffer[(readPosInt+1)%d.bufferSize]
	
	// Linear interpolation
	return s1*(1.0-frac) + s2*frac
}

// ReadMs gets a delayed sample (delay in milliseconds)
func (d *Line) ReadMs(delayMs float64) float32 {
	delaySamples := delayMs * d.sampleRate / 1000.0
	return d.Read(delaySamples)
}

// Tap reads without writing (for multi-tap delays)
func (d *Line) Tap(delaySamples float64) float32 {
	return d.Read(delaySamples)
}

// Process writes and reads in one operation
func (d *Line) Process(input float32, delaySamples float64) float32 {
	output := d.Read(delaySamples)
	d.Write(input)
	return output
}

// ProcessMs writes and reads with delay in milliseconds
func (d *Line) ProcessMs(input float32, delayMs float64) float32 {
	delaySamples := delayMs * d.sampleRate / 1000.0
	return d.Process(input, delaySamples)
}

// ProcessBuffer processes a buffer with fixed delay - no allocations
func (d *Line) ProcessBuffer(buffer []float32, delaySamples float64) {
	for i := range buffer {
		delayed := d.Read(delaySamples)
		d.Write(buffer[i])
		buffer[i] = delayed
	}
}

// ProcessBufferMix processes with dry/wet mix - no allocations
func (d *Line) ProcessBufferMix(buffer []float32, delaySamples float64, mix float32) {
	dryGain := 1.0 - mix
	for i := range buffer {
		dry := buffer[i]
		wet := d.Process(dry, delaySamples)
		buffer[i] = dry*dryGain + wet*mix
	}
}

// AllpassDelay implements an allpass delay for reverb effects
type AllpassDelay struct {
	Line
	feedback float32
}

// NewAllpass creates a new allpass delay
func NewAllpass(maxDelaySeconds, sampleRate float64) *AllpassDelay {
	return &AllpassDelay{
		Line:     *New(maxDelaySeconds, sampleRate),
		feedback: 0.5,
	}
}

// SetFeedback sets the allpass feedback coefficient
func (a *AllpassDelay) SetFeedback(feedback float32) {
	a.feedback = feedback
}

// Process runs the allpass filter
func (a *AllpassDelay) Process(input float32, delaySamples float64) float32 {
	delayed := a.Read(delaySamples)
	output := -input + delayed
	a.Write(input + delayed*a.feedback)
	return output
}

// ProcessBuffer processes a buffer through the allpass - no allocations
func (a *AllpassDelay) ProcessBuffer(buffer []float32, delaySamples float64) {
	for i := range buffer {
		buffer[i] = a.Process(buffer[i], delaySamples)
	}
}

// CombDelay implements a comb filter delay
type CombDelay struct {
	Line
	feedback float32
	damp     float32
	dampVal  float32
}

// NewComb creates a new comb filter delay
func NewComb(maxDelaySeconds, sampleRate float64) *CombDelay {
	return &CombDelay{
		Line:     *New(maxDelaySeconds, sampleRate),
		feedback: 0.5,
		damp:     0.5,
		dampVal:  0.0,
	}
}

// SetFeedback sets the comb feedback
func (c *CombDelay) SetFeedback(feedback float32) {
	c.feedback = feedback
}

// SetDamp sets the damping factor (0=no damping, 1=full damping)
func (c *CombDelay) SetDamp(damp float32) {
	c.damp = damp
}

// Process runs the comb filter
func (c *CombDelay) Process(input float32, delaySamples float64) float32 {
	output := c.Read(delaySamples)
	
	// Apply damping (simple one-pole lowpass)
	c.dampVal = output*(1.0-c.damp) + c.dampVal*c.damp
	
	// Write input + filtered feedback
	c.Write(input + c.dampVal*c.feedback)
	
	return output
}

// ProcessBuffer processes a buffer through the comb - no allocations
func (c *CombDelay) ProcessBuffer(buffer []float32, delaySamples float64) {
	for i := range buffer {
		buffer[i] = c.Process(buffer[i], delaySamples)
	}
}

// MultiTapDelay provides multiple delay taps
type MultiTapDelay struct {
	Line
	numTaps int
}

// TapOutput represents a single tap configuration
type TapOutput struct {
	DelaySamples float64
	Gain         float32
	Pan          float32 // -1 (left) to 1 (right)
}

// NewMultiTap creates a multi-tap delay
func NewMultiTap(maxDelaySeconds, sampleRate float64, numTaps int) *MultiTapDelay {
	return &MultiTapDelay{
		Line:    *New(maxDelaySeconds, sampleRate),
		numTaps: numTaps,
	}
}

// ProcessMultiTap processes with multiple taps - returns stereo output
func (m *MultiTapDelay) ProcessMultiTap(input float32, taps []TapOutput, outL, outR *float32) {
	// Write input to delay line
	m.Write(input)
	
	// Clear outputs
	*outL = 0
	*outR = 0
	
	// Sum all taps
	for i := range taps {
		if i >= m.numTaps {
			break
		}
		
		tap := &taps[i]
		delayed := m.Tap(tap.DelaySamples) * tap.Gain
		
		// Pan (constant power)
		panAngle := (tap.Pan + 1.0) * 0.25 * math.Pi // 0 to π/2
		leftGain := float32(math.Cos(float64(panAngle)))
		rightGain := float32(math.Sin(float64(panAngle)))
		
		*outL += delayed * leftGain
		*outR += delayed * rightGain
	}
}

// ModulatedDelay implements a delay with LFO modulation
type ModulatedDelay struct {
	Line
	lfoPhase   float64
	lfoRate    float64
	lfoDepth   float64
	centerDelay float64
}

// NewModulated creates a modulated delay line
func NewModulated(maxDelaySeconds, sampleRate float64) *ModulatedDelay {
	return &ModulatedDelay{
		Line:        *New(maxDelaySeconds, sampleRate),
		lfoPhase:    0.0,
		lfoRate:     0.5,
		lfoDepth:    5.0, // milliseconds
		centerDelay: 10.0, // milliseconds
	}
}

// SetLFO configures the modulation
func (m *ModulatedDelay) SetLFO(rateHz, depthMs float64) {
	m.lfoRate = rateHz
	m.lfoDepth = depthMs
}

// SetCenterDelay sets the center delay time in milliseconds
func (m *ModulatedDelay) SetCenterDelay(delayMs float64) {
	m.centerDelay = delayMs
}

// Process with modulation
func (m *ModulatedDelay) Process(input float32) float32 {
	// Calculate modulated delay
	lfo := math.Sin(2.0 * math.Pi * m.lfoPhase)
	delayMs := m.centerDelay + m.lfoDepth*lfo
	delaySamples := delayMs * m.sampleRate / 1000.0
	
	// Ensure delay is positive
	if delaySamples < 1.0 {
		delaySamples = 1.0
	}
	
	// Process with modulated delay
	output := m.Read(delaySamples)
	m.Write(input)
	
	// Update LFO phase
	m.lfoPhase += m.lfoRate / m.sampleRate
	if m.lfoPhase >= 1.0 {
		m.lfoPhase -= 1.0
	}
	
	return output
}

// ProcessBuffer with modulation - no allocations
func (m *ModulatedDelay) ProcessBuffer(buffer []float32) {
	for i := range buffer {
		buffer[i] = m.Process(buffer[i])
	}
}