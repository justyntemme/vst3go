package debug_test

import (
	"github.com/justyntemme/vst3go/pkg/dsp/debug"
	"github.com/justyntemme/vst3go/pkg/dsp/gain"
)

// Example of using debug utilities in an audio processor
func ExampleCheckAllocation() {
	// Enable tracking in debug builds
	debug.EnableAllocationTracking()
	defer debug.DisableAllocationTracking()
	
	// Pre-allocate buffers
	input := make([]float32, 512)
	output := make([]float32, 512)
	
	// In your ProcessAudio function
	processAudio := func() {
		// Check that buffers are pre-allocated
		debug.CheckAllocation(input, "input")
		debug.CheckAllocation(output, "output")
		
		// Track frame allocations
		debug.StartFrame()
		
		// Do actual processing
		copy(output, input)
		gain.ApplyBuffer(output, 0.5)
		
		// Check if any allocations occurred
		allocs, bytes := debug.EndFrame()
		if allocs > 0 {
			// Log warning about allocations in audio path
			_ = allocs
			_ = bytes
		}
	}
	
	processAudio()
}

// Example of verifying buffer reuse across process calls
func ExampleVerifyBufferReuse() {
	debug.EnableAllocationTracking()
	defer debug.DisableAllocationTracking()
	
	// Processor with internal buffer
	type Processor struct {
		buffer    []float32
		bufferPtr uintptr
	}
	
	p := &Processor{
		buffer: make([]float32, 256),
	}
	
	// In ProcessAudio, verify buffer hasn't been reallocated
	process := func() {
		p.bufferPtr = debug.VerifyBufferReuse(p.buffer, "internal_buffer", p.bufferPtr)
		// Process using the buffer...
	}
	
	// Multiple process calls should reuse the same buffer
	process()
	process()
	process()
}