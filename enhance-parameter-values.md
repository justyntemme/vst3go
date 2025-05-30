# Enhancing Parameter Value Display

This guide explains how to implement parameter value formatting and different parameter types (sliders, dropdowns) to provide better UI feedback and control in VST3 plugins.

## Overview

VST3 hosts query plugins for parameter display strings to show formatted values (e.g., "440 Hz" instead of "440.0"). This guide covers:
1. Implementing value-to-string conversion
2. Implementing string-to-value parsing
3. Using step counts for dropdown-style parameters
4. Real-world examples

## Parameter Value Formatting

### Basic Implementation

First, extend the parameter system to support value formatting:

```go
// pkg/framework/param/parameter.go

type Parameter struct {
    // Existing fields...
    
    // Value formatting
    formatFunc   func(float64) string
    parseFunc    func(string) (float64, error)
    stepCount    int32  // 0 = continuous, >0 = discrete steps
}

// SetFormatter sets custom value formatting
func (p *Parameter) SetFormatter(format func(float64) string, parse func(string) (float64, error)) {
    p.formatFunc = format
    p.parseFunc = parse
}

// FormatValue returns formatted parameter value
func (p *Parameter) FormatValue(normalized float64) string {
    if p.formatFunc != nil {
        plain := p.Denormalize(normalized)
        return p.formatFunc(plain)
    }
    // Default formatting
    return fmt.Sprintf("%.2f", p.Denormalize(normalized))
}

// ParseValue parses string to normalized value
func (p *Parameter) ParseValue(str string) (float64, error) {
    if p.parseFunc != nil {
        plain, err := p.parseFunc(str)
        if err != nil {
            return 0, err
        }
        return p.Normalize(plain), nil
    }
    // Default parsing
    plain, err := strconv.ParseFloat(str, 64)
    if err != nil {
        return 0, err
    }
    return p.Normalize(plain), nil
}
```

### Builder API Enhancement

```go
// pkg/framework/param/builder.go

func (b *Builder) Steps(count int32) *Builder {
    b.param.stepCount = count
    return b
}

func (b *Builder) Formatter(format func(float64) string, parse func(string) (float64, error)) *Builder {
    b.param.formatFunc = format
    b.param.parseFunc = parse
    return b
}
```

## Common Formatters

### Frequency Formatter
```go
// pkg/framework/param/formatters.go

func FrequencyFormatter(hz float64) string {
    if hz >= 1000 {
        return fmt.Sprintf("%.2f kHz", hz/1000)
    }
    return fmt.Sprintf("%.1f Hz", hz)
}

func FrequencyParser(str string) (float64, error) {
    str = strings.TrimSpace(str)
    
    // Handle kHz
    if strings.HasSuffix(str, "kHz") || strings.HasSuffix(str, "khz") {
        numStr := strings.TrimSuffix(strings.TrimSuffix(str, "kHz"), "khz")
        numStr = strings.TrimSpace(numStr)
        val, err := strconv.ParseFloat(numStr, 64)
        if err != nil {
            return 0, err
        }
        return val * 1000, nil
    }
    
    // Handle Hz
    str = strings.TrimSuffix(strings.TrimSuffix(str, "Hz"), "hz")
    str = strings.TrimSpace(str)
    return strconv.ParseFloat(str, 64)
}
```

### Decibel Formatter
```go
func DecibelFormatter(db float64) string {
    if db <= -60 {
        return "-∞ dB"
    }
    return fmt.Sprintf("%.1f dB", db)
}

func DecibelParser(str string) (float64, error) {
    if str == "-∞ dB" || str == "-inf" {
        return -96.0, nil  // Practical minimum
    }
    str = strings.TrimSuffix(strings.TrimSpace(str), "dB")
    return strconv.ParseFloat(strings.TrimSpace(str), 64)
}
```

### Percentage Formatter
```go
func PercentFormatter(value float64) string {
    return fmt.Sprintf("%.0f%%", value)
}

func PercentParser(str string) (float64, error) {
    str = strings.TrimSuffix(strings.TrimSpace(str), "%")
    return strconv.ParseFloat(str, 64)
}
```

### Time Formatter
```go
func TimeFormatter(ms float64) string {
    if ms < 1 {
        return fmt.Sprintf("%.2f µs", ms*1000)
    } else if ms < 1000 {
        return fmt.Sprintf("%.1f ms", ms)
    }
    return fmt.Sprintf("%.2f s", ms/1000)
}
```

## Dropdown Parameters (Discrete Steps)

### Gate Type Example

For a gate plugin with different gate types:

