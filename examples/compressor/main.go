package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"fmt"
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/filter"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// MasterCompressorPlugin implements the Plugin interface
type MasterCompressorPlugin struct{}

func (p *MasterCompressorPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.mastercompressor",
		Name:     "Master Compressor",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics",
	}
}

func (p *MasterCompressorPlugin) CreateProcessor() vst3plugin.Processor {
	return NewMasterCompressorProcessor()
}

// MasterCompressorProcessor handles the audio processing
type MasterCompressorProcessor struct {
	params      *param.Registry
	buses       *bus.Configuration
	sampleRate  float64
	compressor  *dynamics.Compressor
	sideHPF     *filter.Biquad // Sidechain high-pass filter
	autoMakeup  bool
}

// Parameter IDs
const (
	ParamThreshold = iota
	ParamRatio
	ParamAttack
	ParamRelease
	ParamKnee
	ParamMakeup
	ParamAutoMakeup
	ParamLookahead
	ParamSideHPF
	ParamSideHPFFreq
	ParamGainReduction // Read-only meter
)

func NewMasterCompressorProcessor() *MasterCompressorProcessor {
	p := &MasterCompressorProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewStereoConfiguration(),
		compressor: dynamics.NewCompressor(44100), // Will be updated in Initialize
		sideHPF:    filter.NewBiquad(2),          // Stereo
		autoMakeup: false,
	}

	// Add parameters
	p.params.Add(
		param.New(ParamThreshold, "Threshold").
			Range(-60, 0).
			Default(-20).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRatio, "Ratio").
			Range(1, 20).
			Default(4).
			Unit(":1").
			Formatter(func(value float64) string {
				if value >= 20 {
					return "∞:1"
				}
				return fmt.Sprintf("%.1f:1", value)
			}, nil).
			Build(),
	)

	p.params.Add(
		param.New(ParamAttack, "Attack").
			Range(0.1, 100).
			Default(10).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRelease, "Release").
			Range(10, 1000).
			Default(100).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamKnee, "Knee").
			Range(0, 10).
			Default(2).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamMakeup, "Makeup Gain").
			Range(-20, 20).
			Default(0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamAutoMakeup, "Auto Makeup").
			Range(0, 1).
			Default(0).
			Steps(2).
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamLookahead, "Lookahead").
			Range(0, 10).
			Default(0).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamSideHPF, "Sidechain HPF").
			Range(0, 1).
			Default(0).
			Steps(2).
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamSideHPFFreq, "HPF Frequency").
			Range(20, 500).
			Default(100).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)

	// Gain reduction meter (read-only)
	p.params.Add(
		param.New(ParamGainReduction, "Gain Reduction").
			Range(-60, 0).
			Default(0).
			Unit("dB").
			Formatter(param.DecibelFormatter, nil).
			Flags(param.IsReadOnly).
			Build(),
	)

	return p
}

func (p *MasterCompressorProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create new compressor with correct sample rate
	p.compressor = dynamics.NewCompressor(sampleRate)
	
	// Initialize sidechain filter
	p.sideHPF.SetHighpass(sampleRate, 100, 0.707) // Default 100Hz
	
	// Apply initial parameter values
	p.updateCompressor()
	
	return nil
}

