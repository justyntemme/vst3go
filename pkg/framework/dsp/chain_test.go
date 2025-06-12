package dsp

import (
	"math"
	"testing"
)

// TestProcessor is a simple test processor that multiplies by a value.
type TestProcessor struct {
	multiplier float32
	processed  bool
}

func (t *TestProcessor) Process(buffer []float32) {
	for i := range buffer {
		buffer[i] *= t.multiplier
	}
	t.processed = true
}

func (t *TestProcessor) Reset() {
	t.processed = false
}

// TestStereoProcessor is a simple test stereo processor.
type TestStereoProcessor struct {
	leftGain  float32
	rightGain float32
	processed bool
}

func (t *TestStereoProcessor) ProcessStereo(left, right []float32) {
	for i := range left {
		left[i] *= t.leftGain
		right[i] *= t.rightGain
	}
	t.processed = true
}

func (t *TestStereoProcessor) Reset() {
	t.processed = false
}

func TestChain(t *testing.T) {
	t.Run("BasicChain", func(t *testing.T) {
		chain := NewChain("test")
		
		// Add processors
		p1 := &TestProcessor{multiplier: 2.0}
		p2 := &TestProcessor{multiplier: 0.5}
		chain.Add(p1).Add(p2)
		
		// Process buffer
		buffer := []float32{1.0, 2.0, 3.0, 4.0}
		chain.Process(buffer)
		
		// Check result (1*2*0.5 = 1, 2*2*0.5 = 2, etc.)
		expected := []float32{1.0, 2.0, 3.0, 4.0}
		for i, v := range buffer {
			if math.Abs(float64(v-expected[i])) > 0.001 {
				t.Errorf("Sample %d: expected %f, got %f", i, expected[i], v)
			}
		}
		
		// Check processors were called
		if !p1.processed || !p2.processed {
			t.Error("Processors were not called")
		}
	})
	
	t.Run("AddFunc", func(t *testing.T) {
		chain := NewChain("test")
		
		// Add function processor
		called := false
		chain.AddFunc("doubler", func(buffer []float32) {
			for i := range buffer {
				buffer[i] *= 2.0
			}
			called = true
		})
		
		buffer := []float32{1.0, 2.0, 3.0}
		chain.Process(buffer)
		
		if !called {
			t.Error("Function processor was not called")
		}
		
		// Check values doubled
		expected := []float32{2.0, 4.0, 6.0}
		for i, v := range buffer {
			if v != expected[i] {
				t.Errorf("Sample %d: expected %f, got %f", i, expected[i], v)
			}
		}
	})
	
	t.Run("Bypass", func(t *testing.T) {
		chain := NewChain("test")
		p := &TestProcessor{multiplier: 2.0}
		chain.Add(p)
		
		// Enable bypass
		chain.SetBypass(true)
		
		buffer := []float32{1.0, 2.0, 3.0}
		original := make([]float32, len(buffer))
		copy(original, buffer)
		
		chain.Process(buffer)
		
		// Buffer should be unchanged
		for i, v := range buffer {
			if v != original[i] {
				t.Error("Buffer was modified when bypassed")
			}
		}
		
		if p.processed {
			t.Error("Processor was called when bypassed")
		}
	})
	
	t.Run("Reset", func(t *testing.T) {
		chain := NewChain("test")
		p1 := &TestProcessor{processed: true}
		p2 := &TestProcessor{processed: true}
		chain.Add(p1).Add(p2)
		
		chain.Reset()
		
		if p1.processed || p2.processed {
			t.Error("Processors were not reset")
		}
	})
	
	t.Run("Empty", func(t *testing.T) {
		chain := NewChain("test")
		
		if !chain.IsEmpty() {
			t.Error("New chain should be empty")
		}
		
		chain.Add(&TestProcessor{})
		
		if chain.IsEmpty() {
			t.Error("Chain with processor should not be empty")
		}
		
		if chain.Count() != 1 {
			t.Errorf("Expected count 1, got %d", chain.Count())
		}
	})
}

func TestStereoChain(t *testing.T) {
	t.Run("BasicStereoChain", func(t *testing.T) {
		chain := NewStereoChain("test")
		
		p := &TestStereoProcessor{
			leftGain:  0.5,
			rightGain: 2.0,
		}
		chain.Add(p)
		
		left := []float32{1.0, 2.0, 3.0}
		right := []float32{1.0, 2.0, 3.0}
		
		chain.ProcessStereo(left, right)
		
		// Check left channel (halved)
		for i, v := range left {
			expected := float32(i+1) * 0.5
			if v != expected {
				t.Errorf("Left[%d]: expected %f, got %f", i, expected, v)
			}
		}
		
		// Check right channel (doubled)
		for i, v := range right {
			expected := float32(i+1) * 2.0
			if v != expected {
				t.Errorf("Right[%d]: expected %f, got %f", i, expected, v)
			}
		}
	})
}

