# Shared Buffer Architecture for RT-Safe Go DSP

## Core Concept

Go developers write normal DSP code that outputs to shared buffers. The C engine handles all timing-critical operations (mixing, event processing, output) while Go code runs ahead of time in larger chunks.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Go DSP Code                            │
│  - Runs in separate thread/goroutine                     │
│  - Processes in larger chunks (e.g., 2048 samples)       │
│  - Can allocate, use GC, etc.                           │
│  - Writes to ring buffers                               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼ Ring Buffers
┌─────────────────────────────────────────────────────────┐
│  Voice 0: [████████████████░░░░░░░░] (75% full)        │
│  Voice 1: [████████████░░░░░░░░░░░░] (50% full)        │
│  Voice 2: [██████████████████░░░░░░] (80% full)        │
│  ...                                                     │
│  Voice N: [████████░░░░░░░░░░░░░░░░] (30% full)        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼ C reads
┌─────────────────────────────────────────────────────────┐
│                   C RT Engine                            │
│  - Reads from ring buffers                              │
│  - Handles all timing (events, mixing)                  │
│  - Never blocks, never allocates                        │
│  - Outputs to host                                      │
└─────────────────────────────────────────────────────────┘
```

## Implementation

### Shared Memory Structure

```c
// C header - shared with Go
typedef struct {
    float* buffer;           // The audio data
    uint32_t size;          // Buffer size (power of 2)
    atomic_uint32_t write;  // Write position
    atomic_uint32_t read;   // Read position
} RingBuffer;

typedef struct {
    // Per-voice ring buffers
    RingBuffer voice_buffers[MAX_VOICES];
    
    // Event communication
    struct {
        Event events[EVENT_QUEUE_SIZE];
        atomic_uint32_t write;
        atomic_uint32_t read;
    } event_queue;
    
    // Voice allocation status
    atomic_uint32_t voice_active[MAX_VOICES];
    
    // Parameter updates
    atomic_float params[MAX_PARAMS];
    
} SharedState;
```

### Go Side - Voice Processing

```go
// This is what developers write - normal Go code!
type MySynthVoice struct {
    osc1    *Oscillator
    osc2    *Oscillator  
    filter  *MoogFilter
    env     *ADSR
    
    // Private - managed by framework
    buffer  *RingBuffer
    active  *atomic.Bool
}

// Developers implement this - can allocate, use maps, whatever!
func (v *MySynthVoice) Process(samples int) []float32 {
    // Allocate output buffer (this is fine! We're not in RT thread)
    output := make([]float32, samples)
    
    // Generate audio using normal Go code
    for i := 0; i < samples; i++ {
        // Complex processing with no RT constraints
        osc1Out := v.osc1.Process()
        osc2Out := v.osc2.Process()
        mixed := osc1Out + osc2Out
        filtered := v.filter.Process(mixed)
        output[i] = filtered * v.env.Process()
    }
    
    return output
}

