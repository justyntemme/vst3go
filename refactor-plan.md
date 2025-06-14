# VST3Go Comprehensive Refactor Plan

## Executive Summary

This document outlines critical architectural refactoring needed to align the VST3Go codebase with its intended 4-layer architecture. The primary issues are layer violations, mixed responsibilities, and missing abstractions that lead to code duplication and developer friction.

## Critical Architectural Issues

### 1. Layer Violations and Confusion

#### Current State
- **C Bridge contamination**: The `pkg/plugin/` directory mixes C bridge code with Go abstractions
- **Duplicate bridges**: Both `pkg/bridge/` and `pkg/plugin/cbridge/` exist
- **Business logic in C**: Helper functions in `wrapper.go` implement logic in C

#### Required Changes
```
CURRENT (BROKEN):
pkg/
├── bridge/          # Minimal but unused
├── plugin/          # MIXED: C bridge + Go abstractions
│   ├── cbridge/     # Duplicate bridge
│   ├── wrapper.go   # C code with business logic
│   └── *.go         # Mixed C imports and Go code
└── framework/       # Correct Go abstractions

TARGET (FIXED):
pkg/
├── cbridge/         # ALL C bridge code (Layer 1)
│   ├── bridge.go    # C imports only
│   └── wrapper.c    # Minimal C helpers
├── core/            # Core plugin infrastructure (Layer 2)
│   ├── plugin.go    # Plugin interfaces
│   ├── processor.go # Processor base
│   └── lifecycle.go # Lifecycle management
└── framework/       # Rich abstractions (Layer 2-3)
```

### 2. Missing Core Abstractions

#### BaseProcessor Implementation
```go
// pkg/core/processor.go
type BaseProcessor struct {
    params     *param.Registry
    buses      *bus.Configuration
    sampleRate float64
    mu         sync.RWMutex
}

func (bp *BaseProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    bp.sampleRate = sampleRate
    return nil
}

func (bp *BaseProcessor) GetLatencySamples() int32 { return 0 }
func (bp *BaseProcessor) GetTailSamples() int32 { return 0 }
```

#### Plugin Registration Helper
```go
// pkg/core/registry.go
type PluginRegistry struct {
    factoryInfo FactoryInfo
}

func (r *PluginRegistry) RegisterPlugin(p Plugin) {
    SetFactoryInfo(r.factoryInfo)
    Register(p)
}

// Example usage:
var ExamplesRegistry = &PluginRegistry{
    factoryInfo: FactoryInfo{
        Vendor: "VST3Go Examples",
        URL:    "https://github.com/vst3go/examples",
        Email:  "examples@vst3go.com",
    },
}
```

### 3. API Inconsistencies

#### Naming Issues
- **Problem**: Duplicate constants with different casing
- **Fix**: Single, consistent naming convention
```go
// BAD (current)
const (
    ResultOK = 0
    ResultOk = 0  // Duplicate!
    ResultTrue = 0
)

// GOOD (fixed)
const (
    ResultOK = 0
    // Use aliases for compatibility
    ResultOk = ResultOK // Deprecated: use ResultOK
)
```

#### Error Handling
```go
// pkg/core/errors.go
type PluginError struct {
    Op  string // Operation
    Err error  // Underlying error
}

func (e *PluginError) Error() string {
    return fmt.Sprintf("plugin %s: %v", e.Op, e.Err)
}

func (e *PluginError) Unwrap() error { return e.Err }

// Usage:
if err := processor.Initialize(sampleRate, blockSize); err != nil {
    return &PluginError{Op: "initialize", Err: err}
}
```

### 4. Thread Safety Issues

#### Current Problems
- Global state with complex locking in wrapper.go
- Undocumented thread safety guarantees
- Potential deadlocks in component handler

#### Solution
```go
// Document thread safety clearly
type Processor interface {
    // ProcessAudio is called from the audio thread.
    // It MUST NOT allocate or block.
    // Parameters are guaranteed stable during processing.
    ProcessAudio(ctx *process.Context)
    
    // Initialize is called from the main thread.
    // Thread-safe with respect to ProcessAudio.
    Initialize(sampleRate float64, maxBlockSize int32) error
}
```

### 5. Code Duplication in Examples

#### Current State
All 18 examples repeat:
- 6-line init() function
- Empty main() function
- GetLatencySamples() returning 0
- GetTailSamples() returning 0

#### Solution
```go
// Using BaseProcessor eliminates 4 methods
type GainProcessor struct {
    *core.BaseProcessor
    // Only plugin-specific fields
}

// Using registry eliminates init() boilerplate
func init() {
    core.ExamplesRegistry.RegisterPlugin(&GainPlugin{})
}
```

## Refactor Plan

### Phase 1: Critical Architecture Fixes (Week 1)

1. **Consolidate C Bridge**
   - Move all C code to `pkg/cbridge/`
   - Remove business logic from C helpers
   - Delete duplicate bridge packages

