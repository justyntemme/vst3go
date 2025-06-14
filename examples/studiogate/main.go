package main

import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp"
	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/filter"
	"github.com/justyntemme/vst3go/pkg/dsp/gain"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
	
	// Import C bridge - required for VST3 plugin to work
	_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
)

// StudioGatePlugin implements the Plugin interface
type StudioGatePlugin struct{}

func (s *StudioGatePlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.studiogate",
		Name:     "Studio Gate",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics",
	}
}

func (s *StudioGatePlugin) CreateProcessor() vst3plugin.Processor {
	return NewStudioGateProcessor()
}

// StudioGateProcessor handles the audio processing
type StudioGateProcessor struct {
	params         *param.Registry
	buses          *bus.Configuration
	gate           *dynamics.Gate
	sidechainHPF   *filter.Biquad
	sampleRate     float64
	
	// Pre-allocated buffers to avoid allocations in ProcessAudio
	sidechainL     []float32
	sidechainR     []float32
	sidechainMono  []float32
}

// Parameter IDs
const (
	ParamThreshold = iota
	ParamHysteresis
	ParamAttack
	ParamHold
	ParamRelease
	ParamRange
	ParamSidechainHPF
	ParamSidechainHPFEnabled
	ParamGateState
	ParamGainReduction
	ParamOutputLevel
)

func NewStudioGateProcessor() *StudioGateProcessor {
	p := &StudioGateProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}

	// Add parameters
	p.params.Add(
		param.New(ParamThreshold, "Threshold").
			Range(dsp.GateMinThreshold, dsp.GateMaxThreshold).
			Default(-40).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamHysteresis, "Hysteresis").
			Range(0, 10).
			Default(5).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamAttack, "Attack").
			Range(0.1, 100).
			Default(1).
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamHold, "Hold").
			Range(0, 100).
			Default(10).
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRelease, "Release").
			Range(1, 1000).
			Default(100).
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRange, "Range").
			Range(dsp.GateMinRange, dsp.GateMaxRange).
			Default(dsp.GateMinRange).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamSidechainHPF, "Sidechain HPF").
			Range(dsp.MinFrequency, 500).
			Default(dsp.DefaultLowFreq).
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamSidechainHPFEnabled, "HPF Enable").
			Range(0, 1).
			Default(0).
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Flags(param.IsBypass).
			Build(),
	)

	// Read-only parameters for metering
	p.params.Add(
		param.New(ParamGateState, "Gate State").
			Range(0, 1).
			Default(0).
			Formatter(func(v float64) string {
				if v > 0.5 {
					return "Open"
				}
				return "Closed"
			}, nil).
			Flags(param.IsReadOnly).
			Build(),
	)

	p.params.Add(
		param.New(ParamGainReduction, "Gain Reduction").
			Range(dsp.GateMinRange, dsp.GateMaxRange).
			Default(0).
			Formatter(param.DecibelFormatter, nil).
			Flags(param.IsReadOnly).
			Build(),
	)

	p.params.Add(
		param.New(ParamOutputLevel, "Output Level").
			Range(dsp.DefaultMinThresholdDB, dsp.DefaultMaxThresholdDB).
			Default(dsp.DefaultMinThresholdDB).
			Formatter(param.DecibelFormatter, nil).
			Flags(param.IsReadOnly).
			Build(),
	)

	return p
}

func (p *StudioGateProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Initialize gate
	p.gate = dynamics.NewGate(sampleRate)
	
	// Initialize sidechain HPF (2 channels for stereo)
	p.sidechainHPF = filter.NewBiquad(2)
	p.sidechainHPF.SetHighpass(sampleRate, 80, 0.707)
	
	// Pre-allocate buffers to avoid allocations in ProcessAudio
	p.sidechainL = make([]float32, maxBlockSize)
	p.sidechainR = make([]float32, maxBlockSize)
	p.sidechainMono = make([]float32, maxBlockSize)
	
	return nil
}

