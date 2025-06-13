package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"fmt"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&MasterLimiterPlugin{})
}

// Required for c-shared build mode
func main() {}

// MasterLimiterPlugin implements the Plugin interface
type MasterLimiterPlugin struct{}

func (p *MasterLimiterPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.masterlimiter",
		Name:     "Master Limiter",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics|Mastering",
	}
}

func (p *MasterLimiterPlugin) CreateProcessor() vst3plugin.Processor {
	return NewMasterLimiterProcessor()
}

// Parameter IDs
const (
	ParamCeiling uint32 = iota
	ParamRelease
	ParamTruePeak
	ParamLookahead
	ParamGainReduction
)

// MasterLimiterProcessor implements the audio processing
type MasterLimiterProcessor struct {
	// DSP
	limiterL *dynamics.Limiter
	limiterR *dynamics.Limiter
	
	// Parameters
	params *param.Registry
	buses  *bus.Configuration
	
	// Parameter values
	ceiling   float64
	release   float64
	truePeak  bool
	lookahead float64
	
	// State
	sampleRate float64
	active     bool
	
	// Stereo linking
	stereoBuffer []float32
}

// NewMasterLimiterProcessor creates a new processor
func NewMasterLimiterProcessor() *MasterLimiterProcessor {
	p := &MasterLimiterProcessor{
		params:    param.NewRegistry(),
		buses:     bus.NewStereoConfiguration(),
		ceiling:   -0.3,
		release:   0.05,
		truePeak:  true,
		lookahead: 0.005,
	}
	
	// Initialize parameters
	p.initializeParameters()
	
	return p
}

func (p *MasterLimiterProcessor) initializeParameters() {
	// Ceiling control (-3 to 0 dB)
	p.params.Add(
		param.New(ParamCeiling, "Ceiling").
			Range(-3.0, 0.0).
			Default(-0.3).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	// Release time (1-100ms)
	p.params.Add(
		param.New(ParamRelease, "Release").
			Range(0.001, 0.1).
			Default(0.05).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.1f ms", value*1000.0)
			}, func(text string) (float64, error) {
				val, err := param.TimeParser(text)
				if err != nil {
					return 0, err
				}
				return val / 1000.0, nil // Convert ms to seconds
			}).
			Build(),
	)
	
	// True peak detection on/off
	p.params.Add(
		param.New(ParamTruePeak, "True Peak").
			Range(0.0, 1.0).
			Default(1.0).
			Unit("").
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)
	
	// Lookahead time (0-10ms)
	p.params.Add(
		param.New(ParamLookahead, "Lookahead").
			Range(0.0, 0.010).
			Default(0.005).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.1f ms", value*1000.0)
			}, func(text string) (float64, error) {
				val, err := param.TimeParser(text)
				if err != nil {
					return 0, err
				}
				return val / 1000.0, nil // Convert ms to seconds
			}).
			Build(),
	)
	
	// Gain reduction meter (read-only)
	p.params.Add(
		param.New(ParamGainReduction, "GR").
			Range(-20.0, 0.0).
			Default(0.0).
			Unit("dB").
			Flags(param.IsReadOnly).
			Build(),
	)
}

// Initialize is called when the plugin is created
func (p *MasterLimiterProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create limiters for stereo processing
	p.limiterL = dynamics.NewLimiter(sampleRate)
	p.limiterR = dynamics.NewLimiter(sampleRate)
	
	// Configure limiters
	p.configureLimiters()
	
	// Allocate stereo buffer for linked processing
	p.stereoBuffer = make([]float32, maxBlockSize*2)
	
	return nil
}

// configureLimiters sets up the limiters with current parameters
func (p *MasterLimiterProcessor) configureLimiters() {
	p.limiterL.SetThreshold(p.ceiling)
	p.limiterL.SetRelease(p.release)
	p.limiterL.SetTruePeak(p.truePeak)
	p.limiterL.SetLookahead(p.lookahead)
	
	p.limiterR.SetThreshold(p.ceiling)
	p.limiterR.SetRelease(p.release)
	p.limiterR.SetTruePeak(p.truePeak)
	p.limiterR.SetLookahead(p.lookahead)
}

// ProcessAudio processes audio
func (p *MasterLimiterProcessor) ProcessAudio(ctx *process.Context) {
	if !p.active {
		ctx.PassThrough()
		return
	}
	
	// Update parameters
	p.updateParameters(ctx)
	
	// Get number of samples
	numSamples := ctx.NumSamples()
	if numSamples == 0 || len(ctx.Input) < 2 || len(ctx.Output) < 2 {
		return
	}
	
	// Process stereo with linked limiting
	// The limiter has stereo processing but we need to handle it properly
	for i := 0; i < numSamples; i++ {
		// Process each channel with its own limiter but linked detection
		ctx.Output[0][i] = p.limiterL.Process(ctx.Input[0][i])
		ctx.Output[1][i] = p.limiterR.Process(ctx.Input[1][i])
	}
	
	// Update gain reduction meter
	// Use the maximum gain reduction from both channels
	grL := p.limiterL.GetGainReduction()
	grR := p.limiterR.GetGainReduction()
	maxGR := -max(grL, grR) // Negate because GR is positive but we display negative
	
	if grParam := p.params.Get(ParamGainReduction); grParam != nil {
		grParam.SetPlainValue(maxGR)
	}
}

// updateParameters checks for parameter changes
func (p *MasterLimiterProcessor) updateParameters(ctx *process.Context) {
	// Check ceiling parameter
	newCeiling := ctx.ParamPlain(ParamCeiling)
	if newCeiling != p.ceiling {
		p.ceiling = newCeiling
		p.limiterL.SetThreshold(p.ceiling)
		p.limiterR.SetThreshold(p.ceiling)
	}
	
	// Check release parameter
	newRelease := ctx.ParamPlain(ParamRelease)
	if newRelease != p.release {
		p.release = newRelease
		p.limiterL.SetRelease(p.release)
		p.limiterR.SetRelease(p.release)
	}
	
	// Check true peak parameter
	truePeakValue := ctx.Param(ParamTruePeak)
	newTruePeak := truePeakValue > 0.5
	if newTruePeak != p.truePeak {
		p.truePeak = newTruePeak
		p.limiterL.SetTruePeak(p.truePeak)
		p.limiterR.SetTruePeak(p.truePeak)
	}
	
	// Check lookahead parameter
	newLookahead := ctx.ParamPlain(ParamLookahead)
	if newLookahead != p.lookahead {
		p.lookahead = newLookahead
		p.limiterL.SetLookahead(p.lookahead)
		p.limiterR.SetLookahead(p.lookahead)
	}
}

// GetParameters returns the parameter registry
func (p *MasterLimiterProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *MasterLimiterProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// SetActive is called when processing starts/stops
func (p *MasterLimiterProcessor) SetActive(active bool) error {
	p.active = active
	if !active {
		// Reset limiters when deactivated
		if p.limiterL != nil {
			p.limiterL.Reset()
		}
		if p.limiterR != nil {
			p.limiterR.Reset()
		}
	}
	return nil
}

// GetLatencySamples returns the plugin latency in samples
func (p *MasterLimiterProcessor) GetLatencySamples() int32 {
	// Return lookahead time in samples
	return int32(p.lookahead * p.sampleRate)
}

// GetTailSamples returns the tail length in samples
func (p *MasterLimiterProcessor) GetTailSamples() int32 {
	// Return release time in samples
	return int32(p.release * p.sampleRate)
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}