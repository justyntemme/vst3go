# VST3Go Real-Time Architecture

## Core Problem Statement

Go's garbage collector runs unpredictably and cannot be disabled per-goroutine. Even with zero allocations in our code, the GC can still pause our audio thread because:
- The Go runtime itself may allocate
- Other goroutines may trigger collection
- GC must scan all goroutines, including ours

**Critical Insight**: Atomic operations don't help if the thread reading them gets paused by GC.

## Architectural Solution: The RT Bridge

### Design Principles

1. **Complete RT/Non-RT Separation** - Real-time code runs in a GC-free environment
2. **Memory Safety First** - All memory is pre-allocated and bounds-checked
3. **Single Writer Principle** - Each memory region has exactly one writer
4. **Zero-Copy Communication** - Data flows through shared memory, not function calls

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          VST3 Host                               │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                     C RT-Bridge Layer                            │
│  Purpose: Routes VST3 calls to appropriate handlers             │
│  Memory: Static allocation only                                  │
│  GC Impact: None (pure C)                                        │
├──────────────────────────────────────────────────────────────────┤
│ Components:                                                       │
│ - VST3 C API interface (component.c)                            │
│ - RT Engine dispatcher                                           │
│ - Shared memory manager                                          │
└────────────┬─────────────────────────────────────┬──────────────┘
             │                                     │
             │ Real-Time Path                      │ Control Path
             │ (No Go code executed)               │ (Go code allowed)
             │                                     │
┌────────────▼──────────────┐         ┌───────────▼───────────────┐
│    RT Processing Core     │         │    Go Control Layer       │
│                           │         │                           │
│ Purpose: Audio processing │ ◄────── │ Purpose: Configuration    │
│ Language: C only          │ shared  │ Language: Go              │
│ GC Impact: None           │ memory  │ GC Impact: Isolated       │
├───────────────────────────┤         ├───────────────────────────┤
│ Components:               │         │ Components:               │
│ - Voice pool              │         │ - Parameter manager       │
│ - Event queue             │         │ - Preset system           │
│ - DSP processors          │         │ - State persistence      │
│ - Mixer                   │         │ - Plugin lifecycle       │
└───────────────────────────┘         └───────────────────────────┘
```

## Memory Architecture

### Shared Memory Regions

All shared memory is allocated once during initialization using jemalloc:

```c
typedef struct {
    // Region 1: Control Parameters (Go writes, C reads)
    struct {
        atomic_uint32_t sequence;        // Sequence number for updates
        float params[MAX_PARAMS];        // Parameter values
        uint32_t param_changed_flags;    // Bitfield of changed params
    } control;
    
    // Region 2: Audio State (C writes, Go reads for display)
    struct {
        atomic_uint32_t sequence;
        float output_levels[MAX_CHANNELS];
        float voice_levels[MAX_VOICES];
        uint32_t active_voice_count;
    } audio_state;
    
    // Region 3: Event Queue (Go writes, C reads)
    struct {
        Event events[EVENT_QUEUE_SIZE];
        atomic_uint32_t write_pos;
        atomic_uint32_t read_pos;
    } event_queue;
    
    // Region 4: Preset Data (Go writes, C reads)
    struct {
        atomic_uint32_t active_preset;
        PresetData presets[MAX_PRESETS];
    } preset_data;
    
} SharedMemory;
```

### Memory Safety Guarantees

1. **Bounds Checking**: All array accesses use compile-time constants
2. **No Pointers in Shared Memory**: Only POD types and fixed arrays
3. **Sequence Numbers**: Detect torn reads/writes
4. **Single Writer**: Each field has exactly one writer

## Communication Protocols

### 1. Parameter Updates (Go → C)

```go
// Go side - runs in non-RT thread
func (p *PluginController) SetParameter(id int, value float64) {
    if id < 0 || id >= MAX_PARAMS {
        return // Bounds check
    }
    
    // Get exclusive write access
    seq := atomic.AddUint32(&p.shared.control.sequence, 1)
    
    // Update parameter
    p.shared.control.params[id] = float32(value)
    p.shared.control.param_changed_flags |= (1 << id)
    
    // Commit with sequence
    atomic.StoreUint32(&p.shared.control.sequence, seq+1)
}
```

```c
// C side - runs in RT thread
void process_parameters(SharedMemory* shared, RTEngine* engine) {
    uint32_t seq;
    do {
        seq = atomic_load(&shared->control.sequence);
        if (seq & 1) continue; // Odd = write in progress
        
        // Copy changed parameters
        uint32_t changed = shared->control.param_changed_flags;
        for (int i = 0; i < MAX_PARAMS; i++) {
            if (changed & (1 << i)) {
                engine->params[i] = shared->control.params[i];
            }
        }
        
    } while (atomic_load(&shared->control.sequence) != seq);
    
    // Clear changed flags after processing
    shared->control.param_changed_flags = 0;
}
```

### 2. Event Processing (Go → C)

Lock-free SPSC (Single Producer Single Consumer) queue:

```go
// Go side - produces events
func (p *PluginController) SendNoteOn(note, velocity byte) {
    event := Event{
        Type:     EVENT_NOTE_ON,
        Note:     note,
        Velocity: velocity,
        Time:     p.getCurrentSampleTime(),
    }
    
    writePos := atomic.LoadUint32(&p.shared.event_queue.write_pos)
    nextPos := (writePos + 1) % EVENT_QUEUE_SIZE
    
    if nextPos != atomic.LoadUint32(&p.shared.event_queue.read_pos) {
        p.shared.event_queue.events[writePos] = event
        atomic.StoreUint32(&p.shared.event_queue.write_pos, nextPos)
    }
    // Note: Events dropped if queue full (better than blocking)
}
```

### 3. State Reporting (C → Go)

```c
// C side - updates state after each process cycle
void update_audio_state(SharedMemory* shared, RTEngine* engine) {
    uint32_t seq = atomic_fetch_add(&shared->audio_state.sequence, 1) + 1;
    
    // Update levels
    for (int i = 0; i < engine->num_channels; i++) {
        shared->audio_state.output_levels[i] = engine->output_levels[i];
    }
    
    shared->audio_state.active_voice_count = engine->active_voices;
    
    atomic_store(&shared->audio_state.sequence, seq + 1);
}
```

## RT Processing Core Design

### Voice Architecture

```c
typedef struct {
    // Fixed-size buffers for zero allocation
    float osc_buffer[MAX_BLOCK_SIZE];
    float filter_buffer[MAX_BLOCK_SIZE];
    float env_buffer[MAX_BLOCK_SIZE];
    
    // Voice state
    struct {
        uint8_t note;
        float frequency;
        float velocity;
        uint32_t sample_position;
        VoiceState state; // OFF, ATTACK, DECAY, SUSTAIN, RELEASE
    } state;
    
    // DSP components (all pre-allocated)
    Oscillator oscillators[OSCS_PER_VOICE];
    SVFilter filter;
    ADSR envelope;
    
} Voice;

