package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"fmt"
	"log"
	"os"

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

	// Debug logging
	debugLogger *log.Logger
	callCount   int
}

// Parameter IDs
const (
	ParamFilterType = 0
	ParamCutoff     = 1
	ParamResonance  = 2
	ParamMix        = 3
)

func NewFilterProcessor() *FilterProcessor {
	// Create debug logger that writes to stderr
	debugFile, err := os.OpenFile("/tmp/vst3go_filter_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		debugFile = os.Stderr // Fallback to stderr
	}

	logger := log.New(debugFile, "[FILTER] ", log.LstdFlags|log.Lmicroseconds)
	logger.Println("=== Creating new FilterProcessor ===")

	p := &FilterProcessor{
		params:      param.NewRegistry(),
		buses:       bus.NewStereoConfiguration(),
		svFilter:    filter.NewMultiModeSVF(2), // stereo
		sampleRate:  48000,
		debugLogger: logger,
		callCount:   0,
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
			Range(80, 8000).
			Default(800).
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),

		param.New(ParamResonance, "Resonance").
			Range(0.5, 10).
			Default(0.707).
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
	p.callCount++

	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}

	numSamples := ctx.NumSamples()

	// Debug every 1000 calls to avoid spam
	if p.callCount%1000 == 1 {
		p.debugLogger.Printf("ProcessAudio call #%d: channels=%d, samples=%d, sampleRate=%.1f",
			p.callCount, numChannels, numSamples, ctx.SampleRate)
	}

	// Get parameter values
	filterType := ctx.ParamPlain(ParamFilterType)
	cutoff := ctx.ParamPlain(ParamCutoff)
	resonance := ctx.ParamPlain(ParamResonance)
	mix := ctx.ParamPlain(ParamMix) / 100.0 // Convert percentage to 0-1

	// Also get normalized parameter values for comparison
	filterTypeNorm := ctx.Param(ParamFilterType)
	cutoffNorm := ctx.Param(ParamCutoff)
	resonanceNorm := ctx.Param(ParamResonance)
	mixNorm := ctx.Param(ParamMix)

	// Debug parameter values every 1000 calls
	if p.callCount%1000 == 1 {
		p.debugLogger.Printf("Parameters: type=%.2f, cutoff=%.1fHz, resonance=%.2f, mix=%.2f",
			filterType, cutoff, resonance, mix)
		p.debugLogger.Printf("Normalized: typeN=%.3f, cutoffN=%.3f, resonanceN=%.3f, mixN=%.3f",
			filterTypeNorm, cutoffNorm, resonanceNorm, mixNorm)
	}

	// Check if we have valid input
	if numChannels == 0 || numSamples == 0 {
		p.debugLogger.Printf("WARNING: No channels (%d) or samples (%d) to process!", numChannels, numSamples)
		return
	}

	// Update filter parameters
	p.svFilter.SetFrequencyAndQ(ctx.SampleRate, cutoff, resonance)
	p.svFilter.SetMode(filterType / 3.0) // Convert 0-3 to 0-1

	// Sample input level for debugging
	var inputLevel, outputLevel float64

	// Process each channel
	for ch := 0; ch < numChannels; ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]

		if len(input) != numSamples || len(output) != numSamples {
			p.debugLogger.Printf("WARNING: Buffer size mismatch! input=%d, output=%d, expected=%d",
				len(input), len(output), numSamples)
			continue
		}

		// Sample input level (first few samples)
		if ch == 0 && p.callCount%1000 == 1 {
			for i := 0; i < min(10, numSamples); i++ {
				inputLevel += float64(input[i] * input[i])
			}
		}

		// Copy input to output first
		copy(output, input)

		// Apply filter
		p.svFilter.Process(output, ch)

		// Sample output level (first few samples)
		if ch == 0 && p.callCount%1000 == 1 {
			for i := 0; i < min(10, numSamples); i++ {
				outputLevel += float64(output[i] * output[i])
			}
		}

		// Apply mix (wet/dry blend)
		if mix < 1.0 {
			for i := range output {
				output[i] = input[i]*(1.0-float32(mix)) + output[i]*float32(mix)
			}
		}
	}

	// Debug audio levels every 1000 calls
	if p.callCount%1000 == 1 {
		p.debugLogger.Printf("Audio levels: input RMS=%.6f, filtered RMS=%.6f",
			inputLevel/10.0, outputLevel/10.0)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
