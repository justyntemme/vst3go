package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"github.com/justyntemme/vst3go/pkg/dsp/filter"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// FilterPlugin implements the Plugin interface
type FilterPlugin struct{}

func (f *FilterPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.filter",
		Name:     "Multi-Mode Filter",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Filter",
	}
}

func (f *FilterPlugin) CreateProcessor() vst3plugin.Processor {
	return NewFilterProcessor()
}

// FilterProcessor handles the audio processing
type FilterProcessor struct {
	params     *param.Registry
	buses      *bus.Configuration
	
	// DSP state
	svFilter   *filter.MultiModeSVF
	sampleRate float64
}

// Parameter IDs
const (
	ParamCutoff    = 0
	ParamResonance = 1
	ParamMode      = 2
	ParamMix       = 3
)

func NewFilterProcessor() *FilterProcessor {
	p := &FilterProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewStereoConfiguration(),
		svFilter:   filter.NewMultiModeSVF(2), // stereo
		sampleRate: 48000,
	}
	
	// Add parameters
	p.params.Add(
		param.New(ParamCutoff, "Cutoff").
			Range(20, 20000).
			Default(1000).
			Unit("Hz").
			Build(),
		
		param.New(ParamResonance, "Resonance").
			Range(0.5, 20).
			Default(1).
			Build(),
		
		param.New(ParamMode, "Mode").
			Range(0, 1).
			Default(0).
			Build(),
		
		param.New(ParamMix, "Mix").
			Range(0, 1).
			Default(1).
			Build(),
	)
	
	return p
}

func (p *FilterProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	p.svFilter.Reset()
	return nil
}

func (p *FilterProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameter values
	cutoff := ctx.ParamPlain(ParamCutoff)
	resonance := ctx.ParamPlain(ParamResonance)
	mode := ctx.Param(ParamMode)
	mix := float32(ctx.Param(ParamMix))
	
	// Update filter parameters
	p.svFilter.SetFrequencyAndQ(p.sampleRate, cutoff, resonance)
	p.svFilter.SetMode(mode)
	
	// Process each channel
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	if numChannels > 2 {
		numChannels = 2 // We only support stereo
	}
	
	numSamples := ctx.NumSamples()
	dryGain := 1.0 - mix
	
	// Use work buffer for processing
	workBuffer := ctx.WorkBuffer()
	
	for ch := 0; ch < numChannels; ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]
		
		// Copy input to work buffer
		copy(workBuffer[:numSamples], input[:numSamples])
		
		// Process through filter
		p.svFilter.Process(workBuffer[:numSamples], ch)
		
		// Mix dry and wet
		for i := 0; i < numSamples; i++ {
			output[i] = input[i]*dryGain + workBuffer[i]*mix
		}
	}
}

func (p *FilterProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *FilterProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *FilterProcessor) SetActive(active bool) error {
	if !active {
		p.svFilter.Reset()
	}
	return nil
}

func (p *FilterProcessor) GetLatencySamples() int32 {
	return 0
}

func (p *FilterProcessor) GetTailSamples() int32 {
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
	vst3plugin.Register(&FilterPlugin{})
}

// Required for c-shared build mode
func main() {}