// +build !debug

// Package debug provides debugging utilities for DSP development.
// This file contains no-op implementations when building without the 'debug' tag.
package debug

// EnableAllocationTracking is a no-op when not in debug mode
func EnableAllocationTracking() {}

// DisableAllocationTracking is a no-op when not in debug mode
func DisableAllocationTracking() {}

// ResetAllocationTracking is a no-op when not in debug mode
func ResetAllocationTracking() {}

// CheckAllocation is a no-op when not in debug mode
func CheckAllocation(buffer []float32, name string) {}

// CheckAllocation64 is a no-op when not in debug mode
func CheckAllocation64(buffer []float64, name string) {}

// StartFrame is a no-op when not in debug mode
func StartFrame() {}

// EndFrame is a no-op when not in debug mode
func EndFrame() (allocations uint64, bytes uint64) {
	return 0, 0
}

// GetAllocationReport returns empty string when not in debug mode
func GetAllocationReport() string {
	return ""
}

// VerifyBufferReuse is a no-op when not in debug mode
func VerifyBufferReuse(buffer []float32, name string, expectedPtr uintptr) uintptr {
	return 0
}

// DetectAllocation is a no-op when not in debug mode
func DetectAllocation(fn func()) {
	fn()
}