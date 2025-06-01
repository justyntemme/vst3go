package buffer

import (
	"errors"
	"math"
	"sync/atomic"
	"time"
)

// WriteAheadBuffer implements a lock-free circular buffer with enforced write-ahead distance
// for absorbing garbage collection pauses in real-time audio processing.
type WriteAheadBuffer struct {
	data           []float32
	readPos        uint64
	writePos       uint64
	size           uint32
	mask           uint32
	latencySamples uint32
	sampleRate     float64
	channels       int
	
	// Statistics for monitoring
	underruns   uint64
	overruns    uint64
	adjustments uint64
}

// BufferStats provides health monitoring information
type BufferStats struct {
	Underruns      uint64
	Overruns       uint64
	Adjustments    uint64
	FillPercentage float32
	CurrentLatency time.Duration
}

// NewWriteAheadBuffer creates a new write-ahead buffer with the specified sample rate and channel count
func NewWriteAheadBuffer(sampleRate float64, channels int) *WriteAheadBuffer {
	// Calculate 50ms latency in samples (per channel)
	latencyMs := 50.0
	latencySamplesPerChannel := uint32(math.Round(latencyMs * sampleRate / 1000.0))
	
	// For interleaved audio, multiply by channels
	latencySamples := latencySamplesPerChannel * uint32(channels)
	
	// Buffer size is 4x the latency for safety, rounded up to power of 2
	minSize := latencySamples * 4
	size := nextPowerOf2(minSize)
	mask := size - 1
	
	buf := &WriteAheadBuffer{
		data:           make([]float32, size),
		size:           size,
		mask:           mask,
		latencySamples: latencySamples,
		sampleRate:     sampleRate,
		channels:       channels,
		readPos:        0,
		writePos:       uint64(latencySamples), // Start write position ahead
	}
	
	return buf
}

// Write adds samples to the buffer
func (buf *WriteAheadBuffer) Write(samples []float32) error {
	if len(samples) == 0 {
		return nil
	}
	
	// Load positions atomically
	writePos := atomic.LoadUint64(&buf.writePos)
	readPos := atomic.LoadUint64(&buf.readPos)
	
	// Check available space
	available := buf.availableSpace(readPos, writePos)
	if available < uint32(len(samples)) {
		atomic.AddUint64(&buf.overruns, 1)
		return errors.New("buffer overrun: not enough space available")
	}
	
	// Copy samples with wrap-around handling
	remaining := len(samples)
	srcOffset := 0
	
	for remaining > 0 {
		dstIdx := uint32(writePos) & buf.mask
		copySize := remaining
		
		// Handle wrap-around
		if dstIdx+uint32(copySize) > buf.size {
			copySize = int(buf.size - dstIdx)
		}
		
		copy(buf.data[dstIdx:dstIdx+uint32(copySize)], samples[srcOffset:srcOffset+copySize])
		
		srcOffset += copySize
		remaining -= copySize
		writePos += uint64(copySize)
	}
	
	// Update write position atomically
	atomic.StoreUint64(&buf.writePos, writePos)
	
	return nil
}

// Read retrieves samples from the buffer, enforcing the minimum latency
func (buf *WriteAheadBuffer) Read(output []float32) int {
	if len(output) == 0 {
		return 0
	}
	
	// Enforce minimum delay before reading
	buf.maintainDelay()
	
	// Load positions atomically
	readPos := atomic.LoadUint64(&buf.readPos)
	writePos := atomic.LoadUint64(&buf.writePos)
	
	// Check available data
	available := buf.availableData(readPos, writePos)
	toRead := len(output)
	if available < uint32(toRead) {
		toRead = int(available)
		atomic.AddUint64(&buf.underruns, 1)
	}
	
	// Copy samples with wrap-around handling
	remaining := toRead
	dstOffset := 0
	
	for remaining > 0 {
		srcIdx := uint32(readPos) & buf.mask
		copySize := remaining
		
		// Handle wrap-around
		if srcIdx+uint32(copySize) > buf.size {
			copySize = int(buf.size - srcIdx)
		}
		
		copy(output[dstOffset:dstOffset+copySize], buf.data[srcIdx:srcIdx+uint32(copySize)])
		
		dstOffset += copySize
		remaining -= copySize
		readPos += uint64(copySize)
	}
	
	// Update read position atomically
	atomic.StoreUint64(&buf.readPos, readPos)
	
	// Zero any unfilled portion
	for i := toRead; i < len(output); i++ {
		output[i] = 0
	}
	
	return toRead
}

