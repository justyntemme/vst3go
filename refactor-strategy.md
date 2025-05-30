# VST3Go Refactor Strategy

## Core Development Principles

### No Backwards Compatibility - Move Fast, Break Things
- **This is a POC/MVP** - We're building towards 1.0, not maintaining legacy code
- **Delete old code immediately** - No backups, no deprecation warnings, no migration paths
- **Make breaking changes freely** - If something needs to change, change it
- **Keep old code only when actively migrating** - And delete it as soon as the migration is complete

## Executive Summary

This document outlines a comprehensive refactoring strategy to align the VST3Go codebase with the architectural principles defined in `guardrails.md`. The goal is to transform the current implementation into a layered, Go-idiomatic framework that makes VST3 plugin development fast, enjoyable, and powerful while maintaining a minimal C bridge layer.

## Current State Analysis

### Identified Issues
1. **Bridge Layer Complexity**: The C bridge contains business logic and framework features that belong in Go layers
2. **Code Duplication**: Common patterns are repeated across examples instead of being extracted to framework packages
3. **Missing Abstractions**: Lack of Go-idiomatic wrappers for common VST3 operations
4. **No DSP Package**: Missing audio processing utilities that developers commonly need
5. **Inconsistent Error Handling**: Mix of bool returns and error patterns

## Target Architecture

### Critical Principle: "Bridge Not Framework"

The most important architectural decision is maintaining a clear separation between the C bridge layer and the Go framework layers. This separation is absolute and non-negotiable.

#### Layer 1: Minimal C Bridge - "Just a Bridge"
- **Purpose**: Direct, thin mapping from VST3 C API to Go
- **Scope**: Function calls, data marshaling, manifest discovery
- **Constraints**: Zero business logic, no framework features
- **Implementation**: So simple it could be auto-generated

```c
// This is what EVERY C bridge function should look like:
tresult SMTG_STDMETHODCALLTYPE IComponentProcess(void* thisInterface, void* data) {
    return goProcess(thisInterface, data);
}
// That's it. Nothing more. Ever.
```

#### Layers 2-4: Go Framework - "Rich and Comprehensive"
All intelligence, convenience, and developer experience lives in Go:

### Layer 2: Go-Native Abstractions
- **Purpose**: Go-idiomatic wrappers around VST3 concepts
- **Scope**: Parameter management, event handling, state management
- **Design**: Interface-based, composable, testable
- **Example**: Thread-safe parameter registry, automatic state versioning

### Layer 3: DSP & Audio Utilities
- **Purpose**: Common audio processing building blocks
- **Scope**: Filters, oscillators, envelopes, effects, utilities
- **Design**: High-performance, **zero allocations in audio path**
- **Example**: Biquad filters, ADSR envelopes, delay lines
- **Critical**: All allocations happen at initialization, never during processing

### Layer 4: Developer Conveniences
- **Purpose**: Productivity tools and helpers
- **Scope**: Builders, templates, code generation, common patterns
- **Design**: Optional, extensible, documentation-focused
- **Example**: Plugin builder API, project templates, debug helpers

## Zero-Allocation Audio Processing Design

### Core Principle: Pre-allocate Everything

The audio processing path must have **zero allocations** to ensure:
- Consistent low latency
- No garbage collection pauses
- Predictable performance
- Real-time safety

### Allocation Strategy

#### 1. Initialization Phase (Allocations Allowed)
```go
// Plugin initialization - allocate all buffers here
func (p *Plugin) Initialize(sampleRate float64, maxBlockSize int) error {
    // Pre-allocate all working buffers
    p.workBuffer = make([]float32, maxBlockSize)
    p.tempBuffer = make([]float32, maxBlockSize)
    
    // Pre-allocate all DSP components
    p.filter = filter.NewBiquad()
    p.delay = delay.NewLine(1000, sampleRate) // 1 second buffer
    p.reverb = reverb.New(reverb.Config{
        RoomSize: 0.8,
        Damping:  0.5,
        PreAllocSize: maxBlockSize,
    })
    
    return nil
}
```

#### 2. Processing Phase (Zero Allocations)
```go
// Audio callback - MUST NOT allocate
func (p *Plugin) ProcessAudio(input, output [][]float32) {
    // Use pre-allocated buffers
    samples := len(input[0])
    
    // Slice existing buffers - no allocation
    work := p.workBuffer[:samples]
    temp := p.tempBuffer[:samples]
    
    // All DSP operations use pre-allocated state
    for ch := range input {
        copy(work, input[ch])        // No allocation
        p.filter.Process(work)       // In-place, no allocation
        p.delay.ProcessBlock(work, temp, p.delayTime) // No allocation
        p.mixer.Mix(work, temp, output[ch], p.mixLevel) // No allocation
    }
}
```

