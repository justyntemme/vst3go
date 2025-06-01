# Latency Compensation for Go Garbage Collection in VST3 Plugins

## Overview

This document outlines a strategy for managing and reporting latency caused by Go's garbage collection in real-time audio plugins. By controlling when GC runs and accurately reporting the resulting latency to the host, we can maintain proper plugin delay compensation (PDC) while avoiding audio glitches.

## The Challenge

Go's garbage collector introduces non-deterministic pauses that can cause audio dropouts in real-time processing. While we cannot eliminate GC entirely, we can:

1. Control when GC runs
2. Measure the impact
3. Report it as processing latency to the host
4. Use the host's PDC to maintain timing accuracy

## Strategy 1: Manual GC Control with Dynamic Latency Reporting

### 1. Disable Automatic GC

Set `GOGC=off` to prevent automatic garbage collection during audio processing:

```go
import (
    "os"
    "runtime"
)

func init() {
    // Disable automatic GC
    os.Setenv("GOGC", "off")
    
    // Or use runtime function
    runtime.SetGCPercent(-1)
}
```

### 2. Implement Manual GC Triggering

Trigger GC manually based on memory pressure or time intervals:

```go
type GCManager struct {
    lastGCTime     time.Time
    gcInterval     time.Duration
    memThreshold   uint64
    isProcessing   atomic.Bool
    gcLatencySamples atomic.Int32
    sampleRate     float64
}

func NewGCManager(sampleRate float64, gcIntervalMs int) *GCManager {
    return &GCManager{
        lastGCTime:   time.Now(),
        gcInterval:   time.Duration(gcIntervalMs) * time.Millisecond,
        memThreshold: 100 * 1024 * 1024, // 100MB threshold
        sampleRate:   sampleRate,
    }
}

func (g *GCManager) MaybeRunGC() {
    // Don't run during audio processing
    if g.isProcessing.Load() {
        return
    }
    
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    shouldGC := false
    
    // Check time threshold
    if time.Since(g.lastGCTime) > g.gcInterval {
        shouldGC = true
    }
    
    // Check memory threshold
    if m.Alloc > g.memThreshold {
        shouldGC = true
    }
    
    if shouldGC {
        startTime := time.Now()
        runtime.GC()
        gcDuration := time.Since(startTime)
        
        // Convert GC duration to samples
        latencySamples := int32(gcDuration.Seconds() * g.sampleRate)
        g.gcLatencySamples.Store(latencySamples)
        
        g.lastGCTime = time.Now()
    }
}
```

### 3. Report GC Latency via VST3

Integrate GC latency reporting with the VST3 latency mechanism:

```go
type MyProcessor struct {
    plugin.BaseProcessor
    gcManager *GCManager
    
    // Audio processing state
    processBuffer [][]float32
    bufferLatency int32
}

func (p *MyProcessor) GetLatencySamples() int32 {
    // Report maximum expected GC latency
    // This ensures the host allocates enough compensation buffer
    baseLatency := int32(0)
    
    // Add any algorithmic latency (e.g., lookahead)
    if p.usesLookahead {
        baseLatency += p.lookaheadSamples
    }
    
    // Add GC latency buffer
    // Report worst-case GC pause time (e.g., 10ms)
    gcBufferSamples := int32(0.010 * p.sampleRate) // 10ms buffer
    
    return baseLatency + gcBufferSamples
}

func (p *MyProcessor) Process(data *process.Data) error {
    // Mark processing active
    p.gcManager.isProcessing.Store(true)
    defer p.gcManager.isProcessing.Store(false)
    
    // Process audio
    for i := 0; i < data.NumSamples; i++ {
        // Your DSP code here
    }
    
    return nil
}
```

### 4. Advanced: Dynamic Latency Compensation

For more sophisticated implementations, track actual GC pauses and adjust reported latency dynamically:

