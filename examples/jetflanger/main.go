package main

import (
	"fmt"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	"github.com/justyntemme/vst3go/pkg/dsp/modulation"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
	
	// Import C bridge - required for VST3 plugin to work
	_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
)

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&JetFlangerPlugin{})
}

// Required for c-shared build mode
func main() {}

// JetFlangerPlugin implements the Plugin interface
type JetFlangerPlugin struct{}

func (p *JetFlangerPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.jetflanger",
		Name:     "Jet Flanger",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Modulation",
	}
}

func (p *JetFlangerPlugin) CreateProcessor() vst3plugin.Processor {
	return NewJetFlangerProcessor()
}

// Parameter IDs
const (
	ParamRate uint32 = iota
	ParamDepth
	ParamFeedback
	ParamMix
	ParamManual
	ParamManualMode
	ParamNegativeFeedback
)

// JetFlangerProcessor implements the audio processing
type JetFlangerProcessor struct {
	// DSP
	flanger *modulation.Flanger
	
	// Parameters
	params *param.Registry
	buses  *bus.Configuration
	
	// Parameter values
	rate             float64
	depth            float64
	feedback         float64
	mix              float64
	manual           float64
	manualMode       bool
	negativeFeedback bool
	
	// State
	sampleRate float64
	active     bool
}

// NewJetFlangerProcessor creates a new processor
func NewJetFlangerProcessor() *JetFlangerProcessor {
	p := &JetFlangerProcessor{
		params:           param.NewRegistry(),
		buses:            bus.NewStereoConfiguration(),
		rate:             0.5,
		depth:            2.0,
		feedback:         0.7,  // High feedback for jet sound
		mix:              0.5,
		manual:           0.5,
		manualMode:       false,
		negativeFeedback: false,
	}
	
	// Initialize parameters
	p.initializeParameters()
	
	return p
}

func (p *JetFlangerProcessor) initializeParameters() {
	// Rate control (0.1-10 Hz)
	p.params.Add(
		param.New(ParamRate, "Rate").
			Range(0.1, 10.0).
			Default(0.5).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)
	
	// Depth control (0.5-10ms)
	p.params.Add(
		param.New(ParamDepth, "Depth").
			Range(0.5, 10.0).
			Default(2.0).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.1f ms", value)
			}, func(text string) (float64, error) {
				return param.TimeParser(text)
			}).
			Build(),
	)
	
	// Feedback control (-99% to +99%)
	p.params.Add(
		param.New(ParamFeedback, "Feedback").
			Range(0.0, 0.99).
			Default(0.7).
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
	
	// Mix control (0-100%)
	p.params.Add(
		param.New(ParamMix, "Mix").
			Range(0.0, 1.0).
			Default(0.5).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)
	
	// Manual control (for static flanging)
	p.params.Add(
		param.New(ParamManual, "Manual").
			Range(0.0, 1.0).
			Default(0.5).
			Unit("").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.2f", value)
			}, func(text string) (float64, error) {
				var val float64
				_, err := fmt.Sscanf(text, "%f", &val)
				return val, err
			}).
			Build(),
	)
	
	// Manual mode on/off
	p.params.Add(
		param.New(ParamManualMode, "Manual Mode").
			Range(0.0, 1.0).
			Default(0.0).
			Unit("").
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)
	
	// Negative feedback option
	p.params.Add(
		param.New(ParamNegativeFeedback, "Negative FB").
			Range(0.0, 1.0).
			Default(0.0).
			Unit("").
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)
}

// Initialize is called when the plugin is created
func (p *JetFlangerProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create flanger effect
	p.flanger = modulation.NewFlanger(sampleRate)
	
	// Configure with current parameters
	p.configureFlanger()
	
	return nil
}

// configureFlanger sets up the flanger with current parameters
func (p *JetFlangerProcessor) configureFlanger() {
	p.flanger.SetRate(p.rate)
	p.flanger.SetDepth(p.depth)
	p.flanger.SetDelay(3.0) // Fixed short delay for flanger (3ms center)
	
	// Apply negative feedback if enabled
	actualFeedback := p.feedback
	if p.negativeFeedback {
		actualFeedback = -p.feedback
	}
	p.flanger.SetFeedback(actualFeedback)
	
	p.flanger.SetMix(p.mix)
	p.flanger.SetManual(p.manual)
	p.flanger.SetManualMode(p.manualMode)
}

// ProcessAudio processes audio
func (p *JetFlangerProcessor) ProcessAudio(ctx *process.Context) {
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
	p.flanger.ProcessStereoBuffer(
		ctx.Input[0][:numSamples],
		ctx.Input[1][:numSamples],
		ctx.Output[0][:numSamples],
		ctx.Output[1][:numSamples],
	)
}

// updateParameters checks for parameter changes
func (p *JetFlangerProcessor) updateParameters(ctx *process.Context) {
	// Check rate parameter
	newRate := ctx.ParamPlain(ParamRate)
	if newRate != p.rate {
		p.rate = newRate
		p.flanger.SetRate(p.rate)
	}
	
	// Check depth parameter
	newDepth := ctx.ParamPlain(ParamDepth)
	if newDepth != p.depth {
		p.depth = newDepth
		p.flanger.SetDepth(p.depth)
	}
	
	// Check feedback parameter
	newFeedback := ctx.ParamPlain(ParamFeedback)
	feedbackChanged := newFeedback != p.feedback
	p.feedback = newFeedback
	
	// Check negative feedback parameter
	negFBValue := ctx.Param(ParamNegativeFeedback)
	newNegativeFeedback := negFBValue > 0.5
	if newNegativeFeedback != p.negativeFeedback || feedbackChanged {
		p.negativeFeedback = newNegativeFeedback
		// Apply feedback with correct sign
		actualFeedback := p.feedback
		if p.negativeFeedback {
			actualFeedback = -p.feedback
		}
		p.flanger.SetFeedback(actualFeedback)
	}
	
	// Check mix parameter
	newMix := ctx.Param(ParamMix)
	if newMix != p.mix {
		p.mix = newMix
		p.flanger.SetMix(p.mix)
	}
	
	// Check manual parameter
	newManual := ctx.Param(ParamManual)
	if newManual != p.manual {
		p.manual = newManual
		p.flanger.SetManual(p.manual)
	}
	
	// Check manual mode parameter
	manualModeValue := ctx.Param(ParamManualMode)
	newManualMode := manualModeValue > 0.5
	if newManualMode != p.manualMode {
		p.manualMode = newManualMode
		p.flanger.SetManualMode(p.manualMode)
	}
}

// GetParameters returns the parameter registry
func (p *JetFlangerProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *JetFlangerProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// SetActive is called when processing starts/stops
func (p *JetFlangerProcessor) SetActive(active bool) error {
	p.active = active
	if !active {
		// Reset flanger when deactivated
		if p.flanger != nil {
			p.flanger.Reset()
		}
	}
	return nil
}

// GetLatencySamples returns the plugin latency in samples
func (p *JetFlangerProcessor) GetLatencySamples() int32 {
	return 0 // No lookahead
}

// GetTailSamples returns the tail length in samples
func (p *JetFlangerProcessor) GetTailSamples() int32 {
	// Return maximum delay time in samples
	maxDelayMs := 3.0 + p.depth // Center delay + depth
	return int32(maxDelayMs * p.sampleRate / 1000.0)
}