### DSP Component Design Rules

#### 1. State Allocation
```go
type Oscillator struct {
    // All state pre-allocated
    phase      float64
    sampleRate float64
    // Pre-computed values to avoid allocations
    phaseIncrement float64
}

func (o *Oscillator) SetFrequency(hz float64) {
    // Pre-compute to avoid division in audio path
    o.phaseIncrement = hz / o.sampleRate
}
```

#### 2. Buffer Management
```go
type BufferPool struct {
    buffers [][]float32
    size    int
}

// Get buffer from pool - no allocation
func (p *BufferPool) Get() []float32 {
    if len(p.buffers) > 0 {
        buf := p.buffers[len(p.buffers)-1]
        p.buffers = p.buffers[:len(p.buffers)-1]
        return buf
    }
    panic("buffer pool empty - this should never happen")
}
```

#### 3. Parameter Smoothing
```go
type Smoother struct {
    current, target float32
    rate           float32  // Pre-calculated
}

// Update without allocation
func (s *Smoother) Process() float32 {
    s.current += (s.target - s.current) * s.rate
    return s.current
}
```

### Framework Guarantees

The framework will ensure zero allocations by:

1. **Pre-allocation Hooks**: Framework calls initialization before audio starts
2. **Buffer Reuse**: Framework manages buffer pools for common sizes
3. **Static Dispatch**: No interface allocations in audio path
4. **Compile-Time Checks**: Build tags to detect allocations in debug builds

```go
// +build debug

// Debug mode - detect allocations
func (p *Plugin) ProcessAudio(input, output [][]float32) {
    before := runtime.MemStats.Alloc
    p.processAudioImpl(input, output)
    after := runtime.MemStats.Alloc
    
    if after > before {
        panic("allocation detected in audio path!")
    }
}
```

## Implementation Phases

### Phase 1: Bridge Layer Cleanup
**Goal**: Strip all business logic from C bridge, making it a pure mapping layer

#### Current Problem: Business Logic in C Bridge
```c
// bridge/component.c - WRONG: Business logic in C bridge
typedef struct {
    Vst3ParamID id;
    double value;
    double min;
    double max;
    char* name;
    int flags;
} ParameterInfo;

static ParameterInfo* parameters = NULL;
static int parameter_count = 0;

void bridge_register_parameter(Vst3ParamID id, const char* name, 
                              double min, double max, double defaultValue) {
    // WRONG: Framework feature in C bridge
    parameters = realloc(parameters, sizeof(ParameterInfo) * (parameter_count + 1));
    parameters[parameter_count].id = id;
    parameters[parameter_count].name = strdup(name);
    // ... more business logic
}
```

#### Target: Pure C-to-Go Mapping
```c
// bridge/component.c - CORRECT: Pure mapping, zero logic
tresult SMTG_STDMETHODCALLTYPE IEditControllerSetParamNormalized(void* thisInterface, 
                                                                 Vst3ParamID id, 
                                                                 Vst3ParamValue value) {
    // CORRECT: Direct call to Go, no business logic
    return goSetParamNormalized(thisInterface, id, value);
}

// The ENTIRE C bridge function - nothing more!
tresult SMTG_STDMETHODCALLTYPE IComponentGetState(void* thisInterface, void* state) {
    return goGetState(thisInterface, state);
}
```

#### Business Logic Moves to Go Framework
```go
// pkg/framework/param/registry.go - CORRECT: Business logic in Go
type Registry struct {
    params map[vst3.ParamID]*Parameter
    mu     sync.RWMutex
}

func (r *Registry) Register(p *Parameter) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.params[p.ID]; exists {
        return fmt.Errorf("parameter %d already registered", p.ID)
    }
    r.params[p.ID] = p
    return nil
}
```

#### Tasks:
1. **Audit Current Bridge**
   - Remove ALL parameter storage from C
   - Remove ALL state management from C
   - Remove ALL business logic from C

2. **Implement Manifest Discovery**
```json
{
    "plugins": [{
        "cid": "12345678901234567890123456789012",
        "name": "Simple Gain",
        "version": "1.0.0",
        "vendor": "Example Inc",
        "entrypoint": "CreatePlugin"
    }]
}
```

3. **Simplify C Interface**
   - Each C function becomes a one-line Go call
   - No data storage in C
   - No memory management beyond basic marshaling

