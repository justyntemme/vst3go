# VST3Go API Inconsistencies - Detailed Analysis and Refactor Guide

## Overview

This document provides a comprehensive analysis of API inconsistencies in the VST3Go codebase and detailed implementation guides to resolve them. These inconsistencies create confusion, increase the likelihood of bugs, and make the framework harder to learn and use.

## 1. Constant Naming Inconsistencies

### Current State

In `pkg/vst3/types.go`:
```go
const (
    ResultOK    = 0
    ResultOk    = 0  // Duplicate with different casing
    ResultTrue  = 0  // Same value, different name
    ResultFalse = 1
)
```

### Problems
- Multiple names for the same value create confusion
- Developers might use inconsistent names across the codebase
- Go convention is to use CamelCase, not multiple variations

### Implementation Guide

#### Step 1: Audit All Constants
```bash
# Find all constant definitions
grep -r "const.*Result" pkg/
grep -r "const.*Param" pkg/
grep -r "const.*Bus" pkg/
```

#### Step 2: Create Canonical Constants
```go
// pkg/vst3/types.go
const (
    // Primary definitions - these are the canonical names
    ResultOK    = 0
    ResultFalse = 1
    
    // Deprecated aliases for backward compatibility
    // TODO: Remove in v2.0
    ResultOk   = ResultOK   // Deprecated: use ResultOK
    ResultTrue = ResultOK   // Deprecated: use ResultOK
)
```

#### Step 3: Update All Usage
```go
// Create a migration script
// tools/fix-constants.go
package main

import (
    "go/ast"
    "go/parser"
    "go/token"
    "golang.org/x/tools/go/ast/astutil"
)

var replacements = map[string]string{
    "ResultOk":   "ResultOK",
    "ResultTrue": "ResultOK",
}

// ... implement AST-based replacement
```

## 2. Parameter ID Conventions

### Current State

Different examples use different conventions:
```go
// Example 1: Individual constants
const ParamGain = 0
const ParamVolume = 1

// Example 2: iota block
const (
    ParamGain = iota
    ParamVolume
)

// Example 3: Typed constants
const (
    ParamGain uint32 = iota
    ParamVolume
)
```

### Problems
- Inconsistent type safety (some typed, some not)
- Different styles make examples hard to compare
- No clear "best practice" for developers

### Implementation Guide

#### Step 1: Define Standard Convention
```go
// pkg/framework/param/conventions.go
package param

// ParameterID is the standard type for parameter identifiers
type ID = uint32

// Standard parameter ID definition pattern:
// const (
//     ParamName ID = iota
//     ParamOther
// )
```

#### Step 2: Create Linting Rule
```yaml
# .golangci.yml
linters-settings:
  custom:
    param-id-convention:
      pattern: "const.*Param.*=.*iota"
      must-have-type: "param.ID"
```

#### Step 3: Update All Examples
```go
// Standardized approach in all examples
const (
    ParamGain param.ID = iota
    ParamVolume
    ParamBypass
)
```

## 3. Interface Hierarchy Confusion

### Current State

The relationship between interfaces is unclear:
```go
// Where do these interfaces belong?
type Plugin interface { /* ... */ }
type Processor interface { /* ... */ }
type Component interface { /* ... */ }
type StatefulProcessor interface { /* ... */ }
```

### Problems
- Unclear which interface to implement
- Optional interfaces (StatefulProcessor) not well integrated
- No clear documentation of interface relationships

### Implementation Guide

#### Step 1: Document Interface Hierarchy
```go
// pkg/core/interfaces.go
package core

// Plugin is the top-level interface that all VST3 plugins must implement.
// A Plugin is a factory that creates Processor instances.
type Plugin interface {
    GetInfo() Info
    CreateProcessor() Processor
}

// Processor handles the actual audio processing.
// Every plugin must implement this interface.
type Processor interface {
    // Core audio processing
    ProcessAudio(ctx *process.Context)
    
    // Lifecycle methods
    Initialize(sampleRate float64, maxBlockSize int32) error
    SetActive(active bool) error
    
    // Configuration access
    GetParameters() *param.Registry
    GetBuses() *bus.Configuration
    
    // Timing information
    GetLatencySamples() int32
    GetTailSamples() int32
}

// StatefulProcessor extends Processor with state persistence.
// Implement this interface if your plugin needs to save state
// beyond parameter values.
type StatefulProcessor interface {
    Processor
    SaveState(w io.Writer) error
    LoadState(r io.Reader) error
}
```