```go
type AdaptiveGCManager struct {
    *GCManager
    
    // Ring buffer of recent GC durations
    gcHistory      []time.Duration
    historyIndex   int
    historySize    int
    
    // Statistics
    avgGCDuration  time.Duration
    maxGCDuration  time.Duration
    
    mu sync.RWMutex
}

func (g *AdaptiveGCManager) UpdateGCStats(duration time.Duration) {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // Update history
    g.gcHistory[g.historyIndex] = duration
    g.historyIndex = (g.historyIndex + 1) % g.historySize
    
    // Calculate statistics
    var total time.Duration
    g.maxGCDuration = 0
    
    for _, d := range g.gcHistory {
        total += d
        if d > g.maxGCDuration {
            g.maxGCDuration = d
        }
    }
    
    g.avgGCDuration = total / time.Duration(g.historySize)
}

func (g *AdaptiveGCManager) GetRecommendedLatencySamples(sampleRate float64) int32 {
    g.mu.RLock()
    defer g.mu.RUnlock()
    
    // Use 99th percentile or max with safety margin
    safetyFactor := 1.5
    recommendedLatency := float64(g.maxGCDuration) * safetyFactor
    
    return int32(recommendedLatency.Seconds() * sampleRate)
}
```

## Strategy 2: Deterministic Fixed Latency with Ring Buffers

This approach introduces a constant, predictable latency that exceeds the worst-case GC pause time. By using ring buffers and fixed delays, we create a deterministic system where GC pauses are always hidden within the buffer latency.

### Benefits

1. **Predictable Performance**: Always the same latency, regardless of GC behavior
2. **Simple Implementation**: No complex GC scheduling or monitoring
3. **Guaranteed Glitch-Free**: GC pauses are always shorter than buffer delay
4. **Host Compatibility**: Works perfectly with all DAW latency compensation

### Implementation

```go
type DeterministicLatencyProcessor struct {
    plugin.BaseProcessor
    
    // Fixed latency configuration
    fixedLatencyMs   float64
    fixedLatencySamples int32
    
    // Ring buffers for each channel
    ringBuffers      []RingBuffer
    numChannels      int
    
    // Processing state
    sampleRate       float64
    writePos         int64
    readPos          int64
}

type RingBuffer struct {
    buffer []float32
    size   int
    mask   int // For fast modulo using bitwise AND
}

func NewRingBuffer(size int) RingBuffer {
    // Ensure size is power of 2 for fast modulo
    size = nextPowerOf2(size)
    return RingBuffer{
        buffer: make([]float32, size),
        size:   size,
        mask:   size - 1,
    }
}

func (r *RingBuffer) Write(pos int64, sample float32) {
    r.buffer[int(pos)&r.mask] = sample
}

func (r *RingBuffer) Read(pos int64) float32 {
    return r.buffer[int(pos)&r.mask]
}

func NewDeterministicLatencyProcessor() *DeterministicLatencyProcessor {
    return &DeterministicLatencyProcessor{
        fixedLatencyMs: 20.0, // 20ms fixed latency (exceeds typical GC pause)
    }
}

func (p *DeterministicLatencyProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    p.sampleRate = sampleRate
    p.fixedLatencySamples = int32(p.fixedLatencyMs * sampleRate / 1000.0)
    
    // Initialize ring buffers with size for 2x the fixed latency
    bufferSize := int(p.fixedLatencySamples * 2)
    p.ringBuffers = make([]RingBuffer, p.numChannels)
    for i := range p.ringBuffers {
        p.ringBuffers[i] = NewRingBuffer(bufferSize)
    }
    
    // Set read position behind write position by fixed latency
    p.writePos = 0
    p.readPos = -int64(p.fixedLatencySamples)
    
    return nil
}

func (p *DeterministicLatencyProcessor) GetLatencySamples() int32 {
    // Always report the same fixed latency
    return p.fixedLatencySamples
}

func (p *DeterministicLatencyProcessor) Process(data *process.Data) error {
    inputs := data.Inputs[0].Buffers
    outputs := data.Outputs[0].Buffers
    numSamples := int(data.NumSamples)
    
    // Process each sample through the ring buffer
    for i := 0; i < numSamples; i++ {
        for ch := 0; ch < len(inputs); ch++ {
            // Write input to ring buffer at write position
            p.ringBuffers[ch].Write(p.writePos, inputs[ch][i])
            
            // Read from ring buffer at read position (delayed)
            if p.readPos >= 0 {
                outputs[ch][i] = p.ringBuffers[ch].Read(p.readPos)
            } else {
                // Still in initial latency period
                outputs[ch][i] = 0
            }
        }
        
        p.writePos++
        p.readPos++
    }
    
    return nil
}
```

