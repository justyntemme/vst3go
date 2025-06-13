// Package dsp provides digital signal processing utilities and algorithms.
package dsp

// Common audio constants used throughout the DSP package and plugins.
const (
	// Gain/Level constants
	MinDB     = -200.0 // Minimum dB value (effectively silence)
	UnityGain = 1.0    // Unity gain (0 dB)
	
	// Common parameter ranges for dynamics processors
	DefaultMinThresholdDB = -60.0
	DefaultMaxThresholdDB = 0.0
	DefaultMinRatio       = 1.0
	DefaultMaxRatio       = 20.0
	
	// Attack/Release time ranges (in seconds)
	DefaultMinAttack  = 0.0001  // 0.1ms
	DefaultMaxAttack  = 1.0     // 1s
	DefaultMinRelease = 0.001   // 1ms
	DefaultMaxRelease = 5.0     // 5s
	
	// Frequency ranges
	MinFrequency      = 20.0     // 20 Hz
	MaxFrequency      = 20000.0  // 20 kHz
	DefaultLowFreq    = 100.0    // Common low shelf frequency
	DefaultMidFreq    = 1000.0   // Common mid frequency
	DefaultHighFreq   = 10000.0  // Common high shelf frequency
	
	// Q factor ranges
	MinQ     = 0.1
	MaxQ     = 20.0
	DefaultQ = 0.707 // Butterworth response
	
	// Channel counts
	Mono   = 1
	Stereo = 2
	
	// Common sample rates
	SampleRate32k  = 32000.0
	SampleRate44k1 = 44100.0
	SampleRate48k  = 48000.0
	SampleRate88k2 = 88200.0
	SampleRate96k  = 96000.0
	SampleRate192k = 192000.0
	
	// Buffer sizes
	MinBufferSize     = 32
	DefaultBufferSize = 512
	MaxBufferSize     = 8192
	
	// Smoothing times
	FastSmoothing   = 0.001  // 1ms
	MediumSmoothing = 0.010  // 10ms
	SlowSmoothing   = 0.050  // 50ms
	
	// Common mix ranges
	MinMix = 0.0   // Dry
	MaxMix = 1.0   // Wet
	HalfMix = 0.5  // 50/50
	
	// Modulation ranges
	DefaultMinDepth = 0.0
	DefaultMaxDepth = 1.0
	DefaultMinRate  = 0.1   // Hz
	DefaultMaxRate  = 20.0  // Hz
	
	// Phase constants
	TwoPi   = 6.283185307179586
	Pi      = 3.141592653589793
	HalfPi  = 1.5707963267948966
	
	// Conversion factors
	DegreesToRadians = Pi / 180.0
	RadiansToDegrees = 180.0 / Pi
	
	// Small values for comparisons
	Epsilon      = 1e-6
	SmallFloat32 = 1e-30
	
	// Clipping thresholds
	ClipThreshold    = 0.999
	SoftClipThreshold = 0.95
)

// Common preset values for different processor types
const (
	// Gate presets
	GateMinThreshold = -80.0
	GateMaxThreshold = 0.0
	GateMinRange     = -80.0
	GateMaxRange     = 0.0
	
	// Compressor presets
	CompMinThreshold = -60.0
	CompMaxThreshold = 0.0
	CompMinRatio     = 1.0
	CompMaxRatio     = 20.0
	CompMinKnee      = 0.0
	CompMaxKnee      = 10.0
	
	// Limiter presets
	LimiterMinCeiling = -30.0
	LimiterMaxCeiling = 0.0
	LimiterMinRelease = 0.001
	LimiterMaxRelease = 1.0
	
	// EQ presets
	EQMinGain = -24.0
	EQMaxGain = 24.0
	EQLowShelfFreq  = 100.0
	EQHighShelfFreq = 10000.0
	EQPeakingFreq   = 1000.0
	
	// Reverb presets
	ReverbMinDecay = 0.1
	ReverbMaxDecay = 30.0
	ReverbMinSize  = 0.0
	ReverbMaxSize  = 1.0
	ReverbMinDamp  = 0.0
	ReverbMaxDamp  = 1.0
	
	// Delay presets
	DelayMinTime = 0.001    // 1ms
	DelayMaxTime = 5.0      // 5s
	DelayMinFeedback = 0.0
	DelayMaxFeedback = 0.99
)

// ProcessorType identifies the type of audio processor
type ProcessorType int

const (
	ProcessorTypeUnknown ProcessorType = iota
	ProcessorTypeGain
	ProcessorTypeFilter
	ProcessorTypeDynamics
	ProcessorTypeModulation
	ProcessorTypeDelay
	ProcessorTypeReverb
	ProcessorTypeDistortion
	ProcessorTypeAnalysis
	ProcessorTypeUtility
)

// String returns the string representation of a ProcessorType
func (pt ProcessorType) String() string {
	switch pt {
	case ProcessorTypeGain:
		return "Gain"
	case ProcessorTypeFilter:
		return "Filter"
	case ProcessorTypeDynamics:
		return "Dynamics"
	case ProcessorTypeModulation:
		return "Modulation"
	case ProcessorTypeDelay:
		return "Delay"
	case ProcessorTypeReverb:
		return "Reverb"
	case ProcessorTypeDistortion:
		return "Distortion"
	case ProcessorTypeAnalysis:
		return "Analysis"
	case ProcessorTypeUtility:
		return "Utility"
	default:
		return "Unknown"
	}
}