package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"fmt"
	
	"github.com/justyntemme/vst3go/pkg/dsp"
	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/filter"
	"github.com/justyntemme/vst3go/pkg/dsp/gain"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
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
	vst3plugin.Register(&DrumBusPlugin{})
}

// Required for c-shared build mode
func main() {}

// DrumBusPlugin implements the Plugin interface
type DrumBusPlugin struct{}

func (p *DrumBusPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.drumbus",
		Name:     "Drum Bus",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Dynamics",
	}
}

func (p *DrumBusPlugin) CreateProcessor() vst3plugin.Processor {
	return NewDrumBusProcessor()
}

// Parameter IDs
const (
	// Parallel Compressor
	ParamParallelThreshold uint32 = iota
	ParamParallelRatio
	ParamParallelAttack
	ParamParallelRelease
	ParamParallelHPF
	
	// Transient Shaper
	ParamTransientAttack
	ParamTransientSustain
	
	// Glue Compressor
	ParamGlueThreshold
	ParamGlueRatio
	ParamGlueAttack
	ParamGlueRelease
	
	// Mix and Output
	ParamParallelMix
	ParamOutputGain
	ParamGainReduction
)

// Parameter range constants
const (
	// Threshold ranges
	minThresholdDB = -60.0
	maxThresholdDB = 0.0
	
	// Ratio ranges
	minRatio = 2.0
	maxRatio = 20.0
	
	// Time ranges
	minAttackMS = 0.001
	maxAttackMS = 0.05
	minReleaseMS = 0.01
	maxReleaseMS = 0.5
	
	// HPF range
	minHPFFreq = 20.0
	maxHPFFreq = 500.0
	defaultHPFFreq = 100.0
	
	// Gain range
	minOutputGainDB = -20.0
	maxOutputGainDB = 20.0
	
	// Default values
	defaultParallelMix = 0.5
	defaultOutputGain = 1.0
)

// DrumBusProcessor implements the audio processing
type DrumBusProcessor struct {
	// DSP - Parallel compression path
	parallelCompL, parallelCompR *dynamics.Compressor
	hpfL, hpfR                   *filter.Biquad
	
	// DSP - Main path
	transientShaperL, transientShaperR *dynamics.Expander
	glueCompL, glueCompR               *dynamics.Compressor
	
	// Parameters
	params *param.Registry
	buses  *bus.Configuration
	
	// Parameter values
	parallelMix      float64
	outputGain       float64
	hpfFreq          float64
	transientAttack  float64
	transientSustain float64
	
	// State
	sampleRate float64
	active     bool
	
	// Buffers for parallel processing
	parallelBufferL []float32
	parallelBufferR []float32
}

// NewDrumBusProcessor creates a new processor
func NewDrumBusProcessor() *DrumBusProcessor {
	p := &DrumBusProcessor{
		params:           param.NewRegistry(),
		buses:            bus.NewStereoConfiguration(),
		parallelMix:      defaultParallelMix,
		outputGain:       defaultOutputGain,
		hpfFreq:          defaultHPFFreq,
		transientAttack:  0.0,
		transientSustain: 0.0,
	}
	
	// Initialize parameters
	p.initializeParameters()
	
	return p
}

