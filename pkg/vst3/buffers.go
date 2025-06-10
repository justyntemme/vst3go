package vst3

// #include "../../include/vst3/vst3_c_api.h"
//
// static inline Steinberg_Vst_Sample32** getChannelBuffers32(struct Steinberg_Vst_AudioBusBuffers* buffers) {
//     return buffers->Steinberg_Vst_AudioBusBuffers_channelBuffers32;
// }
import "C"
import (
	"unsafe"
)

// AudioBuffer provides safe access to VST3 audio buffers
type AudioBuffer struct {
	channelBuffers [][]float32
	numSamples     int32
}

// NewAudioBuffer creates a new audio buffer from C audio bus buffers
func NewAudioBuffer(cBuffers *C.struct_Steinberg_Vst_AudioBusBuffers, numSamples int32) *AudioBuffer {
	if cBuffers == nil {
		return nil
	}

	numChannels := int(cBuffers.numChannels)
	if numChannels == 0 {
		return nil
	}

	// Create Go slices for each channel
	channelBuffers := make([][]float32, numChannels)

	// Access the channel buffer pointers through the union using helper function
	channelBuffers32 := C.getChannelBuffers32(cBuffers)

	if channelBuffers32 != nil {
		// Convert the double pointer to a slice of pointers
		channelPtrs := (*[MaxArraySize]*C.Steinberg_Vst_Sample32)(unsafe.Pointer(channelBuffers32))[:numChannels:numChannels]

		for i := 0; i < numChannels; i++ {
			if channelPtrs[i] != nil {
				// Create a Go slice from the C buffer without copying
				channelBuffers[i] = (*[MaxArraySize]float32)(unsafe.Pointer(channelPtrs[i]))[:numSamples:numSamples]
			}
		}
	}

	return &AudioBuffer{
		channelBuffers: channelBuffers,
		numSamples:     numSamples,
	}
}

// GetChannel returns a specific channel's buffer
func (b *AudioBuffer) GetChannel(index int) []float32 {
	if index < 0 || index >= len(b.channelBuffers) {
		return nil
	}
	return b.channelBuffers[index]
}

// NumChannels returns the number of channels
func (b *AudioBuffer) NumChannels() int {
	return len(b.channelBuffers)
}

// NumSamples returns the number of samples per channel
func (b *AudioBuffer) NumSamples() int32 {
	return b.numSamples
}

// ProcessContextFlags defines which fields in ProcessContext are valid
type ProcessContextFlags uint32

const (
	ProcessContextFlagPlaying          ProcessContextFlags = 1 << 1
	ProcessContextFlagCycleActive      ProcessContextFlags = 1 << 2
	ProcessContextFlagRecording        ProcessContextFlags = 1 << 3
	ProcessContextFlagSystemTimeValid  ProcessContextFlags = 1 << 8
	ProcessContextFlagContTimeValid    ProcessContextFlags = 1 << 17
	ProcessContextFlagProjectTimeValid ProcessContextFlags = 1 << 9
	ProcessContextFlagBarPositionValid ProcessContextFlags = 1 << 11
	ProcessContextFlagCycleValid       ProcessContextFlags = 1 << 12
	ProcessContextFlagTempoValid       ProcessContextFlags = 1 << 10
	ProcessContextFlagTimeSigValid     ProcessContextFlags = 1 << 13
	ProcessContextFlagChordValid       ProcessContextFlags = 1 << 18
	ProcessContextFlagSmpteValid       ProcessContextFlags = 1 << 14
	ProcessContextFlagClockValid       ProcessContextFlags = 1 << 15
)

// ProcessContext wraps the VST3 process context with all transport and timing information
type ProcessContext struct {
	State                 ProcessContextFlags // Flags indicating which fields are valid
	SampleRate            float64             // Current sample rate
	ProjectTimeSamples    int64               // Project time in samples
	SystemTime            int64               // System time in nanoseconds
	ContinuousTimeSamples int64               // Continuous time in samples
	ProjectTimeMusic      float64             // Musical position in quarter notes
	BarPositionMusic      float64             // Bar position in quarter notes
	CycleStartMusic       float64             // Cycle/loop start in quarter notes
	CycleEndMusic         float64             // Cycle/loop end in quarter notes
	Tempo                 float64             // Current tempo in BPM
	TimeSigNumerator      int32               // Time signature numerator (e.g., 4 for 4/4)
	TimeSigDenominator    int32               // Time signature denominator (e.g., 4 for 4/4)
	SamplesToNextClock    int32               // Samples to next clock/beat
}

