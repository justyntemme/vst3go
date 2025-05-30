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
		channelPtrs := (*[1 << 30]*C.Steinberg_Vst_Sample32)(unsafe.Pointer(channelBuffers32))[:numChannels:numChannels]

		for i := 0; i < numChannels; i++ {
			if channelPtrs[i] != nil {
				// Create a Go slice from the C buffer without copying
				channelBuffers[i] = (*[1 << 30]float32)(unsafe.Pointer(channelPtrs[i]))[:numSamples:numSamples]
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

// ProcessContext wraps the VST3 process context
type ProcessContext struct {
	State           uint32
	SampleRate      float64
	ProjectTimeSecs float64
	BarPositionPPQ  float64
	Tempo           float64
}

// NewProcessContext creates a process context from C struct
func NewProcessContext(ctx *C.struct_Steinberg_Vst_ProcessContext) *ProcessContext {
	if ctx == nil {
		return nil
	}

	return &ProcessContext{
		State:           uint32(ctx.state),
		SampleRate:      float64(ctx.sampleRate),
		ProjectTimeSecs: float64(ctx.projectTimeMusic),
		BarPositionPPQ:  float64(ctx.barPositionMusic),
		Tempo:           float64(ctx.tempo),
	}
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
		inputPtrs := (*[1 << 30]C.struct_Steinberg_Vst_AudioBusBuffers)(unsafe.Pointer(data.inputs))[:numInputs:numInputs]

		for i := 0; i < numInputs; i++ {
			wrapper.inputBuffers[i] = NewAudioBuffer(&inputPtrs[i], int32(data.numSamples))
		}
	}

	// Wrap output buffers
	if data.numOutputs > 0 && data.outputs != nil {
		numOutputs := int(data.numOutputs)
		wrapper.outputBuffers = make([]*AudioBuffer, numOutputs)
		outputPtrs := (*[1 << 30]C.struct_Steinberg_Vst_AudioBusBuffers)(unsafe.Pointer(data.outputs))[:numOutputs:numOutputs]

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
