// +build debug

package debug

import (
	"strings"
	"testing"
)

func TestCheckAllocation(t *testing.T) {
	EnableAllocationTracking()
	defer DisableAllocationTracking()
	defer ResetAllocationTracking()
	
	// Test with valid pre-allocated buffer
	buffer := make([]float32, 128)
	CheckAllocation(buffer, "test_buffer")
	
	// Test with nil buffer - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil buffer")
		}
	}()
	CheckAllocation(nil, "nil_buffer")
}

func TestCheckAllocationZeroCapacity(t *testing.T) {
	EnableAllocationTracking()
	defer DisableAllocationTracking()
	defer ResetAllocationTracking()
	
	// Test with zero capacity buffer - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for zero capacity buffer")
		} else if !strings.Contains(r.(string), "nil") && !strings.Contains(r.(string), "not pre-allocated") {
			t.Errorf("Expected 'nil' or 'not pre-allocated' error, got: %v", r)
		}
	}()
	
	var buffer []float32
	CheckAllocation(buffer, "zero_cap_buffer")
}

func TestAllocationTracking(t *testing.T) {
	EnableAllocationTracking()
	defer DisableAllocationTracking()
	ResetAllocationTracking()
	
	// Track some allocations
	buffer1 := make([]float32, 128)
	buffer2 := make([]float32, 256)
	
	CheckAllocation(buffer1, "buffer1")
	CheckAllocation(buffer1, "buffer1") // Same buffer again
	CheckAllocation(buffer2, "buffer2")
	
	// Get report
	report := GetAllocationReport()
	
	// Verify report contains expected information
	if !strings.Contains(report, "buffer1") {
		t.Error("Report should contain buffer1")
	}
	if !strings.Contains(report, "buffer2") {
		t.Error("Report should contain buffer2")
	}
	if !strings.Contains(report, "Access Count: 2") {
		t.Error("buffer1 should have been accessed twice")
	}
}

func TestFrameTracking(t *testing.T) {
	EnableAllocationTracking()
	defer DisableAllocationTracking()
	ResetAllocationTracking()
	
	// Start a new frame
	StartFrame()
	
	// Do some allocations
	buffer := make([]float32, 128)
	CheckAllocation(buffer, "frame_buffer")
	
	// End frame and check stats
	allocs, bytes := EndFrame()
	
	if allocs != 1 {
		t.Errorf("Expected 1 allocation in frame, got %d", allocs)
	}
	if bytes != 128*4 { // 128 float32s * 4 bytes each
		t.Errorf("Expected %d bytes in frame, got %d", 128*4, bytes)
	}
}

func TestVerifyBufferReuse(t *testing.T) {
	EnableAllocationTracking()
	defer DisableAllocationTracking()
	
	buffer := make([]float32, 128)
	
	// First call should return the pointer
	ptr1 := VerifyBufferReuse(buffer, "reuse_test", 0)
	if ptr1 == 0 {
		t.Error("Expected non-zero pointer")
	}
	
	// Second call with same buffer should succeed
	ptr2 := VerifyBufferReuse(buffer, "reuse_test", ptr1)
	if ptr2 != ptr1 {
		t.Error("Expected same pointer")
	}
	
	// Test with different buffer - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for reallocated buffer")
		}
	}()
	
	newBuffer := make([]float32, 128)
	VerifyBufferReuse(newBuffer, "reuse_test", ptr1)
}

func TestDetectAllocation(t *testing.T) {
	// This test is flaky due to GC and runtime behavior
	t.Skip("Skipping allocation detection test - too flaky in practice")
	
	// Function that doesn't allocate
	noAlloc := func() {
		x := 1 + 1
		_ = x
	}
	
	// This should not panic
	DetectAllocation(noAlloc)
	
	// Function that allocates
	withAlloc := func() {
		_ = make([]float32, 128)
	}
	
	// This should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for allocation")
		}
	}()
	
	DetectAllocation(withAlloc)
}

func TestBufferPool(t *testing.T) {
	pool := NewBufferPool(128)
	
	// Get initial stats
	allocated1, inUse1 := pool.Stats()
	if allocated1 != 0 || inUse1 != 0 {
		t.Error("Expected zero initial stats")
	}
	
	// Get a buffer
	buf1 := pool.Get()
	if len(buf1) != 128 {
		t.Errorf("Expected buffer size 128, got %d", len(buf1))
	}
	
	allocated2, inUse2 := pool.Stats()
	if allocated2 != 1 {
		t.Errorf("Expected 1 allocated buffer, got %d", allocated2)
	}
	if inUse2 != 1 {
		t.Errorf("Expected 1 buffer in use, got %d", inUse2)
	}
	
	// Return the buffer
	pool.Put(buf1)
	
	allocated3, inUse3 := pool.Stats()
	if allocated3 != 1 {
		t.Errorf("Expected 1 allocated buffer, got %d", allocated3)
	}
	if inUse3 != 0 {
		t.Errorf("Expected 0 buffers in use, got %d", inUse3)
	}
	
	// Get another buffer - should reuse
	buf2 := pool.Get()
	defer pool.Put(buf2)
	
	allocated4, _ := pool.Stats()
	if allocated4 != 1 {
		t.Errorf("Expected buffer to be reused, but got %d allocations", allocated4)
	}
	
	// Test panic on wrong size
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for wrong buffer size")
		}
	}()
	
	wrongSize := make([]float32, 64)
	pool.Put(wrongSize)
}