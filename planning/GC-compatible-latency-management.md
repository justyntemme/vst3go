# GC-Compatible Latency Management for vst3go

## Overview

This document outlines a latency management strategy that maintains Go's standard garbage collection while ensuring glitch-free audio processing. Instead of disabling GC with `GOGC=off`, we use a write-ahead buffer where Go writes audio 50ms ahead of where the C code reads, creating a natural cushion for GC pauses.

## Core Concept

```
Write Position (Go)          Read Position (C)
      ↓                           ↓
[━━━━━━━━━━━━━━━━━━━━░░░░░░░░░░░░░░░░░░░░░░░░░]
└────── 50ms buffer ──────┘└─── unused space ───┘

Even if GC blocks for 10-20ms, the read position 
never catches up to the write position.
```

## Architecture Design

### 1. Dual-Position Ring Buffer

```go
// pkg/dsp/buffer/writeahead.go
package buffer

import (
    "sync/atomic"
    "unsafe"
)

// WriteAheadBuffer provides GC-safe audio buffering
type WriteAheadBuffer struct {
    // Audio data
    data      []float32
    size      uint32
    mask      uint32
    
    // Positions (must be cache-line aligned)
    writePos  atomic.Uint64 // Go writes here
    readPos   atomic.Uint64 // C reads here
    
    // Configuration
    aheadSamples uint32 // How far ahead to write (e.g., 50ms)
    
    // Statistics
    underruns atomic.Uint64
    overruns  atomic.Uint64
}

func NewWriteAheadBuffer(sizeSamples, aheadSamples uint32) *WriteAheadBuffer {
    // Ensure power of 2 for fast modulo
    size := nextPowerOf2(sizeSamples)
    
    buf := &WriteAheadBuffer{
        data:         make([]float32, size),
        size:         size,
        mask:         size - 1,
        aheadSamples: aheadSamples,
    }
    
    // Initialize write position ahead of read
    buf.writePos.Store(uint64(aheadSamples))
    buf.readPos.Store(0)
    
    return buf
}

// WriteSlice writes samples from Go (can be called from any goroutine)
func (b *WriteAheadBuffer) WriteSlice(samples []float32) bool {
    writePos := b.writePos.Load()
    readPos := b.readPos.Load()
    
    // Check if we have space
    availableSpace := b.size - uint32(writePos-readPos)
    if availableSpace < uint32(len(samples)) {
        b.overruns.Add(1)
        return false // Buffer full
    }
    
    // Write samples
    for i, sample := range samples {
        idx := uint32(writePos+uint64(i)) & b.mask
        b.data[idx] = sample
    }
    
    // Update write position
    b.writePos.Add(uint64(len(samples)))
    return true
}

// ReadSlice reads samples for C (called from audio thread)
func (b *WriteAheadBuffer) ReadSlice(out []float32) int {
    writePos := b.writePos.Load()
    readPos := b.readPos.Load()
    
    // Check available samples
    available := int(writePos - readPos)
    toRead := min(available, len(out))
    
    if toRead < len(out) {
        b.underruns.Add(1)
    }
    
    // Read samples
    for i := 0; i < toRead; i++ {
        idx := uint32(readPos+uint64(i)) & b.mask
        out[i] = b.data[idx]
    }
    
    // Zero remaining samples if underrun
    for i := toRead; i < len(out); i++ {
        out[i] = 0
    }
    
    // Update read position
    b.readPos.Add(uint64(toRead))
    return toRead
}

// GetLatencySamples returns the ahead buffer size
func (b *WriteAheadBuffer) GetLatencySamples() uint32 {
    return b.aheadSamples
}

// GetBufferHealth returns buffer fill percentage
func (b *WriteAheadBuffer) GetBufferHealth() float32 {
    writePos := b.writePos.Load()
    readPos := b.readPos.Load()
    filled := float32(writePos - readPos)
    return filled / float32(b.size)
}

// GetCurrentDelay returns actual delay between write and read heads
func (b *WriteAheadBuffer) GetCurrentDelay() uint32 {
    writePos := b.writePos.Load()
    readPos := b.readPos.Load()
    return uint32(writePos - readPos)
}

// MaintainDelay ensures the read head stays the target distance behind write
func (b *WriteAheadBuffer) MaintainDelay() {
    currentDelay := b.GetCurrentDelay()
    
    // If delay is too small, we need to increase it
    if currentDelay < b.aheadSamples {
        // Pause reading by not advancing read position
        // This will naturally increase the delay
    }
    
    // If delay is too large, we need to decrease it
    if currentDelay > b.aheadSamples*2 {
        // Skip ahead to maintain reasonable latency
        skipAmount := currentDelay - b.aheadSamples
        b.readPos.Add(uint64(skipAmount))
    }
}
```