#### Step 2: Create Interface Compliance Helpers
```go
// pkg/core/helpers.go
package core

// Compile-time interface compliance checks
var (
    _ Processor = (*BaseProcessor)(nil)
    _ StatefulProcessor = (*StatefulBaseProcessor)(nil)
)

// ProcessorAdapter provides default implementations
type ProcessorAdapter struct {
    BaseProcessor
}

// Ensure all methods have sensible defaults
func (p *ProcessorAdapter) ProcessAudio(ctx *process.Context) {
    // Default: pass-through
    ctx.PassThrough()
}
```

## 4. Method Naming Inconsistencies

### Current State

Similar operations have different names:
```go
// Getting values
param.GetValue()      // Normalized
param.GetPlainValue() // Denormalized
param.Value()         // Which one?

// Setting values  
param.SetValue(v)     // Normalized
param.SetPlainValue(v) // Does this exist?
```

### Problems
- Unclear which method to use
- Inconsistent naming patterns
- Missing symmetry (get/set pairs)

### Implementation Guide

#### Step 1: Define Naming Convention
```go
// pkg/framework/param/parameter.go

// Normalized (0-1) accessors
func (p *Parameter) GetNormalized() float64
func (p *Parameter) SetNormalized(value float64) error

// Plain (denormalized) accessors  
func (p *Parameter) GetPlain() float64
func (p *Parameter) SetPlain(value float64) error

// Deprecated aliases
func (p *Parameter) GetValue() float64 { 
    return p.GetNormalized() 
}
func (p *Parameter) GetPlainValue() float64 { 
    return p.GetPlain() 
}
```

#### Step 2: Update Documentation
```go
// Package param provides parameter management for VST3 plugins.
//
// Parameters can be accessed in two forms:
//   - Normalized: 0.0 to 1.0 range (use GetNormalized/SetNormalized)
//   - Plain: Actual value in parameter's range (use GetPlain/SetPlain)
//
// Example:
//   gain := param.New(0, "Gain").Range(-12, 12).Build()
//   gain.SetPlain(-6.0)     // Set to -6 dB
//   gain.SetNormalized(0.5) // Set to middle of range
package param
```

## 5. Bus Configuration API

### Current State

Multiple ways to create bus configurations:
```go
// Method 1
buses := bus.NewStereoConfiguration()

// Method 2  
buses := &bus.Configuration{
    Inputs:  []bus.Info{{Channels: 2}},
    Outputs: []bus.Info{{Channels: 2}},
}

// Method 3
buses := bus.NewBuilder().
    AddInput("Main", 2).
    AddOutput("Main", 2).
    Build()
```

### Problems
- Too many ways to do the same thing
- Unclear which approach is recommended
- Some methods may not set all required fields

### Implementation Guide

#### Step 1: Define Primary API
```go
// pkg/framework/bus/config.go

// Primary API - factory functions for common cases
func Stereo() *Configuration           // Standard stereo I/O
func Mono() *Configuration             // Mono I/O  
func Generator() *Configuration        // No input, stereo output
func Effect(in, out int) *Configuration // Custom channel counts

// Advanced API - builder for complex cases
type Builder struct { /* ... */ }
func NewBuilder() *Builder

// Deprecated - remove eventually
func NewStereoConfiguration() *Configuration {
    return Stereo()
}
```

#### Step 2: Update Examples
```go
// Simple cases use factory functions
buses: bus.Stereo(),

// Complex cases use builder
buses: bus.NewBuilder().
    AddInput("Main", 2).
    AddInput("Sidechain", 2).
    AddOutput("Main", 2).
    Build(),
```

