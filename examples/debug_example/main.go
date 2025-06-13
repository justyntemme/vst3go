// Package main demonstrates the use of debug utilities in a VST3 plugin.
package main

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justyntemme/vst3go/pkg/dsp/gain"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/debug"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

const (
	paramBypass = iota
	paramDebugLevel
)

// DebugExamplePlugin demonstrates debug utilities
type DebugExamplePlugin struct{}

func (p *DebugExamplePlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.debugexample",
		Name:     "DebugExample",
		Vendor:   "VST3Go Examples",
		Version:  "1.0.0",
		Category: "Fx",
	}
}

func (p *DebugExamplePlugin) CreateProcessor() vst3plugin.Processor {
	return &DebugExampleProcessor{}
}

// DebugExampleProcessor demonstrates various debug features
type DebugExampleProcessor struct {
	params *param.Registry
	
	// Debug components
	logger       *debug.Logger
	profiler     *debug.AudioProcessProfiler
	analyzer     *debug.AudioAnalyzer
	
	processCount uint64
	sampleRate   float64
	bufferSize   int
}

func (p *DebugExampleProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	p.bufferSize = int(maxBlockSize)
	
	// Initialize debug components
	
	// Create a file logger
	logDir := filepath.Join(os.TempDir(), "vst3go_debug")
	logFile := filepath.Join(logDir, "debug_example.log")
	
	var err error
	p.logger, err = debug.NewFileLogger(logFile, "DebugExample", debug.DefaultFlags)
	if err != nil {
		// Fall back to stderr
		p.logger = debug.New(os.Stderr, "DebugExample", debug.DefaultFlags)
		p.logger.Warn("Failed to create file logger: %v", err)
	}
	
	p.logger.Info("Plugin initialized - Sample Rate: %.0f Hz, Max Block Size: %d", sampleRate, maxBlockSize)
	
	// Create audio profiler
	p.profiler = debug.NewAudioProcessProfiler(sampleRate, p.bufferSize)
	
	// Create audio analyzer
	p.analyzer = debug.NewAudioAnalyzer()
	
	// Initialize parameter registry
	p.params = param.NewRegistry()

	// Register parameters
	registry := p.params
	
	registry.Add(
		param.BypassParameter(paramBypass, "Bypass").Build(),
	)
	
	registry.Add(
		param.Choice(paramDebugLevel, "Debug Level", []param.ChoiceOption{
			{Value: 0, Name: "Debug"},
			{Value: 1, Name: "Info"},
			{Value: 2, Name: "Warn"},
			{Value: 3, Name: "Error"},
			{Value: 4, Name: "Off"},
		}).Default(1).Build(),
	)
	
	// Set up global debug settings
	debug.SetPrefix("DebugExample")
	debug.EnableProfiling()
	
	return nil
}

func (p *DebugExampleProcessor) ProcessAudio(ctx *process.Context) {
	// Profile the entire process call
	stop := p.profiler.Start("ProcessAudio")
	defer stop()
	
	p.processCount++
	
	// Handle parameter changes
	for _, change := range ctx.GetParameterChanges() {
		switch change.ParamID {
		case paramDebugLevel:
			level := int(change.Value * 4) // 0-4
			p.logger.SetLevel(debug.LogLevel(level))
			p.logger.Info("Debug level changed to: %s", debug.LogLevel(level).String())
		}
	}
	
	// Log every 100th process call
	if p.processCount%100 == 0 {
		p.logger.Debug("Process call #%d", p.processCount)
		// Transport info not available in context
		p.logger.Debug("Process context: NumSamples=%d", ctx.NumSamples())
	}
	
	// Handle bypass
	bypassParam := p.params.Get(paramBypass)
	if bypassParam != nil && bypassParam.GetValue() > 0.5 {
		// Copy input to output
		for ch := 0; ch < ctx.NumInputChannels(); ch++ {
			copy(ctx.Output[ch], ctx.Input[ch])
		}
		return
	}
	
	// Process audio (simple gain reduction for demonstration)
	for ch := 0; ch < ctx.NumInputChannels() && ch < ctx.NumOutputChannels(); ch++ {
		func() {
			stop := p.profiler.Start(fmt.Sprintf("Channel%d", ch))
			defer stop()
			
			input := ctx.Input[ch]
			output := ctx.Output[ch]
			
			// Analyze input
			if p.processCount%1000 == 0 { // Every 1000 blocks
				result := p.analyzer.Analyze(input[:ctx.NumSamples()])
				
				if result.Clipping {
					p.logger.Warn("Input clipping on channel %d: %d samples", ch, result.ClippedSamples)
				}
				
				if result.HasNaN {
					p.logger.Error("NaN detected in input channel %d: %d samples", ch, result.NaNCount)
				}
				
				p.logger.Debug("Channel %d stats - Peak: %.3f, RMS: %.3f, DC: %.6f", 
					ch, result.Peak, result.RMS, result.DC)
			}
			
			// Simple processing (gain reduction)
			debug.Time("GainReduction", func() {
				// First copy input to output
				copy(output[:ctx.NumSamples()], input[:ctx.NumSamples()])
				// Then apply gain
				gain.ApplyBuffer(output[:ctx.NumSamples()], 0.7)
			})
			
			// Check output
			issues := debug.CheckBuffer(output[:ctx.NumSamples()], fmt.Sprintf("Output%d", ch))
			for _, issue := range issues {
				p.logger.Warn("%s", issue)
			}
		}()
	}
	
	// Update CPU load periodically
	if p.processCount%100 == 0 {
		p.profiler.UpdateCPULoad()
		cpuLoad := p.profiler.GetCPULoad()
		
		if cpuLoad > 80.0 {
			p.logger.Warn("High CPU load: %.2f%%", cpuLoad)
		} else {
			p.logger.Debug("CPU load: %.2f%%", cpuLoad)
		}
	}
	
	// Log performance report every 10000 blocks
	if p.processCount%10000 == 0 {
		p.logger.Info("Performance Report:\n%s", p.profiler.AudioReport())
		
		// Also log global profiling data
		p.logger.Debug("Global Profiling:\n%s", debug.ProfilingReport())
	}
}

func (p *DebugExampleProcessor) GetParameters() *param.Registry {
	return p.params
}

func (p *DebugExampleProcessor) GetBuses() *bus.Configuration {
	return bus.NewStereoConfiguration()
}

func (p *DebugExampleProcessor) SetActive(active bool) error {
	if active {
		p.logger.Info("Plugin activated")
		p.processCount = 0
		p.profiler.Reset()
		debug.ResetProfiling()
	} else {
		p.logger.Info("Plugin deactivated")
		
		// Log final performance report
		p.logger.Info("Final Performance Report:\n%s", p.profiler.AudioReport())
	}
	return nil
}

func (p *DebugExampleProcessor) GetLatencySamples() int32 {
	return 0
}

func (p *DebugExampleProcessor) GetTailSamples() int32 {
	return 0
}

// VST3 exported functions
func init() {
	// Set factory info
	vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
		Vendor: "VST3Go Examples",
		URL:    "https://github.com/vst3go/examples",
		Email:  "examples@vst3go.com",
	})

	// Register our plugin
	vst3plugin.Register(&DebugExamplePlugin{})
}

// Required for c-shared build mode
func main() {}