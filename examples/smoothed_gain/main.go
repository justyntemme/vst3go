// Package main implements a gain plugin with parameter smoothing to prevent zipper noise.
package main

import (
	"C"
	"math"

	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

const (
	paramGain = iota
	paramBypass
)

// SmoothedGainPlugin implements a gain plugin with parameter smoothing
type SmoothedGainPlugin struct{}

func (p *SmoothedGainPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.smoothedgain",
		Name:     "SmoothedGain",
		Vendor:   "VST3Go Examples",
		Version:  "1.0.0",
		Category: "Fx",
	}
}

func (p *SmoothedGainPlugin) CreateProcessor() vst3plugin.Processor {
	return &SmoothedGainProcessor{}
}

// SmoothedGainProcessor handles the audio processing with parameter smoothing
type SmoothedGainProcessor struct {
	params *param.Registry
	
	// Parameter smoother
	smoother *param.ParameterSmoother
	
	// Sample rate for updating smoothing time
	sampleRate float64
}

func (p *SmoothedGainProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Initialize parameter smoother
	p.smoother = param.NewParameterSmoother()
	
	// Initialize parameter registry
	p.params = param.NewRegistry()

	// Register parameters
	registry := p.params
	
	// Gain parameter with smoothing
	gainParam := param.New(paramGain, "Gain").
		Range(-60, 12).
		Default(0).
		Unit("dB").
		Build()
	registry.Add(gainParam)
	
	// Add to smoother with exponential smoothing for 5ms transition
	p.smoother.Add(paramGain, gainParam, param.ExponentialSmoothing, 0.999)
	
	// Update smoothing rate based on sample rate (5ms target time)
	if sp, ok := p.smoother.Get(paramGain); ok {
		sp.UpdateSampleRate(sampleRate, 5.0) // 5ms smoothing time
	}
	
	// Bypass parameter (no smoothing needed)
	registry.Add(
		param.BypassParameter(paramBypass, "Bypass").Build(),
	)
	
	return nil
}

func (p *SmoothedGainProcessor) ProcessAudio(ctx *process.Context) {
	// Handle bypass
	bypassParam := p.params.Get(paramBypass)
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
		p.smoother.SetValue(change.ParamID, change.Value)
	}
	
	// Process audio with smoothed gain
	for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]
		
		// Process each sample with smoothed gain
		for i := 0; i < ctx.NumSamples(); i++ {
			// Get smoothed gain value (normalized)
			gainNorm := p.smoother.GetSmoothed(paramGain)
			
			// Convert to dB then to linear
			gainDB := -60.0 + gainNorm * 72.0 // -60 to +12 dB range
			gainLinear := float32(math.Pow(10, gainDB/20))
			
			// Apply gain
			output[i] = input[i] * gainLinear
		}
	}
}

func (p *SmoothedGainProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *SmoothedGainProcessor) GetBuses() *bus.Configuration {
	// Simple stereo in/out configuration
	return bus.NewStereoConfiguration()
}

func (p *SmoothedGainProcessor) SetActive(active bool) error {
	if !active {
		// Reset smoothers when deactivating
		if sp, ok := p.smoother.Get(paramGain); ok {
			gainParam := p.params.Get(paramGain)
			gainValue := 0.0
			if gainParam != nil {
				gainValue = gainParam.GetValue()
			}
			// Reset using the current parameter value
			sp.SetValue(gainValue)
		}
	}
	return nil
}

func (p *SmoothedGainProcessor) GetLatencySamples() int32 {
	// No latency in gain processing
	return 0
}

func (p *SmoothedGainProcessor) GetTailSamples() int32 {
	// Gain has no tail
	return 0
}

// VST3 exported functions
func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&SmoothedGainPlugin{})
}

// Required for c-shared build mode
func main() {}