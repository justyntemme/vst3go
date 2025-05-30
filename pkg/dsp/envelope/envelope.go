// Package envelope provides envelope generators for audio synthesis
package envelope

import "math"

// Stage represents the current envelope stage
type Stage int

const (
	// StageIdle represents envelope idle state
	StageIdle Stage = iota
	// StageAttack represents envelope attack phase
	StageAttack
	// StageDecay represents envelope decay phase
	StageDecay
	// StageSustain represents envelope sustain phase
	StageSustain
	// StageRelease represents envelope release phase
	StageRelease
)

// ADSR implements an Attack-Decay-Sustain-Release envelope generator
type ADSR struct {
	sampleRate float64

	// Parameters (in seconds for A,D,R and 0-1 for S)
	attack  float64
	decay   float64
	sustain float64
	release float64

	// Coefficients (pre-calculated for efficiency)
	attackCoef  float64
	decayCoef   float64
	releaseCoef float64

	// State
	stage  Stage
	value  float64
	target float64
}

// New creates a new ADSR envelope
func New(sampleRate float64) *ADSR {
	env := &ADSR{
		sampleRate: sampleRate,
		attack:     0.01,
		decay:      0.1,
		sustain:    0.7,
		release:    0.3,
		stage:      StageIdle,
		value:      0.0,
		target:     0.0,
	}
	env.updateCoefficients()
	return env
}

// SetAttack sets the attack time in seconds
func (e *ADSR) SetAttack(seconds float64) {
	e.attack = math.Max(0.001, seconds) // Minimum 1ms
	e.updateCoefficients()
}

// SetDecay sets the decay time in seconds
func (e *ADSR) SetDecay(seconds float64) {
	e.decay = math.Max(0.001, seconds)
	e.updateCoefficients()
}

// SetSustain sets the sustain level (0-1)
func (e *ADSR) SetSustain(level float64) {
	e.sustain = math.Max(0.0, math.Min(1.0, level))
}

// SetRelease sets the release time in seconds
func (e *ADSR) SetRelease(seconds float64) {
	e.release = math.Max(0.001, seconds)
	e.updateCoefficients()
}

// SetADSR sets all parameters at once
func (e *ADSR) SetADSR(attack, decay, sustain, release float64) {
	e.attack = math.Max(0.001, attack)
	e.decay = math.Max(0.001, decay)
	e.sustain = math.Max(0.0, math.Min(1.0, sustain))
	e.release = math.Max(0.001, release)
	e.updateCoefficients()
}

// updateCoefficients recalculates the exponential coefficients
func (e *ADSR) updateCoefficients() {
	// Using exponential curves for natural sound
	// coef = exp(-1 / (time * sampleRate))
	e.attackCoef = calcCoef(e.attack, e.sampleRate)
	e.decayCoef = calcCoef(e.decay, e.sampleRate)
	e.releaseCoef = calcCoef(e.release, e.sampleRate)
}

// calcCoef calculates exponential coefficient for a given time
func calcCoef(timeSeconds, sampleRate float64) float64 {
	if timeSeconds <= 0.0 {
		return 0.0
	}
	return math.Exp(-1.0 / (timeSeconds * sampleRate))
}

// Trigger starts the envelope (note on)
func (e *ADSR) Trigger() {
	e.stage = StageAttack
	e.target = 1.0
}

// Release starts the release stage (note off)
func (e *ADSR) Release() {
	if e.stage != StageIdle {
		e.stage = StageRelease
		e.target = 0.0
	}
}

// Reset immediately returns the envelope to idle
func (e *ADSR) Reset() {
	e.stage = StageIdle
	e.value = 0.0
	e.target = 0.0
}

// IsActive returns true if the envelope is generating output
func (e *ADSR) IsActive() bool {
	return e.stage != StageIdle
}

// GetStage returns the current envelope stage
func (e *ADSR) GetStage() Stage {
	return e.stage
}

// Next generates the next envelope value
func (e *ADSR) Next() float32 {
	switch e.stage {
	case StageAttack:
		e.value = e.target + (e.value-e.target)*e.attackCoef
		if e.value >= 0.999 {
			e.value = 1.0
			e.stage = StageDecay
			e.target = e.sustain
		}

	case StageDecay:
		e.value = e.target + (e.value-e.target)*e.decayCoef
		if e.value <= e.sustain+0.001 {
			e.value = e.sustain
			e.stage = StageSustain
		}

	case StageSustain:
		e.value = e.sustain

	case StageRelease:
		e.value = e.target + (e.value-e.target)*e.releaseCoef
		if e.value <= 0.001 {
			e.value = 0.0
			e.stage = StageIdle
		}

	case StageIdle:
		e.value = 0.0
	}

	return float32(e.value)
}