### Phase 2: Core Framework Development
**Goal**: Build foundational Go packages for VST3 development

#### Domain-Driven Package Structure (No Generic "Helpers")
```
pkg/
├── framework/
│   ├── plugin/          # Plugin lifecycle and base types
│   ├── param/           # Parameter management domain
│   ├── state/           # State persistence domain
│   ├── event/           # Event processing domain
│   ├── bus/             # Audio bus configuration
│   └── midi/            # MIDI event handling
├── dsp/                 # Domain-specific audio processing
│   ├── filter/          # Filter implementations
│   ├── oscillator/      # Oscillator types
│   ├── envelope/        # Envelope generators
│   ├── delay/           # Delay line implementations
│   ├── dynamics/        # Compressors, limiters
│   └── analysis/        # FFT, peak detection
└── vst3/               # Low-level VST3 types (Layer 1)
```

#### 1. Plugin Base Framework
```go
// pkg/framework/plugin/base.go
package plugin

type Base struct {
    info    Info
    params  *param.Registry
    state   *state.Manager
    buses   *bus.Configuration
}

// Developers only implement this interface
type AudioProcessor interface {
    ProcessAudio(input, output [][]float32)
}

// Framework handles ALL the boilerplate - with zero allocations
func (b *Base) Process(data *vst3.ProcessData) vst3.Result {
    // Framework handles (all zero-allocation):
    // - Parameter updates (pre-allocated queues)
    // - Event processing (reusable event buffers)
    // - Buffer management (slice existing arrays)
    // - Sample rate changes (update pre-allocated state)
    // - Bypass handling (in-place buffer copy)
    
    // Developer only implements ProcessAudio
    processor := b.getProcessor().(AudioProcessor)
    processor.ProcessAudio(b.getInputBuffers(data), b.getOutputBuffers(data))
    
    return vst3.ResultOK
}
```

#### 2. Parameter Builder API
```go
// pkg/framework/param/builder.go
package param

// Fluent API for parameter creation
type Builder struct {
    param *Parameter
}

func New(id vst3.ParamID, name string) *Builder {
    return &Builder{
        param: &Parameter{
            ID:   id,
            Name: name,
        },
    }
}

func (b *Builder) Range(min, max float64) *Builder {
    b.param.Min = min
    b.param.Max = max
    return b
}

func (b *Builder) Default(value float64) *Builder {
    b.param.DefaultValue = value
    return b
}

func (b *Builder) Unit(unit string) *Builder {
    b.param.Unit = unit
    return b
}

// Usage: Clean, readable parameter definitions
params := param.NewRegistry()
params.Add(
    param.New(GainParamID, "Gain").
        Range(-24, 24).
        Default(0).
        Unit("dB").
        Flags(param.CanAutomate),
    
    param.New(MixParamID, "Mix").
        Range(0, 100).
        Default(50).
        Unit("%"),
)
```

#### 3. State Management
```go
// pkg/framework/state/manager.go
package state

type Manager struct {
    version  int
    registry *param.Registry
    custom   CustomStateFunc
}

// Automatic versioned state management
func (m *Manager) Save(stream vst3.IBStream) error {
    writer := newWriter(stream)
    
    // Write version header
    writer.WriteInt32(m.version)
    
    // Automatically save all parameters
    for _, param := range m.registry.All() {
        writer.WriteParamID(param.ID)
        writer.WriteFloat64(param.Value)
    }
    
    // Optional custom state
    if m.custom != nil {
        return m.custom(writer)
    }
    
    return writer.Error()
}
```

#### 4. Event Processing
```go
// pkg/framework/event/processor.go
package event

type Processor struct {
    paramQueue *param.Queue
    midiQueue  *midi.Queue
}

// Framework processes events, developer gets clean callbacks
func (p *Processor) Process(eventList vst3.IEventList) {
    for i := 0; i < eventList.GetEventCount(); i++ {
        event := eventList.GetEvent(i)
        
        switch event.Type {
        case vst3.EventTypeParameter:
            p.paramQueue.Add(event.ParamID, event.Value, event.SampleOffset)
        case vst3.EventTypeMIDI:
            p.midiQueue.Add(event.Data, event.SampleOffset)
        }
    }
}

### Phase 3: DSP Package Implementation
**Goal**: Provide comprehensive audio processing utilities with zero allocations

#### Performance Critical Requirements
- **Zero Allocations**: No memory allocations in the audio processing path
- **Pre-allocation**: All buffers and state allocated during initialization
- **Cache-Friendly**: Data structures optimized for CPU cache efficiency
- **SIMD-Ready**: Structures aligned for potential SIMD optimizations

#### 1. Filter Package - Clean, Performant APIs with Zero Allocations
```go
// pkg/dsp/filter/biquad.go
package filter