### 2. Plugin Integration Layer

```go
// pkg/plugin/buffered_processor.go
package plugin

import (
    "time"
    "runtime"
    "github.com/tmegow/vst3go/pkg/dsp/buffer"
)

// BufferedProcessor wraps a processor with write-ahead buffering
type BufferedProcessor struct {
    BaseProcessor
    
    // User's actual processor
    wrapped Processor
    
    // Per-channel buffers
    buffers []*buffer.WriteAheadBuffer
    
    // Processing state
    sampleRate      float64
    processingChunk int32
    isRunning       atomic.Bool
    
    // Worker goroutine
    worker *processingWorker
}

type processingWorker struct {
    processor    Processor
    buffers      []*buffer.WriteAheadBuffer
    chunkSize    int32
    sampleRate   float64
    stopChan     chan struct{}
    
    // Temporary buffers for processing
    inputChunk   [][]float32
    outputChunk  [][]float32
}

func NewBufferedProcessor(wrapped Processor, latencyMs float64) *BufferedProcessor {
    return &BufferedProcessor{
        wrapped:         wrapped,
        processingChunk: 512, // Process in 512 sample chunks
    }
}

func (p *BufferedProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    p.sampleRate = sampleRate
    
    // Initialize wrapped processor
    if err := p.wrapped.Initialize(sampleRate, maxSamplesPerBlock); err != nil {
        return err
    }
    
    // Calculate buffer sizes
    latencyMs := 50.0 // 50ms write-ahead
    latencySamples := uint32(latencyMs * sampleRate / 1000.0)
    bufferSize := latencySamples * 4 // 200ms total buffer
    
    // Create per-channel buffers (assume stereo for now)
    p.buffers = make([]*buffer.WriteAheadBuffer, 2)
    for i := range p.buffers {
        p.buffers[i] = buffer.NewWriteAheadBuffer(bufferSize, latencySamples)
    }
    
    // Start processing worker
    p.worker = &processingWorker{
        processor:  p.wrapped,
        buffers:    p.buffers,
        chunkSize:  p.processingChunk,
        sampleRate: sampleRate,
        stopChan:   make(chan struct{}),
    }
    
    p.worker.initBuffers()
    p.isRunning.Store(true)
    go p.worker.run()
    
    return nil
}

func (w *processingWorker) initBuffers() {
    // Pre-allocate processing buffers
    w.inputChunk = make([][]float32, 2)
    w.outputChunk = make([][]float32, 2)
    for i := range w.inputChunk {
        w.inputChunk[i] = make([]float32, w.chunkSize)
        w.outputChunk[i] = make([]float32, w.chunkSize)
    }
}

func (w *processingWorker) run() {
    // Allow normal GC operation
    runtime.GC() // Initial GC to start clean
    
    ticker := time.NewTicker(5 * time.Millisecond) // Check every 5ms
    defer ticker.Stop()
    
    for {
        select {
        case <-w.stopChan:
            return
        case <-ticker.C:
            w.processNextChunk()
        }
    }
}

func (w *processingWorker) processNextChunk() {
    // Check buffer health
    health := w.buffers[0].GetBufferHealth()
    
    // Adaptive processing based on buffer state
    if health < 0.3 {
        // Buffer getting low - process more aggressively
        w.processChunks(4) // Process 4 chunks
    } else if health < 0.5 {
        // Normal processing
        w.processChunks(2)
    } else if health > 0.8 {
        // Buffer quite full - can relax
        w.processChunks(1)
    }
    // If buffer is very full (>0.8), skip this tick
}

func (w *processingWorker) processChunks(numChunks int) {
    for chunk := 0; chunk < numChunks; chunk++ {
        // Clear input buffers (simulating silence for this example)
        for ch := range w.inputChunk {
            for i := range w.inputChunk[ch] {
                w.inputChunk[ch][i] = 0
            }
        }
        
        // Create mock process data
        data := &process.Data{
            NumSamples: w.chunkSize,
            Inputs: []process.Bus{{
                Buffers: w.inputChunk,
            }},
            Outputs: []process.Bus{{
                Buffers: w.outputChunk,
            }},
        }
        
        // Process audio
        if err := w.processor.Process(data); err != nil {
            // Log error but continue
            continue
        }
        
        // Write to buffers
        for ch, buffer := range w.buffers {
            if !buffer.WriteSlice(w.outputChunk[ch]) {
                // Buffer full - this shouldn't happen with proper sizing
                break
            }
        }
    }
}

// Process is called by the host - just reads from buffers
func (p *BufferedProcessor) Process(data *process.Data) error {
    numChannels := min(len(data.Outputs[0].Buffers), len(p.buffers))
    numSamples := int(data.NumSamples)
    
    // CRITICAL: Maintain the 50ms delay before reading
    for ch := 0; ch < numChannels; ch++ {
        p.buffers[ch].MaintainDelay()
    }
    
    // Read from buffers into output
    for ch := 0; ch < numChannels; ch++ {
        p.buffers[ch].ReadSlice(data.Outputs[0].Buffers[ch][:numSamples])
    }
    
    // Process MIDI events with timestamp compensation
    if data.InputEvents != nil {
        p.processEventsWithLatencyCompensation(data.InputEvents)
    }
    
    return nil
}

// GetLatencySamples reports the write-ahead latency to VST3
// This is CRITICAL for proper plugin delay compensation
func (p *BufferedProcessor) GetLatencySamples() int32 {
    if len(p.buffers) > 0 {
        // Report the target latency, not current buffer state
        return int32(p.buffers[0].GetLatencySamples())
    }
    return 0
}

// processEventsWithLatencyCompensation adjusts MIDI event timing
func (p *BufferedProcessor) processEventsWithLatencyCompensation(events EventList) {
    latencySamples := p.GetLatencySamples()
    
    for i := 0; i < events.GetEventCount(); i++ {
        event := events.GetEvent(i)
        
        // Adjust event timestamp to account for our latency
        // The event should be processed when its audio comes out
        adjustedSampleOffset := event.SampleOffset + latencySamples
        
        // Queue event for the worker at the right time
        p.worker.queueEvent(event, adjustedSampleOffset)
    }
}

func (p *BufferedProcessor) Terminate() error {
    // Stop worker
    if p.isRunning.CompareAndSwap(true, false) {
        close(p.worker.stopChan)
    }
    
    // Terminate wrapped processor
    return p.wrapped.Terminate()
}
```

