package dynamics

import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/envelope"
)

// Gate implements a noise gate with hysteresis and smooth operation
type Gate struct {
	sampleRate float64

	// Parameters
	threshold  float64 // Open threshold in dB
	hysteresis float64 // Hysteresis in dB (threshold difference for closing)
	attack     float64 // Attack time in seconds
	hold       float64 // Hold time in seconds
	release    float64 // Release time in seconds
	range_     float64 // Range in dB (max attenuation when closed)
	
	// Side-chain filter (optional)
	hpfEnabled   bool
	hpfFrequency float64
	hpfState     float64 // Simple 1-pole HPF state

	// Envelope detection (not currently used, using instant detection)
	detector *envelope.Detector

	// Gate state machine
	state        gateState
	holdCounter  int
	holdSamples  int
	currentGain  float64
	targetGain   float64

	// Smooth gain transitions
	attackCoeff  float64
	releaseCoeff float64

	// State
	lastInput     float32
	gateOpen      bool
	gainReduction float64 // For metering
}

// gateState represents the current state of the gate
type gateState int

const (
	gateStateClosed gateState = iota
	gateStateAttack
	gateStateOpen
	gateStateHold
	gateStateRelease
)

// NewGate creates a new noise gate
func NewGate(sampleRate float64) *Gate {
	g := &Gate{
		sampleRate:   sampleRate,
		threshold:    -40.0,  // -40 dB default
		hysteresis:   5.0,    // 5 dB hysteresis
		attack:       0.001,  // 1ms attack
		hold:         0.010,  // 10ms hold
		release:      0.100,  // 100ms release
		range_:       -80.0,  // -80 dB range (practically mute)
		state:        gateStateClosed,
		detector:     envelope.NewDetector(sampleRate, envelope.ModePeak),
	}

	// Initialize gain to closed state
	g.currentGain = math.Pow(10.0, g.range_/20.0)
	g.targetGain = g.currentGain
	g.gainReduction = g.range_

	// Configure detector
	g.detector.SetType(envelope.TypeLinear)
	g.detector.SetAttack(0.0001)  // Very fast for gate detection
	g.detector.SetRelease(0.010)  // 10ms release

	// Update coefficients
	g.updateCoefficients()
	g.SetHold(g.hold) // Initialize hold samples

	return g
}

// SetThreshold sets the gate opening threshold in dB
func (g *Gate) SetThreshold(dB float64) {
	g.threshold = dB
}

// SetHysteresis sets the hysteresis in dB
func (g *Gate) SetHysteresis(dB float64) {
	g.hysteresis = math.Max(0.0, dB)
}

// SetAttack sets the attack time in seconds
func (g *Gate) SetAttack(seconds float64) {
	g.attack = math.Max(0.0, seconds)
	g.updateCoefficients()
}

// SetHold sets the hold time in seconds
func (g *Gate) SetHold(seconds float64) {
	g.hold = math.Max(0.0, seconds)
	g.holdSamples = int(g.hold * g.sampleRate)
}

// SetRelease sets the release time in seconds
func (g *Gate) SetRelease(seconds float64) {
	g.release = math.Max(0.0, seconds)
	g.updateCoefficients()
}

// SetRange sets the gate range (max attenuation) in dB
func (g *Gate) SetRange(dB float64) {
	g.range_ = math.Min(0.0, dB) // Can't be positive
	
	// Update current gain if gate is closed
	if g.state == gateStateClosed {
		g.currentGain = math.Pow(10.0, g.range_/20.0)
		g.targetGain = g.currentGain
		g.gainReduction = g.range_
	}
}

// SetSidechainFilter enables/disables the sidechain high-pass filter
func (g *Gate) SetSidechainFilter(enabled bool, frequency float64) {
	g.hpfEnabled = enabled
	g.hpfFrequency = math.Max(20.0, math.Min(frequency, g.sampleRate/2))
}

// updateCoefficients updates the smoothing coefficients
func (g *Gate) updateCoefficients() {
	// Attack and release coefficients for smooth gain changes
	// Using one-pole smoothing: coeff = exp(-1 / (time * sampleRate))
	if g.attack > 0 {
		g.attackCoeff = math.Exp(-1.0 / (g.attack * g.sampleRate))
	} else {
		g.attackCoeff = 0.0 // Instant attack
	}

	if g.release > 0 {
		g.releaseCoeff = math.Exp(-1.0 / (g.release * g.sampleRate))
	} else {
		g.releaseCoeff = 0.0 // Instant release
	}
}

