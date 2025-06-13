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
	"github.com/justyntemme/vst3go/pkg/dsp/gain"
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
	vst3plugin.Register(&TransientShaperPlugin{})
}

// Required for c-shared build mode
func main() {}

// TransientShaperPlugin implements the Plugin interface
type TransientShaperPlugin struct{}

func (p *TransientShaperPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.transientshaper",
		Name:     "TransientShaper",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics",
	}
}

func (p *TransientShaperPlugin) CreateProcessor() vst3plugin.Processor {
	return NewTransientShaperProcessor()
}

// Parameter IDs
const (
	ParamAttack uint32 = iota
	ParamSustain
	ParamMix
	ParamOutput
	ParamGainReduction
)

// TransientShaperProcessor implements the audio processing
type TransientShaperProcessor struct {
	// DSP
	expanderL *dynamics.Expander
	expanderR *dynamics.Expander
	
	// Parameters
	params *param.Registry
	buses  *bus.Configuration
	
	// Parameter values
	attack      float64
	sustain     float64
	mix         float64
	outputGain  float64
	
	// State
	sampleRate   float64
	active       bool
	
	// Parallel processing buffers (for mix control)
	dryBufferL   []float32
	dryBufferR   []float32
	wetBufferL   []float32
	wetBufferR   []float32
}

// NewTransientShaperProcessor creates a new processor
func NewTransientShaperProcessor() *TransientShaperProcessor {
	p := &TransientShaperProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewStereoConfiguration(),
		attack:     0.5,
		sustain:    0.5,
		mix:        1.0,
		outputGain: 1.0,
	}
	
	// Initialize parameters
	p.initializeParameters()
	
	return p
}

func (p *TransientShaperProcessor) initializeParameters() {
	// Attack enhancement (-100% to +100%)
	p.params.Add(
		param.New(ParamAttack, "Attack").
			Range(-1.0, 1.0).
			Default(0.0).
			Unit("%").
			Formatter(func(value float64) string {
				percentage := value * 100.0
				return fmt.Sprintf("%.0f%%", percentage)
			}, func(text string) (float64, error) {
				return param.PercentParser(text)
			}).
			Build(),
	)
	
	// Sustain control (-100% to +100%)
	p.params.Add(
		param.New(ParamSustain, "Sustain").
			Range(-1.0, 1.0).
			Default(0.0).
			Unit("%").
			Formatter(func(value float64) string {
				percentage := value * 100.0
				return fmt.Sprintf("%.0f%%", percentage)
			}, func(text string) (float64, error) {
				return param.PercentParser(text)
			}).
			Build(),
	)
	
	// Mix control (0-100%)
	p.params.Add(
		param.New(ParamMix, "Mix").
			Range(0.0, 1.0).
			Default(1.0).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)
	
	// Output gain
	p.params.Add(
		param.New(ParamOutput, "Output").
			Range(0.0, 2.0).
			Default(1.0).
			Unit("dB").
			Formatter(func(value float64) string {
				db := gain.LinearToDb(value)
				return fmt.Sprintf("%.1f dB", db)
			}, func(text string) (float64, error) {
				db, err := param.DecibelParser(text)
				if err != nil {
					return 0, err
				}
				return gain.DbToLinear(db), nil
			}).
			Build(),
	)
	
	// Gain reduction meter (read-only)
	p.params.Add(
		param.New(ParamGainReduction, "GR").
			Range(-40.0, 0.0).
			Default(0.0).
			Unit("dB").
			Flags(param.IsReadOnly).
			Build(),
	)
}

// Initialize is called when the plugin is created
func (p *TransientShaperProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create expanders for stereo processing
	p.expanderL = dynamics.NewExpander(sampleRate)
	p.expanderR = dynamics.NewExpander(sampleRate)
	
	// Configure expanders for transient shaping
	p.configureExpanders()
	
	// Allocate processing buffers
	bufferSize := int(maxBlockSize)
	p.dryBufferL = make([]float32, bufferSize)
	p.dryBufferR = make([]float32, bufferSize)
	p.wetBufferL = make([]float32, bufferSize)
	p.wetBufferR = make([]float32, bufferSize)
	
	return nil
}