typedef struct {
    Voice voice_pool[MAX_VOICES];
    uint32_t voice_alloc_index; // Round-robin allocation
    
    // Pre-allocated work buffers
    float* mix_buffer;
    float* temp_buffer;
    
    // Current parameters (copied from shared memory)
    float params[MAX_PARAMS];
    
} RTEngine;
```

### Process Flow

```c
void rt_process(RTEngine* engine, float** outputs, int num_samples, 
                SharedMemory* shared) {
    // 1. Process parameter updates (non-blocking)
    process_parameters(shared, engine);
    
    // 2. Process events sample-accurately
    process_events(shared, engine, num_samples);
    
    // 3. Clear mix buffer
    memset(engine->mix_buffer, 0, num_samples * sizeof(float));
    
    // 4. Process all active voices
    for (int v = 0; v < MAX_VOICES; v++) {
        if (engine->voice_pool[v].state.state != VOICE_OFF) {
            process_voice(&engine->voice_pool[v], engine->params, 
                         engine->mix_buffer, num_samples);
        }
    }
    
    // 5. Copy to output buffers
    for (int ch = 0; ch < 2; ch++) {
        memcpy(outputs[ch], engine->mix_buffer, num_samples * sizeof(float));
    }
    
    // 6. Update audio state for Go
    update_audio_state(shared, engine);
}
```

## Go Control Layer Design

### Plugin Structure

```go
type RealtimePlugin struct {
    // Shared memory mapped from C
    shared *SharedMemory
    
    // Go-only data (not accessed from RT thread)
    presetManager *PresetManager
    stateManager  *StateManager
    uiState       *UIState
    
    // RT engine handle (opaque)
    rtEngine unsafe.Pointer
}

