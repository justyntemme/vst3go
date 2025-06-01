package buffer

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestGCResilience verifies buffer handles GC pauses without losing data
func TestGCResilience(t *testing.T) {
	sampleRate := 44100.0
	buf := NewWriteAheadBuffer(sampleRate, 2) // stereo
	
	// Configuration
	blockSize := 512
	totalBlocks := 1000
	
	// Generate test pattern - ascending values
	var expectedValue float32 = 1.0
	var lastReadValue float32 = 0.0
	glitches := int32(0)
	
	// Writer goroutine (simulates plugin processing)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		
		data := make([]float32, blockSize)
		for i := 0; i < totalBlocks; i++ {
			// Fill with ascending pattern
			for j := range data {
				data[j] = expectedValue
				expectedValue++
			}
			
			// Write to buffer
			for retry := 0; retry < 10; retry++ {
				err := buf.Write(data)
				if err == nil {
					break
				}
				// Buffer full, wait a bit
				time.Sleep(100 * time.Microsecond)
			}
			
			// Simulate processing time
			time.Sleep(time.Duration(float64(blockSize)/sampleRate*1000) * time.Millisecond)
		}
	}()
	
	// Reader goroutine (simulates host/DAW)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		
		output := make([]float32, blockSize)
		totalRead := 0
		
		// Give writer time to fill buffer
		time.Sleep(100 * time.Millisecond)
		
		for totalRead < totalBlocks*blockSize {
			n := buf.Read(output)
			
			// Check for glitches (gaps in sequence)
			for i := 0; i < n; i++ {
				if output[i] != 0 { // Skip initial zeros
					if lastReadValue != 0 && output[i] != lastReadValue+1 {
						// Detected a gap
						atomic.AddInt32(&glitches, 1)
						t.Logf("Glitch detected: expected %.0f, got %.0f", lastReadValue+1, output[i])
					}
					lastReadValue = output[i]
				}
			}
			
			totalRead += n
			
			// Simulate DAW processing time
			time.Sleep(time.Duration(float64(blockSize)/sampleRate*1000) * time.Millisecond)
		}
	}()
	
	// GC pause simulator
	go func() {
		time.Sleep(200 * time.Millisecond) // Let processing stabilize
		
		for i := 0; i < 10; i++ {
			// Force GC
			runtime.GC()
			runtime.Gosched()
			
			// Simulate 20ms pause
			time.Sleep(20 * time.Millisecond)
			
			// Wait between GC pauses
			time.Sleep(100 * time.Millisecond)
		}
	}()
	
	// Wait for completion
	<-writerDone
	<-readerDone
	
	// Check results
	stats := buf.GetBufferHealth()
	t.Logf("Buffer stats: Underruns=%d, Overruns=%d, Adjustments=%d",
		stats.Underruns, stats.Overruns, stats.Adjustments)
	
	finalGlitches := atomic.LoadInt32(&glitches)
	if finalGlitches > 0 {
		t.Errorf("Detected %d glitches during GC pauses", finalGlitches)
	}
}

// TestLatencyAtDifferentRates verifies consistent 50ms latency across sample rates
func TestLatencyAtDifferentRates(t *testing.T) {
	rates := []float64{22050, 44100, 48000, 88200, 96000, 192000}
	
	for _, rate := range rates {
		t.Run(fmt.Sprintf("%.0fHz", rate), func(t *testing.T) {
			buf := NewWriteAheadBuffer(rate, 1)
			
			// Calculate expected latency (with rounding)
			expectedSamples := uint32(math.Round(50.0 * rate / 1000.0)) // 50ms
			
			if buf.latencySamples != expectedSamples {
				t.Errorf("At %vHz: expected %d samples, got %d",
					rate, expectedSamples, buf.latencySamples)
			}
			
			// Verify actual latency matches
			latency := buf.GetCurrentLatency()
			expectedDuration := 50 * time.Millisecond
			tolerance := time.Millisecond
			
			if latency < expectedDuration-tolerance || latency > expectedDuration+tolerance {
				t.Errorf("At %vHz: latency %v, expected %v ± %v",
					rate, latency, expectedDuration, tolerance)
			}
		})
	}
}

// TestBufferSizing verifies buffer sizes are appropriate
func TestBufferSizing(t *testing.T) {
	rates := []float64{44100, 48000, 96000}
	channels := []int{1, 2, 6} // mono, stereo, 5.1
	
	for _, rate := range rates {
		for _, ch := range channels {
			t.Run(fmt.Sprintf("%.0fHz_%dch", rate, ch), func(t *testing.T) {
				buf := NewWriteAheadBuffer(rate, ch)
				
				// Buffer should be at least 4x latency
				minSize := buf.latencySamples * 4
				
				if buf.size < minSize {
					t.Errorf("Buffer too small: %d < %d (4x latency)",
						buf.size, minSize)
				}
				
				// Buffer should be power of 2
				if (buf.size & (buf.size - 1)) != 0 {
					t.Errorf("Buffer size %d is not power of 2", buf.size)
				}
				
				// Mask should be size-1
				if buf.mask != buf.size-1 {
					t.Errorf("Mask %d != size-1 %d", buf.mask, buf.size-1)
				}
			})
		}
	}
}

// TestConcurrentStress tests buffer under heavy concurrent load
func TestConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	buf := NewWriteAheadBuffer(44100, 2)
	
	var wg sync.WaitGroup
	numWriters := 4
	numReaders := 4
	duration := 2 * time.Second
	
	stop := make(chan struct{})
	errors := int32(0)
	
	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			data := make([]float32, 128)
			for j := range data {
				data[j] = float32(id) + 0.1*float32(j)
			}
			
			for {
				select {
				case <-stop:
					return
				default:
					err := buf.Write(data)
					if err != nil {
						// Expected when buffer is full
						time.Sleep(time.Microsecond)
					}
				}
			}
		}(i)
	}
	
	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			output := make([]float32, 256)
			
			for {
				select {
				case <-stop:
					return
				default:
					n := buf.Read(output)
					if n == 0 {
						atomic.AddInt32(&errors, 1)
					}
					time.Sleep(time.Microsecond)
				}
			}
		}()
	}
	
	// Run test
	time.Sleep(duration)
	close(stop)
	wg.Wait()
	
	// Check results
	stats := buf.GetBufferHealth()
	t.Logf("Stress test stats: Underruns=%d, Overruns=%d, Adjustments=%d, Errors=%d",
		stats.Underruns, stats.Overruns, stats.Adjustments, atomic.LoadInt32(&errors))
}