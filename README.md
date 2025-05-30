# VST3Go

A Go framework for building VST3 audio plugins with minimal boilerplate and zero allocations in the audio path.

## Features

- **Minimal C Bridge** - Thin C wrapper with all logic in Go
- **Zero Allocations** - Pre-allocated buffers for real-time safety
- **Rich DSP Library** - Filters, oscillators, envelopes, delays
- **Simple API** - Build effects in under 200 lines
- **Thread-Safe** - Lock-free parameter system

## Project Structure

```
vst3go/
├── bridge/           # Minimal C bridge (just routing)
├── pkg/
│   ├── framework/   # Core plugin framework
│   │   ├── plugin/  # Plugin base types
│   │   ├── param/   # Parameter management
│   │   ├── process/ # Audio processing context
│   │   ├── bus/     # Bus configuration
│   │   └── state/   # State persistence
│   ├── dsp/         # DSP utilities
│   │   ├── buffer/  # Buffer operations
│   │   ├── filter/  # Filters (biquad, SVF)
│   │   ├── oscillator/ # Oscillators
│   │   ├── envelope/   # Envelopes (ADSR)
│   │   └── delay/      # Delay lines
│   └── plugin/      # VST3 wrapper
├── examples/        # Example plugins
│   ├── gain/        # Simple gain effect
│   ├── delay/       # Delay with feedback
│   └── filter/      # Multi-mode filter
└── include/         # VST3 C API headers
```

## Quick Start

### Building Examples

```bash
# Build all example plugins
make all-examples

# Build specific plugin
make gain
make delay
make filter

# Run VST3 validation
make test-validate PLUGIN_NAME=SimpleGain

# Install to ~/.vst3
make install
```

### Creating Your First Plugin

```go
package main

import (
    "github.com/justyntemme/vst3go/pkg/framework/plugin"
    "github.com/justyntemme/vst3go/pkg/framework/param"
    "github.com/justyntemme/vst3go/pkg/framework/process"
)

type MyPlugin struct{}

func (p *MyPlugin) GetInfo() plugin.Info {
    return plugin.Info{
        ID:       "com.example.myplugin",
        Name:     "My Plugin",
        Version:  "1.0.0",
        Vendor:   "My Company",
        Category: "Fx",
    }
}

func (p *MyPlugin) CreateProcessor() Processor {
    return &MyProcessor{
        params: param.NewRegistry(
            param.New(0, "Gain").Range(-24, 24).Default(0).Unit("dB"),
        ),
    }
}

type MyProcessor struct {
    params *param.Registry
}

func (p *MyProcessor) ProcessAudio(ctx *process.Context) {
    gain := ctx.ParamPlain(0) // Get parameter value
    
    for ch := 0; ch < ctx.NumChannels(); ch++ {
        for i := range ctx.Input[ch] {
            ctx.Output[ch][i] = ctx.Input[ch][i] * gain
        }
    }
}
```

## Documentation

- **[Architecture Guide](docs/architecture.md)** - Comprehensive overview of the framework design
- **[Getting Started](docs/getting-started.md)** - Step-by-step tutorial for your first plugin
- **[API Documentation](docs/)** - Detailed documentation for each component
- **[Development Roadmap](TODO.md)** - Current status and future plans

### Working
- ✅ Basic plugin framework
- ✅ Zero-allocation audio processing
- ✅ Comprehensive DSP library
- ✅ Three example plugins
- ✅ VST3 validation passing

### In Progress
- 🚧 Parameter automation from host
- 🚧 State save/load
- 🚧 Process context (tempo, transport)
- 🚧 Developer experience improvements

### Planned
- 📅 MIDI support
- 📅 Multi-bus configurations
- 📅 Cross-platform support

## Requirements

- Go 1.19+
- GCC (for CGO)
- VST3 SDK headers (included)
- Linux (macOS/Windows coming soon)

## License

This project is licensed under the MIT License. The VST3 SDK headers are licensed under their respective licenses.