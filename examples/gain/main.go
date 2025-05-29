package main

// #include "../../include/vst3/vst3_c_api.h"
import "C"
import (
	"math"
	"unsafe"
	
	"github.com/justyntemme/vst3go/pkg/plugin"
	"github.com/justyntemme/vst3go/pkg/vst3"
)

type GainPlugin struct{}

func (g *GainPlugin) GetPluginName() string {
	return "Simple Gain"
}

func (g *GainPlugin) GetVendorName() string {
	return "VST3Go"
}

func (g *GainPlugin) GetPluginVersion() string {
	return "1.0.0"
}

func (g *GainPlugin) GetUID() [16]byte {
	// Unique ID for this plugin
	return [16]byte{
		0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0,
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
	}
}

func (g *GainPlugin) CreateComponent() plugin.Component {
	return NewGainComponent()
}

// GainComponent implements the actual audio processing
type GainComponent struct {
	plugin.BaseComponent
}

// NewGainComponent creates a new gain component
func NewGainComponent() *GainComponent {
	g := &GainComponent{
		BaseComponent: *plugin.NewBaseComponent(),
	}
	
	// Initialize with gain parameter
	gainParam := plugin.NewParameter(vst3.ParameterInfo{
		ID:           plugin.ParamIDGain,
		Title:        "Gain",
		ShortTitle:   "Gain",
		Units:        "dB",
		StepCount:    0,
		DefaultValue: 0.5, // 0dB in normalized form
		UnitID:       0,
		Flags:        vst3.ParameterCanAutomate,
	})
	g.Params.AddParameter(gainParam)
	
	return g
}

// Override Process method for gain processing
func (g *GainComponent) Process(data unsafe.Pointer) error {
	// Get process data wrapper for safe buffer access
	wrapper := vst3.NewProcessDataWrapper(data)
	if wrapper == nil {
		return vst3.ErrNotImplemented
	}
	
	// Get gain value
	gainNorm := g.GetParamNormalized(plugin.ParamIDGain)
	// Convert normalized (0-1) to linear gain (0-2)
	gain := float32(gainNorm * 2.0)
	
	// Simple stereo processing
	input := wrapper.GetInput(0)
	output := wrapper.GetOutput(0)
	
	if input != nil && output != nil {
		// Process each channel
		for ch := 0; ch < input.NumChannels() && ch < output.NumChannels(); ch++ {
			inChan := input.GetChannel(ch)
			outChan := output.GetChannel(ch)
			
			// Check if channels are valid
			if inChan == nil || outChan == nil {
				continue
			}
			
			// Apply gain to each sample
			for i := 0; i < int(wrapper.NumSamples()); i++ {
				outChan[i] = inChan[i] * gain
			}
		}
	}
	
	return nil
}

// Parameter value conversion helpers
func (g *GainComponent) NormalizedParamToPlain(id uint32, normalized float64) float64 {
	if id == plugin.ParamIDGain {
		// Convert 0-1 to -inf to +12dB
		if normalized <= 0 {
			return -96.0 // -inf dB
		}
		// Linear to dB conversion
		linear := normalized * 2.0 // 0-2 range
		return 20.0 * math.Log10(linear)
	}
	return g.BaseComponent.NormalizedParamToPlain(id, normalized)
}

func (g *GainComponent) PlainParamToNormalized(id uint32, plain float64) float64 {
	if id == plugin.ParamIDGain {
		// Convert dB to normalized
		if plain <= -96.0 {
			return 0.0
		}
		// dB to linear conversion
		linear := math.Pow(10.0, plain/20.0)
		return linear / 2.0 // Normalize to 0-1
	}
	return g.BaseComponent.PlainParamToNormalized(id, plain)
}

func init() {
	// Set factory info
	plugin.SetFactoryInfo(plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})
	
	// Register our plugin
	plugin.RegisterPlugin(&GainPlugin{})
}

// Required for c-shared build mode
func main() {}