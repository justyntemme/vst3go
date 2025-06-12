package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/gain"
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

const (
	ParamGain = iota
	ParamOutputLevel
)

func NewGainProcessor() *GainProcessor {
	p := &GainProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}

	// Add parameters
	p.params.Add(
		param.New(ParamGain, "Gain").
			Range(-24, 24).
			Default(0).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	// Add output meter (read-only)
	p.params.Add(
		param.New(ParamOutputLevel, "Output Level").
			Range(-60, 0).
			Default(-60).
			Formatter(param.DecibelFormatter, nil). // No parser needed for read-only
			Flags(param.IsReadOnly).
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
	gainDB := float32(ctx.ParamPlain(ParamGain))

	// Convert to linear using the DSP library
	gainLinear := gain.DbToLinear32(gainDB)

	// Process and measure
	peak := float32(0)

	// Use the process helper to process all channels
	ctx.ProcessChannels(func(ch int, input, output []float32) {
		// Copy input to output
		copy(output, input)
		
		// Apply gain using the DSP library
		gain.ApplyBuffer(output, gainLinear)

		// Find peak
		for _, sample := range output {
			if abs := float32(math.Abs(float64(sample))); abs > peak {
				peak = abs
			}
		}
	})

	// Update output meter using DSP library conversion
	peakDB := gain.LinearToDb32(peak)
	if peakDB < -60 {
		peakDB = -60
	}
	p.params.Get(ParamOutputLevel).SetValue(p.params.Get(ParamOutputLevel).Normalize(float64(peakDB)))
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
