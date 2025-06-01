package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// StudioGatePlugin implements the Plugin interface
type StudioGatePlugin struct{}

func (p *StudioGatePlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.studiogate",
		Name:     "Studio Gate",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics",
	}
}

func (p *StudioGatePlugin) CreateProcessor() vst3plugin.Processor {
	return NewStudioGateProcessor()
}

// StudioGateProcessor handles the audio processing
type StudioGateProcessor struct {
	params      *param.Registry
	buses       *bus.Configuration
	sampleRate  float64
	gate        *dynamics.Gate
	gateOpen    bool           // For LED display
}

// Parameter IDs
const (
	ParamThreshold = iota
	ParamHysteresis
	ParamAttack
	ParamHold
	ParamRelease
	ParamRange
	ParamSideHPF
	ParamSideHPFFreq
	ParamGateState // Read-only LED indicator
)

func NewStudioGateProcessor() *StudioGateProcessor {
	p := &StudioGateProcessor{
		params:   param.NewRegistry(),
		buses:    bus.NewStereoConfiguration(),
		gate:     dynamics.NewGate(44100), // Will be updated in Initialize
		gateOpen: false,
	}

	// Add parameters
	p.params.Add(
		param.New(ParamThreshold, "Threshold").
			Range(-80, 0).
			Default(-30).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamHysteresis, "Hysteresis").
			Range(0, 10).
			Default(3).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamAttack, "Attack").
			Range(0.01, 50).
			Default(0.1).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamHold, "Hold").
			Range(0, 100).
			Default(10).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRelease, "Release").
			Range(1, 500).
			Default(100).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),
	)

	p.params.Add(
		param.New(ParamRange, "Range").
			Range(-80, 0).
			Default(-40).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
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
			Range(20, 200).
			Default(80).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)

	// Gate state LED (read-only)
	p.params.Add(
		param.New(ParamGateState, "Gate State").
			Range(0, 1).
			Default(0).
			Steps(2).
			Formatter(func(value float64) string {
				if value > 0.5 {
					return "Open"
				}
				return "Closed"
			}, nil).
			Flags(param.IsReadOnly).
			Build(),
	)

	return p
}

func (p *StudioGateProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create new gate with correct sample rate
	p.gate = dynamics.NewGate(sampleRate)
	
	// Configure sidechain filter in gate
	p.gate.SetSidechainFilter(false, 80) // Default 80Hz, initially disabled
	
	// Apply initial parameter values
	p.updateGate()
	
	return nil
}

func (p *StudioGateProcessor) ProcessAudio(ctx *process.Context) {
	// Update parameters if needed
	p.updateGate()
	
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
		p.gate.ProcessBuffer(
			ctx.Input[0][:numSamples],
			ctx.Output[0][:numSamples],
		)
	} else {
		// Stereo processing
		p.gate.ProcessStereo(
			ctx.Input[0][:numSamples],
			ctx.Input[1][:numSamples],
			ctx.Output[0][:numSamples],
			ctx.Output[1][:numSamples],
		)
	}
	
	// Update gate state LED
	isOpen := p.gate.IsOpen()
	if isOpen != p.gateOpen {
		p.gateOpen = isOpen
		if isOpen {
			p.params.Get(ParamGateState).SetValue(1.0)
		} else {
			p.params.Get(ParamGateState).SetValue(0.0)
		}
	}
}

func (p *StudioGateProcessor) updateGate() {
	// Threshold
	threshold := p.params.Get(ParamThreshold).GetPlainValue()
	p.gate.SetThreshold(threshold)
	
	// Hysteresis
	hysteresis := p.params.Get(ParamHysteresis).GetPlainValue()
	p.gate.SetHysteresis(hysteresis)
	
	// Attack (convert from ms to seconds)
	attack := p.params.Get(ParamAttack).GetPlainValue() / 1000.0
	p.gate.SetAttack(attack)
	
	// Hold (convert from ms to seconds)
	hold := p.params.Get(ParamHold).GetPlainValue() / 1000.0
	p.gate.SetHold(hold)
	
	// Release (convert from ms to seconds)
	release := p.params.Get(ParamRelease).GetPlainValue() / 1000.0
	p.gate.SetRelease(release)
	
	// Range
	rangeDB := p.params.Get(ParamRange).GetPlainValue()
	p.gate.SetRange(rangeDB)
	
	// Update sidechain filter
	sidechainEnabled := p.params.Get(ParamSideHPF).GetValue() > 0.5
	freq := p.params.Get(ParamSideHPFFreq).GetPlainValue()
	p.gate.SetSidechainFilter(sidechainEnabled, freq)
}

func (p *StudioGateProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *StudioGateProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *StudioGateProcessor) SetActive(active bool) error {
	if !active {
		p.gate.Reset()
		p.gateOpen = false
		p.params.Get(ParamGateState).SetValue(0.0)
	}
	return nil
}

func (p *StudioGateProcessor) GetLatencySamples() int32 {
	// Gate has no lookahead, so no latency
	return 0
}

func (p *StudioGateProcessor) GetTailSamples() int32 {
	// Gate release time as tail
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
	vst3plugin.Register(&StudioGatePlugin{})
}

// Required for c-shared build mode
func main() {}