# Latency Enforcement: Ensuring We Deliver What We Promise

## The Critical Problem

If we report 50ms latency but actually process in real-time (0ms), we create a disaster:

```
What we promise:     [────50ms delay────][Audio Output]
What actually happens: [Audio Output]
Result: Audio is 50ms EARLY! Everything is out of sync!
```

## The Enforcement Mechanism

### 1. Ring Buffer with Enforced Read Position

```go
type WriteAheadBuffer struct {
    data         []float32
    size         uint32
    writePos     atomic.Uint64
    readPos      atomic.Uint64
    
    // CRITICAL: This enforces our latency
    initialized  atomic.Bool
    startTime    time.Time
    sampleRate   float64
}

// Initialize sets up the buffer with enforced delay
func (b *WriteAheadBuffer) Initialize(latencySamples uint32, sampleRate float64) {
    b.sampleRate = sampleRate
    b.startTime = time.Now()
    
    // CRITICAL: Write position starts at latencySamples
    b.writePos.Store(uint64(latencySamples))
    b.readPos.Store(0)
    
    // Pre-fill with silence to enforce delay
    silence := make([]float32, latencySamples)
    b.WriteSlice(silence)
    
    b.initialized.Store(true)
}

// ReadSlice ENFORCES the delay - cannot read ahead of time
func (b *WriteAheadBuffer) ReadSlice(out []float32, currentSampleTime uint64) int {
    if !b.initialized.Load() {
        // Not initialized - output silence
        for i := range out {
            out[i] = 0
        }
        return 0
    }
    
    // CRITICAL ENFORCEMENT: Calculate maximum allowed read position
    elapsedTime := time.Since(b.startTime)
    elapsedSamples := uint64(elapsedTime.Seconds() * b.sampleRate)
    
    // We can only read samples that are at least latencySamples old
    maxReadPos := elapsedSamples
    currentReadPos := b.readPos.Load()
    
    // Enforce: Never read beyond what maintains our latency
    if currentReadPos > maxReadPos {
        // We're trying to read too early! Output silence
        for i := range out {
            out[i] = 0
        }
        return 0
    }
    
    // Normal read operation
    writePos := b.writePos.Load()
    available := int(writePos - currentReadPos)
    
    // ... rest of read implementation
}
```

### 2. The Three-Buffer Architecture for Guaranteed Latency

A more robust approach uses three conceptual buffers:

```
┌─────────────────────────────────────────────────────────┐
│                  Three-Stage Pipeline                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Stage 1: Future Buffer (Go writes here)                │
│  [████████████████████████████████]                     │
│                                                          │
│  Stage 2: Delay Buffer (50ms of samples)                │
│  [▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓]                     │
│                                                          │
│  Stage 3: Ready Buffer (Host reads here)                │
│  [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░]                     │
│                                                          │
└─────────────────────────────────────────────────────────┘

Sample flow: Future → (wait 50ms) → Delay → Ready → Output
```

Implementation:

```go
type ThreeStageBuffer struct {
    // Circular buffer with three logical regions
    buffer      []float32
    bufferSize  int
    
    // Three pointers defining regions
    writePtr    atomic.Uint64  // Where Go writes
    delayPtr    atomic.Uint64  // 50ms behind write
    readPtr     atomic.Uint64  // Where host reads
    
    // Timing enforcement
    latencySamples uint64
    ticker         *time.Ticker
}

func (t *ThreeStageBuffer) enforcer() {
    // Runs in separate goroutine
    for range t.ticker.C {
        // Move samples from future to delay buffer
        writePos := t.writePtr.Load()
        delayPos := t.delayPtr.Load()
        
        // Only advance delay pointer if we're maintaining distance
        if writePos-delayPos >= t.latencySamples {
            // Move one chunk from future to delay
            t.delayPtr.Add(128) // Move in small chunks
        }
    }
}
```

### 3. Timestamp-Based Enforcement

The most robust approach tags each sample with its "ready time":