type Biquad struct {
    a0, a1, a2 float32  // Feedforward coefficients
    b1, b2     float32  // Feedback coefficients
    x1, x2     float32  // Input delay line
    y1, y2     float32  // Output delay line
}

// High-level filter factory functions
func Lowpass(cutoff, sampleRate float64) *Biquad {
    omega := 2.0 * math.Pi * cutoff / sampleRate
    sin := math.Sin(omega)
    cos := math.Cos(omega)
    alpha := sin / math.Sqrt(2.0)
    
    b0 := (1.0 - cos) / 2.0
    b1 := 1.0 - cos
    b2 := (1.0 - cos) / 2.0
    a0 := 1.0 + alpha
    a1 := -2.0 * cos
    a2 := 1.0 - alpha
    
    return &Biquad{
        a0: float32(b0 / a0),
        a1: float32(b1 / a0),
        a2: float32(b2 / a0),
        b1: float32(a1 / a0),
        b2: float32(a2 / a0),
    }
}

// In-place processing - zero allocations
func (b *Biquad) Process(samples []float32) {
    // No allocations - direct mutation of input buffer
    for i := range samples {
        input := samples[i]
        output := b.a0*input + b.a1*b.x1 + b.a2*b.x2 - b.b1*b.y1 - b.b2*b.y2
        
        b.x2 = b.x1
        b.x1 = input
        b.y2 = b.y1
        b.y1 = output
        
        samples[i] = output
    }
}

// Process stereo - reuses existing buffers
func (b *Biquad) ProcessStereo(left, right []float32) {
    b.Process(left)  // In-place, no allocation
    b.Process(right) // In-place, no allocation
}

// State variable filter for more flexibility
type SVF struct {
    cutoff     float32
    resonance  float32
    sampleRate float32
    lp, bp, hp float32
}

func (f *SVF) Process(input float32) (lowpass, bandpass, highpass float32) {
    // Efficient state variable filter implementation
    // Returns all outputs simultaneously
}
```

#### 2. Oscillator Package - Ready-to-Use Sound Generators
```go
// pkg/dsp/oscillator/basic.go
package oscillator

type Oscillator struct {
    phase      float64
    frequency  float64
    sampleRate float64
}

func (o *Oscillator) SetFrequency(hz float64) {
    o.frequency = hz
}

func (o *Oscillator) Sine() float32 {
    sample := math.Sin(2.0 * math.Pi * o.phase)
    o.advance()
    return float32(sample)
}

func (o *Oscillator) Saw() float32 {
    sample := 2.0*o.phase - 1.0
    o.advance()
    return float32(sample)
}

func (o *Oscillator) advance() {
    o.phase += o.frequency / o.sampleRate
    if o.phase >= 1.0 {
        o.phase -= 1.0
    }
}

// Wavetable oscillator for complex waveforms
type Wavetable struct {
    table      []float32
    phase      float64
    frequency  float64
    sampleRate float64
}

func (w *Wavetable) Process() float32 {
    // Interpolated wavetable lookup
}
```

#### 3. Envelope Package - Musical Dynamics
```go
// pkg/dsp/envelope/adsr.go
package envelope

type ADSR struct {
    attack, decay, sustain, release float64
    sampleRate                      float64
    state                          State
    level                          float64
}

type State int

const (
    Idle State = iota
    Attack
    Decay
    Sustain
    Release
)

func (e *ADSR) Gate(on bool) {
    if on {
        e.state = Attack
    } else if e.state != Idle {
        e.state = Release
    }
}

func (e *ADSR) Process() float32 {
    switch e.state {
    case Attack:
        e.level += 1.0 / (e.attack * e.sampleRate)
        if e.level >= 1.0 {
            e.level = 1.0
            e.state = Decay
        }
    case Decay:
        e.level -= (1.0 - e.sustain) / (e.decay * e.sampleRate)
        if e.level <= e.sustain {
            e.level = e.sustain
            e.state = Sustain
        }
    case Release:
        e.level -= e.level / (e.release * e.sampleRate)
        if e.level <= 0.0001 {
            e.level = 0.0
            e.state = Idle
        }
    }
    return float32(e.level)
}
```

#### 4. Delay Package - Time-Based Effects with Pre-allocated Buffers
```go
// pkg/dsp/delay/line.go
package delay

