package vst3

// #include "../../include/vst3/vst3_c_api.h"
import "C"
import "unsafe"

// Component interface IDs
var (
	IIDIComponent = [16]byte{
		0xE8, 0x31, 0xFF, 0x31, 0xF2, 0xD5, 0x4B, 0x01,
		0x83, 0x6F, 0x5D, 0x38, 0x54, 0x34, 0xAE, 0xC6,
	}
	IIDIAudioProcessor = [16]byte{
		0x42, 0x04, 0x3F, 0x99, 0xB2, 0xA8, 0x4F, 0x3F,
		0xA2, 0x85, 0x7A, 0xA0, 0x39, 0x82, 0x15, 0xC1,
	}
	IIDIEditController = [16]byte{
		0xDD, 0xB1, 0x18, 0x8F, 0x2B, 0x0D, 0x43, 0x11,
		0x9E, 0xD0, 0xAE, 0xB4, 0x38, 0x95, 0x40, 0x52,
	}
)

// IComponent represents the main plugin component interface
type IComponent interface {
	// IPluginBase methods
	Initialize(context interface{}) error
	Terminate() error
	
	// IComponent methods
	GetControllerClassID() [16]byte
	SetIOMode(mode int32) error
	GetBusCount(mediaType, direction int32) int32
	GetBusInfo(mediaType, direction, index int32) (*BusInfo, error)
	GetRoutingInfo(inInfo, outInfo interface{}) error
	ActivateBus(mediaType, direction, index int32, state bool) error
	SetActive(state bool) error
	SetState(state []byte) error
	GetState() ([]byte, error)
}

// IAudioProcessor represents the audio processing interface
type IAudioProcessor interface {
	SetBusArrangements(inputs, outputs []int64) error
	GetBusArrangement(direction, index int32) (int64, error)
	CanProcessSampleSize(symbolicSampleSize int32) error
	GetLatencySamples() uint32
	SetupProcessing(setup *ProcessSetup) error
	SetProcessing(state bool) error
	Process(data unsafe.Pointer) error
	GetTailSamples() uint32
}

// IEditController represents the parameter control interface
type IEditController interface {
	// IPluginBase methods
	Initialize(context interface{}) error
	Terminate() error
	
	// IEditController methods
	SetComponentState(state []byte) error
	SetState(state []byte) error
	GetState() ([]byte, error)
	GetParameterCount() int32
	GetParameterInfo(index int32) (*ParameterInfo, error)
	GetParamStringByValue(id uint32, value float64) (string, error)
	GetParamValueByString(id uint32, str string) (float64, error)
	NormalizedParamToPlain(id uint32, normalized float64) float64
	PlainParamToNormalized(id uint32, plain float64) float64
	GetParamNormalized(id uint32) float64
	SetParamNormalized(id uint32, value float64) error
	SetComponentHandler(handler interface{}) error
	CreateView(name string) (interface{}, error)
}