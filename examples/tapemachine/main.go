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

// TapeMachinePlugin implements the Plugin interface
type TapeMachinePlugin struct{}

func (t *TapeMachinePlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.tapemachine",
		Name:     "TapeMachine",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx",
	}
}

func (t *TapeMachinePlugin) CreateProcessor() vst3plugin.Processor {
	return NewTapeMachineProcessor()
}

// Parameter IDs
const (
	ParamDrive = iota
	ParamSaturation
	ParamBias
	ParamCompression
	ParamWarmth
	ParamFlutter
	ParamMix
	ParamOutput
)

// TapeMachineProcessor is a vintage tape saturation processor
type TapeMachineProcessor struct {
	params *param.Registry
	buses  *bus.Configuration
	
	tapeL *distortion.TapeSaturator
	tapeR *distortion.TapeSaturator
	
	// Parameters
	drive       float64
	saturation  float64
	bias        float64
	compression float64
	warmth      float64
	flutter     float64
	mix         float64
	outputGain  float64
	
	// Temporary buffers for float conversion
	tempInput  []float64
	tempOutput []float64
}

// NewTapeMachineProcessor creates a new tape machine processor
func NewTapeMachineProcessor() *TapeMachineProcessor {
	p := &TapeMachineProcessor{
		params:      param.NewRegistry(),
		buses:       bus.NewStereoConfiguration(),
		drive:       1.0,
		saturation:  0.5,
		bias:        0.15,
		compression: 0.3,
		warmth:      0.5,
		flutter:     0.05,
		mix:         1.0,
		outputGain:  1.0,
	}
	
	p.setupParameters()
	return p
}

func (p *TapeMachineProcessor) setupParameters() {
	p.params.Add(
		param.New(ParamDrive, "Input Drive").
			Range(1, 10).
			Default(3).
			Unit("").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamSaturation, "Saturation").
			Range(0, 100).
			Default(50).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamBias, "Tape Bias").
			Range(0, 100).
			Default(15).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamCompression, "Compression").
			Range(0, 100).
			Default(30).
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
		param.New(ParamFlutter, "Wow & Flutter").
			Range(0, 100).
			Default(5).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamMix, "Mix").
			Range(0, 100).
			Default(80). // Default to 80% wet
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
func (p *TapeMachineProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *TapeMachineProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// Initialize prepares the processor for playback
func (p *TapeMachineProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
	// Create tape saturators for each channel
	p.tapeL = distortion.NewTapeSaturator(sampleRate)
	p.tapeR = distortion.NewTapeSaturator(sampleRate)
	
	// Allocate temp buffers
	p.tempInput = make([]float64, maxSamplesPerBlock)
	p.tempOutput = make([]float64, maxSamplesPerBlock)
	
	// Set initial parameters
	p.updateTapeParameters(p.tapeL)
	p.updateTapeParameters(p.tapeR)
	
	// Add slight variations between channels for stereo width
	p.tapeR.SetFlutter(p.flutter * 1.1) // Slightly different flutter on right channel
	
	return nil
}

// ProcessAudio processes audio through tape saturation
func (p *TapeMachineProcessor) ProcessAudio(ctx *process.Context) {
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
		
		p.tapeL.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		
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
		
		p.tapeR.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		
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
func (p *TapeMachineProcessor) updateParameters(ctx *process.Context) {
	p.drive = ctx.ParamPlain(ParamDrive)
	p.tapeL.SetDrive(p.drive)
	p.tapeR.SetDrive(p.drive)
	
	p.saturation = ctx.ParamPlain(ParamSaturation) / 100.0
	p.tapeL.SetSaturation(p.saturation)
	p.tapeR.SetSaturation(p.saturation)
	
	p.bias = ctx.ParamPlain(ParamBias) / 100.0
	p.tapeL.SetBias(p.bias)
	p.tapeR.SetBias(p.bias)
	
	p.compression = ctx.ParamPlain(ParamCompression) / 100.0
	p.tapeL.SetCompression(p.compression)
	p.tapeR.SetCompression(p.compression)
	
	p.warmth = ctx.ParamPlain(ParamWarmth) / 100.0
	p.tapeL.SetWarmth(p.warmth)
	p.tapeR.SetWarmth(p.warmth)
	
	p.flutter = ctx.ParamPlain(ParamFlutter) / 100.0
	p.tapeL.SetFlutter(p.flutter)
	p.tapeR.SetFlutter(p.flutter * 1.1) // Slight variation for stereo
	
	p.mix = ctx.ParamPlain(ParamMix) / 100.0
	p.tapeL.SetMix(p.mix)
	p.tapeR.SetMix(p.mix)
	
	// Convert dB to linear
	outputDB := ctx.ParamPlain(ParamOutput)
	p.outputGain = math.Pow(10.0, outputDB/20.0)
}

// updateTapeParameters applies all current parameters to a tape instance
func (p *TapeMachineProcessor) updateTapeParameters(tape *distortion.TapeSaturator) {
	tape.SetDrive(p.drive)
	tape.SetSaturation(p.saturation)
	tape.SetBias(p.bias)
	tape.SetCompression(p.compression)
	tape.SetWarmth(p.warmth)
	tape.SetFlutter(p.flutter)
	tape.SetMix(p.mix)
}

// SetActive is called when plugin becomes active/inactive
func (p *TapeMachineProcessor) SetActive(active bool) error {
	return nil
}

// GetLatencySamples returns the plugin latency
func (p *TapeMachineProcessor) GetLatencySamples() int32 {
	return 0
}

// GetTailSamples returns the tail length
func (p *TapeMachineProcessor) GetTailSamples() int32 {
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
	vst3plugin.Register(&TapeMachinePlugin{})
}

// Required for c-shared build mode
func main() {}