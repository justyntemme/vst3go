package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"github.com/justyntemme/vst3go/pkg/dsp"
	"github.com/justyntemme/vst3go/pkg/dsp/delay"
	"github.com/justyntemme/vst3go/pkg/dsp/mix"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// DelayPlugin implements the Plugin interface
type DelayPlugin struct{}

func (d *DelayPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.delay",
		Name:     "Simple Delay",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Delay",
	}
}

func (d *DelayPlugin) CreateProcessor() vst3plugin.Processor {
	return NewDelayProcessor()
}

// DelayProcessor handles the audio processing
type DelayProcessor struct {
	params *param.Registry
	buses  *bus.Configuration

	// Delay lines using DSP library
	delayLines []*delay.Line
	sampleRate float64
}

// Parameter IDs
const (
	ParamDelayTime = 0
	ParamFeedback  = 1
	ParamMix       = 2
)

func NewDelayProcessor() *DelayProcessor {
	p := &DelayProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewStereoConfiguration(),
		sampleRate: dsp.SampleRate48k,
	}

	// Add parameters
	p.params.Add(
		param.New(ParamDelayTime, "Delay Time").
			Range(0, 1000).
			Default(250).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),

		param.New(ParamFeedback, "Feedback").
			Range(0, 100).
			Default(30).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamMix, "Mix").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)

	return p
}

func (p *DelayProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate

	// Create delay lines for 2 channels using DSP library
	// 1 second max delay
	p.delayLines = make([]*delay.Line, 2)
	for i := range p.delayLines {
		p.delayLines[i] = delay.New(1.0, sampleRate)
	}

	return nil
}

func (p *DelayProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameter values
	delayTimeMs := ctx.ParamPlain(ParamDelayTime)
	feedback := float32(ctx.ParamPlain(ParamFeedback) / 100.0) // Convert from percentage
	mixAmount := float32(ctx.ParamPlain(ParamMix) / 100.0)     // Convert from percentage

	numSamples := ctx.NumSamples()

	// Use process helper to handle stereo channels
	ctx.ProcessStereo(func(ch int, input, output []float32) {
		for sample := 0; sample < numSamples; sample++ {
			// Get input sample
			dry := input[sample]

			// Read delayed sample
			delayed := p.delayLines[ch].ReadMs(delayTimeMs)

			// Mix dry and wet signals using the mix utility
			output[sample] = mix.DryWet(dry, delayed, mixAmount)

			// Write to delay line with feedback
			p.delayLines[ch].Write(dry + delayed*feedback)
		}
	})
}

func (p *DelayProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *DelayProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *DelayProcessor) SetActive(active bool) error {
	if !active {
		// Clear delay lines when deactivated
		for _, line := range p.delayLines {
			if line != nil {
				line.Reset()
			}
		}
	}
	return nil
}

func (p *DelayProcessor) GetLatencySamples() int32 {
	return 0
}

func (p *DelayProcessor) GetTailSamples() int32 {
	// Return max delay time as tail
	return int32(p.sampleRate) // 1 second
}

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&DelayPlugin{})
}

// Required for c-shared build mode
func main() {}
