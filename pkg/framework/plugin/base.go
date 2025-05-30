package plugin

import (
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/state"
)

// Base provides core functionality for all plugins
type Base struct {
	Info   Info
	params *param.Registry
	state  *state.Manager
}

// NewBase creates a new plugin base
func NewBase(info Info) *Base {
	b := &Base{
		Info:   info,
		params: param.NewRegistry(),
	}

	// Initialize state manager with parameter registry
	b.state = state.NewManager(b.params)

	return b
}

// Parameters returns the parameter registry for configuration
func (b *Base) Parameters() *param.Registry {
	return b.params
}

// AudioProcessor is the interface plugins implement for audio processing
type AudioProcessor interface {
	// ProcessAudio processes audio buffers - zero allocations allowed!
	ProcessAudio(input, output [][]float32)
}
