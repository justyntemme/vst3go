package dsp

import (
	"math"
	"testing"

	"github.com/justyntemme/vst3go/pkg/dsp/gain"
	"github.com/justyntemme/vst3go/pkg/dsp/mix"
	"github.com/justyntemme/vst3go/pkg/dsp/utility"
)

// Common buffer sizes for benchmarking
var benchmarkSizes = []int{64, 128, 256, 512, 1024, 2048}

// BenchmarkGainOperations benchmarks various gain operations
func BenchmarkGainOperations(b *testing.B) {
	for _, size := range benchmarkSizes {
		buffer := make([]float32, size)
		// Fill with test data
		for i := range buffer {
			buffer[i] = float32(math.Sin(float64(i) * 0.1))
		}

		b.Run("ApplyBuffer_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size * 4)) // float32 is 4 bytes
			for i := 0; i < b.N; i++ {
				gain.ApplyBuffer(buffer, 0.5)
			}
		})

		b.Run("DbToLinear", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = gain.DbToLinear(-6.0)
			}
		})

		b.Run("LinearToDb", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = gain.LinearToDb(0.5)
			}
		})
	}
}

// BenchmarkMixOperations benchmarks mixing operations
func BenchmarkMixOperations(b *testing.B) {
	for _, size := range benchmarkSizes {
		dst := make([]float32, size)
		src := make([]float32, size)
		
		// Fill with test data
		for i := range src {
			src[i] = float32(math.Sin(float64(i) * 0.1))
			dst[i] = float32(math.Cos(float64(i) * 0.1))
		}

		b.Run("AddScaled_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				AddScaled(dst, src, 0.5)
			}
		})

		b.Run("Mix_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size * 4))
			src2 := make([]float32, size)
			copy(src2, dst)
			for i := 0; i < b.N; i++ {
				Mix(dst, src, src2, 0.5)
			}
		})
	}
}

// BenchmarkBufferOperations benchmarks basic buffer operations
func BenchmarkBufferOperations(b *testing.B) {
	for _, size := range benchmarkSizes {
		buffer := make([]float32, size)
		src := make([]float32, size)
		
		// Fill with test data
		for i := range src {
			src[i] = float32(math.Sin(float64(i) * 0.1))
		}

		b.Run("Clear_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				Clear(buffer)
			}
		})

		b.Run("Copy_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				Copy(buffer, src)
			}
		})

		b.Run("Scale_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size * 4))
			copy(buffer, src)
			for i := 0; i < b.N; i++ {
				Scale(buffer, 0.5)
			}
		})
	}
}

// BenchmarkParameterScaling benchmarks parameter scaling operations
func BenchmarkParameterScaling(b *testing.B) {
	b.Run("ScaleParameter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = utility.ScaleParameter(0.5, -60.0, 0.0)
		}
	})

	b.Run("ScaleParameterExp", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = utility.ScaleParameterExp(0.5, 20.0, 20000.0)
		}
	})

	b.Run("ClampParameter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = utility.ClampParameter(0.5, 0.0, 1.0)
		}
	})

	// Benchmark parameter smoothing
	smoother := utility.NewSmoothParameter(0.01, 48000)
	smoother.SetTarget(1.0)
	
	b.Run("SmoothParameter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = smoother.Process()
		}
	})
}

// BenchmarkManualVsLibrary compares manual implementations with library functions
func BenchmarkManualVsLibrary(b *testing.B) {
	buffer := make([]float32, 512)
	for i := range buffer {
		buffer[i] = float32(math.Sin(float64(i) * 0.1))
	}

	// Manual gain application
	b.Run("ManualGain", func(b *testing.B) {
		b.SetBytes(512 * 4)
		for i := 0; i < b.N; i++ {
			g := float32(0.5)
			for j := range buffer {
				buffer[j] *= g
			}
		}
	})

	// Library gain application
	b.Run("LibraryGain", func(b *testing.B) {
		b.SetBytes(512 * 4)
		for i := 0; i < b.N; i++ {
			gain.ApplyBuffer(buffer, 0.5)
		}
	})

	// Manual dB conversion
	b.Run("ManualDbToLinear", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = math.Pow(10.0, -6.0/20.0)
		}
	})

	// Library dB conversion
	b.Run("LibraryDbToLinear", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = gain.DbToLinear(-6.0)
		}
	})
}

// BenchmarkAllocationCheck verifies no allocations in critical paths
func BenchmarkAllocationCheck(b *testing.B) {
	buffer := make([]float32, 512)
	src := make([]float32, 512)
	
	// These operations should have zero allocations
	benchmarks := []struct {
		name string
		fn   func()
	}{
		{"GainApply", func() {
			gain.ApplyBuffer(buffer, 0.5)
		}},
		{"BufferCopy", func() {
			Copy(buffer, src)
		}},
		{"BufferClear", func() {
			Clear(buffer)
		}},
		{"BufferScale", func() {
			Scale(buffer, 0.5)
		}},
		{"AddScaled", func() {
			AddScaled(buffer, src, 0.5)
		}},
		{"ParameterScale", func() {
			_ = utility.ScaleParameter(0.5, -60.0, 0.0)
		}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name+"_Allocs", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				bm.fn()
			}
			// The test will show allocations per operation
			// We expect 0 allocs/op for all these functions
		})
	}
}