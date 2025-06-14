// Package main demonstrates automatic parameter ID management in a VST3 plugin.
package main

import (
	"C"
	"fmt"

	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/filter"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
	
	// Import C bridge - required for VST3 plugin to work
	_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
)

// AutoParamsPlugin demonstrates automatic parameter management
type AutoParamsPlugin struct{}

func (p *AutoParamsPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.autoparams",
		Name:     "AutoParams",
		Vendor:   "VST3Go Examples",
		Version:  "1.0.0",
		Category: "Fx",
	}
}

func (p *AutoParamsPlugin) CreateProcessor() vst3plugin.Processor {
	return &AutoParamsProcessor{}
}

// AutoParamsProcessor demonstrates automatic parameter ID management
type AutoParamsProcessor struct {
	plugin.Base
	
	// Use AutoRegistry instead of regular Registry
	autoParams *param.AutoRegistry
	
	// DSP components
	compressor *dynamics.Compressor
	filters    []*filter.SVF
	
	sampleRate float64
}

func (p *AutoParamsProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create auto registry
	p.autoParams = param.NewAutoRegistry()
	
	// Register standard controls - IDs assigned automatically!
	if err := p.autoParams.RegisterStandardControls(); err != nil {
		return err
	}
	
	// Register compressor controls - IDs continue automatically
	if err := p.autoParams.RegisterCompressorControls(); err != nil {
		return err
	}
	
	// Register 3 EQ bands - each band gets automatic IDs
	for i := 1; i <= 3; i++ {
		if err := p.autoParams.RegisterEQBand(i); err != nil {
			return err
		}
	}
	
	// Can also register individual parameters without tracking IDs
	p.autoParams.Register(
		param.DepthParameter(0, "Warmth").Build(),
		param.RateParameter(0, "Modulation", 0.1, 10, 1).Build(),
		param.ResonanceParameter(0, "Character").Build(),
	)
	
	// Initialize DSP
	p.compressor = dynamics.NewCompressor(sampleRate)
	p.filters = make([]*filter.SVF, 3)
	for i := range p.filters {
		p.filters[i] = filter.NewSVF(2) // Stereo
	}
	
	return nil
}

func (p *AutoParamsProcessor) ProcessAudio(ctx *process.Context) {
	// Handle bypass using parameter name instead of ID
	if bypass := p.autoParams.GetByName("Bypass"); bypass != nil && bypass.GetValue() > 0.5 {
		for ch := 0; ch < ctx.NumInputChannels(); ch++ {
			copy(ctx.Output[ch], ctx.Input[ch])
		}
		return
	}
	
	// Handle parameter changes
	for _, change := range ctx.GetParameterChanges() {
		// Find which parameter changed
		param := p.autoParams.Get(change.ParamID)
		if param == nil {
			continue
		}
		
		// Update based on parameter name
		switch param.Name {
		case "Threshold":
			p.compressor.SetThreshold(param.GetPlainValue())
		case "Ratio":
			p.compressor.SetRatio(param.GetPlainValue())
		case "Attack":
			p.compressor.SetAttack(param.GetPlainValue())
		case "Release":
			p.compressor.SetRelease(param.GetPlainValue())
		}
		
		// Handle EQ parameters
		for i := 1; i <= 3; i++ {
			bandFreqName := fmt.Sprintf("Band %d Frequency", i)
			if param.Name == bandFreqName {
				if i-1 < len(p.filters) {
					p.filters[i-1].SetFrequency(param.GetPlainValue(), p.sampleRate)
				}
			}
		}
	}
	
	// Process audio
	for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
		copy(ctx.Output[ch], ctx.Input[ch])
		
		// Apply compression
		for i := 0; i < ctx.NumSamples(); i++ {
			ctx.Output[ch][i] = p.compressor.Process(ctx.Output[ch][i])
		}
		
		// Apply EQ bands (using bandpass for demonstration)
		for _, f := range p.filters {
			f.ProcessBandpass(ctx.Output[ch], ch)
		}
	}
	
	// Apply mix control
	if mixParam := p.autoParams.GetByName("Mix"); mixParam != nil {
		mix := float32(mixParam.GetPlainValue() / 100.0)
		for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
			for i := 0; i < ctx.NumSamples(); i++ {
				ctx.Output[ch][i] = ctx.Input[ch][i]*(1-mix) + ctx.Output[ch][i]*mix
			}
		}
	}
}

func (p *AutoParamsProcessor) GetParameters() *param.Registry {
	// Return the embedded Registry from AutoRegistry
	return p.autoParams.Registry
}

func (p *AutoParamsProcessor) GetBuses() *bus.Configuration {
	return bus.NewStereoConfiguration()
}

func (p *AutoParamsProcessor) SetActive(active bool) error {
	if !active {
		p.compressor.Reset()
		for _, f := range p.filters {
			f.Reset()
		}
	}
	return nil
}

func (p *AutoParamsProcessor) GetLatencySamples() int32 {
	// Compressor might have lookahead
	return 0
}

func (p *AutoParamsProcessor) GetTailSamples() int32 {
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
	vst3plugin.Register(&AutoParamsPlugin{})
}

// Required for c-shared build mode
func main() {}