type Line struct {
    buffer     []float32  // Pre-allocated at creation
    writeIndex int
    maxDelay   int
    // Pre-calculated for zero-allocation processing
    invSampleRate float32
}

func NewLine(maxDelayMs float64, sampleRate float64) *Line {
    samples := int(maxDelayMs * sampleRate / 1000.0)
    return &Line{
        buffer:        make([]float32, samples), // One-time allocation
        maxDelay:      samples,
        invSampleRate: float32(1000.0 / sampleRate),
    }
}

// Zero allocations during processing
func (d *Line) Process(input float32, delayMs float32) float32 {
    // Write input
    d.buffer[d.writeIndex] = input
    
    // Calculate read position - no allocations
    delaySamples := int(delayMs * d.invSampleRate)
    readIndex := d.writeIndex - delaySamples
    if readIndex < 0 {
        readIndex += len(d.buffer)
    }
    
    // Advance write position
    d.writeIndex++
    if d.writeIndex >= len(d.buffer) {
        d.writeIndex = 0
    }
    
    return d.buffer[readIndex]
}

// Process block - zero allocations
func (d *Line) ProcessBlock(input, output []float32, delayMs float32) {
    for i := range input {
        output[i] = d.Process(input[i], delayMs)
    }
}

// Multi-tap delay for complex effects
type MultiTap struct {
    line     *Line
    taps     []Tap
    feedback float32
    mix      float32
}

type Tap struct {
    DelayMs float64
    Gain    float32
    Pan     float32
}
```

#### 5. Dynamics Package - Level Control
```go
// pkg/dsp/dynamics/compressor.go
package dynamics

type Compressor struct {
    threshold float32
    ratio     float32
    attack    float32
    release   float32
    envelope  float32
}

func (c *Compressor) Process(input float32) float32 {
    // Envelope detection
    inputLevel := math.Abs(float64(input))
    
    if inputLevel > float64(c.envelope) {
        c.envelope += (float32(inputLevel) - c.envelope) * c.attack
    } else {
        c.envelope += (float32(inputLevel) - c.envelope) * c.release
    }
    
    // Gain calculation
    gain := float32(1.0)
    if c.envelope > c.threshold {
        excess := c.envelope - c.threshold
        gain = c.threshold + excess/c.ratio
        gain = gain / c.envelope
    }
    
    return input * gain
}

### Phase 4: Developer Experience Layer
**Goal**: Make plugin development as simple as possible

#### 1. Plugin Builder - Zero Boilerplate Plugin Creation
```go
// pkg/framework/plugin/builder.go
package plugin

type Builder struct {
    plugin *Base
}

func NewEffect(name string) *Builder {
    return &Builder{
        plugin: &Base{
            info: Info{
                Name:     name,
                Category: "Fx",
                Version:  "1.0.0",
            },
        },
    }
}

func (b *Builder) Company(name string) *Builder {
    b.plugin.info.Vendor = name
    return b
}

func (b *Builder) UniqueID(id string) *Builder {
    b.plugin.info.ID = id
    return b
}

func (b *Builder) Parameters(params ...*param.Builder) *Builder {
    registry := param.NewRegistry()
    for _, p := range params {
        registry.Add(p.Build())
    }
    b.plugin.params = registry
    return b
}

func (b *Builder) Process(fn ProcessFunc) *Builder {
    b.plugin.processor = fn
    return b
}

// Usage: Complete plugin in minimal code
func NewDelayPlugin() plugin.Plugin {
    return plugin.NewEffect("Simple Delay").
        Company("Example Audio").
        UniqueID("com.example.delay").
        Parameters(
            param.New(1, "Time").Range(0, 1000).Default(250).Unit("ms"),
            param.New(2, "Feedback").Range(0, 0.99).Default(0.5),
            param.New(3, "Mix").Range(0, 1).Default(0.5),
        ).
        Process(func(p *ProcessContext) {
            // Just implement your DSP here
            delay := p.DSP.(*DelayProcessor)
            delay.SetTime(p.Param(1))
            delay.SetFeedback(p.Param(2))
            delay.SetMix(p.Param(3))
            delay.Process(p.Input, p.Output)
        }).
        Build()
}
```

#### 2. Quick Start Templates
```go
// pkg/framework/templates/effect.go
package templates

// Developer runs: vst3go new effect MyPlugin
// Generates this ready-to-use template:

const EffectTemplate = `package main

import (
    "github.com/vst3go/framework/plugin"
    "github.com/vst3go/framework/param"
)

