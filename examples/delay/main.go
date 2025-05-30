package main

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"unsafe"
	
	"github.com/justyntemme/vst3go/pkg/plugin"
	"github.com/justyntemme/vst3go/pkg/vst3"
)

type DelayPlugin struct{}

func (d *DelayPlugin) GetPluginName() string {
	return "Simple Delay"
}

func (d *DelayPlugin) GetVendorName() string {
	return "VST3Go"
}

func (d *DelayPlugin) GetPluginVersion() string {
	return "1.0.0"
}

func (d *DelayPlugin) GetUID() [16]byte {
	// Unique ID for this plugin
	return [16]byte{
		0x22, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0,
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x99,
	}
}

func (d *DelayPlugin) CreateComponent() plugin.Component {
	return NewDelayComponent()
}

// DelayComponent implements the actual delay processing
type DelayComponent struct {
	plugin.BaseComponent
	delayBuffer [][]float32
	bufferSize  int
	writePos    int
	sampleRate  float64
}

// NewDelayComponent creates a new delay component
func NewDelayComponent() *DelayComponent {
	d := &DelayComponent{
		BaseComponent: *plugin.NewBaseComponent(),
		bufferSize:    48000, // 1 second at 48kHz
		writePos:      0,
	}
	
	// Initialize with delay parameters
	delayTimeParam := plugin.NewParameter(vst3.ParameterInfo{
		ID:           0,
		Title:        "Delay Time",
		ShortTitle:   "Delay",
		Units:        "ms",
		StepCount:    0,
		DefaultValue: 0.25, // 250ms in normalized form (0-1 = 0-1000ms)
		UnitID:       0,
		Flags:        vst3.ParameterCanAutomate,
	})
	d.Params.AddParameter(delayTimeParam)
	
	feedbackParam := plugin.NewParameter(vst3.ParameterInfo{
		ID:           1,
		Title:        "Feedback",
		ShortTitle:   "FB",
		Units:        "%",
		StepCount:    0,
		DefaultValue: 0.3, // 30% feedback
		UnitID:       0,
		Flags:        vst3.ParameterCanAutomate,
	})
	d.Params.AddParameter(feedbackParam)
	
	mixParam := plugin.NewParameter(vst3.ParameterInfo{
		ID:           2,
		Title:        "Mix",
		ShortTitle:   "Mix",
		Units:        "%",
		StepCount:    0,
		DefaultValue: 0.5, // 50% wet
		UnitID:       0,
		Flags:        vst3.ParameterCanAutomate,
	})
	d.Params.AddParameter(mixParam)
	
	return d
}

// Override SetupProcessing to initialize delay buffer
func (d *DelayComponent) SetupProcessing(setup *vst3.ProcessSetup) error {
	d.sampleRate = setup.SampleRate
	d.bufferSize = int(setup.SampleRate) // 1 second max delay
	
	// Allocate delay buffers for 2 channels
	d.delayBuffer = make([][]float32, 2)
	for i := range d.delayBuffer {
		d.delayBuffer[i] = make([]float32, d.bufferSize)
	}
	d.writePos = 0
	
	return d.BaseComponent.SetupProcessing(setup)
}

// Override Process method for delay processing
func (d *DelayComponent) Process(data unsafe.Pointer) error {
	// Get process data wrapper for safe buffer access
	wrapper := vst3.NewProcessDataWrapper(data)
	if wrapper == nil {
		return vst3.ErrNotImplemented
	}
	
	// Get parameter values
	delayTimeNorm := d.GetParamNormalized(0)
	feedback := float32(d.GetParamNormalized(1))
	mix := float32(d.GetParamNormalized(2))
	
	// Convert delay time to samples
	delayTimeMs := delayTimeNorm * 1000.0 // 0-1 -> 0-1000ms
	delaySamples := int(delayTimeMs * d.sampleRate / 1000.0)
	if delaySamples >= d.bufferSize {
		delaySamples = d.bufferSize - 1
	}
	
	// Process audio
	input := wrapper.GetInput(0)
	output := wrapper.GetOutput(0)
	
	if input != nil && output != nil {
		numSamples := int(wrapper.NumSamples())
		
		// Process each channel
		for ch := 0; ch < input.NumChannels() && ch < output.NumChannels() && ch < 2; ch++ {
			inChan := input.GetChannel(ch)
			outChan := output.GetChannel(ch)
			
			if inChan == nil || outChan == nil {
				continue
			}
			
			// Process each sample
			for i := 0; i < numSamples; i++ {
				// Calculate read position
				readPos := d.writePos - delaySamples
				if readPos < 0 {
					readPos += d.bufferSize
				}
				
				// Read from delay buffer
				delayed := d.delayBuffer[ch][readPos]
				
				// Mix dry and wet signals
				dry := inChan[i]
				wet := delayed
				outChan[i] = dry*(1.0-mix) + wet*mix
				
				// Write to delay buffer with feedback
				d.delayBuffer[ch][d.writePos] = dry + delayed*feedback
				
				// Increment write position
				d.writePos++
				if d.writePos >= d.bufferSize {
					d.writePos = 0
				}
			}
		}
	}
	
	return nil
}

// Parameter value conversion helpers
func (d *DelayComponent) NormalizedParamToPlain(id uint32, normalized float64) float64 {
	switch id {
	case 0: // Delay time
		return normalized * 1000.0 // 0-1 -> 0-1000ms
	case 1, 2: // Feedback, Mix
		return normalized * 100.0 // 0-1 -> 0-100%
	}
	return d.BaseComponent.NormalizedParamToPlain(id, normalized)
}

func (d *DelayComponent) PlainParamToNormalized(id uint32, plain float64) float64 {
	switch id {
	case 0: // Delay time
		return plain / 1000.0 // 0-1000ms -> 0-1
	case 1, 2: // Feedback, Mix
		return plain / 100.0 // 0-100% -> 0-1
	}
	return d.BaseComponent.PlainParamToNormalized(id, plain)
}

func init() {
	// Set factory info
	plugin.SetFactoryInfo(plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})
	
	// Register our plugin
	plugin.RegisterPlugin(&DelayPlugin{})
}

// Required for c-shared build mode
func main() {}