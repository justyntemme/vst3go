# Ring Buffer Explanation for Audio Processing

## What is a Ring Buffer?

A ring buffer (also called circular buffer) is a fixed-size buffer that wraps around - when you reach the end, you continue from the beginning. Imagine it as a circle of memory slots where you continuously write and read data.

```
Traditional Buffer (Linear):
[0][1][2][3][4][5][6][7] → End (need to shift or reallocate)

Ring Buffer (Circular):
     Write →
[0][1][2][3]
[7]      [4]
  [6][5]
    ← Read
```

## Why Ring Buffers for Audio?

1. **Constant Memory**: No allocations during audio processing
2. **Fast Access**: O(1) read/write operations
3. **Natural Delay Line**: Perfect for implementing delays and latency compensation
4. **Lock-Free Potential**: Can be implemented without locks for real-time safety

## How It Works

### Basic Concept

```go
// Simplified ring buffer
type RingBuffer struct {
    data  []float32
    size  int
    write int  // Write position
    read  int  // Read position
}

// Write a sample
func (r *RingBuffer) Write(sample float32) {
    r.data[r.write] = sample
    r.write = (r.write + 1) % r.size  // Wrap around
}

// Read a sample
func (r *RingBuffer) Read() float32 {
    sample := r.data[r.read]
    r.read = (r.read + 1) % r.size  // Wrap around
    return sample
}
```

### For Latency Compensation

In our use case, we maintain a fixed distance between read and write positions:

```
Buffer with 4-sample delay:
Time 0: W[0][ ][ ][ ]  (Write at 0)
Time 1: [ ]W[1][ ][ ]  (Write at 1)
Time 2: [ ][ ]W[2][ ]  (Write at 2)
Time 3: [ ][ ][ ]W[3]  (Write at 3)
Time 4: R[4][ ][ ][ ]W (Write at 0, Read at 0 - output first sample)
        ↑             ↑
        Read          Write (wrapped around)
```

## Integration with vst3go Library

### 1. Create Ring Buffer Module

First, let's add a ring buffer implementation to the DSP library:

```go
// pkg/dsp/buffer/ringbuffer.go
package buffer

import (
    "math/bits"
    "sync/atomic"
)

// RingBuffer provides a lock-free circular buffer for audio processing
type RingBuffer struct {
    buffer []float32
    size   uint32
    mask   uint32  // For fast modulo (size must be power of 2)
}

// New creates a ring buffer with the specified size (rounded up to power of 2)
func NewRingBuffer(size int) *RingBuffer {
    // Round up to next power of 2 for fast bitwise operations
    size = int(nextPowerOf2(uint32(size)))
    
    return &RingBuffer{
        buffer: make([]float32, size),
        size:   uint32(size),
        mask:   uint32(size - 1),
    }
}

// WriteRead performs an atomic write and read operation
func (r *RingBuffer) WriteRead(writePos, readPos uint64, input float32) float32 {
    // Write new sample
    r.buffer[uint32(writePos)&r.mask] = input
    
    // Read delayed sample
    return r.buffer[uint32(readPos)&r.mask]
}

// Clear zeroes the buffer
func (r *RingBuffer) Clear() {
    for i := range r.buffer {
        r.buffer[i] = 0
    }
}

func nextPowerOf2(n uint32) uint32 {
    if n == 0 {
        return 1
    }
    return 1 << (32 - bits.LeadingZeros32(n-1))
}
```

### 2. Create Latency Compensation Processor

```go
// pkg/dsp/latency/compensator.go
package latency

import (
    "github.com/tmegow/vst3go/pkg/dsp/buffer"
)

// Compensator provides deterministic latency for GC protection
type Compensator struct {
    buffers        []*buffer.RingBuffer
    numChannels    int
    latencySamples int32
    position       uint64
}

// NewCompensator creates a latency compensator
func NewCompensator(numChannels int, latencySamples int32) *Compensator {
    c := &Compensator{
        numChannels:    numChannels,
        latencySamples: latencySamples,
        buffers:        make([]*buffer.RingBuffer, numChannels),
    }
    
    // Create ring buffer for each channel
    // Size = 2x latency for safety
    bufferSize := int(latencySamples * 2)
    for i := 0; i < numChannels; i++ {
        c.buffers[i] = buffer.NewRingBuffer(bufferSize)
    }
    
    return c
}

// Process applies the fixed latency to audio
func (c *Compensator) Process(inputs, outputs [][]float32, numSamples int) {
    readPos := c.position - uint64(c.latencySamples)
    
    for sample := 0; sample < numSamples; sample++ {
        for ch := 0; ch < c.numChannels; ch++ {
            outputs[ch][sample] = c.buffers[ch].WriteRead(
                c.position+uint64(sample),
                readPos+uint64(sample),
                inputs[ch][sample],
            )
        }
    }
    
    c.position += uint64(numSamples)
}

// GetLatencySamples returns the fixed latency in samples
func (c *Compensator) GetLatencySamples() int32 {
    return c.latencySamples
}

// Reset clears all buffers and resets position
func (c *Compensator) Reset() {
    c.position = 0
    for _, buf := range c.buffers {
        buf.Clear()
    }
}
```

### 3. Integration with Existing DSP Modules

