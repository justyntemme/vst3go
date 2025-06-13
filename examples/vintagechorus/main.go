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
	"github.com/justyntemme/vst3go/pkg/dsp/modulation"
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
	vst3plugin.Register(&VintageChorusPlugin{})
}

// Required for c-shared build mode
func main() {}

// VintageChorusPlugin implements the Plugin interface
type VintageChorusPlugin struct{}

func (p *VintageChorusPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.vintagechorus",
		Name:     "Vintage Chorus",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Modulation",
	}
}

func (p *VintageChorusPlugin) CreateProcessor() vst3plugin.Processor {
	return NewVintageChorusProcessor()
}

// Parameter IDs
const (
	ParamRate uint32 = iota
	ParamDepth
	ParamDelay
	ParamMix
	ParamFeedback
	ParamSpread
	ParamVoices
)

// VintageChorusProcessor implements the audio processing
type VintageChorusProcessor struct {
	// DSP
	chorus *modulation.Chorus
	
	// Parameters
	params *param.Registry
	buses  *bus.Configuration
	
	// Parameter values
	rate     float64
	depth    float64
	delay    float64
	mix      float64
	feedback float64
	spread   float64
	voices   int
	
	// State
	sampleRate float64
	active     bool
}

// NewVintageChorusProcessor creates a new processor
func NewVintageChorusProcessor() *VintageChorusProcessor {
	p := &VintageChorusProcessor{
		params:   param.NewRegistry(),
		buses:    bus.NewStereoConfiguration(),
		rate:     0.5,
		depth:    2.0,
		delay:    20.0,
		mix:      0.5,
		feedback: 0.1,
		spread:   1.0,
		voices:   2,
	}
	
	// Initialize parameters
	p.initializeParameters()
	
	return p
}

func (p *VintageChorusProcessor) initializeParameters() {
	// Rate control (0.1-10 Hz)
	p.params.Add(
		param.New(ParamRate, "Rate").
			Range(0.1, 10.0).
			Default(0.5).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)
	
	// Depth control (0-10ms)
	p.params.Add(
		param.New(ParamDepth, "Depth").
			Range(0.0, 10.0).
			Default(2.0).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.1f ms", value)
			}, func(text string) (float64, error) {
				return param.TimeParser(text)
			}).
			Build(),
	)
	
	// Delay control (10-50ms)
	p.params.Add(
		param.New(ParamDelay, "Delay").
			Range(10.0, 50.0).
			Default(20.0).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.1f ms", value)
			}, func(text string) (float64, error) {
				return param.TimeParser(text)
			}).
			Build(),
	)
	
	// Mix control (0-100%)
	p.params.Add(
		param.New(ParamMix, "Mix").
			Range(0.0, 1.0).
			Default(0.5).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)
	
	// Feedback control (0-50%)
	p.params.Add(
		param.New(ParamFeedback, "Feedback").
			Range(0.0, 0.5).
			Default(0.1).
			Unit("%").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.0f%%", value*100.0)
			}, func(text string) (float64, error) {
				val, err := param.PercentParser(text)
				if err != nil {
					return 0, err
				}
				return val / 100.0, nil
			}).
			Build(),
	)
	
	// Stereo spread (0-100%)
	p.params.Add(
		param.New(ParamSpread, "Spread").
			Range(0.0, 1.0).
			Default(1.0).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)
	
	// Voice selection (1-4)
	p.params.Add(
		param.New(ParamVoices, "Voices").
			Range(1.0, 4.0).
			Default(2.0).
			Unit("").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%d", int(value))
			}, func(text string) (float64, error) {
				var val int
				_, err := fmt.Sscanf(text, "%d", &val)
				if err != nil {
					return 0, err
				}
				return float64(val), nil
			}).
			Build(),
	)
}

// Initialize is called when the plugin is created
func (p *VintageChorusProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create chorus effect
	p.chorus = modulation.NewChorus(sampleRate)
	
	// Configure with current parameters
	p.configureChorus()
	
	return nil
}

// configureChorus sets up the chorus with current parameters
func (p *VintageChorusProcessor) configureChorus() {
	p.chorus.SetRate(p.rate)
	p.chorus.SetDepth(p.depth)
	p.chorus.SetDelay(p.delay)
	p.chorus.SetMix(p.mix)
	p.chorus.SetFeedback(p.feedback)
	p.chorus.SetSpread(p.spread)
	p.chorus.SetVoices(p.voices)
}

// ProcessAudio processes audio
func (p *VintageChorusProcessor) ProcessAudio(ctx *process.Context) {
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
	
	// Process stereo
	p.chorus.ProcessStereoBuffer(
		ctx.Input[0][:numSamples],
		ctx.Input[1][:numSamples],
		ctx.Output[0][:numSamples],
		ctx.Output[1][:numSamples],
	)
}

// updateParameters checks for parameter changes
func (p *VintageChorusProcessor) updateParameters(ctx *process.Context) {
	// Check rate parameter
	newRate := ctx.ParamPlain(ParamRate)
	if newRate != p.rate {
		p.rate = newRate
		p.chorus.SetRate(p.rate)
	}
	
	// Check depth parameter
	newDepth := ctx.ParamPlain(ParamDepth)
	if newDepth != p.depth {
		p.depth = newDepth
		p.chorus.SetDepth(p.depth)
	}
	
	// Check delay parameter
	newDelay := ctx.ParamPlain(ParamDelay)
	if newDelay != p.delay {
		p.delay = newDelay
		p.chorus.SetDelay(p.delay)
	}
	
	// Check mix parameter
	newMix := ctx.Param(ParamMix)
	if newMix != p.mix {
		p.mix = newMix
		p.chorus.SetMix(p.mix)
	}
	
	// Check feedback parameter
	newFeedback := ctx.ParamPlain(ParamFeedback)
	if newFeedback != p.feedback {
		p.feedback = newFeedback
		p.chorus.SetFeedback(p.feedback)
	}
	
	// Check spread parameter
	newSpread := ctx.Param(ParamSpread)
	if newSpread != p.spread {
		p.spread = newSpread
		p.chorus.SetSpread(p.spread)
	}
	
	// Check voices parameter
	newVoices := int(ctx.ParamPlain(ParamVoices))
	if newVoices != p.voices {
		p.voices = newVoices
		p.chorus.SetVoices(p.voices)
	}
}

// GetParameters returns the parameter registry
func (p *VintageChorusProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *VintageChorusProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// SetActive is called when processing starts/stops
func (p *VintageChorusProcessor) SetActive(active bool) error {
	p.active = active
	if !active {
		// Reset chorus when deactivated
		if p.chorus != nil {
			p.chorus.Reset()
		}
	}
	return nil
}

// GetLatencySamples returns the plugin latency in samples
func (p *VintageChorusProcessor) GetLatencySamples() int32 {
	return 0 // No lookahead
}

// GetTailSamples returns the tail length in samples
func (p *VintageChorusProcessor) GetTailSamples() int32 {
	// Return maximum delay time in samples
	maxDelayMs := p.delay + p.depth
	return int32(maxDelayMs * p.sampleRate / 1000.0)
}