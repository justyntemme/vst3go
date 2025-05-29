package vst3

// #include "../../include/vst3/vst3_c_api.h"
import "C"

// ProcessData wraps the VST3 process data
type ProcessData struct {
	ptr *C.struct_Steinberg_Vst_ProcessData
}

// AudioBusBuffers provides access to audio buffers
type AudioBusBuffers struct {
	NumChannels int32
	Buffers     [][]float32
}

// ProcessSetup contains audio processing configuration
type ProcessSetup struct {
	ProcessMode     int32
	SymbolicSampleSize int32
	MaxSamplesPerBlock int32
	SampleRate      float64
}

// ParameterInfo describes a parameter
type ParameterInfo struct {
	ID            uint32
	Title         string
	ShortTitle    string
	Units         string
	StepCount     int32
	DefaultValue  float64
	UnitID        int32
	Flags         int32
}

// BusInfo describes an audio bus
type BusInfo struct {
	MediaType     int32
	Direction     int32
	ChannelCount  int32
	Name          string
	BusType       int32
	Flags         uint32
}

// Constants for media types
const (
	MediaTypeAudio = C.Steinberg_Vst_MediaTypes_kAudio
	MediaTypeEvent = C.Steinberg_Vst_MediaTypes_kEvent
)

// Constants for bus directions
const (
	BusDirectionInput  = C.Steinberg_Vst_BusDirections_kInput
	BusDirectionOutput = C.Steinberg_Vst_BusDirections_kOutput
)

// Constants for bus types
const (
	BusTypeMain = C.Steinberg_Vst_BusTypes_kMain
	BusTypeAux  = C.Steinberg_Vst_BusTypes_kAux
)

// Constants for parameter flags
const (
	ParameterIsReadOnly = C.Steinberg_Vst_ParameterInfo_ParameterFlags_kIsReadOnly
	ParameterIsWrapAround = C.Steinberg_Vst_ParameterInfo_ParameterFlags_kIsWrapAround
	ParameterIsList = C.Steinberg_Vst_ParameterInfo_ParameterFlags_kIsList
	ParameterIsHidden = C.Steinberg_Vst_ParameterInfo_ParameterFlags_kIsHidden
	ParameterCanAutomate = C.Steinberg_Vst_ParameterInfo_ParameterFlags_kCanAutomate
	ParameterIsBypass = C.Steinberg_Vst_ParameterInfo_ParameterFlags_kIsBypass
)