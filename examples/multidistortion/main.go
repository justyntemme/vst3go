package main

import (
	"fmt"

	"github.com/justyntemme/vst3go/pkg/dsp/distortion"
	"github.com/justyntemme/vst3go/pkg/dsp/gain"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
	
	// Import C bridge - required for VST3 plugin to work
	_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
)

// MultiDistortionPlugin implements the Plugin interface
type MultiDistortionPlugin struct{}

func (m *MultiDistortionPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.multidistortion",
		Name:     "Multi-Distortion",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Distortion",
	}
}

func (m *MultiDistortionPlugin) CreateProcessor() vst3plugin.Processor {
	return NewMultiDistortionProcessor()
}

// MultiDistortionProcessor handles the audio processing
type MultiDistortionProcessor struct {
	params *param.Registry
	buses  *bus.Configuration

	// DSP state
	waveshaper *distortion.Waveshaper
	tube       *distortion.TubeSaturation
	tape       *distortion.TapeSaturation
	bitcrusher *distortion.Bitcrusher

	sampleRate float64
}

// Parameter IDs
const (
	ParamDistortionType = 0
	ParamDrive          = 1
	ParamMix            = 2
	ParamOutput         = 3

	// Waveshaper params
	ParamWaveCurve     = 10
	ParamWaveAsymmetry = 11

	// Tube params
	ParamTubeWarmth     = 20
	ParamTubeHarmonics  = 21
	ParamTubeBias       = 22
	ParamTubeHysteresis = 23

	// Tape params
	ParamTapeSaturation  = 30
	ParamTapeCompression = 31
	ParamTapeFlutter     = 32
	ParamTapeWarmth      = 33

	// Bitcrusher params
	ParamBitDepth        = 40
	ParamSampleReduction = 41
	ParamDither          = 42
	ParamAntiAlias       = 43
)

// Distortion types
const (
	DistortionWaveshaper = 0
	DistortionTube       = 1
	DistortionTape       = 2
	DistortionBitcrusher = 3
)

func NewMultiDistortionProcessor() *MultiDistortionProcessor {
	p := &MultiDistortionProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewStereoConfiguration(),
		waveshaper: distortion.NewWaveshaper(),
		tube:       distortion.NewTubeSaturation(),
		tape:       distortion.NewTapeSaturation(48000),
		bitcrusher: distortion.NewBitcrusher(48000),
		sampleRate: 48000,
	}

	// Main parameters
	p.params.Add(
		param.Choice(ParamDistortionType, "Type", []param.ChoiceOption{
			{Value: 0, Name: "Waveshaper", Aliases: []string{"wave", "shaper"}},
			{Value: 1, Name: "Tube", Aliases: []string{"valve", "triode"}},
			{Value: 2, Name: "Tape", Aliases: []string{"analog", "vintage"}},
			{Value: 3, Name: "Bitcrusher", Aliases: []string{"digital", "lofi", "crush"}},
		}).Build(),

		param.New(ParamDrive, "Drive").
			Range(1, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.MixParameter(ParamMix, "Mix").Build(),
		param.GainParameter(ParamOutput, "Output").Build(),
	)

	// Waveshaper parameters
	p.params.Add(
		param.Choice(ParamWaveCurve, "Curve", []param.ChoiceOption{
			{Value: 0, Name: "Hard Clip", Aliases: []string{"hard", "clip"}},
			{Value: 1, Name: "Soft Clip", Aliases: []string{"soft", "tanh"}},
			{Value: 2, Name: "Saturate", Aliases: []string{"sat"}},
			{Value: 3, Name: "Foldback", Aliases: []string{"fold", "wrap"}},
			{Value: 4, Name: "Asymmetric", Aliases: []string{"asym"}},
			{Value: 5, Name: "Sine", Aliases: []string{"sin"}},
			{Value: 6, Name: "Exponential", Aliases: []string{"exp"}},
		}).Build(),

		param.New(ParamWaveAsymmetry, "Asymmetry").
			Range(-100, 100).
			Default(0).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)

	// Tube parameters
	p.params.Add(
		param.New(ParamTubeWarmth, "Warmth").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamTubeHarmonics, "Harmonics").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamTubeBias, "Bias").
			Range(-100, 100).
			Default(0).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamTubeHysteresis, "Hysteresis").
			Range(0, 100).
			Default(10).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)

	// Tape parameters
	p.params.Add(
		param.New(ParamTapeSaturation, "Saturation").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamTapeCompression, "Compression").
			Range(0, 100).
			Default(30).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamTapeFlutter, "Flutter").
			Range(0, 100).
			Default(0).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamTapeWarmth, "Tape Warmth").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)

	// Bitcrusher parameters
	p.params.Add(
		param.New(ParamBitDepth, "Bit Depth").
			Range(1, 32).
			Default(8).
			Steps(32).
			Unit("bits").
			Build(),

		param.New(ParamSampleReduction, "Sample Rate").
			Range(1, 100).
			Default(1).
			Formatter(func(v float64) string {
				if v == 1 {
					return "Full"
				}
				return fmt.Sprintf("1/%d", int(v))
			}, nil).
			Build(),

		param.Choice(ParamDither, "Dither", []param.ChoiceOption{
			{Value: 0, Name: "None", Aliases: []string{"off"}},
			{Value: 1, Name: "White", Aliases: []string{"white noise"}},
			{Value: 2, Name: "Triangular", Aliases: []string{"tri"}},
		}).Build(),

		param.New(ParamAntiAlias, "Anti-Alias").
			Range(0, 1).
			Default(1).
			Steps(2).
			Formatter(param.OnOffFormatter, param.OnOffParser).
			Build(),
	)

	return p
}

