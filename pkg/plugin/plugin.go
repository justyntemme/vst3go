// Package plugin provides the VST3 plugin framework
package plugin

import (
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	"github.com/justyntemme/vst3go/pkg/framework/process"
)

// Plugin is the main interface that users implement
type Plugin interface {
	// GetInfo returns plugin metadata
	GetInfo() plugin.Info
	
	// CreateProcessor creates a new instance of the audio processor
	CreateProcessor() Processor
}

// Processor handles the actual audio processing
type Processor interface {
	// Initialize is called when the plugin is created
	Initialize(sampleRate float64, maxBlockSize int32) error
	
	// ProcessAudio processes audio - ZERO ALLOCATIONS!
	ProcessAudio(ctx *process.Context)
	
	// GetParameters returns the parameter registry
	GetParameters() *param.Registry
	
	// GetBuses returns the bus configuration
	GetBuses() *bus.Configuration
	
	// SetActive is called when processing starts/stops
	SetActive(active bool) error
	
	// GetLatencySamples returns the plugin's latency in samples
	GetLatencySamples() int32
	
	// GetTailSamples returns the tail length in samples
	GetTailSamples() int32
}