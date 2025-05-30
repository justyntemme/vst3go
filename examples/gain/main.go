package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"math"
	
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// GainPlugin implements the Plugin interface
type GainPlugin struct{}

func (g *GainPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.gain",
		Name:     "Simple Gain",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx",
	}
}

func (g *GainPlugin) CreateProcessor() vst3plugin.Processor {
	return NewGainProcessor()
}

// GainProcessor handles the audio processing
type GainProcessor struct {
	params *param.Registry
	buses  *bus.Configuration
}

func NewGainProcessor() *GainProcessor {
	p := &GainProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}
	
	// Add parameters
	p.params.Add(
		param.New(vst3plugin.ParamIDGain, "Gain").
			Range(-24, 24).
			Default(0).
			Unit("dB").
			Build(),
	)
	
	return p
}

func (p *GainProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	// Nothing to initialize for simple gain
	return nil
}

func (p *GainProcessor) ProcessAudio(ctx *process.Context) {
	// Get gain in dB
	gainDB := ctx.ParamPlain(vst3plugin.ParamIDGain)
	
	// Convert to linear
	gain := float32(math.Pow(10.0, gainDB/20.0))
	
	// Process each channel
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	
	for ch := 0; ch < numChannels; ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]
		
		// Apply gain
		for i := range input {
			output[i] = input[i] * gain
		}
	}
}

func (p *GainProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *GainProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *GainProcessor) SetActive(active bool) error {
	return nil
}

func (p *GainProcessor) GetLatencySamples() int32 {
	return 0
}

func (p *GainProcessor) GetTailSamples() int32 {
	return 0
}

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})
	
	// Register our plugin
	vst3plugin.Register(&GainPlugin{})
}

// Required for c-shared build mode
func main() {}