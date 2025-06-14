package main

import (
	"fmt"
	
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

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&VocalStripPlugin{})
}

// Required for c-shared build mode
func main() {}

// VocalStripPlugin implements the Plugin interface
type VocalStripPlugin struct{}

func (p *VocalStripPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.vocalstrip",
		Name:     "Vocal Strip",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Channel Strip",
	}
}

func (p *VocalStripPlugin) CreateProcessor() vst3plugin.Processor {
	return NewVocalStripProcessor()
}

// Parameter IDs
const (
	// Gate section
	ParamGateEnable uint32 = iota
	ParamGateThreshold
	ParamGateRange
	
	// Compressor section
	ParamCompEnable
	ParamCompThreshold
	ParamCompRatio
	ParamCompAttack
	ParamCompRelease
	
	// EQ section
	ParamEQEnable
	ParamEQLowFreq
	ParamEQLowGain
	ParamEQMidFreq
	ParamEQMidGain
	ParamEQMidQ
	ParamEQHighFreq
	ParamEQHighGain
	
	// Limiter section
	ParamLimiterEnable
	ParamLimiterCeiling
	
	// Output
	ParamOutputGain
	ParamGainReduction
)

// Parameter range constants
const (
	// Threshold and gain ranges
	minThresholdDB = -60.0
	maxThresholdDB = 0.0
	minGainDB = -12.0
	maxGainDB = 12.0
	
	// Gate specific
	minGateRangeDB = -60.0
	maxGateRangeDB = 0.0
	
	// Compressor specific
	minRatio = 1.0
	maxRatio = 20.0
	minAttackSec = 0.001
	maxAttackSec = 0.1
	minReleaseSec = 0.01
	maxReleaseSec = 1.0
	
	// EQ specific
	minEQFreq = 20.0
	maxEQFreqLow = 1000.0
	maxEQFreqMid = 10000.0
	maxEQFreqHigh = 20000.0
	minQ = 0.1
	maxQ = 10.0
	
	// Limiter specific
	minCeilingDB = -3.0
	maxCeilingDB = 0.0
	
	// Output
	minOutputGainDB = -12.0
	maxOutputGainDB = 12.0
)

// VocalStripProcessor implements the audio processing
type VocalStripProcessor struct {
	// DSP - stereo processing chains
	gateL, gateR           *dynamics.Gate
	compressorL, compressorR *dynamics.Compressor
	eqLowL, eqLowR         *filter.Biquad
	eqMidL, eqMidR         *filter.Biquad
	eqHighL, eqHighR       *filter.Biquad
	limiterL, limiterR     *dynamics.Limiter
	
	// Parameters
	params *param.Registry
	buses  *bus.Configuration
	
	// Bypass states
	gateEnable      bool
	compEnable      bool
	eqEnable        bool
	limiterEnable   bool
	
	// Parameter values
	outputGain      float64
	
	// EQ parameters
	eqLowFreq       float64
	eqLowGain       float64
	eqMidFreq       float64
	eqMidGain       float64
	eqMidQ          float64
	eqHighFreq      float64
	eqHighGain      float64
	
	// State
	sampleRate float64
	active     bool
}

// NewVocalStripProcessor creates a new processor
func NewVocalStripProcessor() *VocalStripProcessor {
	p := &VocalStripProcessor{
		params:        param.NewRegistry(),
		buses:         bus.NewStereoConfiguration(),
		gateEnable:    true,
		compEnable:    true,
		eqEnable:      true,
		limiterEnable: true,
		outputGain:    1.0,
		eqLowFreq:     100.0,
		eqLowGain:     0.0,
		eqMidFreq:     2000.0,
		eqMidGain:     0.0,
		eqMidQ:        0.7,
		eqHighFreq:    8000.0,
		eqHighGain:    0.0,
	}
	
	// Initialize parameters
	p.initializeParameters()
	
	return p
}

