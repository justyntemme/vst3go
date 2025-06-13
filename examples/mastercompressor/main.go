package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/filter"
	"github.com/justyntemme/vst3go/pkg/dsp/gain"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// MasterCompressorPlugin implements the Plugin interface
type MasterCompressorPlugin struct{}

func (m *MasterCompressorPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.mastercompressor",
		Name:     "Master Compressor",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics",
	}
}

func (m *MasterCompressorPlugin) CreateProcessor() vst3plugin.Processor {
	return NewMasterCompressorProcessor()
}

// MasterCompressorProcessor handles the audio processing
type MasterCompressorProcessor struct {
	params         *param.Registry
	buses          *bus.Configuration
	compressor     *dynamics.Compressor
	sidechainHPF   *filter.Biquad
	sampleRate     float64
	makeupGainAuto bool
}

// Parameter IDs
const (
	ParamThreshold = iota
	ParamRatio
	ParamAttack
	ParamRelease
	ParamKnee
	ParamMakeupGain
	ParamAutoMakeup
	ParamLookahead
	ParamSidechainHPF
	ParamGainReduction
	ParamOutputLevel
)

func NewMasterCompressorProcessor() *MasterCompressorProcessor {
	p := &MasterCompressorProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}

	// Add parameters
	p.params.Add(
		param.New(ParamThreshold, "Threshold").
			Range(-60, 0).
			Default(-12).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRatio, "Ratio").
			Range(1, 20).
			Default(4).
			Formatter(func(v float64) string {
				if v >= 20 {
					return "∞:1"
				}
				return param.RatioFormatter(v)
			}, param.RatioParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamAttack, "Attack").
			Range(0.1, 100).
			Default(10).
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRelease, "Release").
			Range(10, 1000).
			Default(100).
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamKnee, "Knee").
			Range(0, 10).
			Default(2).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamMakeupGain, "Makeup Gain").
			Range(-12, 24).
			Default(0).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamAutoMakeup, "Auto Makeup").
			Range(0, 1).
			Default(0).
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Flags(param.IsBypass).
			Build(),
	)

	p.params.Add(
		param.New(ParamLookahead, "Lookahead").
			Range(0, 10).
			Default(2).
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamSidechainHPF, "Sidechain HPF").
			Range(20, 500).
			Default(20).
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)

	// Read-only parameters for metering
	p.params.Add(
		param.New(ParamGainReduction, "Gain Reduction").
			Range(-24, 0).
			Default(0).
			Formatter(param.DecibelFormatter, nil).
			Flags(param.IsReadOnly).
			Build(),
	)

	p.params.Add(
		param.New(ParamOutputLevel, "Output Level").
			Range(-60, 0).
			Default(-60).
			Formatter(param.DecibelFormatter, nil).
			Flags(param.IsReadOnly).
			Build(),
	)

	return p
}

func (p *MasterCompressorProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Initialize compressor
	p.compressor = dynamics.NewCompressor(sampleRate)
	
	// Initialize sidechain HPF (2 channels for stereo)
	p.sidechainHPF = filter.NewBiquad(2)
	p.sidechainHPF.SetHighpass(sampleRate, 20, 0.707)
	
	return nil
}