type MyPlugin struct {
    plugin.Base
    // Add your DSP fields here
}

func New() *MyPlugin {
    p := &MyPlugin{}
    
    p.Info = plugin.Info{
        ID:      "com.yourcompany.myplugin",
        Name:    "MyPlugin",
        Version: "1.0.0",
        Vendor:  "Your Company",
    }
    
    p.Parameters = param.NewRegistry(
        param.New(1, "Parameter 1").Range(0, 1).Default(0.5),
        // Add more parameters
    )
    
    return p
}

// This is where you implement your audio processing
func (p *MyPlugin) ProcessAudio(input, output [][]float32) {
    // Your DSP code here
    for ch := range input {
        for i := range input[ch] {
            output[ch][i] = input[ch][i] // Pass through for now
        }
    }
}

func main() {
    plugin.Register(New)
}
`
```

#### 3. Development Helpers
```go
// pkg/framework/debug/logger.go
package debug

// Conditional compilation for debug builds
// +build debug

type Logger struct {
    enabled bool
    prefix  string
}

func (l *Logger) Param(id int32, value float64) {
    if l.enabled {
        fmt.Printf("[%s] Param %d = %.3f\n", l.prefix, id, value)
    }
}

// Usage in plugin - automatically removed in release builds
debug.Log.Param(paramID, value)

// pkg/framework/profile/measure.go
package profile

// Simple performance measurement
func Measure(name string, fn func()) {
    if !enabled {
        fn()
        return
    }
    
    start := time.Now()
    fn()
    duration := time.Since(start)
    
    metrics.Record(name, duration)
}

// Usage
profile.Measure("ProcessAudio", func() {
    p.ProcessAudio(input, output)
})
```

#### 4. Common Plugin Patterns
```go
// pkg/framework/patterns/stereo.go
package patterns

// Stereo processing helper
type StereoProcessor struct {
    left  Processor
    right Processor
}

func (s *StereoProcessor) Process(input, output [][]float32) {
    if len(input) >= 2 {
        s.left.Process(input[0], output[0])
        s.right.Process(input[1], output[1])
    }
}

// pkg/framework/patterns/bypass.go
package patterns

// Automatic bypass handling
type Bypassable struct {
    bypassed bool
    process  ProcessFunc
}

func (b *Bypassable) Process(input, output [][]float32) {
    if b.bypassed {
        // Copy input to output
        for ch := range input {
            copy(output[ch], input[ch])
        }
    } else {
        b.process(input, output)
    }
}

### Phase 5: Example Refactoring - Dramatic Simplification
**Goal**: Update examples to use new framework, eliminating duplication

#### Before: Current Gain Plugin (200+ lines of boilerplate)
```go
// examples/gain/main.go - CURRENT VERSION
package main

import (
    "C"
    "sync"
    "unsafe"
    // ... many imports
)

type GainPlugin struct {
    gain     float64
    gainLock sync.RWMutex
    bypass   bool
    // ... many fields for state management
}

func (g *GainPlugin) Initialize(context unsafe.Pointer) int32 {
    // 30+ lines of initialization
    // Parameter registration
    // Bus configuration
    // Extension setup
}

func (g *GainPlugin) Process(data unsafe.Pointer) int32 {
    processData := (*vst3.ProcessData)(data)
    
    // 20 lines of parameter update handling
    if processData.InputParameterChanges != nil {
        // ... parameter change processing
    }
    
    // 10 lines of buffer validation
    if processData.NumInputs == 0 || processData.NumOutputs == 0 {
        return vst3.ResultOK
    }
    
    // 15 lines of audio buffer setup
    inputs := processData.Inputs
    outputs := processData.Outputs
    
    // Finally, 5 lines of actual audio processing
    for i := 0; i < processData.NumSamples; i++ {
        for ch := 0; ch < numChannels; ch++ {
            outputs[ch][i] = inputs[ch][i] * g.gain
        }
    }
    
    return vst3.ResultOK
}

// ... 100+ more lines of boilerplate methods
```

#### After: New Gain Plugin (30 lines total!)
```go
// examples/gain/main.go - NEW VERSION
package main

import (
    "github.com/vst3go/framework/plugin"
    "github.com/vst3go/framework/param"
    "github.com/vst3go/dsp/amp"
)

func main() {
    plugin.Register(NewGainPlugin)
}