func (p *VocalStripProcessor) initializeParameters() {
	// Gate section
	p.params.Add(
		param.New(ParamGateEnable, "Gate Enable").
			Range(0.0, 1.0).
			Default(1.0).
			Unit("").
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamGateThreshold, "Gate Threshold").
			Range(minThresholdDB, maxThresholdDB).
			Default(-40.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamGateRange, "Gate Range").
			Range(minGateRangeDB, maxGateRangeDB).
			Default(-40.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	// Compressor section
	p.params.Add(
		param.New(ParamCompEnable, "Comp Enable").
			Range(0.0, 1.0).
			Default(1.0).
			Unit("").
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamCompThreshold, "Comp Threshold").
			Range(minThresholdDB, maxThresholdDB).
			Default(-20.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamCompRatio, "Comp Ratio").
			Range(minRatio, maxRatio).
			Default(4.0).
			Unit(":1").
			Formatter(param.RatioFormatter, param.RatioParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamCompAttack, "Comp Attack").
			Range(minAttackSec, maxAttackSec).
			Default(0.01).
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
		param.New(ParamCompRelease, "Comp Release").
			Range(minReleaseSec, maxReleaseSec).
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
	
	// EQ section
	p.params.Add(
		param.New(ParamEQEnable, "EQ Enable").
			Range(0.0, 1.0).
			Default(1.0).
			Unit("").
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)
	
	// Low shelf
	p.params.Add(
		param.New(ParamEQLowFreq, "Low Freq").
			Range(20.0, 500.0).
			Default(100.0).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamEQLowGain, "Low Gain").
			Range(-12.0, 12.0).
			Default(0.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	// Mid peaking
	p.params.Add(
		param.New(ParamEQMidFreq, "Mid Freq").
			Range(200.0, 8000.0).
			Default(2000.0).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamEQMidGain, "Mid Gain").
			Range(-12.0, 12.0).
			Default(0.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamEQMidQ, "Mid Q").
			Range(0.1, 10.0).
			Default(0.7).
			Unit("").
			Formatter(func(value float64) string {
				return fmt.Sprintf("%.1f", value)
			}, func(text string) (float64, error) {
				var val float64
				_, err := fmt.Sscanf(text, "%f", &val)
				return val, err
			}).
			Build(),
	)
	
	// High shelf
	p.params.Add(
		param.New(ParamEQHighFreq, "High Freq").
			Range(2000.0, 20000.0).
			Default(8000.0).
			Unit("Hz").
			Formatter(param.FrequencyFormatter, param.FrequencyParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamEQHighGain, "High Gain").
			Range(-12.0, 12.0).
			Default(0.0).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	// Limiter section
	p.params.Add(
		param.New(ParamLimiterEnable, "Limiter Enable").
			Range(0.0, 1.0).
			Default(1.0).
			Unit("").
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamLimiterCeiling, "Ceiling").
			Range(minCeilingDB, maxCeilingDB).
			Default(-0.3).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
	
	// Output
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
func (p *VocalStripProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Create DSP processors for stereo
	p.gateL = dynamics.NewGate(sampleRate)
	p.gateR = dynamics.NewGate(sampleRate)
	
	p.compressorL = dynamics.NewCompressor(sampleRate)
	p.compressorR = dynamics.NewCompressor(sampleRate)
	
	p.eqLowL = filter.NewBiquad(1)
	p.eqLowR = filter.NewBiquad(1)
	p.eqMidL = filter.NewBiquad(1)
	p.eqMidR = filter.NewBiquad(1)
	p.eqHighL = filter.NewBiquad(1)
	p.eqHighR = filter.NewBiquad(1)
	
	p.limiterL = dynamics.NewLimiter(sampleRate)
	p.limiterR = dynamics.NewLimiter(sampleRate)
	
	// Configure processors with default values
	p.configureProcessors()
	
	return nil
}

// configureProcessors sets up all processors with current parameters
func (p *VocalStripProcessor) configureProcessors() {
	// Configure gates for vocals
	p.gateL.SetAttack(0.001)    // 1ms
	p.gateL.SetHold(0.010)      // 10ms
	p.gateL.SetRelease(0.050)   // 50ms
	p.gateL.SetHysteresis(3.0)  // 3dB
	
	p.gateR.SetAttack(0.001)
	p.gateR.SetHold(0.010)
	p.gateR.SetRelease(0.050)
	p.gateR.SetHysteresis(3.0)
	
	// Configure compressors for vocals
	p.compressorL.SetKnee(dynamics.KneeSoft, 2.0)
	p.compressorR.SetKnee(dynamics.KneeSoft, 2.0)
	
	// Configure EQ filters
	p.updateEQFilters()
	
	// Configure limiters
	p.limiterL.SetRelease(0.050)   // 50ms
	p.limiterL.SetTruePeak(true)
	p.limiterL.SetLookahead(0.005) // 5ms
	
	p.limiterR.SetRelease(0.050)
	p.limiterR.SetTruePeak(true)
	p.limiterR.SetLookahead(0.005)
}

// updateEQFilters updates all EQ filter coefficients
func (p *VocalStripProcessor) updateEQFilters() {
	// Low shelf
	p.eqLowL.SetLowShelf(p.sampleRate, p.eqLowFreq, 0.7, p.eqLowGain)
	p.eqLowR.SetLowShelf(p.sampleRate, p.eqLowFreq, 0.7, p.eqLowGain)
	
	// Mid peaking
	p.eqMidL.SetPeakingEQ(p.sampleRate, p.eqMidFreq, p.eqMidQ, p.eqMidGain)
	p.eqMidR.SetPeakingEQ(p.sampleRate, p.eqMidFreq, p.eqMidQ, p.eqMidGain)
	
	// High shelf
	p.eqHighL.SetHighShelf(p.sampleRate, p.eqHighFreq, 0.7, p.eqHighGain)
	p.eqHighR.SetHighShelf(p.sampleRate, p.eqHighFreq, 0.7, p.eqHighGain)
}

// ProcessAudio processes audio
func (p *VocalStripProcessor) ProcessAudio(ctx *process.Context) {
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
	
	// Copy input to output first
	copy(ctx.Output[0][:numSamples], ctx.Input[0][:numSamples])
	copy(ctx.Output[1][:numSamples], ctx.Input[1][:numSamples])
	
	// Process chain: Gate → Compressor → EQ → Limiter
	
	// 1. Gate
	if p.gateEnable {
		p.gateL.ProcessBuffer(ctx.Output[0][:numSamples], ctx.Output[0][:numSamples])
		p.gateR.ProcessBuffer(ctx.Output[1][:numSamples], ctx.Output[1][:numSamples])
	}
	
	// 2. Compressor
	if p.compEnable {
		p.compressorL.ProcessBuffer(ctx.Output[0][:numSamples], ctx.Output[0][:numSamples])
		p.compressorR.ProcessBuffer(ctx.Output[1][:numSamples], ctx.Output[1][:numSamples])
	}
	
	// 3. EQ
	if p.eqEnable {
		// Process through three bands (in-place)
		p.eqLowL.Process(ctx.Output[0][:numSamples], 0)
		p.eqLowR.Process(ctx.Output[1][:numSamples], 0)
		
		p.eqMidL.Process(ctx.Output[0][:numSamples], 0)
		p.eqMidR.Process(ctx.Output[1][:numSamples], 0)
		
		p.eqHighL.Process(ctx.Output[0][:numSamples], 0)
		p.eqHighR.Process(ctx.Output[1][:numSamples], 0)
	}
	
	// 4. Limiter
	if p.limiterEnable {
		p.limiterL.ProcessBuffer(ctx.Output[0][:numSamples], ctx.Output[0][:numSamples])
		p.limiterR.ProcessBuffer(ctx.Output[1][:numSamples], ctx.Output[1][:numSamples])
	}
	
	// 5. Output gain
	if p.outputGain != 1.0 {
		gainValue := float32(p.outputGain)
		gain.ApplyBuffer(ctx.Output[0][:numSamples], gainValue)
		gain.ApplyBuffer(ctx.Output[1][:numSamples], gainValue)
	}
	
	// Update gain reduction meter (combined from compressor and limiter)
	compGR := (p.compressorL.GetGainReduction() + p.compressorR.GetGainReduction()) / 2.0
	limGR := (p.limiterL.GetGainReduction() + p.limiterR.GetGainReduction()) / 2.0
	totalGR := -(compGR + limGR) // Negate for display
	
	if grParam := p.params.Get(ParamGainReduction); grParam != nil {
		grParam.SetPlainValue(totalGR)
	}
}

// updateParameters checks for parameter changes
func (p *VocalStripProcessor) updateParameters(ctx *process.Context) {
	// Gate parameters
	p.gateEnable = ctx.Param(ParamGateEnable) > 0.5
	
	threshold := ctx.ParamPlain(ParamGateThreshold)
	p.gateL.SetThreshold(threshold)
	p.gateR.SetThreshold(threshold)
	
	rangeDB := ctx.ParamPlain(ParamGateRange)
	p.gateL.SetRange(rangeDB)
	p.gateR.SetRange(rangeDB)
	
	// Compressor parameters
	p.compEnable = ctx.Param(ParamCompEnable) > 0.5
	
	compThreshold := ctx.ParamPlain(ParamCompThreshold)
	p.compressorL.SetThreshold(compThreshold)
	p.compressorR.SetThreshold(compThreshold)
	
	ratio := ctx.ParamPlain(ParamCompRatio)
	p.compressorL.SetRatio(ratio)
	p.compressorR.SetRatio(ratio)
	
	attack := ctx.ParamPlain(ParamCompAttack)
	p.compressorL.SetAttack(attack)
	p.compressorR.SetAttack(attack)
	
	release := ctx.ParamPlain(ParamCompRelease)
	p.compressorL.SetRelease(release)
	p.compressorR.SetRelease(release)
	
	// EQ parameters
	p.eqEnable = ctx.Param(ParamEQEnable) > 0.5
	
	// Check if any EQ parameter changed
	eqChanged := false
	
	newLowFreq := ctx.ParamPlain(ParamEQLowFreq)
	if newLowFreq != p.eqLowFreq {
		p.eqLowFreq = newLowFreq
		eqChanged = true
	}
	
	newLowGain := ctx.ParamPlain(ParamEQLowGain)
	if newLowGain != p.eqLowGain {
		p.eqLowGain = newLowGain
		eqChanged = true
	}
	
	newMidFreq := ctx.ParamPlain(ParamEQMidFreq)
	if newMidFreq != p.eqMidFreq {
		p.eqMidFreq = newMidFreq
		eqChanged = true
	}
	
	newMidGain := ctx.ParamPlain(ParamEQMidGain)
	if newMidGain != p.eqMidGain {
		p.eqMidGain = newMidGain
		eqChanged = true
	}
	
	newMidQ := ctx.ParamPlain(ParamEQMidQ)
	if newMidQ != p.eqMidQ {
		p.eqMidQ = newMidQ
		eqChanged = true
	}
	
	newHighFreq := ctx.ParamPlain(ParamEQHighFreq)
	if newHighFreq != p.eqHighFreq {
		p.eqHighFreq = newHighFreq
		eqChanged = true
	}
	
	newHighGain := ctx.ParamPlain(ParamEQHighGain)
	if newHighGain != p.eqHighGain {
		p.eqHighGain = newHighGain
		eqChanged = true
	}
	
	if eqChanged {
		p.updateEQFilters()
	}
	
	// Limiter parameters
	p.limiterEnable = ctx.Param(ParamLimiterEnable) > 0.5
	
	ceiling := ctx.ParamPlain(ParamLimiterCeiling)
	p.limiterL.SetThreshold(ceiling)
	p.limiterR.SetThreshold(ceiling)
	
	// Output gain
	outputGainDB := ctx.ParamPlain(ParamOutputGain)
	p.outputGain = gain.DbToLinear(outputGainDB)
}

// GetParameters returns the parameter registry
func (p *VocalStripProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *VocalStripProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// SetActive is called when processing starts/stops
func (p *VocalStripProcessor) SetActive(active bool) error {
	p.active = active
	if !active {
		// Reset all processors
		if p.gateL != nil {
			p.gateL.Reset()
			p.gateR.Reset()
		}
		if p.compressorL != nil {
			p.compressorL.Reset()
			p.compressorR.Reset()
		}
		if p.eqLowL != nil {
			p.eqLowL.Reset()
			p.eqLowR.Reset()
			p.eqMidL.Reset()
			p.eqMidR.Reset()
			p.eqHighL.Reset()
			p.eqHighR.Reset()
		}
		if p.limiterL != nil {
			p.limiterL.Reset()
			p.limiterR.Reset()
		}
	}
	return nil
}

// GetLatencySamples returns the plugin latency in samples
func (p *VocalStripProcessor) GetLatencySamples() int32 {
	// Only the limiter adds latency (lookahead)
	return int32(0.005 * p.sampleRate) // 5ms
}

// GetTailSamples returns the tail length in samples
func (p *VocalStripProcessor) GetTailSamples() int32 {
	// Maximum of all release times
	maxRelease := 0.1 // 100ms compressor release
	return int32(maxRelease * p.sampleRate)
}