```go
type TimestampedBuffer struct {
    samples []struct {
        data      float32
        readyTime uint64  // When this sample can be read
    }
    
    currentTime uint64  // Incremented each Process() call
    latency     uint64  // Our promised latency
}

func (t *TimestampedBuffer) Write(sample float32) {
    t.samples = append(t.samples, struct {
        data      float32
        readyTime uint64
    }{
        data:      sample,
        readyTime: t.currentTime + t.latency,  // Can't read until future
    })
}

func (t *TimestampedBuffer) Read() float32 {
    if len(t.samples) == 0 {
        return 0
    }
    
    // ENFORCEMENT: Can only read samples whose time has come
    if t.samples[0].readyTime > t.currentTime {
        return 0  // Not ready yet - return silence
    }
    
    // Sample is ready
    sample := t.samples[0].data
    t.samples = t.samples[1:]  // Remove read sample
    return sample
}
```

## The Complete Enforcement Solution

```go
// pkg/dsp/buffer/enforced.go
package buffer

type EnforcedLatencyBuffer struct {
    // Double buffer for lock-free operation
    buffers     [2][]float32
    activeIdx   atomic.Uint32
    
    // Positions
    writeOffset uint64
    
    // Enforcement mechanism
    latencySamples   uint64
    samplesGenerated atomic.Uint64
    samplesConsumed  atomic.Uint64
    
    // Safety checks
    underruns atomic.Uint64
    overruns  atomic.Uint64
}

// Process is called by the host
func (e *EnforcedLatencyBuffer) Process(out []float32, numSamples int) {
    consumed := e.samplesConsumed.Load()
    generated := e.samplesGenerated.Load()
    
    // CRITICAL: Enforce latency
    maxConsumable := uint64(0)
    if generated > e.latencySamples {
        maxConsumable = generated - e.latencySamples
    }
    
    // Can we provide the requested samples?
    if consumed+uint64(numSamples) > maxConsumable {
        // Would violate latency - provide partial/silence
        canProvide := int(maxConsumable - consumed)
        
        // Read what we can
        e.readSamples(out[:canProvide])
        
        // Fill rest with silence
        for i := canProvide; i < numSamples; i++ {
            out[i] = 0
        }
        
        e.underruns.Add(1)
        return
    }
    
    // Normal case - read the samples
    e.readSamples(out)
    e.samplesConsumed.Add(uint64(numSamples))
}
```

## Testing Latency Enforcement

```go
func TestLatencyEnforcement(t *testing.T) {
    buffer := NewEnforcedLatencyBuffer(2205) // 50ms @ 44.1kHz
    
    // Try to read immediately - should get silence
    out := make([]float32, 512)
    buffer.Process(out, 512)
    
    for _, sample := range out {
        assert.Equal(t, float32(0), sample, "Should get silence before latency")
    }
    
    // Write 3000 samples
    buffer.Write(generateTestSamples(3000))
    
    // Should only be able to read 3000-2205 = 795 samples
    buffer.Process(out, 1000)
    
    // First 795 should have data, rest silence
    assert.NotEqual(t, float32(0), out[0], "Should have data")
    assert.Equal(t, float32(0), out[799], "Should be silence past available")
}
```

## The Golden Rule

**Never allow reading samples younger than the reported latency!**

```go
if sampleAge < reportedLatency {
    return silence
}
```

## Why This Matters

Without enforcement, you get:
- Audio playing 50ms early
- MIDI events out of sync
- Automation playing at wrong time
- Other tracks delayed but yours isn't
- Complete chaos in the mix

With proper enforcement:
- Audio delayed exactly as promised
- Perfect synchronization
- Host's PDC works correctly
- Professional, predictable behavior

## Implementation Priority

1. **Start Simple**: Pre-fill buffer with silence
2. **Add Enforcement**: Check sample age before reading
3. **Monitor Health**: Track underruns and buffer state
4. **Test Thoroughly**: Verify latency is exactly as reported