The ring buffer pattern is already used in several places in vst3go:

#### Delay Module Integration
The delay module could be refactored to use our ring buffer:

```go
// pkg/dsp/delay/delay.go
type Delay struct {
    buffer      *buffer.RingBuffer
    delaySamples int32
    position    uint64
    feedback    float32
    mix         float32
}

func (d *Delay) Process(input float32) float32 {
    readPos := d.position - uint64(d.delaySamples)
    
    // Read delayed sample
    delayed := d.buffer.ReadAt(readPos)
    
    // Write input + feedback
    d.buffer.WriteAt(d.position, input + delayed*d.feedback)
    
    d.position++
    
    // Mix dry/wet
    return input*(1-d.mix) + delayed*d.mix
}
```

#### Plugin Integration Example

Here's how a plugin would use the latency compensator:

```go
// examples/gc-safe-reverb/main.go
package main

import (
    "github.com/tmegow/vst3go/pkg/plugin"
    "github.com/tmegow/vst3go/pkg/dsp/latency"
    "github.com/tmegow/vst3go/pkg/dsp/reverb"
    "github.com/tmegow/vst3go/pkg/framework/process"
)

type GCSafeReverb struct {
    plugin.BaseProcessor
    
    compensator *latency.Compensator
    reverb      *reverb.FreeVerb
    
    // Processing buffers
    tempBuffer  [][]float32
}

func NewGCSafeReverb() *GCSafeReverb {
    return &GCSafeReverb{
        reverb: reverb.NewFreeVerb(),
    }
}

func (p *GCSafeReverb) Initialize(sampleRate float64, maxSamplesPerBlock int32) error {
    // Initialize reverb
    p.reverb.SetSampleRate(sampleRate)
    
    // Create latency compensator with 15ms buffer
    latencyMs := 15.0
    latencySamples := int32(latencyMs * sampleRate / 1000.0)
    p.compensator = latency.NewCompensator(2, latencySamples)
    
    // Allocate temp buffers
    p.tempBuffer = make([][]float32, 2)
    for i := range p.tempBuffer {
        p.tempBuffer[i] = make([]float32, maxSamplesPerBlock)
    }
    
    return nil
}

func (p *GCSafeReverb) GetLatencySamples() int32 {
    // Report the compensator's latency
    return p.compensator.GetLatencySamples()
}

func (p *GCSafeReverb) Process(data *process.Data) error {
    numSamples := int(data.NumSamples)
    inputs := data.Inputs[0].Buffers
    outputs := data.Outputs[0].Buffers
    
    // Process reverb into temp buffer
    for ch := 0; ch < 2; ch++ {
        for i := 0; i < numSamples; i++ {
            if ch == 0 {
                outL, outR := p.reverb.Process(inputs[0][i], inputs[1][i])
                p.tempBuffer[0][i] = outL
                p.tempBuffer[1][i] = outR
            }
        }
    }
    
    // Apply latency compensation
    p.compensator.Process(p.tempBuffer, outputs, numSamples)
    
    return nil
}
```

## Benefits in vst3go Context

1. **GC Protection**: The ring buffer provides a cushion that absorbs GC pauses
2. **Memory Efficiency**: Pre-allocated buffers mean no allocations during processing
3. **Consistent API**: Fits naturally with the existing DSP module design
4. **Reusable**: Can be used for delays, lookahead, smoothing, etc.

## Performance Considerations

### Power-of-2 Sizing
```go
// Fast modulo using bitwise AND
position & mask  // Instead of position % size
```

### Cache-Friendly Access
```go
// Process in chunks for better cache utilization
const chunkSize = 64  // Typical cache line size

for offset := 0; offset < numSamples; offset += chunkSize {
    // Process chunk
}
```

### Memory Alignment
```go
// Ensure buffer is aligned for SIMD operations
type AlignedRingBuffer struct {
    _ [0]func() // Compiler hint for alignment
    buffer []float32
    // ... other fields
}
```

## Common Use Cases in Audio

1. **Delay Effects**: Echo, chorus, flanger
2. **Lookahead Processing**: Limiters, compressors
3. **Latency Compensation**: As we're using for GC
4. **Sample Rate Conversion**: Interpolation buffers
5. **FFT Processing**: Overlap-add buffers

## Testing the Implementation

```go
// pkg/dsp/buffer/ringbuffer_test.go
func TestRingBuffer(t *testing.T) {
    rb := NewRingBuffer(8)
    
    // Write samples
    for i := 0; i < 4; i++ {
        rb.WriteAt(uint64(i), float32(i))
    }
    
    // Read with 4-sample delay
    for i := 4; i < 8; i++ {
        written := rb.WriteRead(uint64(i), uint64(i-4), float32(i))
        expected := float32(i - 4)
        if written != expected {
            t.Errorf("Expected %f, got %f", expected, written)
        }
    }
}
```

## Conclusion

Ring buffers are fundamental to audio processing and integrate naturally with vst3go:

1. They provide the constant-time, allocation-free operations needed for real-time audio
2. They solve the GC latency problem by creating a predictable delay buffer
3. They're already conceptually used in delay-based effects
4. They fit perfectly with the framework's modular DSP design

The implementation can be added as a core DSP primitive that other modules can build upon, making the entire library more robust for real-time processing.