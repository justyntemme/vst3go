# Getting Started with VST3Go

## Prerequisites

Before you begin, ensure you have:
- Go 1.19 or later
- GCC (for CGO compilation)
- VST3 validator (optional, for testing)
- A DAW that supports VST3 (for testing your plugins)

### Platform-Specific Requirements

#### Linux
```bash
sudo apt-get install build-essential  # Debian/Ubuntu
sudo dnf install gcc gcc-c++          # Fedora
```

#### macOS
```bash
xcode-select --install  # Install command line tools
```

#### Windows
- Install MinGW-w64 or use WSL2
- Ensure gcc is in your PATH

## Installation

1. Clone the repository:
```bash
git clone https://github.com/justyntemme/vst3go.git
cd vst3go
```

2. Build the examples:
```bash
make all-examples
```

3. Install plugins to your system:
```bash
make install  # Installs to ~/.vst3 (Linux) or appropriate directory
```

## Your First Plugin

Let's create a simple gain plugin from scratch.

### 1. Create the Directory Structure

```bash
mkdir -p myplugins/simplegain
cd myplugins/simplegain
```

### 2. Create main.go

```go
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

// Define our plugin
type SimpleGainPlugin struct{}

func (p *SimpleGainPlugin) GetInfo() plugin.Info {
    return plugin.Info{
        ID:       "com.example.simplegain",
        Name:     "Simple Gain",
        Version:  "1.0.0",
        Vendor:   "My Company",
        Category: "Fx",
    }
}

func (p *SimpleGainPlugin) CreateProcessor() vst3plugin.Processor {
    return NewSimpleGainProcessor()
}

// Define our processor
type SimpleGainProcessor struct {
    params *param.Registry
    buses  *bus.Configuration
}

// Parameter IDs
const (
    ParamGain = 0
)

func NewSimpleGainProcessor() *SimpleGainProcessor {
    p := &SimpleGainProcessor{
        params: param.NewRegistry(),
        buses:  bus.NewStereoConfiguration(),
    }
    
    // Add gain parameter
    p.params.Add(
        param.New(ParamGain, "Gain").
            Range(-24, 24).
            Default(0).
            Unit("dB").
            Build(),
    )
    
    return p
}

func (p *SimpleGainProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
    // Any initialization code here
    return nil
}

func (p *SimpleGainProcessor) ProcessAudio(ctx *process.Context) {
    // Get gain value in dB
    gainDB := ctx.ParamPlain(ParamGain)
    
    // Convert to linear gain
    gain := float32(math.Pow(10.0, gainDB/20.0))
    
    // Process each channel
    numChannels := ctx.NumInputChannels()
    if ctx.NumOutputChannels() < numChannels {
        numChannels = ctx.NumOutputChannels()
    }
    
    numSamples := ctx.NumSamples()
    
    for ch := 0; ch < numChannels; ch++ {
        input := ctx.Input[ch]
        output := ctx.Output[ch]
        
        for i := 0; i < numSamples; i++ {
            output[i] = input[i] * gain
        }
    }
}

func (p *SimpleGainProcessor) GetParameters() *param.Registry {
    return p.params
}

func (p *SimpleGainProcessor) GetBuses() *bus.Configuration {
    return p.buses
}

func (p *SimpleGainProcessor) SetActive(active bool) error {
    return nil
}

func (p *SimpleGainProcessor) GetLatencySamples() int32 {
    return 0
}

func (p *SimpleGainProcessor) GetTailSamples() int32 {
    return 0
}

// Register the plugin
func init() {
    vst3plugin.SetFactoryInfo(vst3plugin.FactoryInfo{
        Vendor: "My Company",
        URL:    "https://example.com",
        Email:  "support@example.com",
    })
    
    vst3plugin.Register(&SimpleGainPlugin{})
}

// Required for c-shared build mode
func main() {}
```

### 3. Build Your Plugin

Add to the main Makefile or create your own:

```makefile
simplegain:
	CGO_CFLAGS="-I../../include" CGO_LDFLAGS="-shared" \
	go build -buildmode=c-shared -o build/SimpleGain.so
```

Build it:
```bash
make simplegain
```

### 4. Create VST3 Bundle

```bash
make bundle PLUGIN_NAME=SimpleGain
```

### 5. Test Your Plugin

```bash
make test-validate PLUGIN_NAME=SimpleGain
```

## Adding DSP Features

Let's enhance our plugin with a simple filter:

