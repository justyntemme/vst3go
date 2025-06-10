// Package modulation provides modulation effects like LFOs, chorus, flanger, etc.
package modulation

import (
	"math"
)

// Waveform represents the LFO waveform shape
type Waveform int

const (
	// WaveformSine produces a sine wave
	WaveformSine Waveform = iota
	// WaveformTriangle produces a triangle wave
	WaveformTriangle
	// WaveformSquare produces a square wave
	WaveformSquare
	// WaveformSawtooth produces a sawtooth wave (ramp up)
	WaveformSawtooth
	// WaveformRandom produces random values (sample & hold noise)
	WaveformRandom
)

// LFO implements a Low Frequency Oscillator for modulation
type LFO struct {
	sampleRate float64

	// Parameters
	frequency float64  // Frequency in Hz
	phase     float64  // Current phase (0-1)
	waveform  Waveform // Waveform type
	depth     float64  // Modulation depth (0-1)
	offset    float64  // DC offset (-1 to 1)

	// Sync
	syncEnabled bool
	syncPhase   float64 // Phase to reset to on sync

	// Phase increment
	phaseInc float64

	// For random waveform
	currentRandom float64
	randomCounter int
	randomPeriod  int
}

// NewLFO creates a new LFO
func NewLFO(sampleRate float64) *LFO {
	lfo := &LFO{
		sampleRate: sampleRate,
		frequency:  1.0,
		waveform:   WaveformSine,
		depth:      1.0,
		offset:     0.0,
		phase:      0.0,
	}

	lfo.updatePhaseIncrement()
	return lfo
}

// SetFrequency sets the LFO frequency in Hz
func (l *LFO) SetFrequency(hz float64) {
	l.frequency = math.Max(0.01, math.Min(20.0, hz)) // Limit to reasonable LFO range
	l.updatePhaseIncrement()
}

// SetWaveform sets the LFO waveform
func (l *LFO) SetWaveform(waveform Waveform) {
	l.waveform = waveform
	if waveform == WaveformRandom {
		l.updateRandomPeriod()
		// Generate initial random value
		l.currentRandom = 2.0*randFloat() - 1.0
		l.randomCounter = 0
	}
}

// SetDepth sets the modulation depth (0-1)
func (l *LFO) SetDepth(depth float64) {
	l.depth = math.Max(0.0, math.Min(1.0, depth))
}

// SetOffset sets the DC offset (-1 to 1)
func (l *LFO) SetOffset(offset float64) {
	l.offset = math.Max(-1.0, math.Min(1.0, offset))
}

// SetPhase sets the current phase (0-1)
func (l *LFO) SetPhase(phase float64) {
	l.phase = phase - math.Floor(phase) // Wrap to 0-1
}

// EnableSync enables sync with configurable reset phase
func (l *LFO) EnableSync(enabled bool, resetPhase float64) {
	l.syncEnabled = enabled
	l.syncPhase = math.Max(0.0, math.Min(1.0, resetPhase))
}

// Sync resets the LFO phase (for tempo sync or note retrigger)
func (l *LFO) Sync() {
	if l.syncEnabled {
		l.phase = l.syncPhase
	}
}

// updatePhaseIncrement updates the phase increment based on frequency
func (l *LFO) updatePhaseIncrement() {
	l.phaseInc = l.frequency / l.sampleRate
	l.updateRandomPeriod()
}

// updateRandomPeriod updates the period for random waveform
func (l *LFO) updateRandomPeriod() {
	if l.frequency > 0 {
		l.randomPeriod = int(l.sampleRate / l.frequency)
	} else {
		l.randomPeriod = int(l.sampleRate) // 1 second if frequency is 0
	}
}

// generateWaveform generates the raw waveform value for current phase
func (l *LFO) generateWaveform() float64 {
	switch l.waveform {
	case WaveformSine:
		return math.Sin(2.0 * math.Pi * l.phase)

	case WaveformTriangle:
		// Triangle wave: linear from -1 to 1 and back
		if l.phase < 0.5 {
			return 4.0*l.phase - 1.0
		}
		return 3.0 - 4.0*l.phase

	case WaveformSquare:
		if l.phase < 0.5 {
			return 1.0
		}
		return -1.0

	case WaveformSawtooth:
		// Ramp up from -1 to 1
		return 2.0*l.phase - 1.0

	case WaveformRandom:
		// Sample and hold random values
		if l.randomCounter >= l.randomPeriod {
			l.randomCounter = 0
			// Generate new random value between -1 and 1
			l.currentRandom = 2.0*randFloat() - 1.0
		}
		l.randomCounter++
		return l.currentRandom

	default:
		return 0.0
	}
}

// Process generates the next LFO sample
func (l *LFO) Process() float64 {
	// Generate waveform
	wave := l.generateWaveform()

	// Apply depth and offset
	output := wave*l.depth + l.offset

	// Advance phase
	l.phase += l.phaseInc
	if l.phase >= 1.0 {
		l.phase -= 1.0
	}

	// Clamp output to valid range
	return math.Max(-1.0, math.Min(1.0, output))
}

// ProcessBuffer fills a buffer with LFO values
func (l *LFO) ProcessBuffer(output []float64) {
	for i := range output {
		output[i] = l.Process()
	}
}

// GetPhase returns the current phase (0-1)
func (l *LFO) GetPhase() float64 {
	return l.phase
}

// Reset resets the LFO state
func (l *LFO) Reset() {
	l.phase = 0.0
	l.randomCounter = 0
	l.currentRandom = 0.0
}

// Simple random number generator (can be replaced with better RNG)
var randState uint32 = 1

func randFloat() float64 {
	// Simple linear congruential generator
	randState = randState*1664525 + 1013904223
	return float64(randState) / float64(1<<32)
}
