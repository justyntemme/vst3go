package reverb

import (
	"math"
	"math/rand"
)

// FDN implements a Feedback Delay Network reverb
// This is a more sophisticated reverb algorithm that uses multiple delay lines
// with a feedback matrix for rich, dense reverberation
type FDN struct {
	// Number of delay lines (typically 4, 8, or 16)
	numDelays int

	// Delay lines
	delayLines   [][]float32
	delayTimes   []int
	writeIndices []int

	// Feedback matrix (Hadamard matrix for good diffusion)
	feedbackMatrix [][]float64

	// Input and output gains
	inputGains  []float64
	outputGains []float64

	// Damping filters (one per delay line)
	dampingFilters []*DampingFilter

	// Global parameters
	decay      float64
	damping    float64
	diffusion  float64
	modulation float64
	wetLevel   float64
	dryLevel   float64

	// Modulation LFOs
	modLFOs   []float64
	modPhases []float64
	modDepth  float64
	modRate   float64

	sampleRate float64
}

// DampingFilter implements a simple one-pole lowpass filter for damping
type DampingFilter struct {
	state float32
	coeff float64
}

// NewDampingFilter creates a new damping filter
func NewDampingFilter() *DampingFilter {
	return &DampingFilter{
		state: 0,
		coeff: 0.5,
	}
}

// SetDamping sets the damping coefficient (0-1, where 0 = no damping, 1 = maximum damping)
func (d *DampingFilter) SetDamping(damping float64) {
	// Convert damping to lowpass coefficient
	// Higher damping = lower cutoff frequency
	d.coeff = 1.0 - math.Max(0.0, math.Min(1.0, damping))
}

// Process applies the damping filter
func (d *DampingFilter) Process(input float32) float32 {
	// One-pole lowpass: y[n] = x[n] * (1-coeff) + y[n-1] * coeff
	d.state = input*float32(1.0-d.coeff) + d.state*float32(d.coeff)
	return d.state
}

// Reset clears the filter state
func (d *DampingFilter) Reset() {
	d.state = 0
}

// NewFDN creates a new Feedback Delay Network reverb
func NewFDN(numDelays int, sampleRate float64) *FDN {
	f := &FDN{
		numDelays:      numDelays,
		delayLines:     make([][]float32, numDelays),
		delayTimes:     make([]int, numDelays),
		writeIndices:   make([]int, numDelays),
		inputGains:     make([]float64, numDelays),
		outputGains:    make([]float64, numDelays),
		dampingFilters: make([]*DampingFilter, numDelays),
		modLFOs:        make([]float64, numDelays),
		modPhases:      make([]float64, numDelays),
		decay:          0.5,
		damping:        0.5,
		diffusion:      0.5,
		modulation:     0.0,
		wetLevel:       0.3,
		dryLevel:       0.7,
		modDepth:       5.0, // samples
		modRate:        0.5, // Hz
		sampleRate:     sampleRate,
	}

	// Initialize delay times using prime numbers for good diffusion
	// These are scaled for the sample rate
	primes := []int{23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89}
	baseDelay := int(sampleRate * 0.01) // 10ms base delay

	for i := 0; i < numDelays; i++ {
		// Use prime number ratios for delay times
		delayTime := baseDelay * primes[i%len(primes)] / 23
		f.delayTimes[i] = delayTime
		f.delayLines[i] = make([]float32, delayTime+int(f.modDepth)+1)

		// Initialize damping filters
		f.dampingFilters[i] = NewDampingFilter()

		// Initialize modulation phases
		f.modPhases[i] = float64(i) * 2.0 * math.Pi / float64(numDelays)

		// Equal input/output gains
		f.inputGains[i] = 1.0 / math.Sqrt(float64(numDelays))
		f.outputGains[i] = 1.0 / math.Sqrt(float64(numDelays))
	}

	// Create feedback matrix (Hadamard matrix for optimal diffusion)
	f.createHadamardMatrix()

	// Apply initial parameter settings
	f.updateInternalParameters()

	return f
}

// createHadamardMatrix creates a Hadamard matrix for the feedback network
func (f *FDN) createHadamardMatrix() {
	n := f.numDelays
	f.feedbackMatrix = make([][]float64, n)
	for i := range f.feedbackMatrix {
		f.feedbackMatrix[i] = make([]float64, n)
	}

	// For simplicity, use a normalized Hadamard-like matrix
	// This provides good diffusion properties
	if n == 4 {
		// 4x4 Hadamard matrix
		h := [][]float64{
			{1, 1, 1, 1},
			{1, -1, 1, -1},
			{1, 1, -1, -1},
			{1, -1, -1, 1},
		}
		scale := 0.5 // 1/sqrt(4)
		for i := 0; i < 4; i++ {
			for j := 0; j < 4; j++ {
				f.feedbackMatrix[i][j] = h[i][j] * scale
			}
		}
	} else if n == 8 {
		// 8x8 Hadamard matrix (simplified)
		scale := 1.0 / math.Sqrt(8.0)
		for i := 0; i < 8; i++ {
			for j := 0; j < 8; j++ {
				// Create a pattern that provides good diffusion
				if (i+j)%2 == 0 {
					f.feedbackMatrix[i][j] = scale
				} else {
					f.feedbackMatrix[i][j] = -scale
				}
			}
		}
	} else {
		// For other sizes, use a householder reflection matrix
		f.createHouseholderMatrix()
	}
}