func TestParallelChain(t *testing.T) {
	t.Run("BasicParallel", func(t *testing.T) {
		parallel := NewParallelChain("test")
		
		// Add two chains with different gains
		parallel.Add(&TestProcessor{multiplier: 2.0}, 0.5)  // 2x then 0.5 mix
		parallel.Add(&TestProcessor{multiplier: 4.0}, 0.25) // 4x then 0.25 mix
		
		buffer := []float32{1.0, 2.0, 3.0}
		parallel.Process(buffer)
		
		// Result should be: 1*(2*0.5 + 4*0.25) = 1*(1 + 1) = 2
		expected := []float32{2.0, 4.0, 6.0}
		for i, v := range buffer {
			if math.Abs(float64(v-expected[i])) > 0.001 {
				t.Errorf("Sample %d: expected %f, got %f", i, expected[i], v)
			}
		}
	})
	
	t.Run("EmptyParallel", func(t *testing.T) {
		parallel := NewParallelChain("test")
		
		buffer := []float32{1.0, 2.0, 3.0}
		original := make([]float32, len(buffer))
		copy(original, buffer)
		
		parallel.Process(buffer)
		
		// Should be unchanged
		for i, v := range buffer {
			if v != original[i] {
				t.Error("Empty parallel chain modified buffer")
			}
		}
	})
}

func TestBuilder(t *testing.T) {
	t.Run("ValidBuild", func(t *testing.T) {
		chain, err := NewBuilder("test").
			WithProcessor(&TestProcessor{multiplier: 2.0}).
			WithFunc("halver", func(buffer []float32) {
				for i := range buffer {
					buffer[i] *= 0.5
				}
			}).
			Build()
		
		if err != nil {
			t.Errorf("Build failed: %v", err)
		}
		
		if chain.Count() != 2 {
			t.Errorf("Expected 2 processors, got %d", chain.Count())
		}
	})
	
	t.Run("NilProcessor", func(t *testing.T) {
		_, err := NewBuilder("test").
			WithProcessor(nil).
			Build()
		
		if err == nil {
			t.Error("Expected error for nil processor")
		}
	})
	
	t.Run("NilFunc", func(t *testing.T) {
		_, err := NewBuilder("test").
			WithFunc("nil", nil).
			Build()
		
		if err == nil {
			t.Error("Expected error for nil function")
		}
	})
	
	t.Run("EmptyChain", func(t *testing.T) {
		_, err := NewBuilder("test").Build()
		
		if err == nil {
			t.Error("Expected error for empty chain")
		}
	})
}

func TestStereoBuilder(t *testing.T) {
	t.Run("ValidBuild", func(t *testing.T) {
		chain, err := NewStereoBuilder("test").
			WithProcessor(&TestStereoProcessor{leftGain: 1.0, rightGain: 1.0}).
			Build()
		
		if err != nil {
			t.Errorf("Build failed: %v", err)
		}
		
		if len(chain.processors) != 1 {
			t.Error("Expected 1 processor")
		}
	})
	
	t.Run("NilProcessor", func(t *testing.T) {
		_, err := NewStereoBuilder("test").
			WithProcessor(nil).
			Build()
		
		if err == nil {
			t.Error("Expected error for nil processor")
		}
	})
}

func BenchmarkChain(b *testing.B) {
	chain := NewChain("bench")
	chain.Add(&TestProcessor{multiplier: 1.1})
	chain.Add(&TestProcessor{multiplier: 0.9})
	chain.Add(&TestProcessor{multiplier: 1.05})
	
	buffer := make([]float32, 512)
	for i := range buffer {
		buffer[i] = float32(i) / 512.0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain.Process(buffer)
	}
}

func BenchmarkParallelChain(b *testing.B) {
	parallel := NewParallelChain("bench")
	parallel.Add(&TestProcessor{multiplier: 1.1}, 0.5)
	parallel.Add(&TestProcessor{multiplier: 0.9}, 0.5)
	
	buffer := make([]float32, 512)
	for i := range buffer {
		buffer[i] = float32(i) / 512.0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parallel.Process(buffer)
	}
}