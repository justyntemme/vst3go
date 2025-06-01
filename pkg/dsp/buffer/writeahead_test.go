package buffer

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWriteAheadBuffer(t *testing.T) {
	sampleRate := 44100.0
	channels := 2
	
	buf := NewWriteAheadBuffer(sampleRate, channels)
	
	// Check buffer was created
	if buf == nil {
		t.Fatal("Failed to create buffer")
	}
	
	// Check buffer size is power of 2
	if (buf.size & (buf.size - 1)) != 0 {
		t.Errorf("Buffer size %d is not power of 2", buf.size)
	}
	
	// Check latency calculation (50ms)
	expectedLatency := uint32(math.Round(50.0 * sampleRate / 1000.0 * float64(channels)))
	if buf.latencySamples != expectedLatency {
		t.Errorf("Expected latency %d samples, got %d", expectedLatency, buf.latencySamples)
	}
	
	// Check initial write position is ahead of read position
	if buf.writePos != uint64(buf.latencySamples) {
		t.Errorf("Initial write position should be %d, got %d", buf.latencySamples, buf.writePos)
	}
}

func TestWriteRead(t *testing.T) {
	buf := NewWriteAheadBuffer(44100, 1)
	
	// Write some test data
	testData := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	err := buf.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	// Try to read immediately - should get zeros due to latency
	output := make([]float32, len(testData))
	n := buf.Read(output)
	
	// Should have read something, but it should be zeros (silence)
	if n == 0 {
		t.Error("Expected to read some samples")
	}
	
	for i := 0; i < n; i++ {
		if output[i] != 0 {
			t.Errorf("Expected silence at position %d, got %f", i, output[i])
		}
	}
	
	// Write enough data to fill past the latency
	moreData := make([]float32, buf.latencySamples)
	for i := range moreData {
		moreData[i] = float32(i + 10)
	}
	err = buf.Write(moreData)
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}
	
	// Now we need to read enough to consume the initial silence
	// and get to our test data
	silence := make([]float32, buf.latencySamples - uint32(len(testData)))
	buf.Read(silence) // Read out the silence
	
	// Now read again - should get our original test data
	n = buf.Read(output)
	if n != len(output) {
		t.Errorf("Expected to read %d samples, got %d", len(output), n)
	}
	
	// Verify we got our original data
	for i := 0; i < len(testData); i++ {
		if output[i] != testData[i] {
			t.Errorf("At position %d: expected %f, got %f", i, testData[i], output[i])
		}
	}
}

func TestWrapAround(t *testing.T) {
	// Create a small buffer to test wrap-around
	buf := NewWriteAheadBuffer(1000, 1) // Small sample rate for small buffer
	
	// Calculate safe amount to write (accounting for latency gap)
	safeSize := int(buf.size - buf.latencySamples - 100)
	data := make([]float32, safeSize)
	for i := range data {
		data[i] = float32(i)
	}
	
	err := buf.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	// Read some data to make space
	output := make([]float32, buf.size/4)
	n := buf.Read(output)
	
	// Write more data that will wrap around
	moreData := make([]float32, n) // Write exactly what we read
	for i := range moreData {
		moreData[i] = float32(i + 1000)
	}
	
	err = buf.Write(moreData)
	if err != nil {
		t.Fatalf("Wrap-around write failed: %v", err)
	}
	
	// Verify buffer stats show no errors
	stats := buf.GetBufferHealth()
	if stats.Overruns > 0 {
		t.Errorf("Unexpected overruns: %d", stats.Overruns)
	}
}