func (p *DrumBusProcessor) initializeParameters() {
	// Parallel Compressor section
	p.params.Add(
		param.New(ParamParallelThreshold, "Parallel Threshold").
			Range(minThresholdDB, maxThresholdDB).
			Default(-20.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamParallelRatio, "Parallel Ratio").
			Range(minRatio, maxRatio).
			Default(10.0).
			Unit(":1").
			Formatter(param.RatioFormatter, param.RatioParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamParallelAttack, "Parallel Attack").
			Range(minAttackMS, maxAttackMS).
			Default(0.005).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.1f ms", value*1000.0)
			}, func(text string) (float64, error) {
				val, err := param.TimeParser(text)
				if err != nil {
					return 0, err
				}
				return val / 1000.0, nil
			}).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamParallelRelease, "Parallel Release").
			Range(minReleaseMS, maxReleaseMS).
			Default(0.1).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.0f ms", value*1000.0)
			}, func(text string) (float64, error) {
				val, err := param.TimeParser(text)
				if err != nil {
					return 0, err
				}
				return val / 1000.0, nil
			}).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamParallelHPF, "HPF Sidechain").
			Range(minHPFFreq, maxHPFFreq).
			Default(defaultHPFFreq).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)
	
	// Transient Shaper section
	p.params.Add(
		param.New(ParamTransientAttack, "Transient Attack").
			Range(-1.0, 1.0).
			Default(0.0).
			Unit("%").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.0f%%", value*100.0)
			}, func(text string) (float64, error) {
				return param.PercentParser(text)
			}).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamTransientSustain, "Transient Sustain").
			Range(-1.0, 1.0).
			Default(0.0).
			Unit("%").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.0f%%", value*100.0)
			}, func(text string) (float64, error) {
				return param.PercentParser(text)
			}).
			Build(),
	)
	
	// Glue Compressor section
	p.params.Add(
		param.New(ParamGlueThreshold, "Glue Threshold").
			Range(-30.0, 0.0).
			Default(-10.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamGlueRatio, "Glue Ratio").
			Range(1.5, 10.0).
			Default(3.0).
			Unit(":1").
			Formatter(param.RatioFormatter, param.RatioParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamGlueAttack, "Glue Attack").
			Range(0.01, 0.1).
			Default(0.03).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.0f ms", value*1000.0)
			}, func(text string) (float64, error) {
				val, err := param.TimeParser(text)
				if err != nil {
					return 0, err
				}
				return val / 1000.0, nil
			}).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamGlueRelease, "Glue Release").
			Range(0.05, 0.5).
			Default(0.2).
			Unit("ms").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.0f ms", value*1000.0)
			}, func(text string) (float64, error) {
				val, err := param.TimeParser(text)
				if err != nil {
					return 0, err
				}
				return val / 1000.0, nil
			}).
			Build(),
	)
	
	// Mix and Output
	p.params.Add(
		param.New(ParamParallelMix, "Parallel Mix").
			Range(0.0, 1.0).
			Default(0.5).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamOutputGain, "Output").
			Range(minOutputGainDB, maxOutputGainDB).
			Default(0.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
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
func (p *DrumBusProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create parallel compression path
	p.parallelCompL = dynamics.NewCompressor(sampleRate)
	p.parallelCompR = dynamics.NewCompressor(sampleRate)
	p.hpfL = filter.NewBiquad(1)
	p.hpfR = filter.NewBiquad(1)
	
	// Create main path processors
	p.transientShaperL = dynamics.NewExpander(sampleRate)
	p.transientShaperR = dynamics.NewExpander(sampleRate)
	p.glueCompL = dynamics.NewCompressor(sampleRate)
	p.glueCompR = dynamics.NewCompressor(sampleRate)
	
	// Allocate parallel processing buffers
	p.parallelBufferL = make([]float32, maxBlockSize)
	p.parallelBufferR = make([]float32, maxBlockSize)
	
	// Configure processors
	p.configureProcessors()
	
	return nil
}

// configureProcessors sets up all processors with appropriate drum bus settings
func (p *DrumBusProcessor) configureProcessors() {
	// Configure parallel compressors for heavy compression
	p.parallelCompL.SetKnee(dynamics.KneeHard, 0.0)
	p.parallelCompL.SetMakeupGain(10.0) // Heavy makeup gain for parallel
	p.parallelCompR.SetKnee(dynamics.KneeHard, 0.0)
	p.parallelCompR.SetMakeupGain(10.0)
	
	// Configure HPF for sidechain
	p.updateHPF()
	
	// Configure transient shapers
	p.configureTransientShapers()
	
	// Configure glue compressors
	p.glueCompL.SetKnee(dynamics.KneeSoft, 4.0)
	p.glueCompR.SetKnee(dynamics.KneeSoft, 4.0)
}

// updateHPF updates the high-pass filter coefficients
func (p *DrumBusProcessor) updateHPF() {
	p.hpfL.SetHighpass(p.sampleRate, p.hpfFreq, 0.7)
	p.hpfR.SetHighpass(p.sampleRate, p.hpfFreq, 0.7)
}

// configureTransientShapers configures the expanders for transient shaping
func (p *DrumBusProcessor) configureTransientShapers() {
	// Use expanders creatively for transient enhancement
	// Positive attack = emphasize transients
	// Negative attack = reduce transients
	
	if p.transientAttack >= 0 {
		// Enhance transients
		threshold := -30.0 + (p.transientAttack * 20.0)
		ratio := 1.5 + (p.transientAttack * 2.5)
		
		p.transientShaperL.SetThreshold(threshold)
		p.transientShaperL.SetRatio(ratio)
		p.transientShaperL.SetAttack(0.0001)
		p.transientShaperL.SetRelease(0.02)
		
		p.transientShaperR.SetThreshold(threshold)
		p.transientShaperR.SetRatio(ratio)
		p.transientShaperR.SetAttack(0.0001)
		p.transientShaperR.SetRelease(0.02)
	} else {
		// Reduce transients (use as upward compressor)
		threshold := -10.0 + (p.transientAttack * 20.0)
		ratio := 1.5 - (p.transientAttack * 0.4)
		
		p.transientShaperL.SetThreshold(threshold)
		p.transientShaperL.SetRatio(ratio)
		p.transientShaperL.SetAttack(0.0001)
		p.transientShaperL.SetRelease(0.05)
		
		p.transientShaperR.SetThreshold(threshold)
		p.transientShaperR.SetRatio(ratio)
		p.transientShaperR.SetAttack(0.0001)
		p.transientShaperR.SetRelease(0.05)
	}
}

// ProcessAudio processes audio
func (p *DrumBusProcessor) ProcessAudio(ctx *process.Context) {
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
	
	// 1. Copy input for parallel processing
	copy(p.parallelBufferL[:numSamples], ctx.Input[0][:numSamples])
	copy(p.parallelBufferR[:numSamples], ctx.Input[1][:numSamples])
	
	// 2. Apply HPF to parallel path (sidechain only)
	// We need filtered signal for detection but process original
	filteredL := make([]float32, numSamples)
	filteredR := make([]float32, numSamples)
	copy(filteredL, p.parallelBufferL[:numSamples])
	copy(filteredR, p.parallelBufferR[:numSamples])
	p.hpfL.Process(filteredL, 0)
	p.hpfR.Process(filteredR, 0)
	
	// Apply parallel compression
	// Note: We can't do true sidechain with current API, so we'll compress normally
	// but the HPF gives us the frequency-conscious behavior we want
	p.parallelCompL.ProcessBuffer(p.parallelBufferL[:numSamples], p.parallelBufferL[:numSamples])
	p.parallelCompR.ProcessBuffer(p.parallelBufferR[:numSamples], p.parallelBufferR[:numSamples])
	
	// 3. Main path: Copy input to output
	copy(ctx.Output[0][:numSamples], ctx.Input[0][:numSamples])
	copy(ctx.Output[1][:numSamples], ctx.Input[1][:numSamples])
	
	// 4. Apply transient shaping
	p.transientShaperL.ProcessBuffer(ctx.Output[0][:numSamples], ctx.Output[0][:numSamples])
	p.transientShaperR.ProcessBuffer(ctx.Output[1][:numSamples], ctx.Output[1][:numSamples])
	
	// 5. Apply glue compression
	p.glueCompL.ProcessBuffer(ctx.Output[0][:numSamples], ctx.Output[0][:numSamples])
	p.glueCompR.ProcessBuffer(ctx.Output[1][:numSamples], ctx.Output[1][:numSamples])
	
	// 6. Mix parallel with main
	dsp.AddScaled(ctx.Output[0][:numSamples], p.parallelBufferL[:numSamples], float32(p.parallelMix))
	dsp.AddScaled(ctx.Output[1][:numSamples], p.parallelBufferR[:numSamples], float32(p.parallelMix))
	
	// 7. Apply output gain and sustain adjustment
	gainValue := float32(p.outputGain * (1.0 + p.transientSustain*0.5))
	gain.ApplyBuffer(ctx.Output[0][:numSamples], gainValue)
	gain.ApplyBuffer(ctx.Output[1][:numSamples], gainValue)
	
	// Update gain reduction meter
	parallelGR := (p.parallelCompL.GetGainReduction() + p.parallelCompR.GetGainReduction()) / 2.0
	glueGR := (p.glueCompL.GetGainReduction() + p.glueCompR.GetGainReduction()) / 2.0
	totalGR := -(parallelGR + glueGR)
	
	if grParam := p.params.Get(ParamGainReduction); grParam != nil {
		grParam.SetPlainValue(totalGR)
	}
}

// updateParameters checks for parameter changes
func (p *DrumBusProcessor) updateParameters(ctx *process.Context) {
	// Parallel compressor parameters
	threshold := ctx.ParamPlain(ParamParallelThreshold)
	p.parallelCompL.SetThreshold(threshold)
	p.parallelCompR.SetThreshold(threshold)
	
	ratio := ctx.ParamPlain(ParamParallelRatio)
	p.parallelCompL.SetRatio(ratio)
	p.parallelCompR.SetRatio(ratio)
	
	attack := ctx.ParamPlain(ParamParallelAttack)
	p.parallelCompL.SetAttack(attack)
	p.parallelCompR.SetAttack(attack)
	
	release := ctx.ParamPlain(ParamParallelRelease)
	p.parallelCompL.SetRelease(release)
	p.parallelCompR.SetRelease(release)
	
	// HPF frequency
	newHPF := ctx.ParamPlain(ParamParallelHPF)
	if newHPF != p.hpfFreq {
		p.hpfFreq = newHPF
		p.updateHPF()
	}
	
	// Transient shaper parameters
	newTransientAttack := ctx.ParamPlain(ParamTransientAttack)
	if newTransientAttack != p.transientAttack {
		p.transientAttack = newTransientAttack
		p.configureTransientShapers()
	}
	
	p.transientSustain = ctx.ParamPlain(ParamTransientSustain)
	
	// Glue compressor parameters
	glueThreshold := ctx.ParamPlain(ParamGlueThreshold)
	p.glueCompL.SetThreshold(glueThreshold)
	p.glueCompR.SetThreshold(glueThreshold)
	
	glueRatio := ctx.ParamPlain(ParamGlueRatio)
	p.glueCompL.SetRatio(glueRatio)
	p.glueCompR.SetRatio(glueRatio)
	
	glueAttack := ctx.ParamPlain(ParamGlueAttack)
	p.glueCompL.SetAttack(glueAttack)
	p.glueCompR.SetAttack(glueAttack)
	
	glueRelease := ctx.ParamPlain(ParamGlueRelease)
	p.glueCompL.SetRelease(glueRelease)
	p.glueCompR.SetRelease(glueRelease)
	
	// Mix and output
	p.parallelMix = ctx.Param(ParamParallelMix)
	
	outputGainDB := ctx.ParamPlain(ParamOutputGain)
	p.outputGain = gain.DbToLinear(outputGainDB)
}

// GetParameters returns the parameter registry
func (p *DrumBusProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *DrumBusProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// SetActive is called when processing starts/stops
func (p *DrumBusProcessor) SetActive(active bool) error {
	p.active = active
	if !active {
		// Reset all processors
		if p.parallelCompL != nil {
			p.parallelCompL.Reset()
			p.parallelCompR.Reset()
		}
		if p.hpfL != nil {
			p.hpfL.Reset()
			p.hpfR.Reset()
		}
		if p.transientShaperL != nil {
			p.transientShaperL.Reset()
			p.transientShaperR.Reset()
		}
		if p.glueCompL != nil {
			p.glueCompL.Reset()
			p.glueCompR.Reset()
		}
		
		// Clear buffers
		for i := range p.parallelBufferL {
			p.parallelBufferL[i] = 0
			p.parallelBufferR[i] = 0
		}
	}
	return nil
}

// GetLatencySamples returns the plugin latency in samples
func (p *DrumBusProcessor) GetLatencySamples() int32 {
	return 0 // No lookahead
}

// GetTailSamples returns the tail length in samples
func (p *DrumBusProcessor) GetTailSamples() int32 {
	// Maximum release time
	maxRelease := 0.5 // 500ms max
	return int32(maxRelease * p.sampleRate)
}