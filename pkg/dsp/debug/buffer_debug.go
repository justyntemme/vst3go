// +build debug

// Package debug provides debugging utilities for DSP development.
// These utilities are only included when building with the 'debug' build tag.
package debug

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// AllocationTracker tracks buffer allocations to help detect unwanted allocations
// in the audio processing path.
type AllocationTracker struct {
	allocations   map[string]*AllocationInfo
	mu            sync.RWMutex
	enabled       atomic.Bool
	totalAllocs   atomic.Uint64
	totalBytes    atomic.Uint64
	frameAllocs   atomic.Uint64
	frameBytes    atomic.Uint64
}

// AllocationInfo contains information about a buffer allocation
type AllocationInfo struct {
	Name       string
	Size       int
	Capacity   int
	StackTrace string
	Count      uint64
	TotalBytes uint64
}

var (
	globalTracker = &AllocationTracker{
		allocations: make(map[string]*AllocationInfo),
	}
)

// EnableAllocationTracking enables global allocation tracking
func EnableAllocationTracking() {
	globalTracker.enabled.Store(true)
}

// DisableAllocationTracking disables global allocation tracking
func DisableAllocationTracking() {
	globalTracker.enabled.Store(false)
}

// ResetAllocationTracking resets all allocation statistics
func ResetAllocationTracking() {
	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()
	
	globalTracker.allocations = make(map[string]*AllocationInfo)
	globalTracker.totalAllocs.Store(0)
	globalTracker.totalBytes.Store(0)
	globalTracker.frameAllocs.Store(0)
	globalTracker.frameBytes.Store(0)
}

// CheckAllocation verifies that a buffer is pre-allocated and tracks its usage.
// This should be called at the beginning of audio processing functions.
func CheckAllocation(buffer []float32, name string) {
	if !globalTracker.enabled.Load() {
		return
	}
	
	if buffer == nil {
		panic(fmt.Sprintf("Buffer %s is nil", name))
	}
	
	if cap(buffer) == 0 {
		panic(fmt.Sprintf("Buffer %s is not pre-allocated (capacity is 0)", name))
	}
	
	// Track the allocation
	trackAllocation(name, len(buffer), cap(buffer))
}

// CheckAllocation64 is the float64 version of CheckAllocation
func CheckAllocation64(buffer []float64, name string) {
	if !globalTracker.enabled.Load() {
		return
	}
	
	if buffer == nil {
		panic(fmt.Sprintf("Buffer %s is nil", name))
	}
	
	if cap(buffer) == 0 {
		panic(fmt.Sprintf("Buffer %s is not pre-allocated (capacity is 0)", name))
	}
	
	// Track the allocation (size in bytes is double for float64)
	trackAllocation(name, len(buffer)*2, cap(buffer)*2)
}

// trackAllocation records buffer usage information
func trackAllocation(name string, size, capacity int) {
	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()
	
	info, exists := globalTracker.allocations[name]
	if !exists {
		// Capture stack trace for new allocations
		buf := make([]byte, 1024)
		n := runtime.Stack(buf, false)
		
		info = &AllocationInfo{
			Name:       name,
			Size:       size,
			Capacity:   capacity,
			StackTrace: string(buf[:n]),
		}
		globalTracker.allocations[name] = info
	}
	
	info.Count++
	info.TotalBytes += uint64(size * 4) // float32 is 4 bytes
	
	// Update global counters
	globalTracker.totalAllocs.Add(1)
	globalTracker.totalBytes.Add(uint64(size * 4))
	globalTracker.frameAllocs.Add(1)
	globalTracker.frameBytes.Add(uint64(size * 4))
}

// StartFrame marks the beginning of a new audio processing frame
func StartFrame() {
	globalTracker.frameAllocs.Store(0)
	globalTracker.frameBytes.Store(0)
}

// EndFrame marks the end of an audio processing frame and returns allocation stats
func EndFrame() (allocations uint64, bytes uint64) {
	return globalTracker.frameAllocs.Load(), globalTracker.frameBytes.Load()
}

// GetAllocationReport returns a detailed report of all tracked allocations
func GetAllocationReport() string {
	globalTracker.mu.RLock()
	defer globalTracker.mu.RUnlock()
	
	report := fmt.Sprintf("=== Buffer Allocation Report ===\n")
	report += fmt.Sprintf("Total Allocations: %d\n", globalTracker.totalAllocs.Load())
	report += fmt.Sprintf("Total Bytes: %d\n", globalTracker.totalBytes.Load())
	report += fmt.Sprintf("\nDetailed Allocations:\n")
	
	for name, info := range globalTracker.allocations {
		report += fmt.Sprintf("\nBuffer: %s\n", name)
		report += fmt.Sprintf("  Size: %d, Capacity: %d\n", info.Size, info.Capacity)
		report += fmt.Sprintf("  Access Count: %d\n", info.Count)
		report += fmt.Sprintf("  Total Bytes: %d\n", info.TotalBytes)
		report += fmt.Sprintf("  Stack Trace:\n%s\n", info.StackTrace)
	}
	
	return report
}

// VerifyBufferReuse checks that a buffer is being reused across multiple calls
func VerifyBufferReuse(buffer []float32, name string, expectedPtr uintptr) uintptr {
	if !globalTracker.enabled.Load() {
		return 0
	}
	
	ptr := uintptr(0)
	if len(buffer) > 0 {
		ptr = uintptr(unsafe.Pointer(&buffer[0]))
	}
	
	if expectedPtr != 0 && ptr != expectedPtr {
		panic(fmt.Sprintf("Buffer %s was reallocated! Expected ptr %x, got %x", 
			name, expectedPtr, ptr))
	}
	
	return ptr
}

// DetectAllocation runs a function and panics if any allocations occur
func DetectAllocation(fn func()) {
	var m1, m2 runtime.MemStats
	
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	fn()
	
	runtime.ReadMemStats(&m2)
	
	if m2.Alloc > m1.Alloc {
		panic(fmt.Sprintf("Allocation detected! %d bytes allocated", m2.Alloc-m1.Alloc))
	}
}

// BufferPool provides a debug-enabled buffer pool for testing
type BufferPool struct {
	pool      sync.Pool
	size      int
	allocated atomic.Int32
	inUse     atomic.Int32
}

// NewBufferPool creates a new debug buffer pool
func NewBufferPool(size int) *BufferPool {
	bp := &BufferPool{
		size: size,
	}
	bp.pool.New = func() interface{} {
		bp.allocated.Add(1)
		return make([]float32, size)
	}
	return bp
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() []float32 {
	bp.inUse.Add(1)
	return bp.pool.Get().([]float32)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []float32) {
	if len(buf) != bp.size {
		panic(fmt.Sprintf("Buffer size mismatch: expected %d, got %d", bp.size, len(buf)))
	}
	bp.inUse.Add(-1)
	bp.pool.Put(buf)
}

// Stats returns pool statistics
func (bp *BufferPool) Stats() (allocated, inUse int32) {
	return bp.allocated.Load(), bp.inUse.Load()
}