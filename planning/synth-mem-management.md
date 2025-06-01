# this is the approach we will use
Dynamic Latency Reporting (Best for modern DAWs)

Report GC duration as latency changes
Host compensates automatically

---

compile with GOGC=off

then when we detect an optimal state for the application to call the GC to ensure no latency we use
```runtime.GC()
```
runtime.GC()

Then we use the following approaches 

# Alternative approach to the Problem - May be much cleaner and not as complicated as the other solutins provided
1.  Disable at runtime, and call manually - This looks to be the cleanest option which still uses the GC but only when we allow it to be used
```
At runtime you can change the GOGC ratio by calling debug.SetGCPercent(), pass a negative value to disable it:
debug.SetGCPercent(-1)
You may trigger a garbage collection "manually" with runtime.GC().
```

2. Look into go areanas using go 1.2
3. look at tinygo garbage collection 


## Modular GC Architecture - Selective GC Control via Shared Objects
  ### Breakthrough Concept
     Instead of fighting Go's "all-or-nothing" GC, we compile different parts of our plugin as separate shared objects (.so
     files) with different GC settings. The C bridge orchestrates between them.



# Synth Memory Management Strategy

## The Problem

Real-time audio synthesis in Go faces a fundamental challenge: Go's garbage collector can introduce unpredictable pauses that cause audio dropouts. While our effect plugins can avoid allocations during processing, synthesizers have dynamic requirements:

- Voice allocation/deallocation on note events
- Dynamic modulation routing
- Event processing with sample-accurate timing
- Voice stealing and management

Even a 1-2ms GC pause during audio processing causes audible glitches.

## Proposed Architecture: Hybrid Memory Management

### Core Design Principles

1. **Critical Path in C** - Voice allocation, event processing, and voice mixing happen in C
2. **Business Logic in Go** - Parameter management, preset handling, UI logic stay in Go
3. **Zero-Copy Communication** - Share memory between C and Go without copying
4. **Developer Ergonomics** - Hide complexity behind clean Go APIs

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        VST3 Host                             │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                    C Bridge Layer                            │
│  - Receives process() calls                                  │
│  - Routes to appropriate handler                             │
└─────────────────────────┬───────────────────────────────────┘
                          │
        ┌─────────────────┴─────────────────┐
        │                                   │
┌───────▼────────┐              ┌──────────▼────────────┐
│ C Voice Engine │              │   Go Controller       │
│                │              │                       │
│ - Voice pool   │◄─────────────┤ - Parameter mapping   │
│ - Event queue  │              │ - Preset management   │
│ - Mixer        │              │ - High-level logic    │
│ - RT safe only │              │ - Non-RT operations   │
└────────────────┘              └───────────────────────┘
```

### Implementation Strategy

#### 1. C-Based Voice Engine (`voice_engine.c`)

```c
// Pre-allocated voice pool
typedef struct {
    float* buffer;          // Audio output buffer
    int note;              // MIDI note number
    float frequency;       // Current frequency
    float phase;           // Oscillator phase
    float envelope;        // Envelope value
    int state;            // OFF, ATTACK, DECAY, SUSTAIN, RELEASE
    // ... other voice state
} Voice;

typedef struct {
    Voice voices[MAX_VOICES];
    int num_active;
    float* mix_buffer;
    
    // Lock-free event queue
    Event event_queue[EVENT_QUEUE_SIZE];
    atomic_int event_write_pos;
    atomic_int event_read_pos;
} VoiceEngine;

// Called from VST3 process()
void voice_engine_process(VoiceEngine* engine, float** outputs, int num_samples) {
    // Process events sample-accurately
    // Mix active voices
    // No allocations, no Go calls
}
```

#### 2. Go Voice Controller (`voice_controller.go`)

```go
package synth

// #cgo CFLAGS: -I../voice_engine
// #include "voice_engine.h"
import "C"

type VoiceController struct {
    engine *C.VoiceEngine
    params atomic.Value // Parameter snapshot
}

// Called from non-RT thread
func (vc *VoiceController) SetParameter(id int, value float64) {
    // Update parameter atomically
    // C engine reads these atomically during process
}