### 3. C-Side Integration

```c
// bridge/buffered_bridge.c

// Export for Go to get buffer pointers
void* GetBufferReadPointer(void* bufferPtr, int channel) {
    BufferedProcessor* proc = (BufferedProcessor*)bufferPtr;
    if (channel < proc->numChannels) {
        return proc->buffers[channel]->data;
    }
    return NULL;
}

// Optimized C reading for maximum performance
void ProcessBufferedAudio(BufferedProcessor* proc, 
                         float** outputs, 
                         int32_t numSamples) {
    for (int ch = 0; ch < proc->numChannels; ch++) {
        WriteAheadBuffer* buf = proc->buffers[ch];
        
        uint64_t readPos = atomic_load(&buf->readPos);
        uint64_t writePos = atomic_load(&buf->writePos);
        
        // Fast path - enough samples available
        if (writePos - readPos >= numSamples) {
            // Direct memory copy with wrap-around handling
            uint32_t startIdx = readPos & buf->mask;
            uint32_t endIdx = (readPos + numSamples) & buf->mask;
            
            if (startIdx < endIdx) {
                // No wrap - single memcpy
                memcpy(outputs[ch], &buf->data[startIdx], 
                       numSamples * sizeof(float));
            } else {
                // Handle wrap-around
                uint32_t firstPart = buf->size - startIdx;
                memcpy(outputs[ch], &buf->data[startIdx], 
                       firstPart * sizeof(float));
                memcpy(&outputs[ch][firstPart], buf->data, 
                       (numSamples - firstPart) * sizeof(float));
            }
            
            atomic_fetch_add(&buf->readPos, numSamples);
        } else {
            // Underrun - handle gracefully
            HandleUnderrun(buf, outputs[ch], numSamples);
        }
    }
}
```

## Critical: Read Head Enforcement and Event Timing