func (p *MultiDistortionProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	p.tape = distortion.NewTapeSaturation(sampleRate)
	p.bitcrusher = distortion.NewBitcrusher(sampleRate)
	return nil
}

func (p *MultiDistortionProcessor) ProcessAudio(ctx *process.Context) {
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}

	numSamples := ctx.NumSamples()
	if numChannels == 0 || numSamples == 0 {
		return
	}

	// Get common parameters
	distType := int(ctx.ParamPlain(ParamDistortionType))
	drive := float32(ctx.ParamPlain(ParamDrive) / 100.0)
	mix := float32(ctx.ParamPlain(ParamMix) / 100.0)
	outputGain := float32(ctx.ParamPlain(ParamOutput))

	// Convert output gain from dB to linear using DSP library
	outputLinear := gain.DbToLinear32(outputGain)

	// Process based on distortion type
	switch distType {
	case DistortionWaveshaper:
		p.processWaveshaper(ctx, drive, mix, outputLinear)
	case DistortionTube:
		p.processTube(ctx, drive, mix, outputLinear)
	case DistortionTape:
		p.processTape(ctx, drive, mix, outputLinear)
	case DistortionBitcrusher:
		p.processBitcrusher(ctx, drive, mix, outputLinear)
	}
}

func (p *MultiDistortionProcessor) processWaveshaper(ctx *process.Context, drive, mix, outputGain float32) {
	// Get waveshaper-specific parameters
	curve := int(ctx.ParamPlain(ParamWaveCurve))
	asymmetry := ctx.ParamPlain(ParamWaveAsymmetry) / 100.0

	// Configure waveshaper
	p.waveshaper.SetCurveType(distortion.CurveType(curve))
	p.waveshaper.SetDrive(float64(1.0 + drive*9.0)) // 1-10x drive
	p.waveshaper.SetAsymmetry(asymmetry)
	p.waveshaper.SetMix(float64(mix))
	p.waveshaper.SetOutput(float64(outputGain))

	// Process each channel
	for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]

		for i := range input {
			output[i] = float32(p.waveshaper.Process(float64(input[i])))
		}
	}
}

func (p *MultiDistortionProcessor) processTube(ctx *process.Context, drive, mix, outputGain float32) {
	// Get tube-specific parameters
	warmth := ctx.ParamPlain(ParamTubeWarmth) / 100.0
	harmonics := ctx.ParamPlain(ParamTubeHarmonics) / 100.0
	bias := ctx.ParamPlain(ParamTubeBias) / 100.0
	hysteresis := ctx.ParamPlain(ParamTubeHysteresis) / 100.0

	// Configure tube
	p.tube.SetWarmth(warmth)
	p.tube.SetHarmonics(harmonics * float64(drive)) // Scale harmonics with drive
	p.tube.SetBias(bias)
	p.tube.SetHysteresis(hysteresis)
	p.tube.SetMix(float64(mix))
	p.tube.SetOutput(float64(outputGain))

	// Process each channel
	for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]

		for i := range input {
			// Apply drive as input gain
			driven := input[i] * (1.0 + drive*2.0)
			output[i] = float32(p.tube.Process(float64(driven)))
		}
	}
}