## 6. Error Type Inconsistencies

### Current State

Errors are handled inconsistently:
```go
// Some functions return bool
ok := param.Validate()

// Some return error
err := processor.Initialize()

// Some panic
bus.MustBuild() // panics on error

// Some fail silently
registry.Add(param) // ignores duplicates
```

### Problems
- No consistent error handling strategy
- Difficult to handle errors properly
- Silent failures hide bugs

### Implementation Guide

#### Step 1: Define Error Types
```go
// pkg/core/errors.go
package core

// Error represents a plugin framework error
type Error struct {
    Op   string // Operation
    Kind ErrorKind
    Err  error
}

type ErrorKind int

const (
    ErrorInvalid ErrorKind = iota
    ErrorNotFound
    ErrorDuplicate
    ErrorConfig
)

func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Op, e.Err)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Kind)
}
```

#### Step 2: Consistent Error Returns
```go
// Always return error, never bool
func (p *Parameter) Validate() error {
    if p.min >= p.max {
        return &core.Error{
            Op:   "validate",
            Kind: core.ErrorInvalid,
            Err:  fmt.Errorf("min >= max: %f >= %f", p.min, p.max),
        }
    }
    return nil
}

// Replace panics with error returns
func (b *Builder) Build() (*Configuration, error) {
    if err := b.validate(); err != nil {
        return nil, err
    }
    return b.config, nil
}

// Replace silent failures
func (r *Registry) Add(p *Parameter) error {
    if _, exists := r.params[p.ID]; exists {
        return &core.Error{
            Op:   "add parameter",  
            Kind: core.ErrorDuplicate,
            Err:  fmt.Errorf("parameter %d already exists", p.ID),
        }
    }
    r.params[p.ID] = p
    return nil
}
```

## 7. Package Function vs Method Inconsistency

### Current State

Some operations are package functions, others are methods:
```go
// Package function
gain.ApplyBuffer(buffer, gainValue)

// Method
buffer.ApplyGain(gainValue)

// Static method style
dsp.ProcessGain(buffer, gainValue)
```

### Problems
- Inconsistent API style
- Unclear when to use which approach
- Makes the API feel incoherent

### Implementation Guide

#### Step 1: Define Consistent Rules
```go
// Rules:
// 1. Operations on single values: package functions
// 2. Operations on buffers: package functions (for flexibility)
// 3. Stateful operations: methods on types
// 4. Builders/fluent APIs: methods

// Examples following rules:

// Rule 1: Single value operations
gainLinear := gain.DbToLinear(db)

// Rule 2: Buffer operations (can work on any []float32)
gain.ApplyBuffer(buffer, gainValue)

// Rule 3: Stateful operations
compressor.Process(sample)

// Rule 4: Builders
param := param.New(0, "Gain").Range(-12, 12).Build()
```

## Implementation Timeline

### Phase 1: Critical Fixes (Week 1)
1. Fix duplicate constants
2. Standardize error handling
3. Document interface hierarchy

### Phase 2: API Cleanup (Week 2)
1. Standardize parameter IDs
2. Fix method naming
3. Consolidate bus configuration

### Phase 3: Polish (Week 3)
1. Update all examples
2. Add migration guide
3. Deprecation notices

## Migration Guide for Users

### Constants
```go
// Old
if result == vst3.ResultOk { }

// New  
if result == vst3.ResultOK { }
```

### Parameters
```go
// Old
value := param.GetValue()

// New
value := param.GetNormalized()
```

### Error Handling
```go
// Old
if !param.Validate() { }

// New
if err := param.Validate(); err != nil { }
```

### Bus Configuration
```go
// Old
buses := bus.NewStereoConfiguration()

// New
buses := bus.Stereo()
```

## Success Metrics

1. **Consistency**: Single way to perform each operation
2. **Clarity**: Clear naming that follows Go conventions
3. **Discoverability**: Easy to find the right API
4. **Type Safety**: Proper use of Go's type system
5. **Error Handling**: No silent failures or unexpected panics