func (p *MasterCompressorProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameters
	threshold := float32(ctx.ParamPlain(ParamThreshold))
	ratio := float32(ctx.ParamPlain(ParamRatio))
	attack := float32(ctx.ParamPlain(ParamAttack)) / 1000.0     // Convert ms to seconds
	release := float32(ctx.ParamPlain(ParamRelease)) / 1000.0   // Convert ms to seconds
	knee := float32(ctx.ParamPlain(ParamKnee))
	makeupGain := float32(ctx.ParamPlain(ParamMakeupGain))
	autoMakeup := ctx.ParamPlain(ParamAutoMakeup) > 0.5
	lookahead := float32(ctx.ParamPlain(ParamLookahead)) / 1000.0 // Convert ms to seconds
	hpfFreq := float32(ctx.ParamPlain(ParamSidechainHPF))

	// Update compressor parameters
	p.compressor.SetThreshold(float64(threshold))
	p.compressor.SetRatio(float64(ratio))
	p.compressor.SetAttack(float64(attack))
	p.compressor.SetRelease(float64(release))
	
	// Set knee type based on knee width
	if knee < 0.1 {
		p.compressor.SetKnee(dynamics.KneeHard, 0)
	} else {
		p.compressor.SetKnee(dynamics.KneeSoft, float64(knee))
	}
	
	// Calculate auto makeup gain if enabled
	if autoMakeup {
		// Simple auto makeup calculation based on threshold and ratio
		// This estimates the average amount of gain reduction
		avgReduction := (threshold / 2) * (1 - 1/ratio)
		makeupGain = -avgReduction
	}
	p.compressor.SetMakeupGain(float64(makeupGain))
	
	// Set lookahead
	p.compressor.SetLookahead(float64(lookahead))
	
	// Update sidechain HPF frequency
	p.sidechainHPF.SetHighpass(p.sampleRate, float64(hpfFreq), 0.707)

	// Process audio
	peak := float32(0)
	gainReduction := float32(0)

	// Check if we have stereo input
	if ctx.NumInputChannels() >= 2 && ctx.NumOutputChannels() >= 2 {
		// Get stereo buffers
		inputL := ctx.Input[0]
		inputR := ctx.Input[1]
		outputL := ctx.Output[0]
		outputR := ctx.Output[1]
		
		// Copy input to output
		copy(outputL, inputL)
		copy(outputR, inputR)
		
		// Create sidechain buffers for HPF processing
		sidechainL := make([]float32, len(inputL))
		sidechainR := make([]float32, len(inputR))
		copy(sidechainL, inputL)
		copy(sidechainR, inputR)
		
		// Apply HPF to sidechain if frequency is above 20Hz
		if hpfFreq > 20.1 {
			p.sidechainHPF.Process(sidechainL, 0)
			p.sidechainHPF.Process(sidechainR, 1)
		}
		
		// Process stereo linked compression with sidechain
		// We need to create output buffers since ProcessSidechain takes input, sidechain, output
		tempL := make([]float32, len(outputL))
		tempR := make([]float32, len(outputR))
		
		// Create linked sidechain by taking max of L/R
		linkedSidechain := make([]float32, len(sidechainL))
		for i := range linkedSidechain {
			if math.Abs(float64(sidechainL[i])) > math.Abs(float64(sidechainR[i])) {
				linkedSidechain[i] = sidechainL[i]
			} else {
				linkedSidechain[i] = sidechainR[i]
			}
		}
		
		// Process both channels with the same linked sidechain
		p.compressor.ProcessSidechain(outputL, linkedSidechain, tempL)
		copy(outputL, tempL)
		p.compressor.ProcessSidechain(outputR, linkedSidechain, tempR)
		copy(outputR, tempR)
		
		// Get gain reduction
		gainReduction = float32(p.compressor.GetGainReduction())
		
		// Find peak for metering
		for i := range outputL {
			if abs := float32(math.Abs(float64(outputL[i]))); abs > peak {
				peak = abs
			}
			if abs := float32(math.Abs(float64(outputR[i]))); abs > peak {
				peak = abs
			}
		}
	} else {
		// Fallback to mono processing
		ctx.ProcessChannels(func(ch int, input, output []float32) {
			copy(output, input)
			
			// Create sidechain buffer
			sidechain := make([]float32, len(input))
			copy(sidechain, input)
			
			// Apply HPF to sidechain if needed
			if hpfFreq > 20.1 {
				p.sidechainHPF.Process(sidechain, 0)
			}
			
			// Process with sidechain
			temp := make([]float32, len(output))
			p.compressor.ProcessSidechain(output, sidechain, temp)
			copy(output, temp)
			
			// Find peak
			for _, sample := range output {
				if abs := float32(math.Abs(float64(sample))); abs > peak {
					peak = abs
				}
			}
		})
		
		gainReduction = float32(p.compressor.GetGainReduction())
	}

	// Update meters
	grDB := gain.LinearToDb32(gainReduction)
	if grDB > 0 {
		grDB = 0
	}
	p.params.Get(ParamGainReduction).SetValue(p.params.Get(ParamGainReduction).Normalize(float64(grDB)))
	
	peakDB := gain.LinearToDb32(peak)
	if peakDB < -60 {
		peakDB = -60
	}
	p.params.Get(ParamOutputLevel).SetValue(p.params.Get(ParamOutputLevel).Normalize(float64(peakDB)))
}

func (p *MasterCompressorProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *MasterCompressorProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *MasterCompressorProcessor) SetActive(active bool) error {
	if !active && p.compressor != nil {
		p.compressor.Reset()
	}
	return nil
}

func (p *MasterCompressorProcessor) GetLatencySamples() int32 {
	if p.compressor != nil {
		// Report lookahead as latency
		lookaheadMS := p.params.Get(ParamLookahead).GetPlainValue()
		return int32(lookaheadMS * p.sampleRate / 1000.0)
	}
	return 0
}

func (p *MasterCompressorProcessor) GetTailSamples() int32 {
	// Compressor doesn't have significant tail
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
	vst3plugin.Register(&MasterCompressorPlugin{})
}

// Required for c-shared build mode
func main() {}