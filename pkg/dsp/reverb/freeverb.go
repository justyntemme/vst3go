package reverb

import (
	"math"
)

// Freeverb tuning constants (scaled for 44.1kHz)
const (
	numCombs     = 8
	numAllpasses = 4
	muted        = 0.0
	fixedGain    = 0.015
	scaleDamping = 0.4
	scaleRoom    = 0.28
	offsetRoom   = 0.7
	initialRoom  = 0.5
	initialDamp  = 0.5
	initialWet   = 1.0 / 3.0
	initialDry   = 0.0
	initialWidth = 1.0
	stereoSpread = 23

	// Freeze mode parameters
	freezeRoom = 1.0
	freezeDamp = 0.0
)

// Comb filter tuning values (in samples at 44.1kHz)
var combTuning = [numCombs]int{
	1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617,
}

// Allpass filter tuning values (in samples at 44.1kHz)
var allpassTuning = [numAllpasses]int{
	556, 441, 341, 225,
}

// Freeverb implements the Freeverb reverb algorithm by Jezar at Dreampoint
type Freeverb struct {
	// Comb filters for left and right channels
	combL [numCombs]*CombFilter
	combR [numCombs]*CombFilter

	// Allpass filters for left and right channels
	allpassL [numAllpasses]*AllPassFilter
	allpassR [numAllpasses]*AllPassFilter

	// Parameters
	gain       float64
	roomSize   float64
	damping    float64
	wetLevel   float64
	dryLevel   float64
	width      float64
	mode       float64 // freeze mode
	sampleRate float64

	// Cached values
	wet1  float64
	wet2  float64
	dry   float64
	damp1 float64
	damp2 float64
}

// NewFreeverb creates a new Freeverb reverb instance
func NewFreeverb(sampleRate float64) *Freeverb {
	f := &Freeverb{
		gain:       fixedGain,
		roomSize:   initialRoom,
		damping:    initialDamp,
		wetLevel:   initialWet,
		dryLevel:   initialDry,
		width:      initialWidth,
		mode:       0.0,
		sampleRate: sampleRate,
	}

	// Scale factor for different sample rates
	scaleFactor := sampleRate / 44100.0

	// Create comb filters with scaled delay times
	for i := 0; i < numCombs; i++ {
		delaySamplesL := int(float64(combTuning[i]) * scaleFactor)
		delaySamplesR := int(float64(combTuning[i]+stereoSpread) * scaleFactor)

		f.combL[i] = NewCombFilter(delaySamplesL)
		f.combR[i] = NewCombFilter(delaySamplesR)
	}

	// Create allpass filters with scaled delay times
	for i := 0; i < numAllpasses; i++ {
		delaySamplesL := int(float64(allpassTuning[i]) * scaleFactor)
		delaySamplesR := int(float64(allpassTuning[i]+stereoSpread) * scaleFactor)

		f.allpassL[i] = NewAllPassFilter(delaySamplesL)
		f.allpassR[i] = NewAllPassFilter(delaySamplesR)

		// Allpass filters use fixed feedback
		f.allpassL[i].SetFeedback(0.5)
		f.allpassR[i].SetFeedback(0.5)
	}

	// Initialize internal parameters
	f.update()

	return f
}

// SetRoomSize sets the room size (0-1)
func (f *Freeverb) SetRoomSize(size float64) {
	f.roomSize = math.Max(0.0, math.Min(1.0, size))
	f.update()
}

// SetDamping sets the damping amount (0-1)
func (f *Freeverb) SetDamping(damping float64) {
	f.damping = math.Max(0.0, math.Min(1.0, damping))
	f.update()
}

// SetWetLevel sets the wet signal level (0-1)
func (f *Freeverb) SetWetLevel(level float64) {
	f.wetLevel = math.Max(0.0, math.Min(1.0, level))
	f.update()
}

// SetDryLevel sets the dry signal level (0-1)
func (f *Freeverb) SetDryLevel(level float64) {
	f.dryLevel = math.Max(0.0, math.Min(1.0, level))
	f.update()
}

// SetWidth sets the stereo width (0-1)
func (f *Freeverb) SetWidth(width float64) {
	f.width = math.Max(0.0, math.Min(1.0, width))
	f.update()
}

