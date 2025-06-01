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
    
    // Read from buffers into output
    for ch := 0; ch < numChannels; ch++ {
        p.buffers[ch].ReadSlice(data.Outputs[0].Buffers[ch][:numSamples])
    }
    
    return nil
}

// GetLatencySamples reports the write-ahead latency
func (p *BufferedProcessor) GetLatencySamples() int32 {
    if len(p.buffers) > 0 {
        return int32(p.buffers[0].GetLatencySamples())
    }
    return 0
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

## Conclusion

This approach provides robust GC compatibility without sacrificing Go's memory management benefits. The 50ms write-ahead buffer easily absorbs typical GC pauses (5-20ms) while maintaining reasonable latency for most audio applications. Plugin developers write normal Go code while the framework handles all the complexity of real-time safety.