func NewGainPlugin() plugin.Plugin {
    gainAmp := amp.New()
    
    return plugin.NewEffect("Simple Gain").
        Company("Example Audio").
        UniqueID("com.example.gain").
        Parameters(
            param.New(1, "Gain").
                Range(-24, 24).
                Default(0).
                Unit("dB").
                Flags(param.CanAutomate),
            param.New(2, "Bypass").
                Toggle().
                Default(false),
        ).
        Process(func(ctx *plugin.ProcessContext) {
            // Get current parameter values
            gainDB := ctx.Param(1)
            bypassed := ctx.Param(2) > 0.5
            
            if bypassed {
                ctx.PassThrough()
            } else {
                gainAmp.SetGainDB(gainDB)
                gainAmp.Process(ctx.Input, ctx.Output)
            }
        }).
        Build()
}
```

#### Complex Example: Delay Plugin Transformation
```go
// Before: 300+ lines with all VST3 boilerplate
// After: 50 lines focusing on delay algorithm

package main

import (
    "github.com/vst3go/framework/plugin"
    "github.com/vst3go/framework/param"
    "github.com/vst3go/dsp/delay"
    "github.com/vst3go/dsp/filter"
)

func main() {
    plugin.Register(NewDelayPlugin)
}

func NewDelayPlugin() plugin.Plugin {
    // Create DSP components
    delayLine := delay.NewStereo(1000, 44100) // 1 second max
    lpFilter := filter.Lowpass(5000, 44100)
    
    return plugin.NewEffect("Filtered Delay").
        Company("Example Audio").
        UniqueID("com.example.delay").
        Parameters(
            param.New(1, "Time").Range(0, 1000).Default(250).Unit("ms"),
            param.New(2, "Feedback").Range(0, 0.95).Default(0.5),
            param.New(3, "Filter").Range(100, 10000).Default(5000).Unit("Hz"),
            param.New(4, "Mix").Range(0, 1).Default(0.5),
        ).
        Process(func(ctx *plugin.ProcessContext) {
            // Update DSP parameters
            delayLine.SetDelayMs(ctx.Param(1))
            delayLine.SetFeedback(ctx.Param(2))
            lpFilter.SetCutoff(ctx.Param(3), ctx.SampleRate)
            
            // Process audio with filtering in feedback path
            delayLine.ProcessWithFilter(
                ctx.Input, 
                ctx.Output,
                lpFilter,
                ctx.Param(4), // mix
            )
        }).
        Build()
}
```

#### Key Improvements Demonstrated

1. **90% Less Code**: From 200-300 lines to 30-50 lines
2. **Zero C Code**: No unsafe pointers or C imports needed
3. **Type Safety**: Full Go type checking throughout
4. **Clear Intent**: Code reads like audio processing description
5. **No Boilerplate**: Framework handles all VST3 protocol details

#### Migration Guide for Existing Plugins

```go
// Step 1: Identify your core DSP algorithm
// This is usually < 10% of your current code

// Step 2: Choose appropriate framework base
plugin.NewEffect()      // For audio effects
plugin.NewInstrument()  // For synthesizers
plugin.NewMIDI()        // For MIDI processors

// Step 3: Define parameters using builder API
param.New(id, name).Range(min, max).Default(value)

// Step 4: Implement ProcessAudio with just your algorithm
Process(func(ctx *ProcessContext) {
    // Your DSP code here
})

