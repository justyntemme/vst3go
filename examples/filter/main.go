package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"fmt"

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
	params *param.Registry
	buses  *bus.Configuration

	// DSP state
	svFilter   *filter.MultiModeSVF
	sampleRate float64
}

// Parameter IDs
const (
	ParamFilterType = 0
	ParamCutoff     = 1
	ParamResonance  = 2
	ParamMix        = 3
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
		param.New(ParamFilterType, "Filter Type").
			Range(0, 3).
			Default(0).
			Steps(4).
			Formatter(param.FilterTypeFormatter, param.FilterTypeParser).
			Build(),

		param.New(ParamCutoff, "Cutoff").
			Range(20, 20000).
			Default(1000).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),

		param.New(ParamResonance, "Resonance").
			Range(0.5, 20).
			Default(1).
			Formatter(func(v float64) string {
				return fmt.Sprintf("Q: %.2f", v)
			}, nil).
			Build(),

		param.New(ParamMix, "Mix").
			Range(0, 100).
			Default(100).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
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
	filterType := int(ctx.Param(ParamFilterType))
	cutoff := ctx.ParamPlain(ParamCutoff)
	resonance := ctx.ParamPlain(ParamResonance)
	mix := float32(ctx.Param(ParamMix) / 100.0) // Convert from percentage

	// Update filter parameters
	p.svFilter.SetFrequencyAndQ(p.sampleRate, cutoff, resonance)

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

		// Process through filter based on type
		switch filterType {
		case param.FilterTypeLowpass:
			p.svFilter.ProcessLowpass(workBuffer[:numSamples], ch)
		case param.FilterTypeHighpass:
			p.svFilter.ProcessHighpass(workBuffer[:numSamples], ch)
		case param.FilterTypeBandpass:
			p.svFilter.ProcessBandpass(workBuffer[:numSamples], ch)
		case param.FilterTypeNotch:
			p.svFilter.ProcessNotch(workBuffer[:numSamples], ch)
		}

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