### Advanced: Adaptive Fixed Latency

Monitor GC behavior during initialization and set appropriate fixed latency:

```go
type AdaptiveDeterministicProcessor struct {
    *DeterministicLatencyProcessor
    gcProfiler *GCProfiler
}

type GCProfiler struct {
    measurements []time.Duration
    mu          sync.Mutex
}

func (g *GCProfiler) ProfileGC(duration time.Duration) []time.Duration {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    measurements := make([]time.Duration, 0, 10)
    
    // Run several GC cycles to measure typical pause times
    for i := 0; i < 10; i++ {
        // Allocate some memory to trigger realistic GC
        _ = make([]byte, 10*1024*1024) // 10MB
        
        start := time.Now()
        runtime.GC()
        pause := time.Since(start)
        measurements = append(measurements, pause)
        
        time.Sleep(50 * time.Millisecond)
    }
    
    return measurements
}

func (p *AdaptiveDeterministicProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    // Profile GC to determine appropriate latency
    measurements := p.gcProfiler.ProfileGC(100 * time.Millisecond)
    
    // Find 99th percentile GC pause
    var maxPause time.Duration
    for _, pause := range measurements {
        if pause > maxPause {
            maxPause = pause
        }
    }
    
    // Set fixed latency to 2x the worst observed pause
    safetyFactor := 2.0
    p.fixedLatencyMs = float64(maxPause.Milliseconds()) * safetyFactor
    
    // Ensure minimum latency
    if p.fixedLatencyMs < 10.0 {
        p.fixedLatencyMs = 10.0
    }
    
    return p.DeterministicLatencyProcessor.Initialize(sampleRate, maxSamplesPerBlock)
}
```

### Optimized Ring Buffer with Lock-Free Design

For maximum performance, use lock-free ring buffers:

```go
type LockFreeRingBuffer struct {
    buffer    []float32
    size      uint32
    mask      uint32
    cacheLine [64]byte // Prevent false sharing
}

func NewLockFreeRingBuffer(size int) *LockFreeRingBuffer {
    size = nextPowerOf2(size)
    return &LockFreeRingBuffer{
        buffer: make([]float32, size),
        size:   uint32(size),
        mask:   uint32(size - 1),
    }
}

func (r *LockFreeRingBuffer) WriteRead(writePos, readPos uint64, input float32) float32 {
    // Write new sample
    r.buffer[uint32(writePos)&r.mask] = input
    
    // Read delayed sample
    return r.buffer[uint32(readPos)&r.mask]
}

// Optimized processor using SIMD-friendly operations
type OptimizedDeterministicProcessor struct {
    plugin.BaseProcessor
    
    buffers         []*LockFreeRingBuffer
    fixedDelay      int32
    positionCounter uint64
}

func (p *OptimizedDeterministicProcessor) Process(data *process.Data) error {
    numSamples := data.NumSamples
    readPos := p.positionCounter - uint64(p.fixedDelay)
    
    // Process in chunks for better cache usage
    const chunkSize = 64
    for offset := int32(0); offset < numSamples; offset += chunkSize {
        end := min(offset+chunkSize, numSamples)
        
        for ch := 0; ch < len(data.Inputs[0].Buffers); ch++ {
            input := data.Inputs[0].Buffers[ch][offset:end]
            output := data.Outputs[0].Buffers[ch][offset:end]
            
            for i := range input {
                pos := p.positionCounter + uint64(i)
                rPos := readPos + uint64(i)
                output[i] = p.buffers[ch].WriteRead(pos, rPos, input[i])
            }
        }
        
        p.positionCounter += uint64(end - offset)
        readPos += uint64(end - offset)
    }
    
    return nil
}
```