// Initialize the C engine
func NewVoiceController() *VoiceController {
    engine := C.voice_engine_create()
    // Set up shared memory regions
    return &VoiceController{engine: engine}
}
```

#### 3. Shared Memory for Parameters

Instead of Go calling into C during audio processing, we share memory regions:

```c
// C side - reads parameters atomically
typedef struct {
    atomic_float cutoff;
    atomic_float resonance;
    atomic_float envelope_attack;
    // ... etc
} SharedParams;
```

```go
// Go side - writes parameters atomically
type SharedParams struct {
    cutoff    atomic.Float64
    resonance atomic.Float64
    attack    atomic.Float64
}
```

#### 4. Event Queue Architecture

MIDI events go through a lock-free queue:

```go
// Go side - produces events
func (vc *VoiceController) HandleMIDI(event MidiEvent) {
    // Convert to C event format
    cEvent := C.Event{
        type: C.int(event.Type),
        note: C.int(event.Note),
        velocity: C.float(event.Velocity),
        sample_offset: C.int(event.SampleOffset),
    }
    
    // Add to lock-free queue (non-blocking)
    C.voice_engine_queue_event(vc.engine, &cEvent)
}
```

### Memory Allocation Strategy

#### Pre-allocated Pools

All memory is allocated upfront during initialization:

```c
VoiceEngine* voice_engine_create() {
    // Allocate everything once
    VoiceEngine* engine = je_calloc(1, sizeof(VoiceEngine));
    
    // Voice buffers
    for (int i = 0; i < MAX_VOICES; i++) {
        engine->voices[i].buffer = je_calloc(MAX_BLOCK_SIZE, sizeof(float));
    }
    
    // Mix buffer
    engine->mix_buffer = je_calloc(MAX_BLOCK_SIZE, sizeof(float));
    
    return engine;
}
```

#### Zero-Copy Audio Buffers

Audio buffers are allocated in C and wrapped in Go:

```go
func wrapAudioBuffer(ptr *C.float, size int) []float32 {
    return (*[1 << 30]float32)(unsafe.Pointer(ptr))[:size:size]
}
```

### Developer-Friendly API

Hide the complexity behind a clean interface:

```go
// What developers see
type Synth interface {
    ProcessAudio(ctx *process.Context)
    SetParameter(id int, value float64)
    LoadPreset(preset Preset)
}

// Hidden implementation
type synthImpl struct {
    voiceEngine  *C.VoiceEngine
    controller   *VoiceController
    paramBuffer  *SharedParams
}

func (s *synthImpl) ProcessAudio(ctx *process.Context) {
    // Just calls C function - no Go logic here
    C.voice_engine_process(
        s.voiceEngine,
        (**C.float)(unsafe.Pointer(&ctx.Output[0])),
        C.int(ctx.NumSamples),
    )
}
```

### GC Mitigation Strategies

#### Option 1: Separate Process (Extreme but Effective)

```
VST3 Plugin Process (C + minimal Go shim, GOGC=off)
    ↕ (shared memory)
Go Control Process (full Go runtime, GC enabled)
```

#### Option 2: Locked OS Thread

```go
func init() {
    // Audio thread never yields to GC
    go func() {
        runtime.LockOSThread()
        // Set up audio processing
        for {
            processAudio()
        }
    }()
}
```

#### Option 3: Manual GC Control

```go
func (s *synthImpl) SetActive(active bool) error {
    if !active {
        // Force GC when audio is stopped
        runtime.GC()
        runtime.Gosched()
    }
    return nil
}
```

### Testing Strategy

1. **Stress Testing**
   - Run 16 voices with complex modulation
   - Monitor for dropouts over hours
   - Measure worst-case latency

2. **GC Pressure Testing**
   - Run background goroutines doing allocations
   - Verify audio thread isn't affected

3. **Memory Leak Detection**
   - Use valgrind on C components
   - Monitor memory usage over time

``

### Tradeoffs

**Pros:**
- Truly real-time safe audio processing
- Predictable performance
- Still get Go's benefits for non-RT code
- Can leverage existing C DSP libraries

**Cons:**
- More complex architecture
- Requires C knowledge
- Harder to debug
- Less "pure Go"

### Summary
This hybrid approach gives us:
- Real-time safe audio processing
- Go's productivity for non-critical paths  
- A path forward for professional-quality synthesizers

But it's a significant architectural change. The question is: Is the complexity worth it for your goals? Only you can answer that.
