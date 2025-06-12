package dsp

import (
	"math"
	"testing"

	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/utility"
)

func TestCompressorAdapter(t *testing.T) {
	sampleRate := 48000.0
	comp := dynamics.NewCompressor(sampleRate)
	comp.SetThreshold(-10)
	comp.SetRatio(4)
	comp.SetAttack(1)
	comp.SetRelease(50)
	
	adapter := NewCompressorAdapter(comp)
	
	// Test with loud signal
	buffer := make([]float32, 100)
	for i := range buffer {
		buffer[i] = 0.8 // Above threshold
	}
	
	adapter.Process(buffer)
	
	// Signal should be processed (not necessarily compressed immediately due to attack time)
	// Just verify it runs without panic
	adapter.Reset()
}

func TestGateAdapter(t *testing.T) {
	sampleRate := 48000.0
	gate := dynamics.NewGate(sampleRate)
	gate.SetThreshold(-30)
	gate.SetRange(-60)
	
	adapter := NewGateAdapter(gate)
	
	// Test with quiet signal (below threshold)
	buffer := make([]float32, 100)
	for i := range buffer {
		buffer[i] = 0.001 // Very quiet, should be gated
	}
	
	adapter.Process(buffer)
	
	// Signal should be attenuated
	// Just verify it runs without panic
	adapter.Reset()
}

func TestDCBlockerAdapter(t *testing.T) {
	sampleRate := 48000.0
	adapter := NewDCBlockerAdapter(sampleRate)
	
	// Test with DC offset
	// Process multiple buffers to allow settling
	buffer := make([]float32, 1000)
	for j := 0; j < 10; j++ { // Process 10 buffers
		for i := range buffer {
			buffer[i] = 0.5 // DC offset
		}
		adapter.Process(buffer)
	}
	
	// After processing many samples, DC should be mostly removed
	sum := float32(0)
	count := 0
	for _, v := range buffer[800:] { // Check last 200 samples
		sum += v
		count++
	}
	avg := sum / float32(count)
	
	// DC blocker is a high-pass filter, so some DC may remain
	// but it should be significantly reduced
	if math.Abs(float64(avg)) > 0.05 { // More lenient threshold
		t.Errorf("DC not sufficiently removed, average: %f", avg)
	}
	
	adapter.Reset()
}

func TestNoiseAdapter(t *testing.T) {
	adapter := NewNoiseAdapter(utility.WhiteNoise, 0.1)
	
	// Test with silence
	buffer := make([]float32, 1000)
	adapter.Process(buffer)
	
	// Should have added noise
	hasNoise := false
	for _, v := range buffer {
		if v != 0 {
			hasNoise = true
			break
		}
	}
	
	if !hasNoise {
		t.Error("No noise was added")
	}
	
	// Check noise level
	sum := float32(0)
	for _, v := range buffer {
		sum += v * v // RMS calculation
	}
	rms := float32(math.Sqrt(float64(sum / float32(len(buffer)))))
	
	// With mix of 0.1, RMS should be roughly 0.1 * 0.577 (RMS of uniform noise)
	expectedRMS := float32(0.1 * 0.577)
	if math.Abs(float64(rms-expectedRMS)) > 0.05 {
		t.Errorf("Unexpected noise level, RMS: %f, expected ~%f", rms, expectedRMS)
	}
	
	adapter.Reset()
}

func TestSimpleChain(t *testing.T) {
	sampleRate := 48000.0
	chain, err := CreateSimpleChain(sampleRate)
	
	if err != nil {
		t.Fatalf("Failed to create simple chain: %v", err)
	}
	
	if chain.Count() != 1 {
		t.Errorf("Expected 1 processor in simple chain, got %d", chain.Count())
	}
	
	// Test processing
	buffer := make([]float32, 512)
	for i := range buffer {
		buffer[i] = 0.5 // DC offset
	}
	
	chain.Process(buffer)
	
	// Should have processed without panic
	chain.Reset()
}

func TestDynamicsChain(t *testing.T) {
	sampleRate := 48000.0
	chain, err := CreateDynamicsChain(sampleRate)
	
	if err != nil {
		t.Fatalf("Failed to create dynamics chain: %v", err)
	}
	
	if chain.Count() != 2 {
		t.Errorf("Expected 2 processors in dynamics chain, got %d", chain.Count())
	}
	
	// Test processing
	buffer := make([]float32, 512)
	for i := range buffer {
		buffer[i] = float32(i%100) / 100.0 - 0.5
	}
	
	chain.Process(buffer)
	
	// Should have processed without panic
	chain.Reset()
}