// Process fills buffer with envelope values - no allocations
func (e *ADSR) Process(buffer []float32) {
	for i := range buffer {
		buffer[i] = e.Next()
	}
}

// ProcessMultiply multiplies buffer by envelope - no allocations
func (e *ADSR) ProcessMultiply(buffer []float32) {
	for i := range buffer {
		buffer[i] *= e.Next()
	}
}

// AR implements a simple Attack-Release envelope
type AR struct {
	sampleRate float64

	// Parameters
	attack  float64
	release float64

	// Coefficients
	attackCoef  float64
	releaseCoef float64

	// State
	active bool
	value  float64
	target float64
}

// NewAR creates a new AR envelope
func NewAR(sampleRate float64) *AR {
	env := &AR{
		sampleRate: sampleRate,
		attack:     0.01,
		release:    0.1,
	}
	env.updateCoefficients()
	return env
}

// SetAttack sets the attack time in seconds
func (e *AR) SetAttack(seconds float64) {
	e.attack = math.Max(0.001, seconds)
	e.updateCoefficients()
}

// SetRelease sets the release time in seconds
func (e *AR) SetRelease(seconds float64) {
	e.release = math.Max(0.001, seconds)
	e.updateCoefficients()
}

// updateCoefficients recalculates the exponential coefficients
func (e *AR) updateCoefficients() {
	e.attackCoef = calcCoef(e.attack, e.sampleRate)
	e.releaseCoef = calcCoef(e.release, e.sampleRate)
}

// Trigger starts the attack phase
func (e *AR) Trigger() {
	e.active = true
	e.target = 1.0
}

// Release starts the release phase
func (e *AR) Release() {
	e.active = false
	e.target = 0.0
}

// Next generates the next envelope value
func (e *AR) Next() float32 {
	if e.active {
		e.value = e.target + (e.value-e.target)*e.attackCoef
	} else {
		e.value = e.target + (e.value-e.target)*e.releaseCoef
	}
	return float32(e.value)
}

// Process fills buffer with envelope values - no allocations
func (e *AR) Process(buffer []float32) {
	for i := range buffer {
		buffer[i] = e.Next()
	}
}

// ProcessMultiply multiplies buffer by envelope - no allocations
func (e *AR) ProcessMultiply(buffer []float32) {
	for i := range buffer {
		buffer[i] *= e.Next()
	}
}

// Follower implements an envelope follower for dynamics processing
type Follower struct {
	sampleRate  float64
	attack      float64
	release     float64
	attackCoef  float64
	releaseCoef float64
	envelope    float64
}

// NewFollower creates a new envelope follower
func NewFollower(sampleRate float64) *Follower {
	f := &Follower{
		sampleRate: sampleRate,
		attack:     0.01,
		release:    0.1,
	}
	f.updateCoefficients()
	return f
}

// SetAttack sets the attack time
func (f *Follower) SetAttack(seconds float64) {
	f.attack = math.Max(0.0001, seconds)
	f.updateCoefficients()
}

// SetRelease sets the release time
func (f *Follower) SetRelease(seconds float64) {
	f.release = math.Max(0.0001, seconds)
	f.updateCoefficients()
}

// updateCoefficients recalculates coefficients
func (f *Follower) updateCoefficients() {
	f.attackCoef = calcCoef(f.attack, f.sampleRate)
	f.releaseCoef = calcCoef(f.release, f.sampleRate)
}

// Process extracts the envelope from a signal - no allocations
func (f *Follower) Process(input, output []float32) {
	for i := range input {
		// Get absolute value of input
		absInput := float64(input[i])
		if absInput < 0 {
			absInput = -absInput
		}

		// Apply attack or release
		if absInput > f.envelope {
			f.envelope = absInput + (f.envelope-absInput)*f.attackCoef
		} else {
			f.envelope = absInput + (f.envelope-absInput)*f.releaseCoef
		}

		output[i] = float32(f.envelope)
	}
}

// Follow processes a single sample
func (f *Follower) Follow(input float32) float32 {
	absInput := float64(input)
	if absInput < 0 {
		absInput = -absInput
	}

	if absInput > f.envelope {
		f.envelope = absInput + (f.envelope-absInput)*f.attackCoef
	} else {
		f.envelope = absInput + (f.envelope-absInput)*f.releaseCoef
	}

	return float32(f.envelope)
}