// SetMode sets the freeze mode (0=normal, 1=frozen)
func (f *Freeverb) SetMode(mode float64) {
	f.mode = math.Max(0.0, math.Min(1.0, mode))
	f.update()
}

// update recalculates internal values after parameter changes
func (f *Freeverb) update() {
	// Calculate wet signal levels based on width
	f.wet1 = f.wetLevel * (f.width/2.0 + 0.5)
	f.wet2 = f.wetLevel * ((1.0 - f.width) / 2.0)

	// Set dry level
	f.dry = f.dryLevel

	// Apply freeze mode if active
	var roomSize, damping float64
	if f.mode >= 0.5 {
		roomSize = freezeRoom
		damping = freezeDamp
	} else {
		roomSize = f.roomSize
		damping = f.damping
	}

	// Calculate feedback and damping values
	feedback := roomSize*scaleRoom + offsetRoom
	f.damp1 = damping * scaleDamping
	f.damp2 = 1.0 - f.damp1

	// Update comb filters
	for i := 0; i < numCombs; i++ {
		f.combL[i].SetFeedback(feedback)
		f.combR[i].SetFeedback(feedback)
		f.combL[i].SetDamping(damping)
		f.combR[i].SetDamping(damping)
	}
}

// ProcessStereo processes stereo input through the reverb
func (f *Freeverb) ProcessStereo(inputL, inputR float32) (outputL, outputR float32) {
	// Mix input to mono for reverb processing
	input := (inputL + inputR) * float32(f.gain)

	// Initialize output accumulators
	var outL, outR float32

	// Process through parallel comb filters
	for i := 0; i < numCombs; i++ {
		outL += f.combL[i].Process(input)
		outR += f.combR[i].Process(input)
	}

	// Process through series allpass filters
	for i := 0; i < numAllpasses; i++ {
		outL = f.allpassL[i].Process(outL)
		outR = f.allpassR[i].Process(outR)
	}

	// Apply wet/dry mix and width
	outputL = outL*float32(f.wet1) + outR*float32(f.wet2) + inputL*float32(f.dry)
	outputR = outR*float32(f.wet1) + outL*float32(f.wet2) + inputR*float32(f.dry)

	return outputL, outputR
}

// Process processes a mono input sample
func (f *Freeverb) Process(input float32) float32 {
	// For mono processing, use left channel only
	outputL, _ := f.ProcessStereo(input, input)
	return outputL
}

// Reset clears all internal state
func (f *Freeverb) Reset() {
	// Reset all comb filters
	for i := 0; i < numCombs; i++ {
		f.combL[i].Reset()
		f.combR[i].Reset()
	}

	// Reset all allpass filters
	for i := 0; i < numAllpasses; i++ {
		f.allpassL[i].Reset()
		f.allpassR[i].Reset()
	}
}

// Preset convenience methods

// SetPresetSmallRoom configures the reverb for a small room sound
func (f *Freeverb) SetPresetSmallRoom() {
	f.SetRoomSize(0.3)
	f.SetDamping(0.75)
	f.SetWetLevel(0.25)
	f.SetDryLevel(0.75)
	f.SetWidth(0.5)
}

// SetPresetMediumHall configures the reverb for a medium hall sound
func (f *Freeverb) SetPresetMediumHall() {
	f.SetRoomSize(0.6)
	f.SetDamping(0.5)
	f.SetWetLevel(0.35)
	f.SetDryLevel(0.65)
	f.SetWidth(0.75)
}

// SetPresetLargeHall configures the reverb for a large hall sound
func (f *Freeverb) SetPresetLargeHall() {
	f.SetRoomSize(0.85)
	f.SetDamping(0.3)
	f.SetWetLevel(0.4)
	f.SetDryLevel(0.6)
	f.SetWidth(1.0)
}

// SetPresetCathedral configures the reverb for a cathedral sound
func (f *Freeverb) SetPresetCathedral() {
	f.SetRoomSize(0.95)
	f.SetDamping(0.1)
	f.SetWetLevel(0.5)
	f.SetDryLevel(0.5)
	f.SetWidth(1.0)
}