// applySidechainFilter applies optional high-pass filtering to the sidechain signal
func (g *Gate) applySidechainFilter(input float32) float32 {
	if !g.hpfEnabled {
		return input
	}

	// Simple 1-pole high-pass filter
	// H(z) = (1 - z^-1) / (1 - a*z^-1)
	// Where a = exp(-2*pi*fc/fs)
	a := math.Exp(-2.0 * math.Pi * g.hpfFrequency / g.sampleRate)
	
	// Difference equation: y[n] = (1+a)/2 * (x[n] - x[n-1]) + a*y[n-1]
	output := float32((1+a)/2) * (input - g.lastInput) + float32(a)*float32(g.hpfState)
	
	g.lastInput = input
	g.hpfState = float64(output)
	
	return output
}

// Process processes a single sample
func (g *Gate) Process(input float32) float32 {
	// Apply sidechain filter if enabled
	detection := g.applySidechainFilter(input)
	
	// Get envelope - for gate, we want fast detection
	envelope := float32(math.Abs(float64(detection)))
	
	// Convert to dB
	inputDB := float64(-96.0)
	if envelope > 0 {
		inputDB = 20.0 * math.Log10(float64(envelope))
	}

	// State machine logic
	switch g.state {
	case gateStateClosed:
		if inputDB > g.threshold {
			// Open gate
			g.state = gateStateAttack
			g.targetGain = 1.0
		}

	case gateStateAttack:
		if g.currentGain >= 0.99 {
			// Fully open
			g.state = gateStateOpen
			g.gateOpen = true
		} else if inputDB < g.threshold-g.hysteresis {
			// Signal dropped during attack, start closing
			g.state = gateStateRelease
			g.targetGain = math.Pow(10.0, g.range_/20.0)
		}

	case gateStateOpen:
		if inputDB < g.threshold-g.hysteresis {
			// Start hold period
			g.state = gateStateHold
			g.holdCounter = g.holdSamples
		}

	case gateStateHold:
		if inputDB > g.threshold-g.hysteresis {
			// Signal came back up, stay open
			g.state = gateStateOpen
		} else if g.holdCounter > 0 {
			g.holdCounter--
		} else {
			// Hold period expired, start closing
			g.state = gateStateRelease
			g.targetGain = math.Pow(10.0, g.range_/20.0)
			g.gateOpen = false
		}

	case gateStateRelease:
		if inputDB > g.threshold {
			// Signal came back up, reopen
			g.state = gateStateAttack
			g.targetGain = 1.0
		} else if g.currentGain <= g.targetGain*1.01 {
			// Fully closed
			g.state = gateStateClosed
		}
	}

	// Smooth gain transitions
	if g.currentGain < g.targetGain {
		// Opening (attack)
		if g.attackCoeff == 0 {
			g.currentGain = g.targetGain // Instant
		} else {
			g.currentGain = g.targetGain + (g.currentGain-g.targetGain)*g.attackCoeff
		}
	} else if g.currentGain > g.targetGain {
		// Closing (release)
		if g.releaseCoeff == 0 {
			g.currentGain = g.targetGain // Instant
		} else {
			g.currentGain = g.targetGain + (g.currentGain-g.targetGain)*g.releaseCoeff
		}
	}

	// Check if we've reached open state after gain update
	if g.state == gateStateAttack && g.currentGain >= 0.99 {
		g.state = gateStateOpen
		g.gateOpen = true
	} else if g.state == gateStateRelease && g.currentGain <= g.targetGain*1.01 {
		g.state = gateStateClosed
	}

	// Calculate gain reduction for metering
	if g.currentGain > 0 {
		g.gainReduction = 20.0 * math.Log10(g.currentGain)
		if g.gainReduction > -0.1 {
			g.gainReduction = 0.0
		}
	} else {
		g.gainReduction = g.range_
	}

	// Apply gain
	return input * float32(g.currentGain)
}

// ProcessBuffer processes a buffer of samples
func (g *Gate) ProcessBuffer(input, output []float32) {
	for i := range input {
		output[i] = g.Process(input[i])
	}
}

