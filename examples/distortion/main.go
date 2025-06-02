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

// MultiDistortionPlugin implements the Plugin interface
type MultiDistortionPlugin struct{}

func (m *MultiDistortionPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.multidistortion",
		Name:     "MultiDistortion",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx",
	}
}

func (m *MultiDistortionPlugin) CreateProcessor() vst3plugin.Processor {
	return NewMultiDistortionProcessor()
}

// Parameter IDs
const (
	ParamDistortionType = iota
	ParamDrive
	ParamMix
	ParamOutput
	ParamWaveshaperCurve
	ParamTubeWarmth
	ParamTubeHarmonics
	ParamTapeSaturation
	ParamTapeCompression
	ParamTapeFlutter
	ParamBitDepth
	ParamSampleRate
	ParamDither
)

// Distortion types - using param package constants
const (
	DistTypeWaveshaper = param.DistortionTypeWaveshaper
	DistTypeTube       = param.DistortionTypeTube
	DistTypeTape       = param.DistortionTypeTape
	DistTypeBitCrusher = param.DistortionTypeBitCrusher
)

// MultiDistortionProcessor demonstrates all distortion types
type MultiDistortionProcessor struct {
	params *param.Registry
	buses  *bus.Configuration
	
	// Current distortion type
	distType int
	
	// Distortion processors
	waveshaper  *distortion.Waveshaper
	tube        *distortion.TubeSaturator
	tape        *distortion.TapeSaturator
	bitcrusher  *distortion.BitCrusher
	
	// Common parameters
	drive      float64
	mix        float64
	outputGain float64
	
	// Type-specific parameters
	waveshaperCurve   distortion.CurveType
	tubeWarmth        float64
	tubeHarmonics     float64
	tapeSaturation    float64
	tapeCompression   float64
	tapeFlutter       float64
	bitDepth          int
	sampleRateRatio   float64
	ditherAmount      float64
	
	// Temporary buffers for float conversion
	tempInput  []float64
	tempOutput []float64
}

// NewMultiDistortionProcessor creates a new multi-distortion processor
func NewMultiDistortionProcessor() *MultiDistortionProcessor {
	p := &MultiDistortionProcessor{
		params:          param.NewRegistry(),
		buses:           bus.NewStereoConfiguration(),
		drive:           1.0,
		mix:             1.0,
		outputGain:      1.0,
		waveshaperCurve: distortion.CurveSoftClip,
		tubeWarmth:      0.5,
		tubeHarmonics:   0.5,
		tapeSaturation:  0.5,
		tapeCompression: 0.3,
		tapeFlutter:     0.0,
		bitDepth:        16,
		sampleRateRatio: 1.0,
		ditherAmount:    0.0,
	}
	
	p.setupParameters()
	return p
}

func (p *MultiDistortionProcessor) setupParameters() {
	// Main controls
	p.params.Add(
		param.New(ParamDistortionType, "Type").
			Range(0, 3).
			Default(0).
			Steps(4).
			Formatter(param.DistortionTypeFormatter, param.DistortionTypeParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamDrive, "Drive").
			Range(0, 100).
			Default(20).
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
	
	// Waveshaper parameters
	p.params.Add(
		param.New(ParamWaveshaperCurve, "Curve").
			Range(0, 6).
			Default(0).
			Steps(7).
			Formatter(param.WaveshaperCurveFormatter, param.WaveshaperCurveParser).
			Build(),
	)
	
	// Tube parameters
	p.params.Add(
		param.New(ParamTubeWarmth, "Warmth").
			Range(0, 100).
			Default(50).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamTubeHarmonics, "Even/Odd").
			Range(0, 100).
			Default(50).
			Unit("%").
			Build(),
	)
	
	// Tape parameters
	p.params.Add(
		param.New(ParamTapeSaturation, "Saturation").
			Range(0, 100).
			Default(50).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamTapeCompression, "Compression").
			Range(0, 100).
			Default(30).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamTapeFlutter, "Flutter").
			Range(0, 100).
			Default(0).
			Unit("%").
			Build(),
	)
	
	// BitCrusher parameters
	p.params.Add(
		param.New(ParamBitDepth, "Bit Depth").
			Range(1, 24).
			Default(16).
			Unit("bits").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamSampleRate, "Sample Rate").
			Range(0, 100).
			Default(100).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamDither, "Dither").
			Range(0, 100).
			Default(0).
			Unit("%").
			Build(),
	)
}