func (p *MasterCompressorProcessor) ProcessAudio(ctx *process.Context) {
	// Update parameters if needed
	p.updateCompressor()
	
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	
	if numChannels == 0 {
		return
	}
	
	numSamples := ctx.NumSamples()
	
	// Handle mono/stereo
	if numChannels == 1 {
		// Mono processing
		if p.params.Get(ParamSideHPF).GetValue() > 0.5 {
			// Apply sidechain filter
			filtered := make([]float32, numSamples)
			copy(filtered, ctx.Input[0][:numSamples])
			p.sideHPF.Process(filtered, 0)
			
			// Process with filtered sidechain
			p.compressor.ProcessSidechain(
				ctx.Input[0][:numSamples],
				filtered,
				ctx.Output[0][:numSamples],
			)
		} else {
			// Direct processing
			p.compressor.ProcessBuffer(
				ctx.Input[0][:numSamples],
				ctx.Output[0][:numSamples],
			)
		}
	} else {
		// Stereo processing
		if p.params.Get(ParamSideHPF).GetValue() > 0.5 {
			// Apply sidechain filter to both channels
			filteredL := make([]float32, numSamples)
			filteredR := make([]float32, numSamples)
			copy(filteredL, ctx.Input[0][:numSamples])
			copy(filteredR, ctx.Input[1][:numSamples])
			p.sideHPF.Process(filteredL, 0)
			p.sideHPF.Process(filteredR, 1)
			
			// For stereo linked compression with sidechain, we need to process manually
			// since the compressor doesn't have a stereo sidechain method
			for i := 0; i < numSamples; i++ {
				// Get max of filtered channels for detection
				maxFiltered := float32(math.Max(
					math.Abs(float64(filteredL[i])),
					math.Abs(float64(filteredR[i])),
				))
				
				// Process left channel with sidechain
				p.compressor.ProcessSidechain(
					ctx.Input[0][i:i+1],
					[]float32{maxFiltered},
					ctx.Output[0][i:i+1],
				)
				
				// Get the gain that was applied
				gain := float32(1.0)
				if ctx.Input[0][i] != 0 {
					gain = ctx.Output[0][i] / ctx.Input[0][i]
				}
				
				// Apply same gain to right channel
				ctx.Output[1][i] = ctx.Input[1][i] * gain
			}
		} else {
			// Direct stereo linked processing
			p.compressor.ProcessStereo(
				ctx.Input[0][:numSamples],
				ctx.Input[1][:numSamples],
				ctx.Output[0][:numSamples],
				ctx.Output[1][:numSamples],
			)
		}
	}
	
	// Update gain reduction meter
	gainReduction := p.compressor.GetGainReduction()
	p.params.Get(ParamGainReduction).SetValue(
		p.params.Get(ParamGainReduction).Normalize(gainReduction),
	)
}

func (p *MasterCompressorProcessor) updateCompressor() {
	// Threshold
	threshold := p.params.Get(ParamThreshold).GetPlainValue()
	p.compressor.SetThreshold(threshold)
	
	// Ratio
	ratio := p.params.Get(ParamRatio).GetPlainValue()
	p.compressor.SetRatio(ratio)
	
	// Attack (convert from ms to seconds)
	attack := p.params.Get(ParamAttack).GetPlainValue() / 1000.0
	p.compressor.SetAttack(attack)
	
	// Release (convert from ms to seconds)
	release := p.params.Get(ParamRelease).GetPlainValue() / 1000.0
	p.compressor.SetRelease(release)
	
	// Knee
	knee := p.params.Get(ParamKnee).GetPlainValue()
	p.compressor.SetKnee(dynamics.KneeSoft, knee)
	
	// Lookahead (convert from ms to seconds)
	lookahead := p.params.Get(ParamLookahead).GetPlainValue() / 1000.0
	p.compressor.SetLookahead(lookahead)
	
	// Makeup gain
	makeup := p.params.Get(ParamMakeup).GetPlainValue()
	autoMakeup := p.params.Get(ParamAutoMakeup).GetValue() > 0.5
	
	if autoMakeup {
		// Calculate approximate makeup gain based on threshold and ratio
		// This is a simplified formula that works reasonably well
		avgReduction := (0 - threshold) * (1.0 - 1.0/ratio) * 0.5
		p.compressor.SetMakeupGain(avgReduction)
	} else {
		p.compressor.SetMakeupGain(makeup)
	}
	
	// Update sidechain filter frequency
	if p.params.Get(ParamSideHPF).GetValue() > 0.5 {
		freq := p.params.Get(ParamSideHPFFreq).GetPlainValue()
		p.sideHPF.SetHighpass(p.sampleRate, freq, 0.707)
	}
}

func (p *MasterCompressorProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *MasterCompressorProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *MasterCompressorProcessor) SetActive(active bool) error {
	if !active {
		p.compressor.Reset()
	}
	return nil
}

func (p *MasterCompressorProcessor) GetLatencySamples() int32 {
	// Report lookahead as latency
	lookaheadMs := p.params.Get(ParamLookahead).GetPlainValue()
	return int32(lookaheadMs * p.sampleRate / 1000.0)
}

func (p *MasterCompressorProcessor) GetTailSamples() int32 {
	// Compressor release time as tail
	releaseMs := p.params.Get(ParamRelease).GetPlainValue()
	return int32(releaseMs * p.sampleRate / 1000.0)
}

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&MasterCompressorPlugin{})
}

// Required for c-shared build mode
func main() {}