// configureExpanders sets up the expanders for transient shaping
func (p *TransientShaperProcessor) configureExpanders() {
	// Attack: Use expander to enhance transients
	// Positive attack = lower threshold to expand quiet parts after transients
	// Negative attack = higher threshold to reduce transients
	
	attackFactor := p.attack
	
	if attackFactor >= 0 {
		// Enhance transients: expand the quiet parts after the transient
		threshold := -30.0 + (attackFactor * 20.0) // -30dB to -10dB
		ratio := 1.5 + (attackFactor * 2.5)        // 1.5:1 to 4:1
		attack := 0.0001                           // Very fast attack (0.1ms)
		release := 0.01 + (attackFactor * 0.04)    // 10ms to 50ms
		
		p.expanderL.SetThreshold(threshold)
		p.expanderL.SetRatio(ratio)
		p.expanderL.SetAttack(attack)
		p.expanderL.SetRelease(release)
		p.expanderL.SetKnee(2.0)
		p.expanderL.SetRange(-20.0)
		
		p.expanderR.SetThreshold(threshold)
		p.expanderR.SetRatio(ratio)
		p.expanderR.SetAttack(attack)
		p.expanderR.SetRelease(release)
		p.expanderR.SetKnee(2.0)
		p.expanderR.SetRange(-20.0)
	} else {
		// Reduce transients: use as upward expander (inverted)
		threshold := -10.0 + (attackFactor * 20.0) // -10dB to -30dB
		ratio := 1.5 - (attackFactor * 0.4)        // 1.5:1 to 1.1:1
		attack := 0.0001                           // Very fast attack
		release := 0.05                            // 50ms release
		
		p.expanderL.SetThreshold(threshold)
		p.expanderL.SetRatio(ratio)
		p.expanderL.SetAttack(attack)
		p.expanderL.SetRelease(release)
		p.expanderL.SetKnee(2.0)
		p.expanderL.SetRange(-10.0)
		
		p.expanderR.SetThreshold(threshold)
		p.expanderR.SetRatio(ratio)
		p.expanderR.SetAttack(attack)
		p.expanderR.SetRelease(release)
		p.expanderR.SetKnee(2.0)
		p.expanderR.SetRange(-10.0)
	}
}

// ProcessAudio processes audio
func (p *TransientShaperProcessor) ProcessAudio(ctx *process.Context) {
	if !p.active {
		ctx.Clear()
		return
	}
	
	// Update parameters
	p.updateParameters(ctx)
	
	// Get number of samples
	numSamples := ctx.NumSamples()
	if numSamples == 0 || len(ctx.Input) < 2 || len(ctx.Output) < 2 {
		return
	}
	
	// Resize buffers if needed
	if len(p.dryBufferL) < numSamples {
		p.dryBufferL = make([]float32, numSamples)
		p.dryBufferR = make([]float32, numSamples)
		p.wetBufferL = make([]float32, numSamples)
		p.wetBufferR = make([]float32, numSamples)
	}
	
	// Copy dry signal
	copy(p.dryBufferL[:numSamples], ctx.Input[0][:numSamples])
	copy(p.dryBufferR[:numSamples], ctx.Input[1][:numSamples])
	
	// Process through expanders
	p.expanderL.ProcessBuffer(ctx.Input[0][:numSamples], p.wetBufferL[:numSamples])
	p.expanderR.ProcessBuffer(ctx.Input[1][:numSamples], p.wetBufferR[:numSamples])
	
	// Apply sustain control (simple gain adjustment on the processed signal)
	sustainGain := float32(1.0 + p.sustain*0.5) // 0.5 to 1.5
	
	// Mix dry and wet signals
	mix := float32(p.mix)
	dryMix := float32(1.0 - p.mix)
	outputGain := float32(p.outputGain)
	
	for i := 0; i < numSamples; i++ {
		// Apply sustain to wet signal
		p.wetBufferL[i] *= sustainGain
		p.wetBufferR[i] *= sustainGain
		
		// Mix and apply output gain
		ctx.Output[0][i] = (p.dryBufferL[i]*dryMix + p.wetBufferL[i]*mix) * outputGain
		ctx.Output[1][i] = (p.dryBufferR[i]*dryMix + p.wetBufferR[i]*mix) * outputGain
	}
	
	// Update gain reduction meter (use average of both channels)
	grL := p.expanderL.GetGainReduction()
	grR := p.expanderR.GetGainReduction()
	avgGR := (grL + grR) / 2.0
	
	// For read-only parameters, we update the value directly
	if grParam := p.params.Get(ParamGainReduction); grParam != nil {
		grParam.SetPlainValue(avgGR)
	}
}

// updateParameters checks for parameter changes
func (p *TransientShaperProcessor) updateParameters(ctx *process.Context) {
	// Check attack parameter
	newAttack := ctx.ParamPlain(ParamAttack)
	if newAttack != p.attack {
		p.attack = newAttack
		p.configureExpanders()
	}
	
	// Check sustain parameter
	p.sustain = ctx.ParamPlain(ParamSustain)
	
	// Check mix parameter
	p.mix = ctx.Param(ParamMix)
	
	// Check output gain
	p.outputGain = ctx.ParamPlain(ParamOutput)
}

// GetParameters returns the parameter registry
func (p *TransientShaperProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *TransientShaperProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// SetActive is called when processing starts/stops
func (p *TransientShaperProcessor) SetActive(active bool) error {
	p.active = active
	if !active {
		// Reset expanders when deactivated
		if p.expanderL != nil {
			p.expanderL.Reset()
		}
		if p.expanderR != nil {
			p.expanderR.Reset()
		}
	}
	return nil
}

// GetLatencySamples returns the plugin latency in samples
func (p *TransientShaperProcessor) GetLatencySamples() int32 {
	return 0
}

// GetTailSamples returns the tail length in samples
func (p *TransientShaperProcessor) GetTailSamples() int32 {
	// Return release time in samples
	return int32(0.05 * p.sampleRate) // 50ms tail
}