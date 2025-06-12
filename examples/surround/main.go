package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/justyntemme/vst3go/pkg/dsp/pan"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// SurroundPlugin implements the Plugin interface
type SurroundPlugin struct{}

func (s *SurroundPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.surround",
		Name:     "Surround Panner 5.1",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Fx|Spatial",
	}
}

func (s *SurroundPlugin) CreateProcessor() vst3plugin.Processor {
	return NewSurroundProcessor()
}

// SurroundProcessor handles the audio processing
type SurroundProcessor struct {
	params *param.Registry
	buses  *bus.Configuration

	// Panning state
	sampleRate float64
}

// Parameter IDs
const (
	ParamAngle = iota      // 0-360 degrees
	ParamDistance          // 0-100 (center to edge)
	ParamWidth             // Stereo width when input is stereo
	ParamLFE               // LFE send amount
	ParamCenterLevel       // Center channel level
	ParamDivergence        // Panning spread
)

// Channel indices for 5.1
const (
	ChLeft = iota
	ChRight
	ChCenter
	ChLFE
	ChLeftSurround
	ChRightSurround
)

func NewSurroundProcessor() *SurroundProcessor {
	p := &SurroundProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewSurroundPanner(), // Stereo in, 5.1 out
		sampleRate: 48000,
	}

	// Add parameters
	p.params.Add(
		param.New(ParamAngle, "Angle").
			Range(0, 360).
			Default(0).
			Unit("°").
			Formatter(func(v float64) string {
				return fmt.Sprintf("%.1f°", v)
			}, func(s string) (float64, error) {
				s = strings.TrimSuffix(s, "°")
				return strconv.ParseFloat(s, 64)
			}).
			Build(),

		param.New(ParamDistance, "Distance").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamWidth, "Width").
			Range(0, 200).
			Default(100).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamLFE, "LFE Send").
			Range(-60, 0).
			Default(-60).
			Unit("dB").
			Formatter(param.DecibelFormatter, param.DecibelParser).
			Build(),

		param.New(ParamCenterLevel, "Center Level").
			Range(0, 100).
			Default(50).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),

		param.New(ParamDivergence, "Divergence").
			Range(0, 100).
			Default(0).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)

	return p
}

func (p *SurroundProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	return nil
}

func (p *SurroundProcessor) ProcessAudio(ctx *process.Context) {
	// Get parameters
	angle := float32(ctx.ParamPlain(ParamAngle))
	distance := float32(ctx.ParamPlain(ParamDistance) / 100.0)
	width := float32(ctx.ParamPlain(ParamWidth) / 100.0)
	// These will be used when full 5.1 support is implemented
	_ = ctx.ParamPlain(ParamLFE)
	_ = ctx.ParamPlain(ParamCenterLevel)
	_ = ctx.ParamPlain(ParamDivergence)

	// For now, we'll implement simple stereo-to-stereo panning
	// Full 5.1 support will be available when multi-bus wrapper is implemented
	
	// Convert angle to radians
	angleRad := angle * math.Pi / 180.0

	// Calculate basic pan position from angle (0° = center, -90° = left, 90° = right)
	panPosition := float32(math.Sin(float64(angleRad)))

	// Apply distance-based attenuation
	attenuation := 1.0 - (distance * 0.5) // Simple linear attenuation

	// Process stereo channels
	numSamples := ctx.NumSamples()
	if ctx.NumInputChannels() >= 2 && ctx.NumOutputChannels() >= 2 {
		for i := 0; i < numSamples; i++ {
			// Get input samples
			left := ctx.Input[0][i]
			right := ctx.Input[1][i]

			// Apply width control
			mid := (left + right) * 0.5
			side := (left - right) * 0.5 * width

			// Reconstruct with width
			left = mid + side
			right = mid - side

			// Apply panning
			leftGain, rightGain := pan.MonoToStereo(panPosition, pan.ConstantPower)

			// Apply distance attenuation
			leftGain *= attenuation
			rightGain *= attenuation

			// Output
			ctx.Output[0][i] = left * leftGain
			ctx.Output[1][i] = right * rightGain
		}
	} else if ctx.NumInputChannels() >= 1 && ctx.NumOutputChannels() >= 2 {
		// Mono to stereo
		for i := 0; i < numSamples; i++ {
			input := ctx.Input[0][i]
			
			// Apply panning
			leftGain, rightGain := pan.MonoToStereo(panPosition, pan.ConstantPower)

			// Apply distance attenuation
			leftGain *= attenuation
			rightGain *= attenuation

			// Output
			ctx.Output[0][i] = input * leftGain
			ctx.Output[1][i] = input * rightGain
		}
	}

	// Note: When multi-bus support is available, we'll distribute to all 5.1 channels:
	// - Front L/R based on angle when in front hemisphere
	// - Surround L/R when in rear hemisphere  
	// - Center channel based on centerLevel and distance
	// - LFE based on lfeDb parameter
	// - Divergence will spread the sound across multiple speakers
}

func (p *SurroundProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *SurroundProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

func (p *SurroundProcessor) SetActive(active bool) error {
	return nil
}

func (p *SurroundProcessor) GetLatencySamples() int32 {
	return 0
}

func (p *SurroundProcessor) GetTailSamples() int32 {
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
	vst3plugin.Register(&SurroundPlugin{})
}

// Required for c-shared build mode
func main() {}