```go
// Define gate types
const (
    GateTypeHard = iota
    GateTypeSoft
    GateTypeExpander
    GateTypeDucker
)

// Gate type names
var gateTypeNames = []string{
    "Hard Gate",
    "Soft Gate",
    "Expander",
    "Ducker",
}

// Custom formatter for gate type
func GateTypeFormatter(value float64) string {
    index := int(value)
    if index >= 0 && index < len(gateTypeNames) {
        return gateTypeNames[index]
    }
    return "Unknown"
}

func GateTypeParser(str string) (float64, error) {
    for i, name := range gateTypeNames {
        if strings.EqualFold(str, name) {
            return float64(i), nil
        }
    }
    return 0, fmt.Errorf("unknown gate type: %s", str)
}

// In your plugin setup:
params.Add(
    param.New(ParamGateType, "Gate Type").
        Range(0, 3).
        Default(0).
        Steps(4).  // This tells the host it's a discrete parameter
        Formatter(GateTypeFormatter, GateTypeParser).
        Build(),
)
```

### Filter Type Example

```go
const (
    FilterTypeLowpass = iota
    FilterTypeHighpass
    FilterTypeBandpass
    FilterTypeNotch
    FilterTypeAllpass
    FilterTypePeaking
    FilterTypeLowShelf
    FilterTypeHighShelf
)

var filterTypeNames = []string{
    "Lowpass",
    "Highpass", 
    "Bandpass",
    "Notch",
    "Allpass",
    "Peaking EQ",
    "Low Shelf",
    "High Shelf",
}

func FilterTypeFormatter(value float64) string {
    index := int(value)
    if index >= 0 && index < len(filterTypeNames) {
        return filterTypeNames[index]
    }
    return "Unknown"
}

func FilterTypeParser(str string) (float64, error) {
    // Handle common variations
    normalizedStr := strings.ToLower(strings.TrimSpace(str))
    
    // Map variations to standard names
    filterAliases := map[string]int{
        "lowpass":     FilterTypeLowpass,
        "low pass":    FilterTypeLowpass,
        "lpf":         FilterTypeLowpass,
        "highpass":    FilterTypeHighpass,
        "high pass":   FilterTypeHighpass,
        "hpf":         FilterTypeHighpass,
        "bandpass":    FilterTypeBandpass,
        "band pass":   FilterTypeBandpass,
        "bpf":         FilterTypeBandpass,
        "notch":       FilterTypeNotch,
        "band reject": FilterTypeNotch,
        "band stop":   FilterTypeNotch,
        "allpass":     FilterTypeAllpass,
        "all pass":    FilterTypeAllpass,
        "apf":         FilterTypeAllpass,
        "peaking":     FilterTypePeaking,
        "peaking eq":  FilterTypePeaking,
        "peak":        FilterTypePeaking,
        "bell":        FilterTypePeaking,
        "low shelf":   FilterTypeLowShelf,
        "lowshelf":    FilterTypeLowShelf,
        "ls":          FilterTypeLowShelf,
        "high shelf":  FilterTypeHighShelf,
        "highshelf":   FilterTypeHighShelf,
        "hs":          FilterTypeHighShelf,
    }
    
    if index, ok := filterAliases[normalizedStr]; ok {
        return float64(index), nil
    }
    
    // Try exact match
    for i, name := range filterTypeNames {
        if strings.EqualFold(str, name) {
            return float64(i), nil
        }
    }
    
    return 0, fmt.Errorf("unknown filter type: %s", str)
}

// Usage in filter plugin
params.Add(
    param.New(ParamFilterType, "Filter Type").
        Range(0, 7).
        Default(0).
        Steps(8).
        Formatter(FilterTypeFormatter, FilterTypeParser).
        Build(),
)
```

### Advanced Filter Mode Example (SVF Style)

For state variable filters that support morphing:

```go
const (
    SVFModeLowpass = iota
    SVFModeBandpass
    SVFModeHighpass
    SVFModeNotch
    SVFModePeak
    SVFModeAllpass
)

var svfModeNames = []string{
    "Lowpass",
    "Bandpass",
    "Highpass",
    "Notch",
    "Peak",
    "Allpass",
}

// For morphable filters, you might also want a continuous mode parameter
params.Add(
    param.New(ParamFilterMode, "Filter Mode").
        Range(0, 1).  // Continuous 0-1 for morphing
        Default(0).
        Formatter(func(value float64) string {
            // Show interpolated mode
            scaledValue := value * float64(len(svfModeNames)-1)
            index := int(scaledValue)
            frac := scaledValue - float64(index)
            
            if frac < 0.1 {
                return svfModeNames[index]
            } else if frac > 0.9 && index < len(svfModeNames)-1 {
                return svfModeNames[index+1]
            } else if index < len(svfModeNames)-1 {
                return fmt.Sprintf("%s → %s", svfModeNames[index], svfModeNames[index+1])
            }
            return svfModeNames[index]
        }, nil).
        Build(),
)
```

