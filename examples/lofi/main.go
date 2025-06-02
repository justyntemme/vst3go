package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"math"

	"github.com/justyntemme/vst3go/pkg/dsp/distortion"
	"github.com/justyntemme/vst3go/pkg/dsp/modulation"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// LoFiPlugin implements the Plugin interface
type LoFiPlugin struct{}

func (l *LoFiPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.lofi",
		Name:     "LoFiProcessor",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx",
	}
}

func (l *LoFiPlugin) CreateProcessor() vst3plugin.Processor {
	return NewLoFiProcessor()
}

// Parameter IDs
const (
	ParamBitDepth = iota
	ParamSampleRate
	ParamAntiAlias
	ParamDither
	ParamDrive
	ParamNoise
	ParamVinyl
	ParamMix
	ParamOutput
)

// LoFiProcessor is a lo-fi degradation processor with bit crushing and vinyl simulation
type LoFiProcessor struct {
	params *param.Registry
	buses  *bus.Configuration
	
	// Bit crushers for each channel
	crusherL *distortion.BitCrusher
	crusherR *distortion.BitCrusher
	
	// Vinyl simulation
	vinylNoiseLevel float64
	vinylCrackle    float64
	lfo             *modulation.LFO // For subtle pitch wobble
	
	// Noise generator state
	noiseState uint32
	
	// Parameters
	bitDepth        int
	sampleRateRatio float64
	antiAlias       bool
	ditherAmount    float64
	drive           float64
	noiseLevel      float64
	vinylAmount     float64
	mix             float64
	outputGain      float64
	
	// Sample rate for initialization
	sampleRate      float64
	
	// Temporary buffers for processing
	tempBuffer []float64
}

// NewLoFiProcessor creates a new lo-fi processor
func NewLoFiProcessor() *LoFiProcessor {
	p := &LoFiProcessor{
		params:          param.NewRegistry(),
		buses:           bus.NewStereoConfiguration(),
		bitDepth:        12,
		sampleRateRatio: 0.5,
		antiAlias:       true,
		ditherAmount:    0.0,
		drive:           1.0,
		noiseLevel:      0.0,
		vinylAmount:     0.0,
		mix:             1.0,
		outputGain:      1.0,
		noiseState:      42,
	}
	
	p.setupParameters()
	return p
}