func TestLatencyMaintenance(t *testing.T) {
	buf := NewWriteAheadBuffer(44100, 1)
	
	// First write some data so there's actually something to read
	testData := make([]float32, buf.latencySamples + 1000)
	for i := range testData {
		testData[i] = float32(i)
	}
	err := buf.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	
	// Now force read position too close to write position
	currentWritePos := atomic.LoadUint64(&buf.writePos)
	atomic.StoreUint64(&buf.readPos, currentWritePos - 100) // Much less than latencySamples
	
	// Read should trigger maintainDelay adjustment
	output := make([]float32, 10)
	n := buf.Read(output)
	t.Logf("Read %d samples", n)
	
	// Check that adjustment was made
	stats := buf.GetBufferHealth()
	if stats.Adjustments == 0 {
		t.Error("Expected at least one position adjustment")
	}
	
	// Verify minimum gap is maintained
	readPos := atomic.LoadUint64(&buf.readPos)
	writePos := atomic.LoadUint64(&buf.writePos)
	gap := writePos - readPos
	
	t.Logf("After read: readPos=%d, writePos=%d, gap=%d, latencySamples=%d, n=%d", 
		readPos, writePos, gap, buf.latencySamples, n)
	
	// The issue is that maintainDelay sets readPos = writePos - latencySamples,
	// but then Read() might not read anything if there's no actual data available
	// Let's check if we actually read any data
	if n == 0 {
		// If no data was read, gap should be exactly latencySamples
		if gap != uint64(buf.latencySamples) {
			t.Errorf("Gap %d does not equal required latency %d (no data read)", gap, buf.latencySamples)
		}
	} else {
		// If data was read, the gap will be latencySamples - n
		// because maintainDelay sets gap to exactly latencySamples,
		// then Read advances by n
		expectedGap := uint64(buf.latencySamples) - uint64(n)
		if gap != expectedGap {
			t.Errorf("Gap %d does not equal expected %d (latencySamples=%d, read=%d)", 
				gap, expectedGap, buf.latencySamples, n)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	buf := NewWriteAheadBuffer(44100, 2)
	const numGoroutines = 10
	const samplesPerGoroutine = 1000
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Writers and readers
	
	// Start writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			data := make([]float32, samplesPerGoroutine)
			for j := range data {
				data[j] = float32(id*samplesPerGoroutine + j)
			}
			
			// Write in chunks
			chunkSize := 100
			for offset := 0; offset < len(data); offset += chunkSize {
				end := offset + chunkSize
				if end > len(data) {
					end = len(data)
				}
				
				for retry := 0; retry < 10; retry++ {
					err := buf.Write(data[offset:end])
					if err == nil {
						break
					}
					time.Sleep(time.Millisecond) // Back off on overrun
				}
			}
		}(i)
	}
	
	// Start reader goroutines
	totalRead := 0
	var readMutex sync.Mutex
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			
			output := make([]float32, 100)
			localRead := 0
			
			for localRead < samplesPerGoroutine {
				n := buf.Read(output)
				localRead += n
				time.Sleep(time.Microsecond * 100) // Simulate processing
			}
			
			readMutex.Lock()
			totalRead += localRead
			readMutex.Unlock()
		}()
	}
	
	wg.Wait()
	
	// Check buffer health
	stats := buf.GetBufferHealth()
	t.Logf("Buffer stats - Underruns: %d, Overruns: %d, Adjustments: %d",
		stats.Underruns, stats.Overruns, stats.Adjustments)
	
	// Some adjustments are expected in concurrent scenario
	if stats.Adjustments == 0 {
		t.Log("Warning: No adjustments made, latency might not be enforced properly")
	}
}

func TestBufferOverrun(t *testing.T) {
	buf := NewWriteAheadBuffer(1000, 1) // Small buffer
	
	// Try to write more than buffer can hold
	tooMuch := make([]float32, buf.size+100)
	err := buf.Write(tooMuch)
	
	if err == nil {
		t.Error("Expected overrun error")
	}
	
	stats := buf.GetBufferHealth()
	if stats.Overruns != 1 {
		t.Errorf("Expected 1 overrun, got %d", stats.Overruns)
	}
}

func TestBufferUnderrun(t *testing.T) {
	buf := NewWriteAheadBuffer(44100, 1)
	
	// Try to read more than available
	output := make([]float32, buf.latencySamples*2)
	n := buf.Read(output)
	
	// Should read less than requested
	if n >= len(output) {
		t.Error("Read more samples than should be available")
	}
	
	stats := buf.GetBufferHealth()
	if stats.Underruns == 0 {
		t.Error("Expected at least one underrun")
	}
}