## Plugin Wrapper Integration

Update the plugin wrapper to handle value strings:

```go
// pkg/plugin/wrapper_controller.go

//export GoController_GetParamStringByValue
func GoController_GetParamStringByValue(handle C.uintptr_t, id C.Steinberg_Vst_ParamID,
    valueNormalized C.Steinberg_Vst_ParamValue, string_ *C.Steinberg_Vst_String128) C.Steinberg_tresult {
    
    wrapper := getComponent(handle)
    params := wrapper.processor.GetParameters()
    
    param := params.Get(ParamID(id))
    if param == nil {
        return C.Steinberg_kResultFalse
    }
    
    // Format the value
    str := param.FormatValue(float64(valueNormalized))
    
    // Copy to VST3 string
    copyStringToString128(str, string_)
    
    return C.Steinberg_kResultOk
}

//export GoController_GetParamValueByString
func GoController_GetParamValueByString(handle C.uintptr_t, id C.Steinberg_Vst_ParamID,
    string_ *C.Steinberg_Vst_TChar, valueNormalized *C.Steinberg_Vst_ParamValue) C.Steinberg_tresult {
    
    wrapper := getComponent(handle)
    params := wrapper.processor.GetParameters()
    
    param := params.Get(ParamID(id))
    if param == nil {
        return C.Steinberg_kResultFalse
    }
    
    // Convert from VST3 string
    str := stringFromString128(string_)
    
    // Parse the value
    value, err := param.ParseValue(str)
    if err != nil {
        return C.Steinberg_kResultFalse
    }
    
    *valueNormalized = C.Steinberg_Vst_ParamValue(value)
    return C.Steinberg_kResultOk
}
```

## Complete Example: Gain Plugin with Value Display

```go
package main

import (
    "fmt"
    "math"
    "github.com/justyntemme/vst3go/pkg/framework/param"
)

type GainPlugin struct {
    params *param.Registry
}

func NewGainPlugin() *GainPlugin {
    p := &GainPlugin{
        params: param.NewRegistry(),
    }
    
    // Add gain parameter with dB formatting
    p.params.Add(
        param.New(0, "Gain").
            Range(-60, 12).
            Default(0).
            Unit("dB").
            Formatter(DecibelFormatter, DecibelParser).
            Build(),
    )
    
    // Add output meter (read-only)
    p.params.Add(
        param.New(1, "Output Level").
            Range(-60, 0).
            Default(-60).
            Unit("dB").
            Formatter(DecibelFormatter, nil). // No parser needed for read-only
            Flags(param.IsReadOnly).
            Build(),
    )
    
    return p
}

func (p *GainPlugin) ProcessAudio(ctx *process.Context) {
    gainDB := ctx.ParamPlain(0)
    gain := float32(math.Pow(10.0, gainDB/20.0))
    
    // Process and measure
    peak := float32(0)
    for ch := 0; ch < ctx.NumChannels(); ch++ {
        for i := 0; i < ctx.NumSamples(); i++ {
            sample := ctx.Input[ch][i] * gain
            ctx.Output[ch][i] = sample
            
            if abs := float32(math.Abs(float64(sample))); abs > peak {
                peak = abs
            }
        }
    }
    
    // Update output meter
    peakDB := 20.0 * math.Log10(float64(peak))
    p.params.Get(1).SetValue(p.params.Get(1).Normalize(peakDB))
}
```

## Complete Example: Gate Plugin with Dropdown