// Framework handles the ring buffer writing
func (v *MySynthVoice) run() {
    for v.active.Load() {
        // Process a chunk
        audioData := v.Process(CHUNK_SIZE)
        
        // Write to ring buffer (non-blocking)
        if !v.buffer.Write(audioData) {
            // Buffer full, skip this chunk
            // Could log warning in debug mode
        }
        
        // Sleep based on buffer fill level
        fillLevel := v.buffer.FillLevel()
        if fillLevel > 0.75 {
            time.Sleep(5 * time.Millisecond)
        }
    }
}
```

### C Side - Real-Time Mixing

```c
// This runs in the RT thread with zero allocations
void process_audio(SharedState* shared, float** outputs, int num_samples) {
    // Clear output buffer
    memset(outputs[0], 0, num_samples * sizeof(float));
    memset(outputs[1], 0, num_samples * sizeof(float));
    
    // Process each active voice
    for (int v = 0; v < MAX_VOICES; v++) {
        if (!atomic_load(&shared->voice_active[v])) {
            continue;
        }
        
        RingBuffer* ring = &shared->voice_buffers[v];
        
        // Read available samples from ring buffer
        int samples_read = ring_buffer_read(ring, temp_buffer, num_samples);
        
        if (samples_read < num_samples) {
            // Underrun - voice couldn't keep up
            // Could fade out or handle gracefully
            atomic_store(&shared->voice_active[v], 0);
        }
        
        // Mix into output
        for (int i = 0; i < samples_read; i++) {
            outputs[0][i] += temp_buffer[i] * 0.5f;
            outputs[1][i] += temp_buffer[i] * 0.5f;
        }
    }
}
```

### Event Handling

```go
// Go side - handles MIDI events
func (s *Synth) HandleNoteOn(note, velocity byte) {
    // Find free voice (can use complex allocation strategy)
    voice := s.findFreeVoice()
    if voice == nil {
        voice = s.stealVoice() // Complex voice stealing
    }
    
    // Configure voice (this can allocate, use maps, etc.)
    voice.SetNote(note)
    voice.SetVelocity(velocity)
    
    // Start voice processing goroutine
    go voice.run()
    
    // Tell C engine this voice is active
    s.shared.voice_active[voice.id].Store(1)
}
```

## Benefits

### 1. **True Go Development**
Developers write normal Go code with full language features:
- Can use `make()`, `append()`, maps, interfaces
- Can import any Go package
- Normal Go debugging and profiling tools work
- Can use goroutines within voice processing

### 2. **Predictable RT Performance**
C engine has completely predictable behavior:
- Fixed processing time per sample
- No allocations
- No blocking operations
- Can guarantee latency

### 3. **Natural Buffering**
Ring buffers provide:
- Smooth out Go's GC pauses
- Allow batch processing in Go (more efficient)
- Natural latency compensation
- Overrun/underrun handling

### 4. **Scalability**
- Can process voices in parallel goroutines
- Can dynamically adjust processing chunk size
- Can priorities voices based on importance

## Advanced Features

### Dynamic Quality Adjustment

```go
func (v *Voice) run() {
    for v.active {
        fillLevel := v.buffer.FillLevel()
        
        // Adjust quality based on buffer state
        if fillLevel < 0.25 {
            // Buffer running low - reduce quality
            v.filter.SetQuality(QUALITY_LOW)
            v.Process(CHUNK_SIZE * 2) // Process more
        } else if fillLevel > 0.75 {
            // Plenty of buffer - increase quality
            v.filter.SetQuality(QUALITY_HIGH)
            v.Process(CHUNK_SIZE / 2) // Process less
        }
    }
}
```

### Voice Pooling

```go
// Pre-warm voices in background
func (s *Synth) Initialize() {
    s.voicePool = make(chan *Voice, MAX_VOICES)
    
    // Start voice processing goroutines
    for i := 0; i < MAX_VOICES; i++ {
        voice := NewVoice(i, s.shared)
        s.voicePool <- voice
        go voice.run() // Already running, waiting for activation
    }
}
```

### Lookahead Processing

```go
// Process modulation ahead of audio
func (v *Voice) processModulation() {
    // Can process envelopes, LFOs, etc. ahead of time
    modBuffer := make([]float32, LOOKAHEAD_SIZE)
    for i := range modBuffer {
        modBuffer[i] = v.lfo.Process() * v.env.Process()
    }
    v.modRingBuffer.Write(modBuffer)
}
```

## Latency Considerations

Total latency = Host Buffer + Ring Buffer Depth

Example:
- Host buffer: 128 samples (2.9ms @ 44.1kHz)
- Ring buffer: 512 samples (11.6ms @ 44.1kHz)
- Total: ~15ms latency

This is acceptable for many use cases:
- Pad/string sounds
- Ambient textures  
- Effects processing
- Background layers

For ultra-low latency needs (live playing), can reduce buffer sizes at the cost of higher CPU usage.

## Error Handling

```c
// C side - graceful degradation
if (ring_buffer_available(ring) < num_samples) {
    // Not enough samples available
    if (voice->priority == PRIORITY_HIGH) {
        // High priority - output silence briefly
        int available = ring_buffer_available(ring);
        ring_buffer_read(ring, outputs[0], available);
        // Fade out the rest
        for (int i = available; i < num_samples; i++) {
            outputs[0][i] = outputs[0][available-1] * (1.0f - (float)i/num_samples);
        }
    } else {
        // Low priority - stop voice
        atomic_store(&shared->voice_active[v], 0);
    }
}
```

## Summary

This shared buffer approach gives us:

1. **Full Go capabilities** - No restrictions on Go code
2. **RT-safe audio output** - C handles all timing critical operations
3. **Natural GC tolerance** - Buffers smooth out pauses
4. **Developer friendly** - Write normal Go code
5. **Production ready** - Used by many audio systems (JACK, ASIO)

The key insight: **Decouple audio generation (Go) from audio timing (C)**. Go generates audio data at its own pace, C ensures it plays back smoothly.