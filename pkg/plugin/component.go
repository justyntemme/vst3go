package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"unsafe"
	
	"github.com/justyntemme/vst3go/pkg/vst3"
)

// Component represents a VST3 plugin component that implements both
// IComponent and IAudioProcessor interfaces
type Component interface {
	vst3.IComponent
	vst3.IAudioProcessor
	vst3.IEditController
}

// BaseComponent provides a base implementation with sensible defaults
type BaseComponent struct {
	buses      []vst3.BusInfo
	active     bool
	processing bool
	sampleRate float64
	Params     *ParameterManager // Exported for access by derived types
}

// NewBaseComponent creates a new base component
func NewBaseComponent() *BaseComponent {
	b := &BaseComponent{
		Params: NewParameterManager(),
	}
	// Initialize buses
	b.buses = []vst3.BusInfo{
		{
			MediaType:    vst3.MediaTypeAudio,
			Direction:    vst3.BusDirectionInput,
			ChannelCount: 2,
			Name:         "Audio Input",
			BusType:      vst3.BusTypeMain,
			Flags:        1, // default active
		},
		{
			MediaType:    vst3.MediaTypeAudio,
			Direction:    vst3.BusDirectionOutput,
			ChannelCount: 2,
			Name:         "Audio Output",
			BusType:      vst3.BusTypeMain,
			Flags:        1, // default active
		},
	}
	return b
}

// IPluginBase methods
func (b *BaseComponent) Initialize(context interface{}) error {
	// Already initialized in NewBaseComponent
	return nil
}

func (b *BaseComponent) Terminate() error {
	return nil
}

// IComponent methods
func (b *BaseComponent) GetControllerClassID() [16]byte {
	// Return the same class ID as the processor
	return [16]byte{}
}

func (b *BaseComponent) SetIOMode(mode int32) error {
	return nil
}

func (b *BaseComponent) GetBusCount(mediaType, direction int32) int32 {
	count := int32(0)
	for _, bus := range b.buses {
		if bus.MediaType == mediaType && bus.Direction == direction {
			count++
		}
	}
	return count
}

func (b *BaseComponent) GetBusInfo(mediaType, direction, index int32) (*vst3.BusInfo, error) {
	busIndex := int32(0)
	for i := range b.buses {
		if b.buses[i].MediaType == mediaType && b.buses[i].Direction == direction {
			if busIndex == index {
				return &b.buses[i], nil
			}
			busIndex++
		}
	}
	return nil, nil
}

func (b *BaseComponent) GetRoutingInfo(inInfo, outInfo interface{}) error {
	return nil
}

func (b *BaseComponent) ActivateBus(mediaType, direction, index int32, state bool) error {
	return nil
}

func (b *BaseComponent) SetActive(state bool) error {
	b.active = state
	return nil
}

func (b *BaseComponent) SetState(state []byte) error {
	return nil
}

func (b *BaseComponent) GetState() ([]byte, error) {
	return []byte{}, nil
}

// IAudioProcessor methods
func (b *BaseComponent) SetBusArrangements(inputs, outputs []int64) error {
	return nil
}

func (b *BaseComponent) GetBusArrangement(direction, index int32) (int64, error) {
	// Return stereo arrangement by default
	return int64(3), nil // Left + Right
}

func (b *BaseComponent) CanProcessSampleSize(symbolicSampleSize int32) error {
	// Accept 32-bit float
	if symbolicSampleSize == C.Steinberg_Vst_SymbolicSampleSizes_kSample32 {
		return nil
	}
	return vst3.ErrNotImplemented
}

func (b *BaseComponent) GetLatencySamples() uint32 {
	return 0
}

func (b *BaseComponent) SetupProcessing(setup *vst3.ProcessSetup) error {
	b.sampleRate = setup.SampleRate
	return nil
}

func (b *BaseComponent) SetProcessing(state bool) error {
	b.processing = state
	return nil
}

func (b *BaseComponent) Process(data unsafe.Pointer) error {
	// Default implementation - pass through
	return nil
}

func (b *BaseComponent) GetTailSamples() uint32 {
	return 0
}

// IEditController methods
func (b *BaseComponent) SetComponentState(state []byte) error {
	return nil
}

func (b *BaseComponent) GetParameterCount() int32 {
	return b.Params.GetParameterCount()
}

func (b *BaseComponent) GetParameterInfo(index int32) (*vst3.ParameterInfo, error) {
	param := b.Params.GetParameterByIndex(index)
	if param == nil {
		return nil, vst3.ErrNotImplemented
	}
	return &param.Info, nil
}

func (b *BaseComponent) GetParamStringByValue(id uint32, value float64) (string, error) {
	return "", nil
}

func (b *BaseComponent) GetParamValueByString(id uint32, str string) (float64, error) {
	return 0, nil
}

func (b *BaseComponent) NormalizedParamToPlain(id uint32, normalized float64) float64 {
	return normalized
}

func (b *BaseComponent) PlainParamToNormalized(id uint32, plain float64) float64 {
	return plain
}

func (b *BaseComponent) GetParamNormalized(id uint32) float64 {
	return b.Params.GetValue(id)
}

func (b *BaseComponent) SetParamNormalized(id uint32, value float64) error {
	b.Params.SetValue(id, value)
	return nil
}

func (b *BaseComponent) SetComponentHandler(handler interface{}) error {
	return nil
}

func (b *BaseComponent) CreateView(name string) (interface{}, error) {
	return nil, nil
}