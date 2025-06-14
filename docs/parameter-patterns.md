# Parameter Patterns Best Practices

## Overview

This document defines the standard patterns for parameter handling in VST3Go plugins to ensure consistency and type safety across the framework.

## Parameter ID Declaration

### ✅ Correct Pattern

Always use typed constants with `uint32` for parameter IDs:

```go
const (
    // Parameter IDs
    ParamGain uint32 = iota
    ParamThreshold
    ParamRatio
    ParamMix
)
```

### ❌ Incorrect Patterns

Avoid these patterns:

```go
// BAD: Untyped constants
const (
    paramGain = iota  // This is int, not uint32!
    paramThreshold
)

// BAD: Direct values
p.params.Get(0).SetValue(0.5)  // Magic number!

// BAD: Mixed naming conventions
const (
    Param_Gain uint32 = iota  // Inconsistent naming
    PARAM_THRESHOLD           // All caps
    param_ratio               // Snake case
)
```

## Naming Conventions

1. **Parameter ID Constants**: Use `ParamXxx` format with PascalCase
   - ✅ `ParamGain`
   - ✅ `ParamThreshold`
   - ✅ `ParamAttackTime`
   - ❌ `paramGain` (lowercase)
   - ❌ `PARAM_GAIN` (all caps)
   - ❌ `Param_Gain` (snake case)

2. **Parameter Ranges**: Use descriptive constant names
   ```go
   const (
       // Gain range in dB
       minGainDB = -60.0
       maxGainDB = 24.0
       defaultGainDB = 0.0
   )
   ```

## Parameter Registration

Use the parameter builder pattern with clear, descriptive names:

```go
func initializeParameters(registry *param.Registry) {
    // Simple parameter
    registry.Add(
        param.New(ParamGain, "Gain").
            Range(minGainDB, maxGainDB).
            Default(defaultGainDB).
            Unit("dB").
            Formatter(param.DecibelFormatter, param.DecibelParser).
            Build(),
    )
    
    // Bypass parameter (special case)
    registry.Add(
        param.BypassParameter(ParamBypass, "Bypass").Build(),
    )
    
    // Choice parameter
    registry.Add(
        param.Choice(ParamFilterType, "Filter Type", []param.ChoiceOption{
            {Value: 0, Name: "Low Pass"},
            {Value: 1, Name: "High Pass"},
            {Value: 2, Name: "Band Pass"},
        }).Build(),
    )
}
```

## Parameter Access

Always check for nil when accessing parameters:

```go
// ✅ Safe parameter access
if param := p.params.Get(ParamGain); param != nil {
    param.SetValue(0.5)
}

// ❌ Unsafe - can panic
p.params.Get(ParamGain).SetValue(0.5)
```

## Parameter Changes in ProcessAudio

Handle parameter changes efficiently:

```go
func (p *Processor) ProcessAudio(ctx *process.Context) {
    // Get normalized value (0-1)
    gain := ctx.Param(ParamGain)
    
    // Get plain value (in parameter's actual range)
    threshold := ctx.ParamPlain(ParamThreshold)
    
    // Handle parameter change events
    for _, change := range ctx.GetParameterChanges() {
        switch change.ParamID {
        case ParamGain:
            // React to gain change
            p.updateGain(change.Value)
        case ParamThreshold:
            // React to threshold change
            p.updateThreshold(change.Value)
        }
    }
}
```

## Common Parameter Types

Use framework-provided parameter builders for common types:

1. **Gain/Volume**: Use decibel scale
   ```go
   param.New(ParamGain, "Gain").
       Range(-60, 12).
       Default(0).
       Unit("dB").
       Formatter(param.DecibelFormatter, param.DecibelParser)
   ```

2. **Frequency**: Use logarithmic scale
   ```go
   param.New(ParamFrequency, "Frequency").
       Range(20, 20000).
       Default(1000).
       Unit("Hz").
       Formatter(param.FrequencyFormatter, param.FrequencyParser)
   ```

3. **Time**: Use appropriate units
   ```go
   param.New(ParamAttack, "Attack").
       Range(0.1, 1000).
       Default(10).
       Unit("ms").
       Formatter(param.TimeFormatter, param.TimeParser)
   ```

4. **Percentage/Mix**: Use 0-100 range
   ```go
   param.New(ParamMix, "Mix").
       Range(0, 100).
       Default(50).
       Unit("%").
       Formatter(param.PercentFormatter, param.PercentParser)
   ```

## Type Safety

The `uint32` type for parameter IDs ensures:
- Consistent type across the framework
- Prevents accidental type mismatches
- Works correctly with VST3 parameter system
- Clear compile-time errors for type issues

## Migration Guide

To update existing plugins:

1. Add `uint32` type to the first parameter constant:
   ```go
   // Before
   const (
       paramGain = iota
       paramThreshold
   )
   
   // After
   const (
       ParamGain uint32 = iota
       ParamThreshold
   )
   ```

2. Update parameter names to PascalCase:
   ```go
   // Before: paramGain, param_gain, PARAM_GAIN
   // After: ParamGain
   ```

3. Update all references throughout the code

## Benefits

Following these patterns provides:
- **Type Safety**: Compile-time checking of parameter IDs
- **Consistency**: Same patterns across all plugins
- **Readability**: Clear, descriptive parameter names
- **Maintainability**: Easy to understand and modify
- **Framework Integration**: Works seamlessly with VST3Go utilities