2. **Create Core Package**
   - Extract interfaces to `pkg/core/`
   - Implement BaseProcessor
   - Add plugin registry

3. **Fix Thread Safety**
   - Document all thread safety guarantees
   - Simplify locking in wrapper
   - Add race detector tests

### Phase 2: API Cleanup (Week 2)

1. **Fix Naming Inconsistencies**
   - Resolve duplicate constants
   - Consistent parameter naming
   - Clear interface hierarchy

2. **Improve Error Handling**
   - Domain-specific error types
   - Error wrapping throughout
   - Remove panic recovery

3. **Add Missing Helpers**
   - Common parameter sets
   - Bus configuration helpers
   - State management utilities

### Phase 3: Developer Experience (Week 3)

1. **Reduce Boilerplate**
   - Update all examples to use BaseProcessor
   - Use plugin registry
   - Extract common patterns

2. **Improve Documentation**
   - Architecture diagrams
   - Thread safety guide
   - Migration guide

3. **Add Convenience APIs**
   - SimpleEffect helper
   - Parameter group builders
   - Common DSP chains

## Migration Impact

### For Plugin Developers

**Before:**
```go
type MyProcessor struct {
    params *param.Registry
    buses  *bus.Configuration
    sampleRate float64
}

func (p *MyProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
    p.sampleRate = sampleRate
    return nil
}

func (p *MyProcessor) GetLatencySamples() int32 { return 0 }
func (p *MyProcessor) GetTailSamples() int32 { return 0 }
// ... more boilerplate
```

**After:**
```go
type MyProcessor struct {
    *core.BaseProcessor
    // Just plugin-specific code
}

// No boilerplate methods needed!
```

### For Framework Users

- All existing code continues to work
- New helpers are opt-in
- Clear migration path provided

## Success Metrics

1. **Code Reduction**: 20-30% less code in examples
2. **Layer Clarity**: Clean separation between C bridge and Go
3. **Safety**: No race conditions detected
4. **Consistency**: Single way to do common tasks
5. **Documentation**: 100% public API documented

## Implementation Priority

1. **Critical** (Must fix):
   - Layer violations
   - Thread safety issues
   - C bridge contamination

2. **Important** (Should fix):
   - Code duplication
   - Missing abstractions
   - Error handling

3. **Nice to have** (Could improve):
   - Convenience APIs
   - Documentation
   - Helper utilities

---

# Original Lifecycle Improvements

## 1. Better Lifecycle Helpers

### Background
Currently, every plugin repeats the same boilerplate code for:
- Field declarations (params, buses, sampleRate)
- SetActive implementations
- GetParameters/GetBuses methods
- Factory registration (6 lines in every init())

### Proposed Implementation

#### 1.1 BaseProcessor
Create a base struct that encapsulates common fields and lifecycle management:

```go
// pkg/plugin/lifecycle.go
type BaseProcessor struct {
    params     *param.Registry
    buses      *bus.Configuration
    sampleRate float64
    
    // Optional lifecycle callbacks
    onInitialize func(sampleRate float64, maxBlockSize int32) error
    onSetActive  func(active bool) error
    onReset      func()
}

// Fluent API for configuration
func (bp *BaseProcessor) WithInitializeCallback(fn func(sampleRate float64, maxBlockSize int32) error) *BaseProcessor
func (bp *BaseProcessor) WithSetActiveCallback(fn func(active bool) error) *BaseProcessor
func (bp *BaseProcessor) WithResetCallback(fn func()) *BaseProcessor

// Standard implementations
func (bp *BaseProcessor) Initialize(sampleRate float64, maxBlockSize int32) error
func (bp *BaseProcessor) SetActive(active bool) error
func (bp *BaseProcessor) GetParameters() *param.Registry
func (bp *BaseProcessor) GetBuses() *bus.Configuration
func (bp *BaseProcessor) GetSampleRate() float64
```

#### 1.2 State Management
Automatic reset management for DSP components:

```go
// pkg/plugin/state.go
type Resettable interface {
    Reset()
}

type StateManager struct {
    components []Resettable
}

func (sm *StateManager) Register(component Resettable)
func (sm *StateManager) RegisterFunc(fn func())
func (sm *StateManager) ResetAll()

// Integration with BaseProcessor
type ProcessorState struct {
    manager *StateManager
}

func (ps *ProcessorState) RegisterResettable(component Resettable)
func (ps *ProcessorState) Reset()
```

##  Process Context Improvements

### Current Issues
- Limited helper methods for common patterns
- No built-in parameter smoothing support

### Proposed Improvements

```go
// pkg/framework/process/context_helpers.go

// Parameter smoothing support
func (ctx *Context) GetSmoothedParameter(id uint32, smoother *param.Smoother) float64

// Common audio patterns
func (ctx *Context) ProcessStereoParallel(fn func(l, r []float32))
func (ctx *Context) ProcessWithSidechain(main, sidechain func(ch int, in, out []float32))

// Buffer management
func (ctx *Context) GetTempBuffer(size int) []float32
func (ctx *Context) ReleaseTempBuffer(buf []float32)
```

