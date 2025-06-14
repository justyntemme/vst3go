package main

import (
	"fmt"
	"log"
	"os"

	"github.com/justyntemme/vst3go/pkg/dsp"
	"github.com/justyntemme/vst3go/pkg/dsp/filter"
	"github.com/justyntemme/vst3go/pkg/dsp/mix"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
	
	// Import C bridge - required for VST3 plugin to work
	_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
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

	// Debug logging
	debugLogger *log.Logger
}

// Parameter IDs
const (
	ParamFilterType = 0
	ParamCutoff     = 1
	ParamResonance  = 2
	ParamMix        = 3
)

func NewFilterProcessor() *FilterProcessor {
	// Create debug logger
	debugFile, err := os.OpenFile("/tmp/vst3go_filter_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		debugFile = os.Stderr
	}

	logger := log.New(debugFile, "[FILTER] ", log.LstdFlags|log.Lmicroseconds)
	logger.Println("=== Creating new FilterProcessor ===")

	p := &FilterProcessor{
		params:      param.NewRegistry(),
		buses:       bus.NewStereoConfiguration(),
		svFilter:    filter.NewMultiModeSVF(2), // stereo
		sampleRate:  48000,
		debugLogger: logger,
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
			Range(dsp.DefaultLowFreq, 8000).
			Default(800).
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),

		param.New(ParamResonance, "Resonance").
			Range(0.5, dsp.MaxQ/2).
			Default(dsp.DefaultQ).
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

	logger.Printf("Parameters added: FilterType=%v, Cutoff=%v, Resonance=%v, Mix=%v",
		p.params.Get(ParamFilterType), p.params.Get(ParamCutoff),
		p.params.Get(ParamResonance), p.params.Get(ParamMix))

	return p
}

func (p *FilterProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.debugLogger.Printf("Initialize called: sampleRate=%.1f, maxBlockSize=%d", sampleRate, maxBlockSize)
	p.sampleRate = sampleRate
	p.svFilter.Reset()
	p.debugLogger.Printf("Filter initialized successfully")
	return nil
}

func (p *FilterProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameter values
	filterType := ctx.ParamPlain(ParamFilterType)
	cutoff := ctx.ParamPlain(ParamCutoff)
	resonance := ctx.ParamPlain(ParamResonance)
	mixAmount := float32(ctx.ParamPlain(ParamMix) / 100.0) // Convert percentage to 0-1

	// Check if we have valid input
	if ctx.GetNumChannels() == 0 || ctx.NumSamples() == 0 {
		p.debugLogger.Printf("WARNING: No channels (%d) or samples (%d) to process!", 
			ctx.GetNumChannels(), ctx.NumSamples())
		return
	}

	// Update filter parameters
	p.svFilter.SetFrequencyAndQ(ctx.SampleRate, cutoff, resonance)
	p.svFilter.SetMode(filterType / 3.0) // Convert 0-3 to 0-1

	// Process each channel using the helper
	ctx.ProcessChannels(func(ch int, input, output []float32) {
		// Create a temporary buffer for the filtered signal
		filtered := make([]float32, len(input))
		copy(filtered, input)

		// Apply filter
		p.svFilter.Process(filtered, ch)

		// Apply mix using the DSP library
		mix.DryWetBufferTo(input, filtered, mixAmount, output)
	})
}

func (p *FilterProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *FilterProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *FilterProcessor) SetActive(active bool) error {
	p.debugLogger.Printf("SetActive called: active=%v", active)
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
