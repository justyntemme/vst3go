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
		Name:     "Gain",
		Version:  "2.0.0",
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
	
	// Optional parameter smoothing
	smoother *param.ParameterSmoother
	sampleRate float64
}

const (
	ParamGain = iota
	ParamOutputLevel
	ParamBypass
	ParamSmoothingEnabled
	ParamSmoothingTime
	
	// Gain range constants
	minGainDB = -24.0
	maxGainDB = 24.0
	
	// Smoothing time range (ms)
	minSmoothingMs = 0.1
	maxSmoothingMs = 100.0
	defaultSmoothingMs = 5.0
)

func NewGainProcessor() *GainProcessor {
	p := &GainProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
		smoother: param.NewParameterSmoother(),
	}

	// Add parameters
	gainParam := param.New(ParamGain, "Gain").
		Range(minGainDB, maxGainDB).
		Default(0).
		Formatter(param.DecibelFormatter, param.DecibelParser).
		Build()
	p.params.Add(gainParam)

	// Add output meter (read-only)
	p.params.Add(
		param.New(ParamOutputLevel, "Output Level").
			Range(-60, 0).
			Default(-60).
			Formatter(param.DecibelFormatter, nil). // No parser needed for read-only
			Flags(param.IsReadOnly).
			Build(),
	)
	
	// Add bypass parameter
	p.params.Add(
		param.BypassParameter(ParamBypass, "Bypass").Build(),
	)
	
	// Add smoothing control parameters
	p.params.Add(
		param.New(ParamSmoothingEnabled, "Enable Smoothing").
			Range(0, 1).
			Default(0).
			Steps(1).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamSmoothingTime, "Smoothing Time").
			Range(minSmoothingMs, maxSmoothingMs).
			Default(defaultSmoothingMs).
			Unit("ms").
			Build(),
	)
	
	// Add gain to smoother (will only be used if smoothing is enabled)
	p.smoother.Add(ParamGain, gainParam, param.ExponentialSmoothing, 0.999)

	return p
}

func (p *GainProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Update smoother sample rate
	if sp, ok := p.smoother.Get(ParamGain); ok {
		smoothingTime := p.params.Get(ParamSmoothingTime).GetValue()
		sp.UpdateSampleRate(sampleRate, smoothingTime)
	}
	
	return nil
}

func (p *GainProcessor) ProcessAudio(ctx *process.Context) {
	// Handle bypass
	if bypassParam := p.params.Get(ParamBypass); bypassParam != nil && bypassParam.GetValue() > 0.5 {
		// Copy input to output
		for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
			copy(ctx.Output[ch], ctx.Input[ch])
		}
		return
	}
	
	// Check if smoothing is enabled
	smoothingEnabled := p.params.Get(ParamSmoothingEnabled).GetValue() > 0.5
	
	// Handle parameter changes
	for _, change := range ctx.GetParameterChanges() {
		switch change.ParamID {
		case ParamGain:
			if smoothingEnabled {
				p.smoother.SetValue(ParamGain, change.Value)
			}
		case ParamSmoothingTime:
			// Update smoothing time
			if sp, ok := p.smoother.Get(ParamGain); ok {
				sp.UpdateSampleRate(p.sampleRate, change.Value)
			}
		}
	}

	// Process and measure
	peak := float32(0)

	if smoothingEnabled {
		// Process with smoothing (sample-by-sample)
		for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
			input := ctx.Input[ch]
			output := ctx.Output[ch]
			
			for i := 0; i < ctx.NumSamples(); i++ {
				// Get smoothed gain value
				gainNorm := p.smoother.GetSmoothed(ParamGain)
				gainDB := minGainDB + gainNorm * (maxGainDB - minGainDB)
				gainLinear := gain.DbToLinear32(float32(gainDB))
				
				// Apply gain
				output[i] = gain.Apply(input[i], gainLinear)
				
				// Track peak
				if abs := float32(math.Abs(float64(output[i]))); abs > peak {
					peak = abs
				}
			}
		}
	} else {
		// Process without smoothing (buffer-based, more efficient)
		gainDB := float32(ctx.ParamPlain(ParamGain))
		gainLinear := gain.DbToLinear32(gainDB)
		
		ctx.ProcessChannels(func(ch int, input, output []float32) {
			copy(output, input)
			gain.ApplyBuffer(output, gainLinear)
			
			// Find peak
			for _, sample := range output {
				if abs := float32(math.Abs(float64(sample))); abs > peak {
					peak = abs
				}
			}
		})
	}

	// Update output meter
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
	if !active {
		// Reset smoother when deactivating
		if sp, ok := p.smoother.Get(ParamGain); ok {
			gainParam := p.params.Get(ParamGain)
			if gainParam != nil {
				sp.SetValue(gainParam.GetValue())
			}
		}
	}
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