// GetParameters returns the parameter registry
func (p *MultiDistortionProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *MultiDistortionProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// Initialize prepares the processor for playback
func (p *MultiDistortionProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
	// Create all distortion processors
	p.waveshaper = distortion.NewWaveshaper(distortion.CurveSoftClip)
	p.tube = distortion.NewTubeSaturator(sampleRate)
	p.tape = distortion.NewTapeSaturator(sampleRate)
	p.bitcrusher = distortion.NewBitCrusher(sampleRate)
	
	// Allocate temp buffers
	p.tempInput = make([]float64, maxSamplesPerBlock)
	p.tempOutput = make([]float64, maxSamplesPerBlock)
	
	return nil
}

// ProcessAudio processes audio buffers
func (p *MultiDistortionProcessor) ProcessAudio(ctx *process.Context) {
	// Update parameters from context
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
	
	// Process each channel
	for ch := 0; ch < numChannels; ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]
		
		// Convert float32 to float64
		for i := 0; i < numSamples; i++ {
			p.tempInput[i] = float64(input[i])
		}
		
		// Apply selected distortion type
		switch p.distType {
		case DistTypeWaveshaper:
			p.waveshaper.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		case DistTypeTube:
			p.tube.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		case DistTypeTape:
			p.tape.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		case DistTypeBitCrusher:
			p.bitcrusher.ProcessBuffer(p.tempInput[:numSamples], p.tempOutput[:numSamples])
		}
		
		// Convert back to float32 and apply output gain
		for i := 0; i < numSamples; i++ {
			output[i] = float32(p.tempOutput[i] * p.outputGain)
		}
	}
}

// updateParameters updates internal state from parameter values
func (p *MultiDistortionProcessor) updateParameters(ctx *process.Context) {
	// Get parameter values
	p.distType = int(ctx.ParamPlain(ParamDistortionType))
	
	drive := ctx.ParamPlain(ParamDrive) / 100.0 // Convert from percentage
	p.drive = 1.0 + drive*9.0 // 1-10
	p.waveshaper.SetDrive(p.drive)
	p.tube.SetDrive(p.drive)
	p.tape.SetDrive(p.drive)
	
	p.mix = ctx.ParamPlain(ParamMix) / 100.0
	p.waveshaper.SetMix(p.mix)
	p.tube.SetMix(p.mix)
	p.tape.SetMix(p.mix)
	p.bitcrusher.SetMix(p.mix)
	
	// Convert dB to linear
	outputDB := ctx.ParamPlain(ParamOutput)
	p.outputGain = math.Pow(10.0, outputDB/20.0)
	
	// Waveshaper params
	curveType := distortion.CurveType(int(ctx.ParamPlain(ParamWaveshaperCurve)))
	p.waveshaper.SetCurveType(curveType)
	
	// Tube params
	p.tubeWarmth = ctx.ParamPlain(ParamTubeWarmth) / 100.0
	p.tube.SetWarmth(p.tubeWarmth)
	
	p.tubeHarmonics = ctx.ParamPlain(ParamTubeHarmonics) / 100.0
	p.tube.SetHarmonicBalance(p.tubeHarmonics)
	
	// Tape params
	p.tapeSaturation = ctx.ParamPlain(ParamTapeSaturation) / 100.0
	p.tape.SetSaturation(p.tapeSaturation)
	
	p.tapeCompression = ctx.ParamPlain(ParamTapeCompression) / 100.0
	p.tape.SetCompression(p.tapeCompression)
	
	p.tapeFlutter = ctx.ParamPlain(ParamTapeFlutter) / 100.0
	p.tape.SetFlutter(p.tapeFlutter)
	
	// BitCrusher params
	p.bitDepth = int(ctx.ParamPlain(ParamBitDepth))
	p.bitcrusher.SetBitDepth(p.bitDepth)
	
	p.sampleRateRatio = ctx.ParamPlain(ParamSampleRate) / 100.0
	p.bitcrusher.SetSampleRateRatio(p.sampleRateRatio)
	
	p.ditherAmount = ctx.ParamPlain(ParamDither) / 100.0
	p.bitcrusher.SetDither(p.ditherAmount)
}

// SetActive is called when plugin becomes active/inactive
func (p *MultiDistortionProcessor) SetActive(active bool) error {
	return nil
}

// GetLatencySamples returns the plugin latency
func (p *MultiDistortionProcessor) GetLatencySamples() int32 {
	return 0
}

// GetTailSamples returns the tail length
func (p *MultiDistortionProcessor) GetTailSamples() int32 {
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
	vst3plugin.Register(&MultiDistortionPlugin{})
}

// Required for c-shared build mode
func main() {}