## 3. Documentation Overhaul

### Current State
- Minimal inline documentation
- No comprehensive guides
- Examples lack explanatory comments

### Implementation Plan

1. **API Documentation**
   - Document all public types and methods
   - Add package-level documentation
   - Include usage examples in godoc

2. **Developer Guides**
   - Getting Started guide
   - DSP Component guide
   - Parameter System guide
   - State Management guide

3. **Architecture Documentation**
   - Layer separation explanation
   - Plugin lifecycle diagrams
   - Data flow documentation

##  Missing Error Handling Patterns

### Areas to Improve

#### 5.1 Parameter Access Safety
```go
// Current unsafe pattern:
value := p.params.Get(ParamID).GetValue() // Can panic if param not found

// Proposed safe pattern in framework:
func (r *Registry) GetSafe(id uint32) (*Parameter, error)
func (r *Registry) GetValueSafe(id uint32) (float64, error)
```

#### 5.2 DSP Component Initialization
```go
// Add validation in DSP constructors:
func NewCompressor(sampleRate float64) (*Compressor, error) {
    if sampleRate <= 0 {
        return nil, fmt.Errorf("invalid sample rate: %f", sampleRate)
    }
    // ...
}
```

#### 5.3 Process Context Validation
```go
// Add to process context:
func (ctx *Context) Validate() error {
    if ctx.NumSamples() == 0 {
        return ErrNoSamples
    }
    if ctx.NumInputChannels() == 0 {
        return ErrNoInputChannels
    }
    return nil
}
```

## 6. Unsafe Parameter Access Patterns

### Current Issues in Library

1. **Registry Access** (`pkg/framework/param/registry.go`)
   - No nil checks in Get() method
   - No bounds checking for parameter IDs

2. **Parameter Updates** (`pkg/framework/param/parameter.go`)
   - SetValue doesn't validate range
   - No thread-safe access

3. **Process Context** (`pkg/framework/process/context.go`)
   - ParamPlain() doesn't check if parameter exists

### Fixes Needed

```go
// pkg/framework/param/registry.go
func (r *Registry) Get(id uint32) *Parameter {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    if param, ok := r.params[id]; ok {
        return param
    }
    return nil // Instead of panic
}

// pkg/framework/param/parameter.go
func (p *Parameter) SetValue(normalized float64) error {
    if normalized < 0 || normalized > 1 {
        return fmt.Errorf("normalized value out of range: %f", normalized)
    }
    
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.value = normalized
    return nil
}

// pkg/framework/process/context.go
func (c *Context) ParamPlainSafe(id uint32) (float64, error) {
    if c.processor == nil {
        return 0, ErrNoProcessor
    }
    
    param := c.processor.GetParameters().Get(id)
    if param == nil {
        return 0, fmt.Errorf("parameter %d not found", id)
    }
    
    return param.GetPlainValue(), nil
}
```

## Implementation Priority

### Phase 1: Critical Safety Fixes
1. Fix unsafe parameter access in framework
2. Add thread safety to parameter updates
3. Validate process context

### Phase 2: Lifecycle Improvements
1. Implement BaseProcessor
2. Add state management helpers

### Phase 3: Developer Experience
1. Process context helpers
2. Documentation overhaul
3. Factory info constants

### Phase 4: Advanced Features
1. Parameter smoothing integration

## Appendix: Specific Safety Issues Found

### Safe Patterns Already in Place
- Registry.Get() calls properly check for nil
- Thread safety with mutexes and atomic operations
- Bounds checking in most array access

### Issues to Address

#### 1. Unsafe Pointer Operations
- `pkg/framework/param/parameter.go:139-143` - Uses unsafe pointer conversion for atomic float operations
- Consider using `math.Float64bits()` and `math.Float64frombits()` instead

#### 2. Missing Error Returns in DSP Constructors
- `dsp/delay/delay.go` - New() doesn't validate parameters
- `dsp/filter/biquad.go` - NewBiquad() doesn't validate channels > 0
- `dsp/oscillator/oscillator.go` - Constructors don't validate sampleRate > 0
- `dsp/envelope/envelope.go` - Constructors don't validate sampleRate > 0

#### 3. Panic Instead of Error Returns
- `dsp/analysis/fft.go:304` - CrossCorrelation panics on invalid input
- `framework/bus/builder.go:232` - MustBuild panics (intentional but problematic)

#### 4. Silent Failures
- `framework/param/registry.go` - Add() silently skips duplicates
- No validation of parameter IDs

#### 5. Missing Nil Checks
- `framework/state/manager.go` - NewManager doesn't validate registry != nil
- `framework/process/multibus.go:39,49,59` - Assumes BusInfo is never nil

#### 6. Buffer Validation
- Many DSP operations assume valid input without checking bounds
- Process context doesn't validate audio buffers are properly allocated