// createHouseholderMatrix creates a Householder reflection matrix
func (f *FDN) createHouseholderMatrix() {
	n := f.numDelays

	// Create a random unit vector
	v := make([]float64, n)
	sum := 0.0
	for i := 0; i < n; i++ {
		v[i] = rand.Float64() - 0.5
		sum += v[i] * v[i]
	}

	// Normalize
	norm := math.Sqrt(sum)
	for i := 0; i < n; i++ {
		v[i] /= norm
	}

	// Create Householder matrix: H = I - 2*v*v^T
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				f.feedbackMatrix[i][j] = 1.0 - 2.0*v[i]*v[j]
			} else {
				f.feedbackMatrix[i][j] = -2.0 * v[i] * v[j]
			}
		}
	}
}

// SetDecay sets the decay time (0-1, where 0 = short, 1 = long)
func (f *FDN) SetDecay(decay float64) {
	f.decay = math.Max(0.0, math.Min(1.0, decay))
	f.updateInternalParameters()
}

// SetDamping sets the damping amount (0-1)
func (f *FDN) SetDamping(damping float64) {
	f.damping = math.Max(0.0, math.Min(1.0, damping))
	f.updateInternalParameters()
}

// SetDiffusion sets the diffusion amount (0-1)
func (f *FDN) SetDiffusion(diffusion float64) {
	f.diffusion = math.Max(0.0, math.Min(1.0, diffusion))
	f.updateInternalParameters()
}

// SetModulation sets the modulation amount (0-1)
func (f *FDN) SetModulation(modulation float64) {
	f.modulation = math.Max(0.0, math.Min(1.0, modulation))
}

// SetWetLevel sets the wet signal level (0-1)
func (f *FDN) SetWetLevel(level float64) {
	f.wetLevel = math.Max(0.0, math.Min(1.0, level))
}

// SetDryLevel sets the dry signal level (0-1)
func (f *FDN) SetDryLevel(level float64) {
	f.dryLevel = math.Max(0.0, math.Min(1.0, level))
}

// updateInternalParameters updates internal parameters after changes
func (f *FDN) updateInternalParameters() {
	// Update damping filters
	for i := 0; i < f.numDelays; i++ {
		f.dampingFilters[i].SetDamping(f.damping)
	}

	// Scale feedback matrix by decay amount
	// Higher decay = more feedback = longer reverb tail
	decayScale := 0.4 + f.decay*0.58 // Range from 0.4 to 0.98

	// Apply diffusion by mixing with identity matrix
	// Low diffusion = more parallel delays, high diffusion = more mixing
	diffusionMix := f.diffusion
}