```go
type GatePlugin struct {
    params      *param.Registry
    gateState   []bool  // Per-channel gate state
}

func NewGatePlugin() *GatePlugin {
    p := &GatePlugin{
        params: param.NewRegistry(),
    }
    
    // Gate type dropdown
    p.params.Add(
        param.New(ParamGateType, "Gate Type").
            Range(0, 3).
            Default(0).
            Steps(4).
            Formatter(GateTypeFormatter, GateTypeParser).
            Build(),
    )
    
    // Threshold with dB display
    p.params.Add(
        param.New(ParamThreshold, "Threshold").
            Range(-60, 0).
            Default(-20).
            Unit("dB").
            Formatter(DecibelFormatter, DecibelParser).
            Build(),
    )
    
    // Attack time with ms display
    p.params.Add(
        param.New(ParamAttack, "Attack").
            Range(0.1, 100).
            Default(1).
            Unit("ms").
            Formatter(TimeFormatter, TimeParser).
            Build(),
    )
    
    // Ratio for expander mode
    p.params.Add(
        param.New(ParamRatio, "Ratio").
            Range(1, 20).
            Default(10).
            Formatter(RatioFormatter, RatioParser).
            Build(),
    )
    
    return p
}

func RatioFormatter(value float64) string {
    return fmt.Sprintf("%.1f:1", value)
}

func (p *GatePlugin) ProcessAudio(ctx *process.Context) {
    gateType := int(ctx.Param(ParamGateType))
    threshold := ctx.ParamPlain(ParamThreshold)
    
    switch gateType {
    case GateTypeHard:
        p.processHardGate(ctx, threshold)
    case GateTypeSoft:
        p.processSoftGate(ctx, threshold)
    case GateTypeExpander:
        p.processExpander(ctx, threshold)
    case GateTypeDucker:
        p.processDucker(ctx, threshold)
    }
}
```

## Host UI Integration

When the host queries parameter info, it will:

1. Check `stepCount` to determine if it should show a slider or dropdown
2. Call `GetParamStringByValue` to get display strings
3. For dropdowns, enumerate all possible values (0 to stepCount-1)
4. Update display in real-time as parameters change

### Testing Your Implementation

```go
func TestParameterFormatting(t *testing.T) {
    param := param.New(0, "Frequency").
        Range(20, 20000).
        Default(1000).
        Formatter(FrequencyFormatter, FrequencyParser).
        Build()
    
    // Test formatting
    tests := []struct {
        plain    float64
        expected string
    }{
        {440, "440.0 Hz"},
        {1000, "1.00 kHz"},
        {15000, "15.00 kHz"},
    }
    
    for _, tt := range tests {
        normalized := param.Normalize(tt.plain)
        got := param.FormatValue(normalized)
        if got != tt.expected {
            t.Errorf("Format(%f) = %s, want %s", tt.plain, got, tt.expected)
        }
    }
}
```

## Best Practices

1. **Always Provide Formatters**: For any parameter with units
2. **Handle Edge Cases**: Like -∞ dB for silence
3. **Be Consistent**: Use the same format across similar parameters
4. **Test Parsers**: Ensure round-trip conversion works
5. **Respect Step Count**: Discrete parameters should have integer values
6. **Localization Ready**: Consider future localization needs

## Common Patterns

### On/Off Switch
```go
param.New(ParamBypass, "Bypass").
    Range(0, 1).
    Default(0).
    Steps(2).
    Formatter(func(v float64) string {
        if v > 0.5 {
            return "On"
        }
        return "Off"
    }, nil).
    Build()
```

### Note Names
```go
func NoteFormatter(noteNumber float64) string {
    notes := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
    note := int(noteNumber) % 12
    octave := int(noteNumber)/12 - 1
    return fmt.Sprintf("%s%d", notes[note], octave)
}
```

### Pan Position
```go
func PanFormatter(pan float64) string {
    if math.Abs(pan) < 0.01 {
        return "C"
    } else if pan < 0 {
        return fmt.Sprintf("%.0fL", -pan*100)
    }
    return fmt.Sprintf("%.0fR", pan*100)
}
```

## Complete Example: Filter Plugin with Dropdown