// maintainDelay ensures the read position is at least latencySamples behind write position
func (buf *WriteAheadBuffer) maintainDelay() {
	for {
		readPos := atomic.LoadUint64(&buf.readPos)
		writePos := atomic.LoadUint64(&buf.writePos)
		
		// Calculate current gap
		gap := writePos - readPos
		
		// If gap is sufficient, we're done
		if gap >= uint64(buf.latencySamples) {
			return
		}
		
		// Adjust read position to maintain minimum gap
		newReadPos := writePos - uint64(buf.latencySamples)
		
		// Try to update atomically
		if atomic.CompareAndSwapUint64(&buf.readPos, readPos, newReadPos) {
			atomic.AddUint64(&buf.adjustments, 1)
			return
		}
		// If CAS failed, another thread updated it, so retry
	}
}

// GetBufferHealth returns current buffer statistics
func (buf *WriteAheadBuffer) GetBufferHealth() BufferStats {
	readPos := atomic.LoadUint64(&buf.readPos)
	writePos := atomic.LoadUint64(&buf.writePos)
	
	available := buf.availableData(readPos, writePos)
	fillPercentage := float32(available) / float32(buf.size) * 100.0
	
	currentLatencySamples := writePos - readPos
	// Convert to seconds - for interleaved audio, divide by channels to get actual frame count
	currentLatencyFrames := float64(currentLatencySamples) / float64(buf.channels)
	currentLatency := time.Duration(currentLatencyFrames / buf.sampleRate * float64(time.Second))
	
	return BufferStats{
		Underruns:      atomic.LoadUint64(&buf.underruns),
		Overruns:       atomic.LoadUint64(&buf.overruns),
		Adjustments:    atomic.LoadUint64(&buf.adjustments),
		FillPercentage: fillPercentage,
		CurrentLatency: currentLatency,
	}
}

// GetCurrentLatency returns the current buffer latency
func (buf *WriteAheadBuffer) GetCurrentLatency() time.Duration {
	readPos := atomic.LoadUint64(&buf.readPos)
	writePos := atomic.LoadUint64(&buf.writePos)
	
	latencySamples := writePos - readPos
	latencyFrames := float64(latencySamples) / float64(buf.channels)
	return time.Duration(latencyFrames / buf.sampleRate * float64(time.Second))
}

// GetBufferUtilization returns the current buffer fill percentage
func (buf *WriteAheadBuffer) GetBufferUtilization() float32 {
	readPos := atomic.LoadUint64(&buf.readPos)
	writePos := atomic.LoadUint64(&buf.writePos)
	
	available := buf.availableData(readPos, writePos)
	return float32(available) / float32(buf.size)
}

// Reset clears the buffer and resets positions
func (buf *WriteAheadBuffer) Reset() {
	// Clear buffer
	for i := range buf.data {
		buf.data[i] = 0
	}
	
	// Reset positions with write ahead
	atomic.StoreUint64(&buf.readPos, 0)
	atomic.StoreUint64(&buf.writePos, uint64(buf.latencySamples))
	
	// Reset statistics
	atomic.StoreUint64(&buf.underruns, 0)
	atomic.StoreUint64(&buf.overruns, 0)
	atomic.StoreUint64(&buf.adjustments, 0)
}

// availableSpace calculates how many samples can be written
func (buf *WriteAheadBuffer) availableSpace(readPos, writePos uint64) uint32 {
	// The buffer is full when write position is one full buffer ahead of read
	used := writePos - readPos
	if used >= uint64(buf.size) {
		return 0
	}
	return buf.size - uint32(used)
}

// availableData calculates how many samples can be read
func (buf *WriteAheadBuffer) availableData(readPos, writePos uint64) uint32 {
	if writePos < readPos {
		return 0 // Should never happen with proper usage
	}
	available := writePos - readPos
	if available > uint64(buf.size) {
		return buf.size
	}
	return uint32(available)
}

// nextPowerOf2 rounds up to the next power of 2
func nextPowerOf2(n uint32) uint32 {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}