// Process processes a mono input sample
func (f *FDN) Process(input float32) float32 {
	// Read outputs from all delay lines
	delayOutputs := make([]float32, f.numDelays)

	for i := 0; i < f.numDelays; i++ {
		// Calculate modulated read position
		modulation := 0.0
		if f.modulation > 0 {
			// Update LFO
			f.modLFOs[i] = math.Sin(f.modPhases[i])
			f.modPhases[i] += 2.0 * math.Pi * f.modRate / f.sampleRate
			if f.modPhases[i] > 2.0*math.Pi {
				f.modPhases[i] -= 2.0 * math.Pi
			}
			modulation = f.modLFOs[i] * f.modDepth * f.modulation
		}

		// Calculate read index with modulation
		readPos := f.writeIndices[i] - f.delayTimes[i]
		modulatedPos := float64(readPos) - modulation

		// Wrap around
		for modulatedPos < 0 {
			modulatedPos += float64(len(f.delayLines[i]))
		}

		// Linear interpolation for smooth modulation
		intPos := int(modulatedPos)
		frac := float32(modulatedPos - float64(intPos))

		idx1 := intPos % len(f.delayLines[i])
		idx2 := (intPos + 1) % len(f.delayLines[i])

		// Interpolate
		delayOutputs[i] = f.delayLines[i][idx1]*(1-frac) + f.delayLines[i][idx2]*frac
	}

	// Apply feedback matrix to create new inputs
	feedbackInputs := make([]float32, f.numDelays)
	decayScale := float32(0.4 + f.decay*0.58)

	for i := 0; i < f.numDelays; i++ {
		sum := float32(0)
		for j := 0; j < f.numDelays; j++ {
			// Mix between feedback matrix and identity based on diffusion
			if i == j {
				// Identity component (parallel delays)
				sum += delayOutputs[j] * float32(1.0-f.diffusion) * decayScale
			}
			// Feedback matrix component (cross-coupling)
			sum += delayOutputs[j] * float32(f.feedbackMatrix[i][j]*f.diffusion) * decayScale
		}
		feedbackInputs[i] = sum
	}

	// Write to delay lines with damping
	for i := 0; i < f.numDelays; i++ {
		// Mix input with feedback
		delayInput := input*float32(f.inputGains[i]) + feedbackInputs[i]

		// Apply damping
		delayInput = f.dampingFilters[i].Process(delayInput)

		// Write to delay line
		f.delayLines[i][f.writeIndices[i]] = delayInput

		// Update write index
		f.writeIndices[i]++
		if f.writeIndices[i] >= len(f.delayLines[i]) {
			f.writeIndices[i] = 0
		}
	}

	// Sum outputs
	output := float32(0)
	for i := 0; i < f.numDelays; i++ {
		output += delayOutputs[i] * float32(f.outputGains[i])
	}

	// Apply wet/dry mix
	return input*float32(f.dryLevel) + output*float32(f.wetLevel)
}

// ProcessStereo processes stereo input
func (f *FDN) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// Mix to mono for processing
	mono := (inputL + inputR) * 0.5

	// Process through FDN
	processed := f.Process(mono)

	// Create stereo output by using different combinations of delay outputs
	// This is a simplified approach - a full implementation would have
	// separate processing for left and right channels

	// For now, create a basic stereo spread
	outputL = processed
	outputR = processed

	// Add some stereo width by phase inverting some of the signal
	if f.numDelays >= 2 {
		// Get two delay outputs for stereo decorrelation
		delay1 := f.delayLines[0][f.writeIndices[0]-1]
		if f.writeIndices[0]-1 < 0 {
			delay1 = f.delayLines[0][len(f.delayLines[0])-1]
		}

		delay2 := f.delayLines[1][f.writeIndices[1]-1]
		if f.writeIndices[1]-1 < 0 {
			delay2 = f.delayLines[1][len(f.delayLines[1])-1]
		}

		// Add decorrelated signals to create width
		spread := float32(0.3)
		outputL += delay1 * spread * float32(f.wetLevel)
		outputR += delay2 * spread * float32(f.wetLevel)
	}

	// Apply final wet/dry mix
	outputL = inputL*float32(f.dryLevel) + outputL*float32(f.wetLevel)
	outputR = inputR*float32(f.dryLevel) + outputR*float32(f.wetLevel)

	return outputL, outputR
}

// Reset clears all internal state
func (f *FDN) Reset() {
	// Clear all delay lines
	for i := 0; i < f.numDelays; i++ {
		for j := range f.delayLines[i] {
			f.delayLines[i][j] = 0
		}
		f.writeIndices[i] = 0
		f.dampingFilters[i].Reset()
		f.modPhases[i] = float64(i) * 2.0 * math.Pi / float64(f.numDelays)
	}
}

// Preset methods

// SetPresetSmallRoom configures the FDN for a small room
func (f *FDN) SetPresetSmallRoom() {
	f.SetDecay(0.2)
	f.SetDamping(0.8)
	f.SetDiffusion(0.7)
	f.SetModulation(0.1)
	f.SetWetLevel(0.25)
	f.SetDryLevel(0.75)
}

// SetPresetMediumHall configures the FDN for a medium hall
func (f *FDN) SetPresetMediumHall() {
	f.SetDecay(0.5)
	f.SetDamping(0.5)
	f.SetDiffusion(0.85)
	f.SetModulation(0.15)
	f.SetWetLevel(0.35)
	f.SetDryLevel(0.65)
}

// SetPresetLargeHall configures the FDN for a large hall
func (f *FDN) SetPresetLargeHall() {
	f.SetDecay(0.8)
	f.SetDamping(0.3)
	f.SetDiffusion(0.9)
	f.SetModulation(0.2)
	f.SetWetLevel(0.4)
	f.SetDryLevel(0.6)
}

// SetPresetCathedral configures the FDN for a cathedral
func (f *FDN) SetPresetCathedral() {
	f.SetDecay(0.95)
	f.SetDamping(0.1)
	f.SetDiffusion(0.95)
	f.SetModulation(0.25)
	f.SetWetLevel(0.5)
	f.SetDryLevel(0.5)
}