// Public API - all non-RT safe
func (p *RealtimePlugin) SetParameter(id ParameterID, value float64) {
    p.validateParameter(id, value)
    p.updateSharedParameter(id, value)
    p.uiState.NotifyParameterChange(id, value)
}

func (p *RealtimePlugin) LoadPreset(preset *Preset) {
    // Update all parameters
    for id, value := range preset.Parameters {
        p.SetParameter(id, value)
    }
    
    // Update preset data in shared memory
    p.updateSharedPreset(preset)
}
```

### Lifecycle Management

```go
func NewRealtimePlugin(pluginType PluginType) (*RealtimePlugin, error) {
    // 1. Create RT engine in C (allocates all memory)
    rtEngine := C.rt_engine_create(C.int(pluginType))
    if rtEngine == nil {
        return nil, errors.New("failed to create RT engine")
    }
    
    // 2. Get shared memory region
    shared := (*SharedMemory)(C.rt_engine_get_shared_memory(rtEngine))
    
    // 3. Initialize Go components
    plugin := &RealtimePlugin{
        shared:        shared,
        rtEngine:      rtEngine,
        presetManager: NewPresetManager(),
        stateManager:  NewStateManager(),
        uiState:       NewUIState(),
    }
    
    // 4. Set up default state
    plugin.loadDefaultPreset()
    
    return plugin, nil
}
```

## Developer Experience

### Simple API for Plugin Developers

```go
// What developers write
type MySynth struct {
    *framework.RealtimeSynth // Embeds all RT functionality
    
    // Define parameters
    params struct {
        Cutoff    framework.Parameter `id:"0" range:"20,20000" default:"1000"`
        Resonance framework.Parameter `id:"1" range:"0,1" default:"0.5"`
        Attack    framework.Parameter `id:"2" range:"0,5" default:"0.01"`
    }
}

func (s *MySynth) DefineVoice() framework.VoiceDefinition {
    return framework.VoiceDefinition{
        Oscillators: []framework.OscillatorDef{
            {Type: framework.Saw, Detune: 0.0},
            {Type: framework.Square, Detune: -0.1},
        },
        Filter: framework.FilterDef{
            Type: framework.LowPass24,
        },
        Envelopes: []framework.EnvelopeDef{
            {Attack: 0.01, Decay: 0.1, Sustain: 0.7, Release: 0.5},
        },
    }
}

// The framework handles all RT complexity
```

### Code Generation

We can use Go code generation to create the C voice processing code:

```bash
//go:generate vst3go generate-rt-engine
```

This generates optimized C code based on the Go voice definition.

## Testing Strategy

### 1. RT Safety Verification

```c
// Test harness that verifies no allocations
void test_rt_safety() {
    // Override malloc to detect any allocation
    void* (*old_malloc)(size_t) = malloc;
    malloc = malloc_detector;
    
    // Run process cycles
    rt_process(engine, outputs, 512, shared);
    
    // Restore malloc
    malloc = old_malloc;
}
```

### 2. GC Interference Testing

```go
func TestGCInterference(t *testing.T) {
    plugin := createTestPlugin()
    
    // Start aggressive GC in background
    go func() {
        for {
            runtime.GC()
            time.Sleep(time.Microsecond)
        }
    }()
    
    // Monitor audio performance
    results := plugin.RunAudioTest(time.Minute)
    
    assert.Zero(t, results.Dropouts)
    assert.Less(t, results.MaxLatency, time.Millisecond)
}
```

## Migration Path

### Phase 1: Proof of Concept
1. Implement basic RT engine with fixed 1-oscillator voice
2. Verify GC isolation works
3. Measure performance

### Phase 2: Full Voice Architecture  
1. Implement complete voice system in C
2. Add modulation routing
3. Create code generation tools

### Phase 3: Developer Experience
1. Build framework abstractions
2. Create plugin templates
3. Write comprehensive docs

### Phase 4: Production Ready
1. Extensive testing across hosts
2. Performance optimization
3. Memory leak verification

## Summary

This architecture completely isolates real-time audio processing from Go's GC by:

1. **Running all audio code in C** - No Go code in audio path
2. **Using shared memory** - Not function calls for communication  
3. **Lock-free data structures** - No blocking between threads
4. **Pre-allocating everything** - No allocations after init

The complexity is hidden from plugin developers who write simple Go code while getting true real-time performance.