// Step 5: Delete all the old boilerplate!
```

## Migration Strategy

### Step 1: Parallel Development
- Keep existing code functional
- Build new framework alongside
- No breaking changes initially

### Step 2: Incremental Adoption
- Update one example at a time
- Validate functionality with each change
- Maintain test coverage

### Step 3: Legacy Cleanup
- Remove old implementations
- Delete unused code
- Update documentation

### Step 4: API Stabilization
- Gather feedback from usage
- Refine based on real needs
- Document stable API surface

## Quality Assurance

### Testing Strategy
1. **Unit Tests**: Each framework package
2. **Integration Tests**: Cross-package interactions
3. **Plugin Tests**: Full plugin validation
4. **Performance Tests**: Audio processing benchmarks

### Validation Criteria
- All examples pass VST3 validator
- No performance regression
- Improved developer experience metrics
- Reduced code duplication

## Success Metrics

### Quantitative
- 80% reduction in example boilerplate
- Zero business logic in C bridge
- 100% test coverage for framework packages
- Sub-microsecond overhead for framework abstractions
- **Zero allocations in audio processing path** (verified by tests)
- Pre-allocated buffer usage < 10MB for typical plugins

### Qualitative
- Intuitive API design (Go-idiomatic)
- Clear separation of concerns
- Easy debugging and error diagnosis
- Comprehensive documentation

## Risk Mitigation

### Technical Risks
1. **Performance Impact**
   - Mitigation: Benchmark critical paths
   - Fallback: Provide escape hatches to lower layers

2. **API Design Mistakes**
   - Mitigation: Iterative development with feedback
   - Fallback: Version 2 with lessons learned

3. **Compatibility Issues**
   - Mitigation: Extensive testing across hosts
   - Fallback: Host-specific workarounds

### Process Risks
1. **Scope Creep**
   - Mitigation: Strict adherence to phases
   - Fallback: Defer features to future versions

2. **Over-Engineering**
   - Mitigation: Start simple, iterate
   - Fallback: Remove unnecessary abstractions

## Timeline Estimation

### Phase Duration (Estimated)
- Phase 1: 1-2 weeks (Bridge cleanup)
- Phase 2: 2-3 weeks (Core framework)
- Phase 3: 3-4 weeks (DSP package)
- Phase 4: 2-3 weeks (Developer tools)
- Phase 5: 1-2 weeks (Example refactoring)

Total: 9-14 weeks for complete refactor

## Next Steps

1. **Immediate Actions**
   - Create feature branches for each phase
   - Set up CI/CD for framework packages
   - Begin Phase 1 bridge audit

2. **Communication**
   - Document progress in project wiki
   - Regular updates on breaking changes
   - Solicit feedback from early adopters

3. **Validation**
   - Continuous testing with VST3 validator
   - Performance benchmarking infrastructure
   - User experience feedback loops

## Complete Developer Experience

### From Zero to Plugin in Minutes

```bash
# Install VST3Go
go install github.com/vst3go/vst3go@latest

# Create new plugin project
vst3go new effect MyReverb
cd myreverb

# Build and install
make install

# That's it! Plugin is ready to use in your DAW
```

### What Developers Write vs What Framework Provides

#### Developer Writes (30 lines):
```go
func NewReverbPlugin() plugin.Plugin {
    reverb := dsp.NewReverb()
    
    return plugin.NewEffect("My Reverb").
        Company("My Company").
        Parameters(
            param.New(1, "Room Size").Range(0, 1).Default(0.5),
            param.New(2, "Damping").Range(0, 1).Default(0.5),
            param.New(3, "Mix").Range(0, 1).Default(0.3),
        ).
        Process(func(ctx *plugin.ProcessContext) {
            reverb.SetRoomSize(ctx.Param(1))
            reverb.SetDamping(ctx.Param(2))
            reverb.Process(ctx.Input, ctx.Output, ctx.Param(3))
        }).
        Build()
}
```

#### Framework Provides (Invisible to Developer):
- Complete VST3 protocol implementation
- Thread-safe parameter management
- State saving/loading with versioning
- Sample-accurate parameter automation
- Proper bus configuration
- MIDI event handling
- Host communication
- Extension support
- Error handling and recovery
- Performance optimizations

## Success Criteria

### For Developers
1. **Time to First Plugin**: < 5 minutes from install to working plugin
2. **Learning Curve**: Go developers productive immediately, no VST3 knowledge required
3. **Code Clarity**: Plugin code reads like audio algorithm description
4. **Debugging**: Clear error messages, no mysterious crashes

### For the Framework
1. **Performance**: < 1% CPU overhead vs raw C implementation
2. **Compatibility**: Works in all major DAWs (Ableton, Logic, Reaper, etc.)
3. **Stability**: Zero crashes in normal use
4. **Maintainability**: Clear separation of concerns, easy to extend

## Conclusion

This refactoring strategy transforms VST3Go from a complex C/Go hybrid into a clean, layered architecture that truly serves Go developers. The key insights are:

1. **Radical Simplification of C Bridge**: By moving ALL business logic to Go, the C bridge becomes trivial to maintain and could even be auto-generated.

2. **Rich Go Framework**: With the bridge simplified, we can focus on building a comprehensive, idiomatic Go framework that makes audio development a joy.

3. **Developer First**: Every decision prioritizes developer experience. Complex VST3 concepts are hidden behind simple, intuitive APIs.

4. **Performance Without Compromise**: The layered architecture ensures developers can drop down to lower levels when needed for performance-critical code.

5. **Community Building**: By making plugin development accessible to Go developers, we open up audio programming to a new community.

The phased approach ensures we can deliver value incrementally while maintaining stability. Most importantly, this architecture makes VST3Go not just a binding to VST3, but a true Go framework for audio development that happens to use VST3 as its plugin format.

With this approach, a Go developer can go from zero knowledge of VST3 to a working plugin in minutes, not days. That's the power of good framework design.