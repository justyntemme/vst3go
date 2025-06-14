// Package main implements a multi-effect plugin using the DSP chain builder.
package main

import (
	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/utility"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/dsp"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
	
	// Import C bridge - required for VST3 plugin to work
	_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
)

const (
	// Parameter IDs
	ParamBypass uint32 = iota
	ParamGateThreshold
	ParamCompThreshold
	ParamCompRatio
	ParamNoiseAmount
	ParamChainSelect
)

// Parameter range constants
const (
	// Parameter IDs
	// Threshold ranges
	minThresholdDB = -60.0
	maxThresholdDB = 0.0
	defaultGateThresholdDB = -30.0
	defaultCompThresholdDB = -20.0
	
	// Ratio ranges
	minRatio = 1.0
	maxRatio = 20.0
	defaultRatio = 4.0
	
	// Noise amount
	noiseScaleFactor = 0.01
)

// ChainFXPlugin implements a multi-effect plugin with selectable chains
type ChainFXPlugin struct{}

func (p *ChainFXPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.chainfx",
		Name:     "ChainFX",
		Vendor:   "VST3Go Examples",
		Version:  "1.0.0",
		Category: "Fx",
	}
}

func (p *ChainFXPlugin) CreateProcessor() vst3plugin.Processor {
	return &ChainFXProcessor{}
}

// ChainFXProcessor handles the audio processing using DSP chains
type ChainFXProcessor struct {
	params *param.Registry
	
	// DSP components
	gate       *dynamics.Gate
	compressor *dynamics.Compressor
	dcBlocker  *dsp.DCBlockerAdapter
	noise      *dsp.NoiseAdapter
	
	// Chains
	dynamicsChain *dsp.Chain
	simpleChain   *dsp.Chain
	currentChain  *dsp.Chain
	
	sampleRate float64
}

func (p *ChainFXProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create DSP components
	p.gate = dynamics.NewGate(sampleRate)
	p.gate.SetAttack(0.1)
	p.gate.SetHold(10)
	p.gate.SetRelease(100)
	
	p.compressor = dynamics.NewCompressor(sampleRate)
	p.compressor.SetAttack(10)
	p.compressor.SetRelease(100)
	p.compressor.SetKnee(dynamics.KneeSoft, 2)
	
	p.dcBlocker = dsp.NewDCBlockerAdapter(sampleRate)
	p.noise = dsp.NewNoiseAdapter(utility.PinkNoise, 0)
	
	// Build chains
	var err error
	
	// Dynamics chain: DC Blocker -> Gate -> Compressor
	p.dynamicsChain, err = dsp.NewBuilder("Dynamics").
		WithProcessor(p.dcBlocker).
		WithProcessor(dsp.NewGateAdapter(p.gate)).
		WithProcessor(dsp.NewCompressorAdapter(p.compressor)).
		Build()
	if err != nil {
		return err
	}
	
	// Simple chain: DC Blocker -> Noise
	p.simpleChain, err = dsp.NewBuilder("Simple").
		WithProcessor(p.dcBlocker).
		WithProcessor(p.noise).
		Build()
	if err != nil {
		return err
	}
	
	// Default to dynamics chain
	p.currentChain = p.dynamicsChain
	
	// Initialize parameter registry
	p.params = param.NewRegistry()

	// Register parameters
	registry := p.params
	
	registry.Add(
		param.BypassParameter(ParamBypass, "Bypass").Build(),
	)
	
	registry.Add(
		param.ThresholdParameter(ParamGateThreshold, "Gate Threshold", minThresholdDB, maxThresholdDB, defaultGateThresholdDB).Build(),
	)
	
	registry.Add(
		param.ThresholdParameter(ParamCompThreshold, "Comp Threshold", minThresholdDB, maxThresholdDB, defaultCompThresholdDB).Build(),
	)
	
	registry.Add(
		param.RatioParameter(ParamCompRatio, "Comp Ratio", minRatio, maxRatio, defaultRatio).Build(),
	)
	
	registry.Add(
		param.MixParameter(ParamNoiseAmount, "Noise Amount").Build(),
	)
	
	registry.Add(
		param.Choice(ParamChainSelect, "Chain Select", []param.ChoiceOption{
			{Value: 0, Name: "Dynamics"},
			{Value: 1, Name: "Simple"},
		}).Build(),
	)
	
	return nil
}

func (p *ChainFXProcessor) ProcessAudio(ctx *process.Context) {
	// Handle bypass
	bypassParam := p.params.Get(ParamBypass)
	if bypassParam != nil && bypassParam.GetValue() > 0.5 {
		// Copy input to output
		for ch := 0; ch < ctx.NumInputChannels(); ch++ {
			input := ctx.Input[ch]
			output := ctx.Output[ch]
			copy(output, input)
		}
		return
	}
	
	// Handle parameter changes
	for _, change := range ctx.GetParameterChanges() {
		switch change.ParamID {
		case ParamGateThreshold:
			dbValue := minThresholdDB + change.Value*(maxThresholdDB-minThresholdDB)
			p.gate.SetThreshold(dbValue)
			
		case ParamCompThreshold:
			dbValue := minThresholdDB + change.Value*(maxThresholdDB-minThresholdDB)
			p.compressor.SetThreshold(dbValue)
			
		case ParamCompRatio:
			ratio := minRatio + change.Value*(maxRatio-minRatio)
			p.compressor.SetRatio(ratio)
			
		case ParamNoiseAmount:
			amount := float32(change.Value * noiseScaleFactor)
			p.noise.SetMix(amount)
			
		case ParamChainSelect:
			if change.Value < 0.5 {
				p.currentChain = p.dynamicsChain
			} else {
				p.currentChain = p.simpleChain
			}
		}
	}
	
	// Process audio through the selected chain
	for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]
		
		// Copy input to output
		copy(output, input)
		
		// Process through chain
		p.currentChain.Process(output)
	}
}

func (p *ChainFXProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *ChainFXProcessor) GetBuses() *bus.Configuration {
	// Simple stereo in/out configuration
	return bus.NewStereoConfiguration()
}

func (p *ChainFXProcessor) SetActive(active bool) error {
	if !active {
		// Reset all chains
		p.dynamicsChain.Reset()
		p.simpleChain.Reset()
	}
	return nil
}

func (p *ChainFXProcessor) GetLatencySamples() int32 {
	// Lookahead in compressor might add latency
	// For now, return 0
	return 0
}

func (p *ChainFXProcessor) GetTailSamples() int32 {
	// Gate release time might create a tail
	// Return a conservative estimate
	return int32(p.sampleRate * 0.1) // 100ms
}

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&ChainFXPlugin{})
}

// Required for c-shared build mode
func main() {}