package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// DelayPlugin implements the Plugin interface
type DelayPlugin struct{}

func (d *DelayPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.delay",
		Name:     "Simple Delay",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Delay",
	}
}

func (d *DelayPlugin) CreateProcessor() vst3plugin.Processor {
	return NewDelayProcessor()
}

// DelayProcessor handles the audio processing
type DelayProcessor struct {
	params *param.Registry
	buses  *bus.Configuration

	// Delay state - pre-allocated
	delayBuffer [][]float32
	bufferSize  int
	writePos    int
	sampleRate  float64
}

// Parameter IDs
const (
	ParamDelayTime = 0
	ParamFeedback  = 1
	ParamMix       = 2
)

func NewDelayProcessor() *DelayProcessor {
	p := &DelayProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewStereoConfiguration(),
		bufferSize: 48000, // 1 second at 48kHz
		writePos:   0,
		sampleRate: 48000,
	}

	// Add parameters
	p.params.Add(
		param.New(ParamDelayTime, "Delay Time").
			Range(0, 1000).
			Default(250).
			Unit("ms").
			Formatter(param.TimeFormatter, param.TimeParser).
			Build(),

		param.New(ParamFeedback, "Feedback").
			Range(0, 100).
			Default(30).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamMix, "Mix").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)

	return p
}

func (p *DelayProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	p.bufferSize = int(sampleRate) // 1 second max delay

	// Pre-allocate delay buffers for 2 channels
	p.delayBuffer = make([][]float32, 2)
	for i := range p.delayBuffer {
		p.delayBuffer[i] = make([]float32, p.bufferSize)
	}
	p.writePos = 0

	return nil
}

func (p *DelayProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameter values
	delayTimeMs := ctx.ParamPlain(ParamDelayTime)
	feedback := float32(ctx.ParamPlain(ParamFeedback) / 100.0) // Convert from percentage
	mix := float32(ctx.ParamPlain(ParamMix) / 100.0)           // Convert from percentage

	// Convert delay time to samples
	delaySamples := int(delayTimeMs * p.sampleRate / 1000.0)
	if delaySamples >= p.bufferSize {
		delaySamples = p.bufferSize - 1
	}

	// Process each channel
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	if numChannels > 2 {
		numChannels = 2 // We only support stereo
	}

	numSamples := ctx.NumSamples()

	for sample := 0; sample < numSamples; sample++ {
		// Calculate read position for this sample
		readPos := p.writePos - delaySamples
		if readPos < 0 {
			readPos += p.bufferSize
		}

		for ch := 0; ch < numChannels; ch++ {
			// Read from delay buffer
			delayed := p.delayBuffer[ch][readPos]

			// Get input sample
			dry := ctx.Input[ch][sample]

			// Mix dry and wet signals
			ctx.Output[ch][sample] = dry*(1.0-mix) + delayed*mix

			// Write to delay buffer with feedback
			p.delayBuffer[ch][p.writePos] = dry + delayed*feedback
		}

		// Increment write position
		p.writePos++
		if p.writePos >= p.bufferSize {
			p.writePos = 0
		}
	}
}

func (p *DelayProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *DelayProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *DelayProcessor) SetActive(active bool) error {
	if !active {
		// Clear delay buffers when deactivated
		for ch := range p.delayBuffer {
			for i := range p.delayBuffer[ch] {
				p.delayBuffer[ch][i] = 0
			}
		}
		p.writePos = 0
	}
	return nil
}

func (p *DelayProcessor) GetLatencySamples() int32 {
	return 0
}

func (p *DelayProcessor) GetTailSamples() int32 {
	// Return max delay time as tail
	return int32(p.sampleRate) // 1 second
}

func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&DelayPlugin{})
}

// Required for c-shared build mode
func main() {}