func TestReset(t *testing.T) {
	buf := NewWriteAheadBuffer(44100, 1)
	
	// Write some data and cause some stats
	data := make([]float32, 1000)
	buf.Write(data)
	buf.Read(data) // This will cause adjustment
	
	// Force an overrun
	bigData := make([]float32, buf.size+100)
	buf.Write(bigData)
	
	// Reset
	buf.Reset()
	
	// Check everything is cleared
	stats := buf.GetBufferHealth()
	if stats.Underruns != 0 || stats.Overruns != 0 || stats.Adjustments != 0 {
		t.Error("Stats not reset properly")
	}
	
	if buf.readPos != 0 {
		t.Errorf("Read position not reset, got %d", buf.readPos)
	}
	
	if buf.writePos != uint64(buf.latencySamples) {
		t.Errorf("Write position not reset properly, got %d", buf.writePos)
	}
	
	// Verify buffer is zeroed
	for i, v := range buf.data {
		if v != 0 {
			t.Errorf("Buffer not zeroed at position %d: %f", i, v)
			break
		}
	}
}

func TestBufferHealthMetrics(t *testing.T) {
	buf := NewWriteAheadBuffer(44100, 1)
	
	// Buffer starts with write position ahead by latencySamples
	// Check initial health metrics
	stats := buf.GetBufferHealth()
	
	// Fill percentage should reflect initial latency buffer
	if stats.FillPercentage <= 0 {
		t.Errorf("Fill percentage should be positive, got %f", stats.FillPercentage)
	}
	
	// Current latency should be exactly 50ms initially
	expectedLatency := 50 * time.Millisecond
	tolerance := 100 * time.Microsecond // Very small tolerance
	
	if stats.CurrentLatency < expectedLatency-tolerance ||
		stats.CurrentLatency > expectedLatency+tolerance {
		t.Errorf("Current latency %v not close to expected %v", 
			stats.CurrentLatency, expectedLatency)
	}
	
	// Test utilization method
	utilization := buf.GetBufferUtilization()
	if utilization <= 0 || utilization > 1 {
		t.Errorf("Buffer utilization out of range: %f", utilization)
	}
	
	// Write some data and check again
	data := make([]float32, 1000)
	buf.Write(data)
	
	stats2 := buf.GetBufferHealth()
	if stats2.FillPercentage <= stats.FillPercentage {
		t.Error("Fill percentage should increase after writing data")
	}
}

func TestPowerOf2Calculation(t *testing.T) {
	tests := []struct {
		input    uint32
		expected uint32
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{5, 8},
		{15, 16},
		{16, 16},
		{17, 32},
		{1000, 1024},
		{2048, 2048},
	}
	
	for _, tc := range tests {
		result := nextPowerOf2(tc.input)
		if result != tc.expected {
			t.Errorf("nextPowerOf2(%d) = %d, expected %d", 
				tc.input, result, tc.expected)
		}
	}
}

func BenchmarkWrite(b *testing.B) {
	buf := NewWriteAheadBuffer(44100, 2)
	data := make([]float32, 512) // Typical block size
	
	for i := range data {
		data[i] = float32(i) / 512.0
	}
	
	// Pre-allocate output buffer
	output := make([]float32, 512)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := buf.Write(data)
		if err != nil {
			// Buffer full, read to make space
			buf.Read(output)
			err = buf.Write(data)
			if err != nil {
				b.Fatal(err)
			}
		}
		
		// Periodically read to keep buffer from filling
		if i%10 == 0 {
			buf.Read(output)
		}
	}
}

func BenchmarkRead(b *testing.B) {
	buf := NewWriteAheadBuffer(44100, 2)
	
	// Pre-fill buffer
	data := make([]float32, buf.size/2)
	buf.Write(data)
	
	output := make([]float32, 512)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		n := buf.Read(output)
		
		// Write to keep buffer fed
		if i%100 == 0 && n > 0 {
			buf.Write(data[:n])
		}
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	buf := NewWriteAheadBuffer(44100, 2)
	data := make([]float32, 256)
	output := make([]float32, 256)
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Alternate between read and write
			if err := buf.Write(data); err == nil {
				buf.Read(output)
			}
		}
	})
}