// NewProcessContext creates a process context from C struct
func NewProcessContext(ctx *C.struct_Steinberg_Vst_ProcessContext) *ProcessContext {
	if ctx == nil {
		return nil
	}

	return &ProcessContext{
		State:                 ProcessContextFlags(ctx.state),
		SampleRate:            float64(ctx.sampleRate),
		ProjectTimeSamples:    int64(ctx.projectTimeSamples),
		SystemTime:            int64(ctx.systemTime),
		ContinuousTimeSamples: int64(ctx.continousTimeSamples),
		ProjectTimeMusic:      float64(ctx.projectTimeMusic),
		BarPositionMusic:      float64(ctx.barPositionMusic),
		CycleStartMusic:       float64(ctx.cycleStartMusic),
		CycleEndMusic:         float64(ctx.cycleEndMusic),
		Tempo:                 float64(ctx.tempo),
		TimeSigNumerator:      int32(ctx.timeSigNumerator),
		TimeSigDenominator:    int32(ctx.timeSigDenominator),
		SamplesToNextClock:    int32(ctx.samplesToNextClock),
	}
}

// IsPlaying returns true if transport is playing
func (c *ProcessContext) IsPlaying() bool {
	return c.State&ProcessContextFlagPlaying != 0
}

// IsRecording returns true if transport is recording
func (c *ProcessContext) IsRecording() bool {
	return c.State&ProcessContextFlagRecording != 0
}

// IsCycleActive returns true if loop/cycle is active
func (c *ProcessContext) IsCycleActive() bool {
	return c.State&ProcessContextFlagCycleActive != 0
}

// HasTempo returns true if tempo information is valid
func (c *ProcessContext) HasTempo() bool {
	return c.State&ProcessContextFlagTempoValid != 0
}

// HasTimeSignature returns true if time signature information is valid
func (c *ProcessContext) HasTimeSignature() bool {
	return c.State&ProcessContextFlagTimeSigValid != 0
}

// HasBarPosition returns true if bar position information is valid
func (c *ProcessContext) HasBarPosition() bool {
	return c.State&ProcessContextFlagBarPositionValid != 0
}

// HasProjectTimeMusic returns true if musical time position is valid
func (c *ProcessContext) HasProjectTimeMusic() bool {
	return c.State&ProcessContextFlagProjectTimeValid != 0
}

// ProcessDataWrapper provides safe access to VST3 process data
type ProcessDataWrapper struct {
	ptr           *C.struct_Steinberg_Vst_ProcessData
	inputBuffers  []*AudioBuffer
	outputBuffers []*AudioBuffer
	context       *ProcessContext
}

// NewProcessDataWrapper creates a wrapper from C process data
func NewProcessDataWrapper(dataPtr unsafe.Pointer) *ProcessDataWrapper {
	if dataPtr == nil {
		return nil
	}

	data := (*C.struct_Steinberg_Vst_ProcessData)(dataPtr)
	wrapper := &ProcessDataWrapper{
		ptr: data,
	}

	// Wrap input buffers
	if data.numInputs > 0 && data.inputs != nil {
		numInputs := int(data.numInputs)
		wrapper.inputBuffers = make([]*AudioBuffer, numInputs)
		inputPtrs := (*[MaxArraySize]C.struct_Steinberg_Vst_AudioBusBuffers)(unsafe.Pointer(data.inputs))[:numInputs:numInputs]

		for i := 0; i < numInputs; i++ {
			wrapper.inputBuffers[i] = NewAudioBuffer(&inputPtrs[i], int32(data.numSamples))
		}
	}

	// Wrap output buffers
	if data.numOutputs > 0 && data.outputs != nil {
		numOutputs := int(data.numOutputs)
		wrapper.outputBuffers = make([]*AudioBuffer, numOutputs)
		outputPtrs := (*[MaxArraySize]C.struct_Steinberg_Vst_AudioBusBuffers)(unsafe.Pointer(data.outputs))[:numOutputs:numOutputs]

		for i := 0; i < numOutputs; i++ {
			wrapper.outputBuffers[i] = NewAudioBuffer(&outputPtrs[i], int32(data.numSamples))
		}
	}

	// Wrap process context
	if data.processContext != nil {
		wrapper.context = NewProcessContext(data.processContext)
	}

	return wrapper
}

// GetInput returns an input buffer by index
func (w *ProcessDataWrapper) GetInput(index int) *AudioBuffer {
	if index < 0 || index >= len(w.inputBuffers) {
		return nil
	}
	return w.inputBuffers[index]
}

// GetOutput returns an output buffer by index
func (w *ProcessDataWrapper) GetOutput(index int) *AudioBuffer {
	if index < 0 || index >= len(w.outputBuffers) {
		return nil
	}
	return w.outputBuffers[index]
}

// NumSamples returns the number of samples to process
func (w *ProcessDataWrapper) NumSamples() int32 {
	return int32(w.ptr.numSamples)
}

// GetContext returns the process context
func (w *ProcessDataWrapper) GetContext() *ProcessContext {
	return w.context
}