```go
import (
    "github.com/justyntemme/vst3go/pkg/dsp/filter"
)

type FilteredGainProcessor struct {
    params     *param.Registry
    buses      *bus.Configuration
    lowpass    *filter.Biquad
    sampleRate float64
}

const (
    ParamGain   = 0
    ParamCutoff = 1
)

func NewFilteredGainProcessor() *FilteredGainProcessor {
    p := &FilteredGainProcessor{
        params:  param.NewRegistry(),
        buses:   bus.NewStereoConfiguration(),
        lowpass: filter.NewBiquad(2), // stereo
    }
    
    p.params.Add(
        param.New(ParamGain, "Gain").
            Range(-24, 24).Default(0).Unit("dB").Build(),
        param.New(ParamCutoff, "Cutoff").
            Range(20, 20000).Default(10000).Unit("Hz").Build(),
    )
    
    return p
}

func (p *FilteredGainProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
    p.sampleRate = sampleRate
    p.lowpass.SetLowpass(sampleRate, 10000, 0.707)
    return nil
}

func (p *FilteredGainProcessor) ProcessAudio(ctx *process.Context) {
    gainDB := ctx.ParamPlain(ParamGain)
    cutoff := ctx.ParamPlain(ParamCutoff)
    
    gain := float32(math.Pow(10.0, gainDB/20.0))
    
    // Update filter if cutoff changed
    p.lowpass.SetLowpass(p.sampleRate, cutoff, 0.707)
    
    numChannels := ctx.NumInputChannels()
    numSamples := ctx.NumSamples()
    
    for ch := 0; ch < numChannels; ch++ {
        // Apply gain
        for i := 0; i < numSamples; i++ {
            ctx.Output[ch][i] = ctx.Input[ch][i] * gain
        }
        
        // Apply filter
        p.lowpass.Process(ctx.Output[ch][:numSamples], ch)
    }
}
```

## Common Patterns

### Using Work Buffers

```go
func (p *MyProcessor) ProcessAudio(ctx *process.Context) {
    work := ctx.WorkBuffer()  // Pre-allocated, no allocation
    
    // Use work buffer for temporary processing
    copy(work, ctx.Input[0][:ctx.NumSamples()])
    
    // Process in work buffer
    p.effect.Process(work)
    
    // Copy to output
    copy(ctx.Output[0][:ctx.NumSamples()], work)
}
```

### Parameter Smoothing

```go
type SmoothProcessor struct {
    currentGain float32
    targetGain  float32
    smoothing   float32
}

func (p *SmoothProcessor) ProcessAudio(ctx *process.Context) {
    p.targetGain = float32(ctx.ParamPlain(ParamGain))
    
    for i := 0; i < ctx.NumSamples(); i++ {
        // Smooth parameter changes
        p.currentGain += (p.targetGain - p.currentGain) * p.smoothing
        
        // Apply smoothed gain
        ctx.Output[0][i] = ctx.Input[0][i] * p.currentGain
    }
}
```

### Stereo Processing

```go
func (p *StereoProcessor) ProcessAudio(ctx *process.Context) {
    if ctx.NumInputChannels() >= 2 && ctx.NumOutputChannels() >= 2 {
        left := ctx.Input[0][:ctx.NumSamples()]
        right := ctx.Input[1][:ctx.NumSamples()]
        
        // Process stereo
        for i := range left {
            // Mid-side processing example
            mid := (left[i] + right[i]) * 0.5
            side := (left[i] - right[i]) * 0.5
            
            // Process mid/side
            mid *= p.midGain
            side *= p.sideGain
            
            // Convert back to L/R
            ctx.Output[0][i] = mid + side
            ctx.Output[1][i] = mid - side
        }
    }
}
```

## Debugging Tips

### Add Debug Logging

```go
// +build debug

package main

import "log"

func debugLog(format string, args ...interface{}) {
    log.Printf("[MyPlugin] "+format, args...)
}
```

### Performance Profiling

```go
func (p *MyProcessor) ProcessAudio(ctx *process.Context) {
    start := time.Now()
    defer func() {
        elapsed := time.Since(start)
        if elapsed > 100*time.Microsecond {
            log.Printf("Process took %v", elapsed)
        }
    }()
    
    // Your processing code
}
```

### Allocation Checking

```go
func TestNoAllocations(t *testing.T) {
    p := NewMyProcessor()
    ctx := createTestContext()
    
    allocs := testing.AllocsPerRun(100, func() {
        p.ProcessAudio(ctx)
    })
    
    if allocs > 0 {
        t.Errorf("ProcessAudio allocated %v times", allocs)
    }
}
```

## Next Steps

1. **Explore the DSP Library**: Check out filters, oscillators, and effects
2. **Study the Examples**: Learn from the included example plugins
3. **Read the Architecture Guide**: Understand the framework design
4. **Join the Community**: Get help and share your plugins

## Troubleshooting

### Plugin doesn't load
- Check the plugin ID is unique
- Verify the shared library architecture matches your DAW
- Run the validator for detailed errors

### Crashes or glitches
- Check for allocations in ProcessAudio
- Verify buffer bounds
- Test with small block sizes

### Parameters not working
- Ensure parameter IDs are unique
- Check value ranges are correct
- Verify thread safety

For more help, see the [VST3 documentation](https://steinbergmedia.github.io/vst3_dev_portal/) and our [architecture guide](./architecture.md).