func (p *StudioGateProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameters
	threshold := float32(ctx.ParamPlain(ParamThreshold))
	hysteresis := float32(ctx.ParamPlain(ParamHysteresis))
	attack := float32(ctx.ParamPlain(ParamAttack)) / 1000.0      // Convert ms to seconds
	hold := float32(ctx.ParamPlain(ParamHold)) / 1000.0          // Convert ms to seconds
	release := float32(ctx.ParamPlain(ParamRelease)) / 1000.0    // Convert ms to seconds
	rangeDB := float32(ctx.ParamPlain(ParamRange))
	hpfFreq := float32(ctx.ParamPlain(ParamSidechainHPF))
	hpfEnabled := ctx.ParamPlain(ParamSidechainHPFEnabled) > 0.5

	// Update gate parameters
	p.gate.SetThreshold(float64(threshold))
	p.gate.SetHysteresis(float64(hysteresis))
	p.gate.SetAttack(float64(attack))
	p.gate.SetHold(float64(hold))
	p.gate.SetRelease(float64(release))
	p.gate.SetRange(float64(rangeDB))
	
	// Configure sidechain filter
	p.gate.SetSidechainFilter(hpfEnabled, float64(hpfFreq))
	
	// Update HPF frequency if needed
	if hpfEnabled {
		p.sidechainHPF.SetHighpass(p.sampleRate, float64(hpfFreq), 0.707)
	}

	// Process audio
	peak := float32(0)

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
		
		// Use pre-allocated sidechain buffers if HPF is enabled
		if hpfEnabled {
			// Use pre-allocated sidechain buffers for external HPF processing
			numSamples := ctx.NumSamples()
			sidechainL := p.sidechainL[:numSamples]
			sidechainR := p.sidechainR[:numSamples]
			copy(sidechainL, inputL[:numSamples])
			copy(sidechainR, inputR[:numSamples])
			
			// Apply HPF to sidechain
			p.sidechainHPF.Process(sidechainL, 0)
			p.sidechainHPF.Process(sidechainR, 1)
			
			// Process stereo linked gate with filtered sidechain
			// We need to manually implement stereo processing with external sidechain
			for i := range outputL {
				// Use max of filtered sidechain for detection
				sidechainMax := sidechainL[i]
				if math.Abs(float64(sidechainR[i])) > math.Abs(float64(sidechainMax)) {
					sidechainMax = sidechainR[i]
				}
				
				// Process through gate with sidechain
				gain := p.gate.Process(sidechainMax)
				
				// Apply gain to both channels
				outputL[i] *= gain
				outputR[i] *= gain
			}
		} else {
			// Process stereo linked gate without external filtering
			p.gate.ProcessStereo(inputL, inputR, outputL, outputR)
		}
		
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
			
			if hpfEnabled {
				// Use pre-allocated sidechain buffer
				numSamples := len(input)
				sidechain := p.sidechainMono[:numSamples]
				copy(sidechain, input)
				
				// Apply HPF to sidechain
				p.sidechainHPF.Process(sidechain, 0)
				
				// Process with filtered sidechain
				for i := range output {
					gain := p.gate.Process(sidechain[i])
					output[i] *= gain
				}
			} else {
				// Process without external filtering
				p.gate.ProcessBuffer(input, output)
			}
			
			// Find peak
			for _, sample := range output {
				if abs := float32(math.Abs(float64(sample))); abs > peak {
					peak = abs
				}
			}
		})
	}

	// Update meters
	// Gate state (open/closed)
	gateState := float64(0)
	if p.gate.IsOpen() {
		gateState = 1
	}
	p.params.Get(ParamGateState).SetValue(gateState)
	
	// Gain reduction
	grDB := p.gate.GetGainReduction()
	p.params.Get(ParamGainReduction).SetValue(p.params.Get(ParamGainReduction).Normalize(grDB))
	
	// Output level
	peakDB := gain.LinearToDb32(peak)
	if peakDB < dsp.DefaultMinThresholdDB {
		peakDB = dsp.DefaultMinThresholdDB
	}
	p.params.Get(ParamOutputLevel).SetValue(p.params.Get(ParamOutputLevel).Normalize(float64(peakDB)))
}

func (p *StudioGateProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *StudioGateProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *StudioGateProcessor) SetActive(active bool) error {
	if !active && p.gate != nil {
		p.gate.Reset()
	}
	return nil
}

func (p *StudioGateProcessor) GetLatencySamples() int32 {
	// Gate doesn't introduce latency
	return 0
}

func (p *StudioGateProcessor) GetTailSamples() int32 {
	// Gate might have a small tail due to release time
	if p.gate != nil {
		releaseMS := p.params.Get(ParamRelease).GetPlainValue()
		return int32(releaseMS * p.sampleRate / 1000.0)
	}
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
	vst3plugin.Register(&StudioGatePlugin{})
}

// Required for c-shared build mode
func main() {}