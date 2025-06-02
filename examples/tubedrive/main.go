package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/distortion"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// TubeDrivePlugin implements the Plugin interface
type TubeDrivePlugin struct{}

func (t *TubeDrivePlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.tubedrive",
		Name:     "TubeDrive",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx",
	}
}

func (t *TubeDrivePlugin) CreateProcessor() vst3plugin.Processor {
	return NewTubeDriveProcessor()
}

// Parameter IDs
const (
	ParamDrive = iota
	ParamBias
	ParamWarmth
	ParamHarmonicBalance
	ParamHysteresis
	ParamMix
	ParamOutput
)

// TubeDriveProcessor is a dedicated tube saturation processor
type TubeDriveProcessor struct {
	params *param.Registry
	buses  *bus.Configuration
	
	tubeL *distortion.TubeSaturator
	tubeR *distortion.TubeSaturator
	
	// Parameters
	drive           float64
	bias            float64
	warmth          float64
	harmonicBalance float64
	hysteresis      float64
	mix             float64
	outputGain      float64
	
	// Temporary buffers for float conversion
	tempInput  []float64
	tempOutput []float64
}

// NewTubeDriveProcessor creates a new tube drive processor
func NewTubeDriveProcessor() *TubeDriveProcessor {
	p := &TubeDriveProcessor{
		params:          param.NewRegistry(),
		buses:           bus.NewStereoConfiguration(),
		drive:           1.0,
		bias:            0.0,
		warmth:          0.5,
		harmonicBalance: 0.5,
		hysteresis:      0.1,
		mix:             1.0,
		outputGain:      1.0,
	}
	
	p.setupParameters()
	return p
}

func (p *TubeDriveProcessor) setupParameters() {
	p.params.Add(
		param.New(ParamDrive, "Drive").
			Range(1, 10).
			Default(2.8).
			Unit("").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamBias, "Bias").
			Range(-100, 100).
			Default(0).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamWarmth, "Warmth").
			Range(0, 100).
			Default(50).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamHarmonicBalance, "Even/Odd").
			Range(0, 100).
			Default(70). // Slightly more even harmonics
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamHysteresis, "Hysteresis").
			Range(0, 100).
			Default(10).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamMix, "Mix").
			Range(0, 100).
			Default(100).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamOutput, "Output").
			Range(-24, 24).
			Default(0).
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),
	)
}

// GetParameters returns the parameter registry
func (p *TubeDriveProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *TubeDriveProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// Initialize prepares the processor for playback
func (p *TubeDriveProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
	// Create tube saturators for each channel
	p.tubeL = distortion.NewTubeSaturator(sampleRate)
	p.tubeR = distortion.NewTubeSaturator(sampleRate)
	
	// Allocate temp buffers
	p.tempInput = make([]float64, maxSamplesPerBlock)
	p.tempOutput = make([]float64, maxSamplesPerBlock)
	
	// Set initial parameters
	p.updateTubeParameters(p.tubeL)
	p.updateTubeParameters(p.tubeR)
	
	return nil
}

// ProcessAudio processes audio through tube saturation
func (p *TubeDriveProcessor) ProcessAudio(ctx *process.Context) {
	// Update parameters
	p.updateParameters(ctx)
	
	// Get channel count
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	
	if numChannels == 0 {
		return
	}
	
	numSamples := len(ctx.Input[0])
	
	// Process left channel
	if numChannels >= 1 {
		input := ctx.Input[0]
		output := ctx.Output[0]
		
		// Convert float32 to float64
		for i := 0; i < numSamples; i++ {
			p.tempInput[i] = float64(input[i])
		}
		
		p.tubeL.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		
		// Convert back to float32 and apply output gain
		for i := 0; i < numSamples; i++ {
			output[i] = float32(p.tempOutput[i] * p.outputGain)
		}
	}
	
	// Process right channel (or copy mono to stereo)
	if numChannels >= 2 && len(ctx.Output) >= 2 {
		input := ctx.Input[1]
		output := ctx.Output[1]
		
		// Convert float32 to float64
		for i := 0; i < numSamples; i++ {
			p.tempInput[i] = float64(input[i])
		}
		
		p.tubeR.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		
		// Convert back to float32 and apply output gain
		for i := 0; i < numSamples; i++ {
			output[i] = float32(p.tempOutput[i] * p.outputGain)
		}
	} else if numChannels == 1 && len(ctx.Output) >= 2 {
		// Mono to stereo
		copy(ctx.Output[1], ctx.Output[0])
	}
}

// updateParameters updates internal state from parameter values
func (p *TubeDriveProcessor) updateParameters(ctx *process.Context) {
	p.drive = ctx.ParamPlain(ParamDrive)
	p.tubeL.SetDrive(p.drive)
	p.tubeR.SetDrive(p.drive)
	
	p.bias = ctx.ParamPlain(ParamBias) / 100.0 // Convert percentage to -1 to 1
	p.tubeL.SetBias(p.bias)
	p.tubeR.SetBias(p.bias)
	
	p.warmth = ctx.ParamPlain(ParamWarmth) / 100.0
	p.tubeL.SetWarmth(p.warmth)
	p.tubeR.SetWarmth(p.warmth)
	
	p.harmonicBalance = ctx.ParamPlain(ParamHarmonicBalance) / 100.0
	p.tubeL.SetHarmonicBalance(p.harmonicBalance)
	p.tubeR.SetHarmonicBalance(p.harmonicBalance)
	
	p.hysteresis = ctx.ParamPlain(ParamHysteresis) / 100.0
	p.tubeL.SetHysteresis(p.hysteresis)
	p.tubeR.SetHysteresis(p.hysteresis)
	
	p.mix = ctx.ParamPlain(ParamMix) / 100.0
	p.tubeL.SetMix(p.mix)
	p.tubeR.SetMix(p.mix)
	
	// Convert dB to linear
	outputDB := ctx.ParamPlain(ParamOutput)
	p.outputGain = math.Pow(10.0, outputDB/20.0)
}

// updateTubeParameters applies all current parameters to a tube instance
func (p *TubeDriveProcessor) updateTubeParameters(tube *distortion.TubeSaturator) {
	tube.SetDrive(p.drive)
	tube.SetBias(p.bias)
	tube.SetWarmth(p.warmth)
	tube.SetHarmonicBalance(p.harmonicBalance)
	tube.SetHysteresis(p.hysteresis)
	tube.SetMix(p.mix)
}

// SetActive is called when plugin becomes active/inactive
func (p *TubeDriveProcessor) SetActive(active bool) error {
	return nil
}

// GetLatencySamples returns the plugin latency
func (p *TubeDriveProcessor) GetLatencySamples() int32 {
	return 0
}

// GetTailSamples returns the tail length
func (p *TubeDriveProcessor) GetTailSamples() int32 {
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
	vst3plugin.Register(&TubeDrivePlugin{})
}

// Required for c-shared build mode
func main() {}