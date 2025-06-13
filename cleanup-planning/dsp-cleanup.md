# DSP Code Cleanup Guide

## Overview

This document identifies all instances where DSP-related calculations are implemented directly in plugin code instead of using the centralized DSP packages. The goal is to eliminate code duplication and ensure consistent, optimized DSP operations across all plugins.

## Available DSP Utilities

### gain package (`pkg/dsp/gain/gain.go`)
- `LinearToDb(float64) float64` - Convert linear to dB
- `DbToLinear(float64) float64` - Convert dB to linear
- `LinearToDb32(float32) float32` - Float32 version
- `DbToLinear32(float32) float32` - Float32 version
- `Apply(sample, gain float32) float32` - Apply gain to sample
- `ApplyDb(sample, db float32) float32` - Apply dB gain to sample
- `ApplyBuffer(buffer []float32, gain float32)` - Apply gain to buffer
- `ApplyDbBuffer(buffer []float32, db float32)` - Apply dB gain to buffer
- `Fade(buffer []float32, startGain, endGain float32)` - Linear fade
- `SoftClip(input, threshold float32) float32` - Soft clipping
- `HardClip(input, threshold float32) float32` - Hard clipping

### buffer utilities (`pkg/dsp/buffer.go`)
- `Clear(buffer []float32)` - Zero a buffer
- `Copy(dst, src []float32)` - Copy buffer
- `Add(dst, src []float32)` - Add buffers
- `AddScaled(dst, src []float32, scale float32)` - Add scaled buffer
- `Scale(buffer []float32, scale float32)` - Scale buffer
- `Mix(dst, src1, src2 []float32, mix float32)` - Mix two buffers

### mix package (`pkg/dsp/mix/mix.go`)
- `DryWet(dry, wet, amount float32) float32` - Dry/wet mixing
- `DryWetBuffer(dry, wet []float32, amount float32)` - Buffer dry/wet
- `CrossfadeCosine(a, b, position float32) float32` - Equal-power crossfade
- `CrossfadeLinear(a, b, position float32) float32` - Linear crossfade

## Issues Found by Plugin

### 1. smoothed_gain/main.go

**Line 114: Manual dB to linear conversion**
```go
// CURRENT (BAD):
gainLinear := float32(math.Pow(10, gainDB/20))

// SHOULD BE:
gainLinear := gain.DbToLinear32(gainDB)
```

**Line 117: Manual gain application**
```go
// CURRENT (BAD):
output[i] = input[i] * gainLinear

// SHOULD BE:
output[i] = gain.Apply(input[i], gainLinear)
// OR for the whole buffer:
gain.ApplyBuffer(output, gainLinear)
```

### 2. drumbus/main.go

**Line 512: Manual dB to linear conversion**
```go
// CURRENT (BAD):
p.outputGain = math.Pow(10.0, outputGainDB/20.0)

// SHOULD BE:
p.outputGain = gain.DbToLinear(outputGainDB)
```

**Lines 433-437: Manual parallel mix**
```go
// CURRENT (BAD):
mix := float32(p.parallelMix)
for i := 0; i < numSamples; i++ {
    ctx.Output[0][i] += p.parallelBufferL[i] * mix
    ctx.Output[1][i] += p.parallelBufferR[i] * mix
}

// SHOULD BE:
dsp.AddScaled(ctx.Output[0][:numSamples], p.parallelBufferL[:numSamples], float32(p.parallelMix))
dsp.AddScaled(ctx.Output[1][:numSamples], p.parallelBufferR[:numSamples], float32(p.parallelMix))
```

**Lines 441-444: Manual gain with sustain**
```go
// CURRENT (BAD):
gain := float32(p.outputGain * (1.0 + p.transientSustain*0.5))
for i := 0; i < numSamples; i++ {
    ctx.Output[0][i] *= gain
    ctx.Output[1][i] *= gain
}

// SHOULD BE:
gainValue := float32(p.outputGain * (1.0 + p.transientSustain*0.5))
gain.ApplyBuffer(ctx.Output[0][:numSamples], gainValue)
gain.ApplyBuffer(ctx.Output[1][:numSamples], gainValue)
```

### 3. vocalstrip/main.go

**Lines 652-654: Duplicate dB conversion function**
```go
// CURRENT (BAD):
func dBToLinear(dB float64) float64 {
    return math.Pow(10.0, dB/20.0)
}

// SHOULD BE REMOVED - use gain.DbToLinear() instead
```