func (p *LoFiProcessor) setupParameters() {
	p.params.Add(
		param.New(ParamBitDepth, "Bit Depth").
			Range(1, 24).
			Default(12).
			Unit("bits").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamSampleRate, "Sample Rate").
			Range(0, 100).
			Default(50). // 50% of original
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamAntiAlias, "Anti-Alias").
			Range(0, 1).
			Default(1).
			Unit("").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamDither, "Dither").
			Range(0, 100).
			Default(0).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamDrive, "Input Drive").
			Range(1, 4).
			Default(1).
			Unit("x").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamNoise, "Noise").
			Range(0, 100).
			Default(0).
			Unit("%").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamVinyl, "Vinyl").
			Range(0, 100).
			Default(0).
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
func (p *LoFiProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *LoFiProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// Initialize prepares the processor for playback
func (p *LoFiProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
	p.sampleRate = sampleRate
	
	// Create bit crushers for each channel
	p.crusherL = distortion.NewBitCrusher(sampleRate)
	p.crusherR = distortion.NewBitCrusher(sampleRate)
	
	// Create LFO for vinyl wobble
	p.lfo = modulation.NewLFO(sampleRate)
	p.lfo.SetFrequency(0.3)  // Slow wobble
	p.lfo.SetWaveform(modulation.WaveformSine)
	
	// Allocate temp buffer
	p.tempBuffer = make([]float64, maxSamplesPerBlock)
	
	// Set initial parameters
	p.updateCrusherParameters(p.crusherL)
	p.updateCrusherParameters(p.crusherR)
	
	return nil
}

// ProcessAudio processes audio through lo-fi effects
func (p *LoFiProcessor) ProcessAudio(ctx *process.Context) {
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
	
	// Process each channel
	for ch := 0; ch < numChannels && ch < len(ctx.Output); ch++ {
		input := ctx.Input[ch]
		output := ctx.Output[ch]
		
		// Get the appropriate crusher
		crusher := p.crusherL
		if ch == 1 {
			crusher = p.crusherR
		}
		
		// Process each sample
		for i := 0; i < numSamples; i++ {
			// Apply input drive
			sample := float64(input[i]) * p.drive
			
			// Add noise if enabled
			if p.noiseLevel > 0 {
				sample += p.generateNoise() * p.noiseLevel * 0.1
			}
			
			// Apply vinyl effects if enabled
			if p.vinylAmount > 0 {
				sample = p.applyVinylEffects(sample, i)
			}
			
			// Store in temp buffer for bit crushing
			p.tempBuffer[i] = sample
		}
		
		// Apply bit crushing to the buffer
		crusher.ProcessBuffer(p.tempBuffer[:numSamples], p.tempBuffer[:numSamples])
		
		// Convert back to float32 and apply output gain
		for i := 0; i < numSamples; i++ {
			output[i] = float32(p.tempBuffer[i] * p.outputGain)
		}
	}
	
	// Handle mono to stereo if needed
	if numChannels == 1 && len(ctx.Output) >= 2 {
		copy(ctx.Output[1], ctx.Output[0])
	}
}

// updateParameters updates internal state from parameter values
func (p *LoFiProcessor) updateParameters(ctx *process.Context) {
	p.bitDepth = int(ctx.ParamPlain(ParamBitDepth))
	p.crusherL.SetBitDepth(p.bitDepth)
	p.crusherR.SetBitDepth(p.bitDepth)
	
	p.sampleRateRatio = ctx.ParamPlain(ParamSampleRate) / 100.0
	p.crusherL.SetSampleRateRatio(p.sampleRateRatio)
	p.crusherR.SetSampleRateRatio(p.sampleRateRatio)
	
	p.antiAlias = ctx.ParamPlain(ParamAntiAlias) > 0.5
	p.crusherL.SetAntiAlias(p.antiAlias)
	p.crusherR.SetAntiAlias(p.antiAlias)
	
	p.ditherAmount = ctx.ParamPlain(ParamDither) / 100.0
	p.crusherL.SetDither(p.ditherAmount)
	p.crusherR.SetDither(p.ditherAmount)
	
	p.drive = ctx.ParamPlain(ParamDrive)
	
	p.noiseLevel = ctx.ParamPlain(ParamNoise) / 100.0
	
	p.vinylAmount = ctx.ParamPlain(ParamVinyl) / 100.0
	// Adjust LFO depth based on vinyl amount
	p.lfo.SetDepth(p.vinylAmount * 0.002) // Subtle pitch wobble
	
	p.mix = ctx.ParamPlain(ParamMix) / 100.0
	p.crusherL.SetMix(p.mix)
	p.crusherR.SetMix(p.mix)
	
	// Convert dB to linear
	outputDB := ctx.ParamPlain(ParamOutput)
	p.outputGain = math.Pow(10.0, outputDB/20.0)
}

// updateCrusherParameters applies all current parameters to a crusher instance
func (p *LoFiProcessor) updateCrusherParameters(crusher *distortion.BitCrusher) {
	crusher.SetBitDepth(p.bitDepth)
	crusher.SetSampleRateRatio(p.sampleRateRatio)
	crusher.SetAntiAlias(p.antiAlias)
	crusher.SetDither(p.ditherAmount)
	crusher.SetMix(p.mix)
}

// generateNoise generates white noise
func (p *LoFiProcessor) generateNoise() float64 {
	// Linear congruential generator
	p.noiseState = (p.noiseState*1664525 + 1013904223) & 0xffffffff
	return (float64(p.noiseState)/float64(0xffffffff) - 0.5) * 2.0
}

// applyVinylEffects simulates vinyl characteristics
func (p *LoFiProcessor) applyVinylEffects(sample float64, sampleIndex int) float64 {
	// Add subtle pitch wobble
	wobble := p.lfo.Process()
	
	// Add vinyl crackle (occasional pops)
	crackle := 0.0
	if p.generateNoise() > (1.0 - p.vinylAmount*0.001) {
		crackle = p.generateNoise() * 0.3 * p.vinylAmount
	}
	
	// Add continuous vinyl noise
	vinylNoise := p.generateNoise() * 0.05 * p.vinylAmount
	
	// Apply subtle high-frequency roll-off (vinyl warmth)
	// Simple one-pole lowpass
	cutoff := 1.0 - p.vinylAmount*0.3
	sample = sample*cutoff + sample*(1.0-cutoff)*wobble
	
	return sample + crackle + vinylNoise
}

// SetActive is called when plugin becomes active/inactive
func (p *LoFiProcessor) SetActive(active bool) error {
	return nil
}

// GetLatencySamples returns the plugin latency
func (p *LoFiProcessor) GetLatencySamples() int32 {
	return 0
}

// GetTailSamples returns the tail length
func (p *LoFiProcessor) GetTailSamples() int32 {
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
	vst3plugin.Register(&LoFiPlugin{})
}

// Required for c-shared build mode
func main() {}