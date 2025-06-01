# VST3 Latency Communication Specification

## How VST3 Handles Latency

### The Core Interface: IAudioProcessor

In VST3, latency is communicated through the `IAudioProcessor` interface via a simple method:

```cpp
// From VST3 SDK
class IAudioProcessor : public FUnknown
{
public:
    virtual uint32 getLatencySamples() = 0;
};
```

### When and How It's Called

1. **Initialization Phase**: The host calls `getLatencySamples()` after the plugin is activated
2. **During Operation**: The host may query this periodically, but typically caches the value
3. **After Parameter Changes**: Some hosts re-query if they suspect latency might have changed

### The VST3 C API Definition

Based on the VST3 C API pattern, it would look like this:

```c
// In vst3_c_api.h (the pattern used by VST3)
struct Vst3IAudioProcessor {
    struct Vst3FUnknown base;
    
    // ... other methods ...
    
    Steinberg_uint32 (PLUGIN_API *getLatencySamples)(void* thisInterface);
};
```

### Implementation in vst3go

Here's how it would be implemented in vst3go's architecture:

#### 1. Go Interface (pkg/plugin/plugin.go)
```go
type Processor interface {
    Initialize(sampleRate float64, maxSamplesPerBlock int32) error
    Process(data *process.Data) error
    GetLatencySamples() int32  // <-- This method
    Terminate() error
}
```

#### 2. C Bridge Implementation (bridge/component.c)
```c
// C function that VST3 host calls
static Steinberg_uint32 PLUGIN_API getLatencySamples(void* thisInterface) {
    struct ComponentContext* context = (struct ComponentContext*)thisInterface;
    
    // Call into Go
    return GoAudioGetLatencySamples(context->goProcessor);
}

// In the vtable setup
static struct Vst3IAudioProcessorVtbl audioProcessorVtbl = {
    // ... other methods ...
    getLatencySamples  // <-- Register our function
};
```

#### 3. Go Export (pkg/plugin/wrapper_audio.go)
```go
//export GoAudioGetLatencySamples
func GoAudioGetLatencySamples(componentPtr unsafe.Pointer) C.uint32_t {
    wrapper := getWrapper(componentPtr)
    if wrapper == nil || wrapper.processor == nil {
        return 0
    }
    
    latency := wrapper.processor.GetLatencySamples()
    return C.uint32_t(latency)
}
```

#### 4. Plugin Implementation
```go
type MyPlugin struct {
    latencySamples int32
}

func (p *MyPlugin) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    // Calculate latency based on sample rate
    latencyMs := 50.0  // 50ms
    p.latencySamples = int32(latencyMs * sampleRate / 1000.0)
    return nil
}

func (p *MyPlugin) GetLatencySamples() int32 {
    return p.latencySamples  // Return fixed latency
}
```

### How the Host Uses This Information

When a plugin reports latency via `getLatencySamples()`, the host:

1. **Delays Other Tracks**: Adds compensating delay to all other tracks
2. **Adjusts Playback**: Shifts audio streams to maintain synchronization  
3. **Updates UI**: Shows latency in plugin info, adjusts waveform display
4. **Handles Recording**: Compensates input monitoring if plugin is used during recording

### Visual Representation

```
Plugin reports: 2205 samples (50ms @ 44.1kHz)
                    ↓
Host's Plugin Delay Compensation (PDC):

Track 1 (No plugins):     [Delay: 2205 samples][Audio═══════>]
Track 2 (Our plugin):     [Plugin Latency     ][Audio═══════>]
Track 3 (No plugins):     [Delay: 2205 samples][Audio═══════>]
Master Output:            [All tracks aligned ][Audio═══════>]
```

### Important VST3 Rules

1. **Report Consistently**: Always return the same value unless latency actually changes
2. **Report in Samples**: Not milliseconds - the host handles conversion
3. **Report Early**: Must be accurate after `Initialize()` is called
4. **Include All Delays**: Lookahead, FFT windows, buffer delays, etc.

### Common Latency Values

- **Zero Latency**: Simple gain, pan, basic filters
- **< 64 samples**: Very low latency processors
- **64-512 samples**: Typical for dynamics with lookahead
- **512-2048 samples**: Linear phase EQs, convolution
- **2048+ samples**: Complex processors, pitch correction

### Implementation Checklist for vst3go

To properly support latency reporting:

- [ ] Add `GetLatencySamples()` to Processor interface
- [ ] Implement C bridge function in component.c
- [ ] Add Go export in wrapper_audio.go
- [ ] Update plugin examples to implement the method
- [ ] Test with various hosts to verify PDC works

### Testing Latency Compensation

To verify it's working:

1. Create two tracks with identical audio
2. Put your plugin on one track
3. Both tracks should stay perfectly in sync
4. Check the host's PDC display shows your reported latency

This is the standard VST3 way - simple, effective, and universally supported by all VST3 hosts.