**Lines 486-491: Manual output gain**
```go
// CURRENT (BAD):
if p.outputGain != 1.0 {
    gain := float32(p.outputGain)
    for i := 0; i < numSamples; i++ {
        ctx.Output[0][i] *= gain
        ctx.Output[1][i] *= gain
    }
}

// SHOULD BE:
if p.outputGain != 1.0 {
    gainValue := float32(p.outputGain)
    gain.ApplyBuffer(ctx.Output[0][:numSamples], gainValue)
    gain.ApplyBuffer(ctx.Output[1][:numSamples], gainValue)
}
```

### 4. chain_fx/main.go

**Lines 156-157: Manual dB range conversion**
```go
// CURRENT (BAD):
dbValue := -60.0 + change.Value*60.0 // -60 to 0 dB

// SHOULD BE:
// Define constants
const (
    MinThresholdDB = -60.0
    MaxThresholdDB = 0.0
)
dbValue := MinThresholdDB + change.Value*(MaxThresholdDB-MinThresholdDB)
```

**Lines 164-165: Manual ratio conversion**
```go
// CURRENT (BAD):
ratio := 1.0 + change.Value*19.0 // 1:1 to 20:1

// SHOULD BE:
const (
    MinRatio = 1.0
    MaxRatio = 20.0
)
ratio := MinRatio + change.Value*(MaxRatio-MinRatio)
```

### 5. debug_example/main.go

**Lines 168-170: Manual gain reduction**
```go
// CURRENT (BAD):
for i := 0; i < ctx.NumSamples(); i++ {
    output[i] = input[i] * 0.7
}

// SHOULD BE:
gain.ApplyBuffer(output[:ctx.NumSamples()], 0.7)
// OR
dsp.Scale(output[:ctx.NumSamples()], 0.7)
```

## Additional Improvements

### 1. Parameter Range Scaling

Create a utility function for parameter scaling:
```go
// Add to dsp/utility package
func ScaleParameter(normalized float64, min, max float64) float64 {
    return min + normalized*(max-min)
}

func ScaleParameterExp(normalized float64, min, max float64) float64 {
    // Exponential scaling for frequency, time parameters
    return min * math.Pow(max/min, normalized)
}
```

### 2. Common Constants

Create a constants file for common audio values:
```go
// dsp/constants.go
package dsp

const (
    // Gain/Level constants
    MinDB = -200.0
    UnityGain = 1.0
    
    // Common parameter ranges
    DefaultMinThresholdDB = -60.0
    DefaultMaxThresholdDB = 0.0
    DefaultMinRatio = 1.0
    DefaultMaxRatio = 20.0
    
    // Channel counts
    Mono = 1
    Stereo = 2
)
```

### 3. Buffer Allocation Tracking

Add debug helpers to detect allocations:
```go
// Add to dsp/debug package (build tag: debug)
func CheckAllocation(buffer []float32, name string) {
    if cap(buffer) == 0 {
        panic(fmt.Sprintf("Buffer %s is not pre-allocated", name))
    }
}
```

## Implementation Priority

### Phase 1: Critical Fixes (Immediate)
1. Replace all manual dB conversions with `gain.DbToLinear*` functions
2. Replace manual gain loops with `gain.ApplyBuffer`
3. Remove duplicate dBToLinear function from vocalstrip

### Phase 2: Consistency (Short term)
1. Replace manual mixing with `dsp.Mix` or `dsp.AddScaled`
2. Use buffer utilities for all buffer operations
3. Define constants for magic numbers

### Phase 3: Optimization (Medium term)
1. Audit for SIMD opportunities in hot paths
2. Ensure proper memory alignment for vectorization
3. Profile and optimize critical DSP chains

## Testing Strategy

After cleanup:
1. **Functional tests**: Ensure audio output remains identical
2. **Performance tests**: Verify no performance regression
3. **Allocation tests**: Confirm zero allocations in audio path

## Migration Checklist

For each plugin:
- [ ] Replace manual dB conversions
- [ ] Replace manual gain applications
- [ ] Replace manual mixing operations
- [ ] Define constants for magic numbers
- [ ] Remove duplicate DSP functions
- [ ] Add appropriate imports for DSP packages
- [ ] Test audio output remains identical
- [ ] Verify no allocations in ProcessAudio

## Benefits

1. **Consistency**: All plugins use the same DSP implementations
2. **Optimization**: DSP packages can be optimized once for all plugins
3. **Maintainability**: Single source of truth for DSP algorithms
4. **Correctness**: Tested DSP functions reduce bugs
5. **Performance**: Potential for SIMD optimizations in DSP packages

## Conclusion

By eliminating these DSP code duplications and using the centralized DSP packages, the VST3Go framework will be more maintainable, consistent, and performant. The cleanup is straightforward - mostly replacing manual implementations with function calls to existing DSP utilities.