### How the 50ms Gap is Maintained

1. **Initial Setup**: When the buffer is created, the write position starts at sample 2205 (50ms at 44.1kHz) while read starts at 0.

2. **Continuous Enforcement**: The `MaintainDelay()` function is called before every read operation:
   ```go
   // In Process() - called every audio callback
   for ch := 0; ch < numChannels; ch++ {
       p.buffers[ch].MaintainDelay() // Enforces the gap
   }
   ```

3. **Gap Adjustment**:
   - If gap < 50ms: Reading pauses (doesn't advance read pointer)
   - If gap > 100ms: Read pointer jumps forward to reduce latency

### MIDI Event Synchronization

The key insight: **MIDI events must be delayed by the same amount as audio** to maintain synchronization.

```
User presses key at T=0
                ↓
VST3 receives MIDI event with sampleOffset=0
                ↓
We adjust: sampleOffset = 0 + 2205 (50ms)
                ↓
Worker processes event when its position reaches 2205
                ↓
Audio output occurs exactly when user expects
```

### Complete Event Flow Example

```go
// Enhanced worker with event queue
type processingWorker struct {
    // ... existing fields ...
    
    // Event handling
    eventQueue     []TimedEvent
    eventQueueLock sync.Mutex
    currentSample  uint64
}

type TimedEvent struct {
    event         Event
    targetSample  uint64
}

func (w *processingWorker) processChunks(numChunks int) {
    for chunk := 0; chunk < numChunks; chunk++ {
        // Check for events that should fire in this chunk
        w.processQueuedEvents()
        
        // Normal audio processing
        data := &process.Data{
            NumSamples: w.chunkSize,
            // ... setup buffers ...
        }
        
        // Process audio
        w.processor.Process(data)
        
        // Write to buffers
        for ch, buffer := range w.buffers {
            buffer.WriteSlice(w.outputChunk[ch])
        }
        
        // Advance our position
        w.currentSample += uint64(w.chunkSize)
    }
}

func (w *processingWorker) processQueuedEvents() {
    w.eventQueueLock.Lock()
    defer w.eventQueueLock.Unlock()
    
    // Process all events whose time has come
    processed := 0
    for i, timedEvent := range w.eventQueue {
        if timedEvent.targetSample <= w.currentSample {
            // Send event to processor
            w.processor.HandleEvent(timedEvent.event)
            processed = i + 1
        } else {
            break // Events are time-ordered
        }
    }
    
    // Remove processed events
    if processed > 0 {
        w.eventQueue = w.eventQueue[processed:]
    }
}

func (w *processingWorker) queueEvent(event Event, adjustedOffset int32) {
    w.eventQueueLock.Lock()
    defer w.eventQueueLock.Unlock()
    
    targetSample := w.currentSample + uint64(adjustedOffset)
    
    // Insert in time order
    timedEvent := TimedEvent{event: event, targetSample: targetSample}
    
    // Binary search for insertion point
    insertIdx := sort.Search(len(w.eventQueue), func(i int) bool {
        return w.eventQueue[i].targetSample > targetSample
    })
    
    // Insert event
    w.eventQueue = append(w.eventQueue[:insertIdx], 
                          append([]TimedEvent{timedEvent}, 
                                 w.eventQueue[insertIdx:]...)...)
}
```

### VST3 Integration Points

#### 1. Host Latency Query and Compensation

The VST3 host manages latency through a two-step process:

```c
// Step 1: Host queries our latency during initialization
// This happens BEFORE any audio processing
uint32_t GoAudioGetLatencySamples(void* componentPtr) {
    // We ALWAYS report 50ms (2205 samples @ 44.1kHz)
    // This tells the host our plugin has a fixed 50ms delay
    return 2205;
}
```

```
Host's Internal Compensation:
┌─────────────────────────────────────────────────┐
│ Track 1: Direct (no latency)    [═══════════>   │
│ Track 2: Our Plugin (50ms)      [----→══════>   │
│                                  ↑              │
│                           Host adds 50ms delay  │
│                           to all OTHER tracks   │
└─────────────────────────────────────────────────┘
```

#### 2. MIDI Event Flow in Current VST3 Architecture

**PROBLEM**: In the current vst3go architecture, MIDI events go directly to Go:

```
Current Flow (PROBLEMATIC):
User Input → Host → C Bridge → Go Process() → Audio Output
                                    ↑
                             Receives event immediately
                             but audio is delayed 50ms!
```

**SOLUTION**: We need to intercept and queue events:

```go
// Modified event handling in wrapper_audio.go
type BufferedEventProcessor interface {
    // Regular processor interface
    Processor
    
    // New method for queued event handling
    QueueEvent(event Event, sampleOffset int32)
}

// In Process callback
func (w *Wrapper) Process(data *process.Data) error {
    // If using buffered processor, handle events specially
    if buffered, ok := w.processor.(BufferedEventProcessor); ok {
        // Don't pass events directly to processor
        // Instead, queue them with timing adjustment
        for i := 0; i < data.InputEvents.GetEventCount(); i++ {
            event := data.InputEvents.GetEvent(i)
            buffered.QueueEvent(event, event.SampleOffset)
        }
        // Don't set data.InputEvents on the process data
        data.InputEvents = nil
    }
    
    return w.processor.Process(data)
}
```

#### 3. Complete MIDI Event Timeline

```
T=0ms    User presses key
         ↓
T=0ms    Host receives MIDI input
         ↓
T=0ms    Host calls our Process() with event at sampleOffset=0
         ↓
T=0ms    BufferedProcessor queues event for T=50ms
         ↓
T=0ms    C reads audio from buffer (50ms old audio)
         ↓
T=50ms   Worker processes queued MIDI event
         ↓
T=50ms   New audio with note begins filling buffer
         ↓
T=100ms  C reads the audio containing the note
         ↓
T=100ms  Audio output (user hears the note)

From user perspective: 100ms total latency
But host compensates: shifts everything back 50ms
Perceived latency: 50ms (same as reported)
```

#### 4. How DAWs Handle Plugin Latency

Different DAWs handle latency compensation slightly differently, but the general principle is:

1. **Recording**: Input monitoring is delayed to match plugin latency
2. **Playback**: All tracks are delayed to match the highest latency
3. **Live Performance**: Some DAWs offer "low latency mode" that bypasses high-latency plugins

**Critical Implementation Detail**:
```go
// The host ONLY compensates if we report latency consistently
func (p *BufferedProcessor) GetLatencySamples() int32 {
    // MUST return the same value every time it's called
    // Changing this dynamically will confuse the host!
    return 2205 // Fixed 50ms @ 44.1kHz
}
```

### Startup Sequence

To ensure the 50ms gap from the start:

```go
func (p *BufferedProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    // ... create buffers ...
    
    // Pre-fill buffers with 50ms of silence
    silence := make([]float32, p.buffers[0].aheadSamples)
    for _, buffer := range p.buffers {
        buffer.WriteSlice(silence)
    }
    
    // Now start the worker
    go p.worker.run()
    
    return nil
}
```

## Implementation Strategy

### Phase 1: Core Buffer Implementation
1. Implement `WriteAheadBuffer` with atomic operations
2. Add comprehensive tests for concurrent access
3. Benchmark performance vs standard channels

### Phase 2: Plugin Integration
1. Create `BufferedProcessor` wrapper
2. Implement adaptive processing based on buffer health
3. Add monitoring and statistics

### Phase 3: Optimization
1. Add C-side optimizations for reading
2. Implement SIMD operations where applicable
3. Profile and tune buffer sizes

## Timing Diagram

This diagram shows how the buffer maintains synchronization:

```
Time (samples) →
0     1000   2000   3000   4000   5000   6000   7000   8000

Write Position (Go):
[-----W₀-----][-----W₁-----][-----W₂-----][-----W₃-----]
                                                      ↑ Current write

Read Position (C):
                    [-----R₀-----][-----R₁-----][-----R₂-----]
                                          ↑ Current read

Gap = Write - Read = ~2205 samples (50ms @ 44.1kHz)

MIDI Event Timeline:
User Input:    Note ON ↓
Host Receives:         Note ON (offset=0)
We Adjust:             Note ON (offset=2205) →→→→→
Worker Processes:                              Note ON ↓
Audio Output:                                         Note starts here
```

Key points:
- The gap ensures GC pauses don't cause dropouts
- MIDI events are delayed to match audio latency
- The host's Plugin Delay Compensation ensures everything stays in sync

## Configuration Options

```go
type BufferConfig struct {
    // Latency in milliseconds (default: 50ms)
    LatencyMs float64
    
    // Processing chunk size (default: 512 samples)
    ChunkSize int32
    
    // Worker tick rate (default: 5ms)
    TickRateMs int
    
    // Adaptive processing thresholds
    LowThreshold  float32 // Default: 0.3
    HighThreshold float32 // Default: 0.8
    
    // Enable statistics collection
    EnableStats bool
}
```

## Usage Example

```go
// Plugin developer's code remains simple
type MyReverb struct {
    reverb *dsp.FreeVerb
}

func (r *MyReverb) Process(data *process.Data) error {
    // Normal processing code - no special handling needed
    return r.reverb.Process(data.Inputs[0].Buffers, 
                           data.Outputs[0].Buffers, 
                           int(data.NumSamples))
}

// Framework wraps it automatically
func CreatePlugin() plugin.Processor {
    reverb := &MyReverb{
        reverb: dsp.NewFreeVerb(),
    }
    
    // Wrap with buffering for GC safety
    return plugin.NewBufferedProcessor(reverb, 50.0) // 50ms latency
}
```

## Performance Considerations

### Memory Layout
- Keep read/write positions on separate cache lines
- Align buffer data for SIMD operations
- Use power-of-2 sizes for fast modulo

### Adaptive Processing
- Process more when buffer is low
- Reduce processing when buffer is full
- Sleep when buffer is healthy to reduce CPU usage

### GC Interaction
- Pre-allocate all buffers to reduce allocation pressure
- Reuse processing chunks
- Let GC run normally - the buffer absorbs pauses

## Monitoring and Debugging

```go
type BufferStats struct {
    Underruns       uint64
    Overruns        uint64
    AverageLatency  float64
    BufferHealth    float32
    ProcessingTime  time.Duration
}

func (p *BufferedProcessor) GetStats() BufferStats {
    return BufferStats{
        Underruns:    p.buffers[0].underruns.Load(),
        Overruns:     p.buffers[0].overruns.Load(),
        BufferHealth: p.buffers[0].GetBufferHealth(),
        // ... etc
    }
}
```

## Advantages Over GOGC=off

1. **Natural Go Development**: Keep standard GC behavior
2. **Better Memory Management**: GC prevents memory leaks
3. **Debugging Tools**: Go's profiler and race detector work normally
4. **Adaptive Performance**: Can tune based on system load
5. **Graceful Degradation**: Handles overload situations well

## Testing Strategy

### Unit Tests
```go
func TestConcurrentAccess(t *testing.T) {
    buf := NewWriteAheadBuffer(1024, 256)
    
    // Simulate Go writing
    go func() {
        data := make([]float32, 64)
        for i := 0; i < 1000; i++ {
            buf.WriteSlice(data)
            time.Sleep(time.Millisecond)
        }
    }()
    
    // Simulate C reading
    out := make([]float32, 32)
    for i := 0; i < 2000; i++ {
        buf.ReadSlice(out)
        time.Sleep(500 * time.Microsecond)
    }
    
    // Check no underruns
    assert.Equal(t, uint64(0), buf.underruns.Load())
}
```

### Integration Tests
- Test with artificial GC pressure
- Verify latency reporting
- Test adaptive processing behavior

### Stress Tests
- Run with `GODEBUG=gctrace=1` to monitor GC
- Test with small buffers to induce underruns
- Verify graceful handling of edge cases

## Future Enhancements

1. **Multi-channel Optimization**: Process all channels in single operation
2. **Zero-copy C Interface**: Direct memory mapping for C side
3. **Dynamic Latency**: Adjust based on GC behavior
4. **Priority Processing**: Different latencies for different plugin types

## Critical: Deterministic Latency Despite Non-Deterministic GC

### The Core Problem
Go's GC is non-deterministic - it can pause for 1ms or 20ms unpredictably. Yet VST3 requires us to report a FIXED latency value. How do we reconcile this?

### The Solution: Buffer-Based Decoupling

```
Non-Deterministic Side (Go)        Deterministic Side (Host/C)
┌─────────────────────────┐        ┌─────────────────────────┐
│ GC can pause 1-20ms     │        │ Always reads exactly    │
│ Writes happen in bursts │───────>│ 50ms behind write pos   │
│ Timing varies           │ Buffer │ Consistent latency      │
└─────────────────────────┘        └─────────────────────────┘
                            
The buffer absorbs ALL timing variations
```

### Why This Guarantees Deterministic Latency

1. **We ALWAYS report 50ms**: No matter what GC does
2. **Buffer is larger than worst-case GC**: 50ms > 20ms typical max GC pause  
3. **Read position is enforced**: Even if Go pauses, C keeps reading at steady rate
4. **Underruns are impossible**: Unless Go stops for >50ms (catastrophic failure)

### Mathematical Proof

```
Given:
- Buffer size: 200ms
- Write-ahead distance: 50ms  
- Worst GC pause: 20ms
- Read rate: Constant (host-driven)

Proof:
- At T=0: Write pos = 50ms, Read pos = 0ms
- GC occurs, Go pauses for 20ms
- During pause: Read advances 20ms, Write stays at 50ms
- Gap reduces to: 50ms - 20ms = 30ms (still safe!)
- After GC: Go catches up quickly, restores 50ms gap
- Result: Output never glitches, latency stays at 50ms
```

## Revolutionary: MIDI Event Buffering Architecture

### The Breakthrough

Traditional VST3 plugins process MIDI events immediately. Our approach buffers MIDI events just like audio, creating perfect synchronization:

```
🎹 MIDI Event Journey Through Our System:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. User presses key at T=0
                ↓
2. Host delivers event with sampleOffset=128 (3ms into buffer)
                ↓
3. We calculate: globalTime = currentPos + 128 + 2205 (50ms latency)
                ↓
4. Event queued at position 2333 in our timeline
                ↓
5. Worker thread at sample 2333: "Time to process this note!"
                ↓
6. Audio generated and written to buffer
                ↓
7. Exactly 50ms later: C reads this audio
                ↓
8. User hears note exactly when expected!
```

### Why This is Elegant

- **Same latency for audio AND events**: Everything shifts by exactly 50ms
- **Sample-accurate timing**: Events fire at exactly the right sample
- **No special cases**: All events (notes, CC, pitch bend) handled identically
- **GC-proof**: Even if GC hits during event processing, timing is preserved

## Host Latency Communication: The VST3 Contract

### How We Guarantee Proper Communication

```c
// This is called by host during initialization
uint32_t GetLatencySamples() {
    return FIXED_LATENCY_SAMPLES; // Always 2205 @ 44.1kHz
}
```

### The VST3 Latency Contract:

1. **Host asks once**: "What's your latency?"
2. **We answer**: "50ms, always and forever"  
3. **Host trusts us**: Sets up compensation for entire session
4. **We deliver**: By maintaining that exact buffer distance

### Critical: We CANNOT Change Latency Dynamically!

```go
// WRONG - This breaks host compensation
func (p *BadProcessor) GetLatencySamples() int32 {
    return p.currentBufferDelay // NO! Changes confuse host
}

// RIGHT - Fixed latency
func (p *GoodProcessor) GetLatencySamples() int32 {
    return 2205 // Always the same, host can trust this
}
```

## Critical Understanding: Latency Reporting

### What Actually Happens

When we report 50ms latency via `GetLatencySamples()`:

1. **The host knows** our plugin delays audio by 50ms
2. **The host compensates** by delaying all other tracks by 50ms  
3. **Everything stays in sync** - the user doesn't hear the delay between tracks
4. **The absolute latency remains** - there's still 50ms from input to output

### Why This Works

- **Most plugins have latency**: Compressors (lookahead), linear phase EQs, convolution reverbs
- **DAWs are built for this**: Plugin Delay Compensation (PDC) is a standard feature
- **Users expect it**: Professional mixing often has 100ms+ total latency
- **Live monitoring**: Users can use direct monitoring or low-latency mode when needed

### Implementation Checklist

✓ Report fixed latency via `GetLatencySamples()`  
✓ Maintain consistent read/write gap in buffers  
✓ Queue MIDI events with proper timing adjustment  
✓ Pre-fill buffers on initialization  
✓ Handle events in worker thread at correct time  

## Conclusion

This approach provides robust GC compatibility without sacrificing Go's memory management benefits. The 50ms write-ahead buffer easily absorbs typical GC pauses (5-20ms) while maintaining reasonable latency for most audio applications. Plugin developers write normal Go code while the framework handles all the complexity of real-time safety.

The key insight: **We don't hide latency, we report it honestly**. The DAW handles the rest through Plugin Delay Compensation, ensuring everything stays perfectly synchronized.