### 5. Non-Blocking GC Strategies

Implement GC during natural processing breaks:

```go
type SmartGCScheduler struct {
    gcManager      *GCManager
    silenceDetector *SilenceDetector
    transportState  *TransportState
}

func (s *SmartGCScheduler) ScheduleGC(ctx context.Context) {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Run GC during optimal times
            if s.shouldRunGC() {
                s.gcManager.MaybeRunGC()
            }
        }
    }
}

func (s *SmartGCScheduler) shouldRunGC() bool {
    // Check if transport is stopped
    if !s.transportState.IsPlaying() {
        return true
    }
    
    // Check if input is silent
    if s.silenceDetector.IsSilent() {
        return true
    }
    
    // Check if we're between process calls
    if !s.gcManager.isProcessing.Load() {
        return true
    }
    
    return false
}
```

## Comparison: Dynamic vs Deterministic Approaches

### Dynamic GC Control (Strategy 1)
**Pros:**
- Lower average latency
- More efficient for simple plugins
- Can achieve near-zero latency when GC is well-managed

**Cons:**
- Complex implementation
- Risk of occasional glitches
- Requires careful tuning

### Deterministic Fixed Latency (Strategy 2)
**Pros:**
- Simple and reliable
- Guaranteed glitch-free operation
- Predictable behavior across all systems
- No GC monitoring overhead

**Cons:**
- Always incurs fixed latency (even when not needed)
- May be overkill for simple plugins

### Recommendation

Use **Strategy 2 (Deterministic Fixed Latency)** for:
- Professional plugins requiring absolute reliability
- Complex processors with high memory allocation
- Plugins that will run on various systems
- When simplicity and predictability are priorities

Use **Strategy 1 (Dynamic GC Control)** for:
- Simple effects with minimal allocations
- When lowest possible latency is critical
- Controlled environments where you can tune GC
- Experimental or specialized use cases

## Best Practices

### 1. Memory Pool Allocation

Pre-allocate memory pools to reduce allocation pressure:

```go
type BufferPool struct {
    pool sync.Pool
    size int
}

func NewBufferPool(bufferSize int) *BufferPool {
    return &BufferPool{
        size: bufferSize,
        pool: sync.Pool{
            New: func() interface{} {
                return make([]float32, bufferSize)
            },
        },
    }
}

func (p *BufferPool) Get() []float32 {
    return p.pool.Get().([]float32)
}

func (p *BufferPool) Put(buf []float32) {
    // Clear the buffer before returning to pool
    for i := range buf {
        buf[i] = 0
    }
    p.pool.Put(buf)
}
```

### 2. Hybrid Approach

Combine manual GC with reduced GC frequency:

```go
func ConfigureGC() {
    // Set high GC percentage to reduce frequency
    runtime.SetGCPercent(400) // GC when heap grows 4x
    
    // Set memory limit to prevent runaway growth
    runtime.SetMemoryLimit(500 * 1024 * 1024) // 500MB limit
}
```

### 3. Monitoring and Metrics

Track GC impact on audio processing:

```go
type GCMetrics struct {
    TotalGCPauses    atomic.Uint64
    TotalGCDuration  atomic.Uint64
    MaxGCDuration    atomic.Uint64
    AudioDropouts    atomic.Uint64
}

func (m *GCMetrics) RecordGC(duration time.Duration) {
    m.TotalGCPauses.Add(1)
    m.TotalGCDuration.Add(uint64(duration))
    
    // Update max duration
    for {
        current := m.MaxGCDuration.Load()
        if uint64(duration) <= current {
            break
        }
        if m.MaxGCDuration.CompareAndSwap(current, uint64(duration)) {
            break
        }
    }
}
```

## Implementation Examples

### Example 1: Dynamic GC Control with VST3

Here's a complete example integrating GC control with VST3 latency reporting:

```go
package main

import (
    "runtime"
    "sync/atomic"
    "time"
    
    "github.com/tmegow/vst3go/pkg/plugin"
    "github.com/tmegow/vst3go/pkg/framework/process"
)

type GCControlledProcessor struct {
    plugin.BaseProcessor
    
    // GC Management
    gcInterval      time.Duration
    lastGCTime      time.Time
    gcLatency       atomic.Int32
    isProcessing    atomic.Bool
    
    // Audio state
    sampleRate      float64
    bufferSize      int
}

func NewGCControlledProcessor() *GCControlledProcessor {
    // Disable automatic GC
    runtime.SetGCPercent(-1)
    
    return &GCControlledProcessor{
        gcInterval: 100 * time.Millisecond, // Run GC every 100ms max
        lastGCTime: time.Now(),
    }
}

func (p *GCControlledProcessor) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    p.sampleRate = sampleRate
    p.bufferSize = int(maxSamplesPerBlock)
    
    // Start GC scheduler
    go p.gcScheduler()
    
    return nil
}

func (p *GCControlledProcessor) GetLatencySamples() int32 {
    // Report worst-case GC latency (e.g., 15ms)
    // This ensures host allocates enough delay compensation
    gcLatencySeconds := 0.015
    return int32(gcLatencySeconds * p.sampleRate)
}

func (p *GCControlledProcessor) Process(data *process.Data) error {
    // Signal that we're processing
    p.isProcessing.Store(true)
    defer p.isProcessing.Store(false)
    
    // Process audio without GC interruption
    inputs := data.Inputs[0].Buffers
    outputs := data.Outputs[0].Buffers
    
    for ch := 0; ch < len(inputs); ch++ {
        copy(outputs[ch][:data.NumSamples], inputs[ch][:data.NumSamples])
    }
    
    return nil
}

func (p *GCControlledProcessor) gcScheduler() {
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()
    
    for range ticker.C {
        // Only run GC when not processing audio
        if !p.isProcessing.Load() && time.Since(p.lastGCTime) > p.gcInterval {
            start := time.Now()
            runtime.GC()
            duration := time.Since(start)
            
            // Update latency measurement
            latencySamples := int32(duration.Seconds() * p.sampleRate)
            p.gcLatency.Store(latencySamples)
            
            p.lastGCTime = time.Now()
        }
    }
}

func main() {
    // Plugin registration code here
}
```

### Example 2: Deterministic Latency with Ring Buffer

Complete implementation of a VST3 plugin using fixed latency approach:

```go
package main

import (
    "math"
    "runtime"
    
    "github.com/tmegow/vst3go/pkg/plugin"
    "github.com/tmegow/vst3go/pkg/framework/process"
    "github.com/tmegow/vst3go/pkg/framework/param"
)

type FixedLatencyPlugin struct {
    plugin.BaseProcessor
    plugin.BaseController
    
    // Ring buffer per channel
    ringBuffers [][]float32
    bufferSize  int
    writeIndex  int
    
    // Fixed latency
    latencySamples int32
    sampleRate     float64
    
    // Parameters
    wetDryMix     float32
    processEnable bool
}

func NewFixedLatencyPlugin() *FixedLatencyPlugin {
    // Allow GC to run normally - we're protected by the buffer
    runtime.SetGCPercent(100)
    
    return &FixedLatencyPlugin{
        wetDryMix:     1.0,
        processEnable: true,
    }
}

func (p *FixedLatencyPlugin) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    p.sampleRate = sampleRate
    
    // Set 15ms fixed latency (typical worst-case GC pause + safety margin)
    latencyMs := 15.0
    p.latencySamples = int32(latencyMs * sampleRate / 1000.0)
    
    // Buffer size must be power of 2 and larger than latency
    p.bufferSize = 1
    for p.bufferSize < int(p.latencySamples*2) {
        p.bufferSize <<= 1
    }
    
    // Pre-allocate ring buffers
    p.ringBuffers = make([][]float32, 2) // Stereo
    for i := range p.ringBuffers {
        p.ringBuffers[i] = make([]float32, p.bufferSize)
    }
    
    p.writeIndex = 0
    
    return nil
}

func (p *FixedLatencyPlugin) GetLatencySamples() int32 {
    return p.latencySamples
}

func (p *FixedLatencyPlugin) Process(data *process.Data) error {
    if !p.processEnable || len(data.Inputs) == 0 || len(data.Outputs) == 0 {
        return nil
    }
    
    numChannels := min(len(data.Inputs[0].Buffers), len(data.Outputs[0].Buffers))
    numSamples := int(data.NumSamples)
    
    // Ensure we have enough ring buffers
    for len(p.ringBuffers) < numChannels {
        p.ringBuffers = append(p.ringBuffers, make([]float32, p.bufferSize))
    }
    
    // Process audio through ring buffers
    for sample := 0; sample < numSamples; sample++ {
        for ch := 0; ch < numChannels; ch++ {
            // Write input to ring buffer
            writeIdx := (p.writeIndex + sample) & (p.bufferSize - 1)
            p.ringBuffers[ch][writeIdx] = data.Inputs[0].Buffers[ch][sample]
            
            // Read delayed output
            readIdx := (p.writeIndex + sample - int(p.latencySamples)) & (p.bufferSize - 1)
            delayed := p.ringBuffers[ch][readIdx]
            
            // Mix dry/wet
            dry := data.Inputs[0].Buffers[ch][sample]
            data.Outputs[0].Buffers[ch][sample] = dry*(1-p.wetDryMix) + delayed*p.wetDryMix
        }
    }
    
    // Update write position
    p.writeIndex = (p.writeIndex + numSamples) & (p.bufferSize - 1)
    
    return nil
}

func (p *FixedLatencyPlugin) GetParameters() []param.Definition {
    return []param.Definition{
        {
            ID:           0,
            Title:        "Mix",
            ShortTitle:   "Mix",
            Units:        "%",
            DefaultValue: 1.0,
            Flags:        param.CanAutomate,
            MinValue:     0.0,
            MaxValue:     1.0,
        },
        {
            ID:           1,
            Title:        "Enable",
            ShortTitle:   "On",
            DefaultValue: 1.0,
            Flags:        param.CanAutomate | param.IsBypass,
            MinValue:     0.0,
            MaxValue:     1.0,
            StepCount:    1,
        },
    }
}

func (p *FixedLatencyPlugin) SetParameter(id param.ID, value param.Value) {
    switch id {
    case 0:
        p.wetDryMix = float32(value)
    case 1:
        p.processEnable = value > 0.5
    }
}

func (p *FixedLatencyPlugin) GetParameter(id param.ID) param.Value {
    switch id {
    case 0:
        return param.Value(p.wetDryMix)
    case 1:
        if p.processEnable {
            return 1.0
        }
        return 0.0
    default:
        return 0.0
    }
}

// Helper function
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func main() {
    plugin.Register(
        &plugin.Info{
            Name:       "Fixed Latency GC Safe",
            Category:   "Fx",
            Vendor:     "Example",
            Version:    "1.0.0",
            SdkVersion: "VST 3.7.4",
        },
        func() plugin.Processor { return NewFixedLatencyPlugin() },
        func() plugin.Controller { return NewFixedLatencyPlugin() },
    )
}
```

## Testing and Validation

1. **Measure GC Impact**: Use `runtime.ReadGCStats()` to monitor GC behavior
2. **Test Under Load**: Simulate various memory allocation patterns
3. **Verify Latency Reporting**: Ensure host receives correct latency values
4. **Monitor Audio Quality**: Check for dropouts or glitches

## Conclusion

By combining manual GC control with VST3's latency reporting mechanism, we can build Go-based audio plugins that:

- Minimize audio dropouts from GC pauses
- Maintain accurate timing through host compensation
- Provide predictable real-time performance
- Scale gracefully with memory usage

The key is to treat GC pauses as a form of processing latency and communicate this to the host, allowing it to compensate appropriately through its plugin delay compensation system.