```go
package main

import (
    "github.com/justyntemme/vst3go/pkg/framework/param"
    "github.com/justyntemme/vst3go/pkg/dsp/filter"
)

type FilterPlugin struct {
    params       *param.Registry
    biquad       *filter.Biquad
    sampleRate   float64
    lastType     int
    lastCutoff   float64
    lastResonance float64
}

const (
    ParamFilterType = iota
    ParamCutoff
    ParamResonance
    ParamGain      // For peaking/shelf filters
    ParamMix
)

func NewFilterPlugin() *FilterPlugin {
    p := &FilterPlugin{
        params: param.NewRegistry(),
        biquad: filter.NewBiquad(2), // stereo
    }
    
    // Filter type dropdown
    p.params.Add(
        param.New(ParamFilterType, "Filter Type").
            Range(0, 7).
            Default(0).
            Steps(8).
            Formatter(FilterTypeFormatter, FilterTypeParser).
            Build(),
    )
    
    // Cutoff with frequency display
    p.params.Add(
        param.New(ParamCutoff, "Cutoff").
            Range(20, 20000).
            Default(1000).
            Unit("Hz").
            Formatter(FrequencyFormatter, FrequencyParser).
            Build(),
    )
    
    // Resonance/Q
    p.params.Add(
        param.New(ParamResonance, "Resonance").
            Range(0.1, 20).
            Default(0.707).
            Formatter(func(q float64) string {
                return fmt.Sprintf("Q: %.2f", q)
            }, nil).
            Build(),
    )
    
    // Gain for EQ filters
    p.params.Add(
        param.New(ParamGain, "Gain").
            Range(-24, 24).
            Default(0).
            Unit("dB").
            Formatter(DecibelFormatter, DecibelParser).
            Build(),
    )
    
    // Mix control
    p.params.Add(
        param.New(ParamMix, "Mix").
            Range(0, 100).
            Default(100).
            Unit("%").
            Formatter(PercentFormatter, PercentParser).
            Build(),
    )
    
    return p
}

func (p *FilterPlugin) ProcessAudio(ctx *process.Context) {
    filterType := int(ctx.Param(ParamFilterType))
    cutoff := ctx.ParamPlain(ParamCutoff)
    resonance := ctx.ParamPlain(ParamResonance)
    gainDB := ctx.ParamPlain(ParamGain)
    mix := float32(ctx.Param(ParamMix))
    
    // Update filter if parameters changed
    if filterType != p.lastType || 
       math.Abs(cutoff - p.lastCutoff) > 0.01 ||
       math.Abs(resonance - p.lastResonance) > 0.01 {
        
        switch filterType {
        case FilterTypeLowpass:
            p.biquad.SetLowpass(p.sampleRate, cutoff, resonance)
        case FilterTypeHighpass:
            p.biquad.SetHighpass(p.sampleRate, cutoff, resonance)
        case FilterTypeBandpass:
            p.biquad.SetBandpass(p.sampleRate, cutoff, resonance)
        case FilterTypeNotch:
            p.biquad.SetNotch(p.sampleRate, cutoff, resonance)
        case FilterTypeAllpass:
            p.biquad.SetAllpass(p.sampleRate, cutoff, resonance)
        case FilterTypePeaking:
            p.biquad.SetPeakingEQ(p.sampleRate, cutoff, resonance, gainDB)
        case FilterTypeLowShelf:
            p.biquad.SetLowShelf(p.sampleRate, cutoff, resonance, gainDB)
        case FilterTypeHighShelf:
            p.biquad.SetHighShelf(p.sampleRate, cutoff, resonance, gainDB)
        }
        
        p.lastType = filterType
        p.lastCutoff = cutoff
        p.lastResonance = resonance
    }
    
    // Process with mix control
    if mix >= 0.999 {
        // 100% wet - process in place
        for ch := 0; ch < ctx.NumChannels(); ch++ {
            p.biquad.Process(ctx.Output[ch][:ctx.NumSamples()], ch)
        }
    } else if mix > 0.001 {
        // Mix dry and wet
        dryGain := 1.0 - mix
        work := ctx.WorkBuffer()
        
        for ch := 0; ch < ctx.NumChannels(); ch++ {
            // Copy to work buffer
            copy(work[:ctx.NumSamples()], ctx.Input[ch][:ctx.NumSamples()])
            
            // Process
            p.biquad.Process(work[:ctx.NumSamples()], ch)
            
            // Mix
            for i := 0; i < ctx.NumSamples(); i++ {
                ctx.Output[ch][i] = ctx.Input[ch][i]*dryGain + work[i]*mix
            }
        }
    }
    // else mix == 0, pure bypass (input already copied to output)
}

// Helper to determine if gain parameter should be visible
func (p *FilterPlugin) IsGainParameterRelevant() bool {
    filterType := int(p.params.Get(ParamFilterType).GetValue())
    return filterType >= FilterTypePeaking // Peaking, Low Shelf, High Shelf
}
```

## UI Behavior with Filter Types

When implementing the filter plugin, the host UI should:

1. **Show/Hide Parameters Based on Filter Type**:
   - Gain parameter only visible for Peaking EQ, Low Shelf, and High Shelf
   - Some hosts support parameter visibility flags

2. **Update Parameter Ranges**:
   - Bandpass might want different Q ranges than lowpass
   - Shelving filters might limit resonance range

3. **Provide Visual Feedback**:
   - Filter response curve display
   - Different curve shapes for different filter types

4. **Smooth Transitions**:
   - When switching filter types, smoothly interpolate coefficients
   - Avoid clicks and pops during type changes

This implementation provides professional parameter value display and control types, making your plugins more user-friendly and precise. The filter type dropdown gives users immediate understanding of the filter behavior, while formatted values for frequency, resonance, and gain provide exact control over the sound.