package main

import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/gain"
	"github.com/justyntemme/vst3go/pkg/dsp/mix"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
	
	// Import C bridge - required for VST3 plugin to work
	_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
)

// SidechainPlugin implements the Plugin interface
type SidechainPlugin struct{}

func (s *SidechainPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.sidechain",
		Name:     "Sidechain Compressor",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics",
	}
}

func (s *SidechainPlugin) CreateProcessor() vst3plugin.Processor {
	return NewSidechainProcessor()
}

// SidechainProcessor handles the audio processing
type SidechainProcessor struct {
	params *param.Registry
	buses  *bus.Configuration

	// DSP components
	compressor *dynamics.Compressor
	sampleRate float64
}

// Parameter IDs
const (
	ParamThreshold = iota
	ParamRatio
	ParamAttack
	ParamRelease
	ParamMakeup
	ParamMix
	ParamSidechainActive
	ParamKnee
)

func NewSidechainProcessor() *SidechainProcessor {
	p := &SidechainProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewEffectStereoSidechain(),
		compressor: dynamics.NewCompressor(48000),
		sampleRate: 48000,
	}

	// Add parameters
	p.params.Add(
		param.New(ParamThreshold, "Threshold").
			Range(-60, 0).
			Default(-20).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),

		param.New(ParamRatio, "Ratio").
			Range(1, 20).
			Default(4).
			Formatter(param.RatioFormatter, param.RatioParser).
			Build(),

		param.New(ParamAttack, "Attack").
			Range(0.1, 100).
			Default(10).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),

		param.New(ParamRelease, "Release").
			Range(1, 1000).
			Default(100).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),

		param.New(ParamMakeup, "Makeup Gain").
			Range(-12, 24).
			Default(0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),

		param.New(ParamMix, "Mix").
			Range(0, 100).
			Default(100).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamSidechainActive, "Sidechain Active").
			Range(0, 1).
			Default(1).
			Steps(2).
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),

		param.New(ParamKnee, "Knee").
			Range(0, 10).
			Default(2).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	return p
}

func (p *SidechainProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Re-create compressor with new sample rate
	p.compressor = dynamics.NewCompressor(sampleRate)
	
	return nil
}

func (p *SidechainProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameters
	threshold := float32(ctx.ParamPlain(ParamThreshold))
	ratio := float32(ctx.ParamPlain(ParamRatio))
	attackMs := float32(ctx.ParamPlain(ParamAttack))
	releaseMs := float32(ctx.ParamPlain(ParamRelease))
	makeupDb := float32(ctx.ParamPlain(ParamMakeup))
	mixAmount := float32(ctx.ParamPlain(ParamMix) / 100.0)
	sidechainActive := ctx.ParamPlain(ParamSidechainActive) > 0.5
	knee := float32(ctx.ParamPlain(ParamKnee))

	// Configure compressor
	p.compressor.SetThreshold(float64(threshold))
	p.compressor.SetRatio(float64(ratio))
	p.compressor.SetAttack(float64(attackMs) / 1000.0) // Convert ms to seconds
	p.compressor.SetRelease(float64(releaseMs) / 1000.0) // Convert ms to seconds
	p.compressor.SetKnee(dynamics.KneeSoft, float64(knee))
	p.compressor.SetMakeupGain(float64(makeupDb))

	// Standard stereo processing for now - multi-bus support will be added when VST3 wrapper is updated
	_ = sidechainActive // Will be used when multi-bus support is available

	// Process main signal through compressor with internal detection
	ctx.ProcessChannels(func(ch int, input, output []float32) {
		// Create temporary buffer for compressed signal
		compressed := make([]float32, len(input))
		copy(compressed, input)

		// Apply compression using ProcessBuffer
		p.compressor.ProcessBuffer(compressed, compressed)

		// Apply makeup gain
		makeupLinear := gain.DbToLinear32(makeupDb)
		gain.ApplyBuffer(compressed, makeupLinear)

		// Mix dry/wet
		mix.DryWetBufferTo(input, compressed, mixAmount, output)
	})

	// Note: When multi-bus wrapper support is available, we'll use:
	// - Main input bus for the signal to be compressed
	// - Sidechain input bus for the detection signal
	// - Process like: compressor.ProcessWithSidechain(main, sidechain, output)
}

func (p *SidechainProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *SidechainProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *SidechainProcessor) SetActive(active bool) error {
	if active {
		// Try to activate sidechain input when plugin becomes active
		p.buses.SetBusActive(bus.MediaTypeAudio, bus.DirectionInput, 1, true)
	} else {
		// Reset DSP state
		p.compressor.Reset()
	}
	return nil
}

func (p *SidechainProcessor) GetLatencySamples() int32 {
	// No latency for now
	return 0
}

func (p *SidechainProcessor) GetTailSamples() int32 {
	// Release time determines tail
	releaseMs := p.params.Get(ParamRelease).GetPlainValue()
	return int32(math.Ceil(releaseMs * p.sampleRate / 1000.0))
}

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&SidechainPlugin{})
}

// Required for c-shared build mode
func main() {}