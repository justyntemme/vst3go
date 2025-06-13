// Package debug provides debugging utilities for DSP development.
//
// This package contains tools to help identify performance issues and
// unwanted allocations in audio processing code. The utilities are
// only active when building with the 'debug' build tag.
//
// Usage:
//
//	// Build with debug support
//	go build -tags debug
//
//	// In your audio processing code
//	func ProcessAudio(input, output []float32) {
//	    debug.CheckAllocation(input, "input")
//	    debug.CheckAllocation(output, "output")
//	    
//	    debug.StartFrame()
//	    // ... audio processing ...
//	    allocs, bytes := debug.EndFrame()
//	    if allocs > 0 {
//	        log.Printf("Warning: %d allocations (%d bytes) in audio frame", allocs, bytes)
//	    }
//	}
//
// The package provides:
//   - Allocation tracking to detect buffer allocations in the audio path
//   - Frame-based statistics to monitor per-frame allocations
//   - Buffer reuse verification to ensure buffers aren't reallocated
//   - Detailed allocation reports with stack traces
//   - A debug buffer pool for testing
//
// When building without the 'debug' tag, all functions become no-ops
// with zero overhead.
package debug