func (p *MultiDistortionProcessor) processTape(ctx *process.Context, drive, mix, outputGain float32) {
	// Get tape-specific parameters
	saturation := ctx.ParamPlain(ParamTapeSaturation) / 100.0
	compression := ctx.ParamPlain(ParamTapeCompression) / 100.0
	flutter := ctx.ParamPlain(ParamTapeFlutter) / 100.0
	warmth := ctx.ParamPlain(ParamTapeWarmth) / 100.0

	// Configure tape
	p.tape.SetSaturation(saturation * float64(drive)) // Scale saturation with drive
	p.tape.SetCompression(compression)
	p.tape.SetFlutter(flutter)
	p.tape.SetWarmth(warmth)
	p.tape.SetMix(float64(mix))
	p.tape.SetOutput(float64(outputGain))

	// Process stereo
	if ctx.NumInputChannels() >= 2 && ctx.NumOutputChannels() >= 2 {
		inputL := make([]float64, len(ctx.Input[0]))
		inputR := make([]float64, len(ctx.Input[1]))
		outputL := make([]float64, len(ctx.Output[0]))
		outputR := make([]float64, len(ctx.Output[1]))

		// Convert to float64
		for i := range inputL {
			inputL[i] = float64(ctx.Input[0][i])
			inputR[i] = float64(ctx.Input[1][i])
		}

		// Process
		p.tape.ProcessStereo(inputL, inputR, outputL, outputR)

		// Convert back
		for i := range outputL {
			ctx.Output[0][i] = float32(outputL[i])
			ctx.Output[1][i] = float32(outputR[i])
		}
	} else if ctx.NumInputChannels() >= 1 && ctx.NumOutputChannels() >= 1 {
		// Mono processing
		input := ctx.Input[0]
		output := ctx.Output[0]

		for i := range input {
			output[i] = float32(p.tape.Process(float64(input[i])))
		}
	}
}

func (p *MultiDistortionProcessor) processBitcrusher(ctx *process.Context, drive, mix, outputGain float32) {
	// Get bitcrusher-specific parameters
	bitDepth := ctx.ParamPlain(ParamBitDepth)
	sampleReduction := ctx.ParamPlain(ParamSampleReduction)
	dither := int(ctx.ParamPlain(ParamDither))
	antiAlias := ctx.ParamPlain(ParamAntiAlias) > 0.5

	// Configure bitcrusher
	p.bitcrusher.SetBitDepth(bitDepth - (bitDepth-1)*float64(drive)*0.8) // Drive reduces bit depth
	p.bitcrusher.SetSampleRateReduction(sampleReduction)
	p.bitcrusher.SetDither(distortion.DitherType(dither))
	p.bitcrusher.SetAntiAlias(antiAlias)
	p.bitcrusher.SetMix(float64(mix))
	p.bitcrusher.SetOutput(float64(outputGain))

	// Process stereo
	if ctx.NumInputChannels() >= 2 && ctx.NumOutputChannels() >= 2 {
		inputL := make([]float64, len(ctx.Input[0]))
		inputR := make([]float64, len(ctx.Input[1]))
		outputL := make([]float64, len(ctx.Output[0]))
		outputR := make([]float64, len(ctx.Output[1]))

		// Convert to float64
		for i := range inputL {
			inputL[i] = float64(ctx.Input[0][i])
			inputR[i] = float64(ctx.Input[1][i])
		}

		// Process
		p.bitcrusher.ProcessStereo(inputL, inputR, outputL, outputR)

		// Convert back
		for i := range outputL {
			ctx.Output[0][i] = float32(outputL[i])
			ctx.Output[1][i] = float32(outputR[i])
		}
	} else if ctx.NumInputChannels() >= 1 && ctx.NumOutputChannels() >= 1 {
		// Mono processing
		input := ctx.Input[0]
		output := ctx.Output[0]

		for i := range input {
			output[i] = float32(p.bitcrusher.Process(float64(input[i])))
		}
	}
}

func (p *MultiDistortionProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *MultiDistortionProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *MultiDistortionProcessor) SetActive(active bool) error {
	if !active {
		p.waveshaper = distortion.NewWaveshaper()
		p.tube.Reset()
		p.tape.Reset()
		p.bitcrusher.Reset()
	}
	return nil
}

func (p *MultiDistortionProcessor) GetLatencySamples() int32 {
	return 0
}

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