// ProcessStereo processes stereo buffers with linked gating
func (g *Gate) ProcessStereo(inputL, inputR, outputL, outputR []float32) {
	for i := range inputL {
		// Use maximum of both channels for detection
		maxInput := float32(math.Max(math.Abs(float64(inputL[i])), math.Abs(float64(inputR[i]))))
		
		// Apply sidechain filter
		detection := g.applySidechainFilter(maxInput)
		
		// Get envelope - for gate, we want fast detection
		envelope := float32(math.Abs(float64(detection)))
		
		// Convert to dB
		inputDB := float64(-96.0)
		if envelope > 0 {
			inputDB = 20.0 * math.Log10(float64(envelope))
		}

		// Run state machine (same as Process method)
		switch g.state {
		case gateStateClosed:
			if inputDB > g.threshold {
				g.state = gateStateAttack
				g.targetGain = 1.0
			}

		case gateStateAttack:
			if g.currentGain >= 0.99 {
				g.state = gateStateOpen
				g.gateOpen = true
			} else if inputDB < g.threshold-g.hysteresis {
				g.state = gateStateRelease
				g.targetGain = math.Pow(10.0, g.range_/20.0)
			}

		case gateStateOpen:
			if inputDB < g.threshold-g.hysteresis {
				g.state = gateStateHold
				g.holdCounter = g.holdSamples
			}

		case gateStateHold:
			if inputDB > g.threshold-g.hysteresis {
				g.state = gateStateOpen
			} else if g.holdCounter > 0 {
				g.holdCounter--
			} else {
				g.state = gateStateRelease
				g.targetGain = math.Pow(10.0, g.range_/20.0)
				g.gateOpen = false
			}

		case gateStateRelease:
			if inputDB > g.threshold {
				g.state = gateStateAttack
				g.targetGain = 1.0
			} else if g.currentGain <= g.targetGain*1.01 {
				g.state = gateStateClosed
			}
		}

		// Smooth gain transitions
		if g.currentGain < g.targetGain {
			if g.attackCoeff == 0 {
				g.currentGain = g.targetGain
			} else {
				g.currentGain = g.targetGain + (g.currentGain-g.targetGain)*g.attackCoeff
			}
		} else if g.currentGain > g.targetGain {
			if g.releaseCoeff == 0 {
				g.currentGain = g.targetGain
			} else {
				g.currentGain = g.targetGain + (g.currentGain-g.targetGain)*g.releaseCoeff
			}
		}

		// Check if we've reached open state after gain update
		if g.state == gateStateAttack && g.currentGain >= 0.99 {
			g.state = gateStateOpen
			g.gateOpen = true
		} else if g.state == gateStateRelease && g.currentGain <= g.targetGain*1.01 {
			g.state = gateStateClosed
		}

		// Update gain reduction
		if g.currentGain > 0 {
			g.gainReduction = 20.0 * math.Log10(g.currentGain)
			if g.gainReduction > -0.1 {
				g.gainReduction = 0.0
			}
		} else {
			g.gainReduction = g.range_
		}

		// Apply same gain to both channels
		gain := float32(g.currentGain)
		outputL[i] = inputL[i] * gain
		outputR[i] = inputR[i] * gain
	}
}

// GetGainReduction returns the current gain reduction in dB
func (g *Gate) GetGainReduction() float64 {
	return g.gainReduction
}

// IsOpen returns true if the gate is currently open
func (g *Gate) IsOpen() bool {
	return g.gateOpen
}

// GetState returns the current gate state for debugging
func (g *Gate) GetState() string {
	switch g.state {
	case gateStateClosed:
		return "closed"
	case gateStateAttack:
		return "attack"
	case gateStateOpen:
		return "open"
	case gateStateHold:
		return "hold"
	case gateStateRelease:
		return "release"
	default:
		return "unknown"
	}
}

// Reset resets the gate state
func (g *Gate) Reset() {
	g.detector.Reset()
	g.state = gateStateClosed
	g.currentGain = math.Pow(10.0, g.range_/20.0)
	g.targetGain = g.currentGain
	g.holdCounter = 0
	g.gateOpen = false
	g.gainReduction = g.range_
	g.hpfState = 0.0
	g.lastInput = 0.0
}