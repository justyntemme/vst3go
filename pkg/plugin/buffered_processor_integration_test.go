package plugin

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/process"
)

// TestGCPauseResilience verifies the buffered processor handles GC pauses without glitches
func TestGCPauseResilience(t *testing.T) {
	sampleRate := 44100.0
	blockSize := 512
	numChannels := 2

	// Create a test processor that generates a sine wave
	testProc := &sineWaveProcessor{
		params:     param.NewRegistry(),
		buses:      bus.NewStereoConfiguration(),
		sampleRate: sampleRate,
		phase:      0,
		frequency:  440.0, // A4
	}

	// Wrap with BufferedProcessor
	buffered := NewBufferedProcessor(testProc, numChannels)
	
	// Initialize
	err := buffered.Initialize(sampleRate, int32(blockSize))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Set active
	err = buffered.SetActive(true)
	if err != nil {
		t.Fatalf("Failed to set active: %v", err)
	}

	// Create process context
	ctx := &process.Context{
		Input:      make([][]float32, numChannels),
		Output:     make([][]float32, numChannels),
		SampleRate: sampleRate,
	}
	for i := 0; i < numChannels; i++ {
		ctx.Input[i] = make([]float32, blockSize)
		ctx.Output[i] = make([]float32, blockSize)
	}
	// Number of samples is determined by buffer size

	// Process audio while triggering GC pauses
	var wg sync.WaitGroup
	stopProcessing := make(chan struct{})
	glitchCount := int32(0)
	lastSample := float32(0)

	// Audio processing goroutine (simulates DAW)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Duration(float64(blockSize)/sampleRate*1000) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Process one block
				buffered.ProcessAudio(ctx)

				// Check for glitches (discontinuities in sine wave)
				for ch := 0; ch < numChannels; ch++ {
					for i := 0; i < blockSize; i++ {
						sample := ctx.Output[ch][i]
						
						// Simple glitch detection: large jumps in consecutive samples
						if i > 0 || ch > 0 || lastSample != 0 {
							diff := sample - lastSample
							if diff > 0.5 || diff < -0.5 {
								// Detected a glitch (sine wave shouldn't jump this much)
								atomic.AddInt32(&glitchCount, 1)
							}
						}
						lastSample = sample
					}
				}

			case <-stopProcessing:
				return
			}
		}
	}()

	// GC pause simulation goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // Let processing stabilize

		// Simulate 5 GC pauses
		for i := 0; i < 5; i++ {
			// Force GC
			runtime.GC()
			runtime.Gosched()

			// Simulate a 20ms pause
			time.Sleep(20 * time.Millisecond)

			// Wait between pauses
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Run for 2 seconds
	time.Sleep(2 * time.Second)
	close(stopProcessing)
	wg.Wait()

	// Check results
	finalGlitches := atomic.LoadInt32(&glitchCount)
	if finalGlitches > 0 {
		t.Errorf("Detected %d glitches during GC pauses", finalGlitches)
	}

	// Verify latency is reported correctly
	latency := buffered.GetLatencySamples()
	expectedLatency := int32(50.0 * sampleRate / 1000.0) // 50ms
	if latency != expectedLatency {
		t.Errorf("Expected latency %d samples, got %d", expectedLatency, latency)
	}
}

// TestLatencyConsistency verifies latency reporting across sample rates
func TestLatencyConsistency(t *testing.T) {
	sampleRates := []float64{44100, 48000, 88200, 96000, 192000}
	
	for _, sr := range sampleRates {
		t.Run(fmt.Sprintf("%.0fHz", sr), func(t *testing.T) {
			// Create test processor
			testProc := &mockProcessor{
				params: param.NewRegistry(),
				buses:  bus.NewStereoConfiguration(),
			}

			// Wrap with BufferedProcessor
			buffered := NewBufferedProcessor(testProc, 2)
			
			// Initialize
			err := buffered.Initialize(sr, 512)
			if err != nil {
				t.Fatalf("Failed to initialize at %vHz: %v", sr, err)
			}

			// Check latency
			latencySamples := buffered.GetLatencySamples()
			latencyMs := float64(latencySamples) * 1000.0 / sr
			
			// Should be 50ms ± 1 sample
			expectedMs := 50.0
			tolerance := 1000.0 / sr // 1 sample in ms
			
			if latencyMs < expectedMs-tolerance || latencyMs > expectedMs+tolerance {
				t.Errorf("Latency at %vHz: %.2fms, expected %.2fms ± %.2fms",
					sr, latencyMs, expectedMs, tolerance)
			}
		})
	}
}

// TestMIDISynchronization verifies MIDI events stay synchronized with audio
func TestMIDISynchronization(t *testing.T) {
	sampleRate := 44100.0
	blockSize := 512
	
	// Create a processor that responds to MIDI
	midiProc := &midiResponsiveProcessor{
		params:       param.NewRegistry(),
		buses:        bus.NewStereoConfiguration(),
		noteOnTimes:  make([]int64, 0),
		audioMarkers: make([]int64, 0),
	}

	// Wrap with BufferedProcessor
	buffered := NewBufferedProcessor(midiProc, 2)
	
	// Initialize
	err := buffered.Initialize(sampleRate, int32(blockSize))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Set active
	err = buffered.SetActive(true)
	if err != nil {
		t.Fatalf("Failed to set active: %v", err)
	}

	// Create process context
	ctx := &process.Context{
		Input:      make([][]float32, 2),
		Output:     make([][]float32, 2),
		SampleRate: sampleRate,
	}
	for i := 0; i < 2; i++ {
		ctx.Input[i] = make([]float32, blockSize)
		ctx.Output[i] = make([]float32, blockSize)
	}
	// Number of samples is determined by buffer size

	// Send MIDI note and mark audio
	totalSamples := int64(0)
	
	// Process some blocks
	for i := 0; i < 10; i++ {
		if i == 5 {
			// Send MIDI note at block 5
			event := MIDIEvent{
				Data:         []byte{0x90, 60, 100}, // Note On C4
				SampleOffset: 100,                   // 100 samples into the block
				Timestamp:    totalSamples + 100,
			}
			buffered.QueueMIDIEvent(event)
			
			// Mark when we expect the audio response
			midiProc.expectedAudioTime = totalSamples + 100 + int64(buffered.GetLatencySamples())
		}
		
		// Process audio
		buffered.ProcessAudio(ctx)
		
		// Update sample counter
		totalSamples += int64(blockSize)
		midiProc.currentSampleTime = totalSamples
	}

	// Verify MIDI and audio stayed synchronized
	if len(midiProc.noteOnTimes) == 0 {
		t.Error("No MIDI events were processed")
	}
	
	if len(midiProc.audioMarkers) == 0 {
		t.Error("No audio markers were generated")
	}
	
	// Check timing alignment
	for i, noteTime := range midiProc.noteOnTimes {
		if i < len(midiProc.audioMarkers) {
			audioTime := midiProc.audioMarkers[i]
			diff := audioTime - noteTime
			
			// Should be exactly the buffer latency
			expectedDiff := int64(buffered.GetLatencySamples())
			if diff != expectedDiff {
				t.Errorf("MIDI-audio sync error: diff=%d, expected=%d", diff, expectedDiff)
			}
		}
	}
}

// Helper processors for testing

type sineWaveProcessor struct {
	params     *param.Registry
	buses      *bus.Configuration
	sampleRate float64
	phase      float64
	frequency  float64
}

func (p *sineWaveProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	return nil
}

func (p *sineWaveProcessor) ProcessAudio(ctx *process.Context) {
	phaseIncrement := 2.0 * 3.14159265359 * p.frequency / p.sampleRate
	
	for ch := 0; ch < ctx.NumOutputChannels(); ch++ {
		phase := p.phase
		for i := 0; i < ctx.NumSamples(); i++ {
			ctx.Output[ch][i] = float32(math.Sin(phase))
			phase += phaseIncrement
			if phase > 2.0*3.14159265359 {
				phase -= 2.0 * 3.14159265359
			}
		}
	}
	
	p.phase += phaseIncrement * float64(ctx.NumSamples())
	if p.phase > 2.0*3.14159265359 {
		p.phase -= 2.0 * 3.14159265359
	}
}

func (p *sineWaveProcessor) GetParameters() *param.Registry { return p.params }
func (p *sineWaveProcessor) GetBuses() *bus.Configuration   { return p.buses }
func (p *sineWaveProcessor) SetActive(active bool) error    { return nil }
func (p *sineWaveProcessor) GetLatencySamples() int32       { return 0 }
func (p *sineWaveProcessor) GetTailSamples() int32          { return 0 }

type midiResponsiveProcessor struct {
	params            *param.Registry
	buses             *bus.Configuration
	noteOnTimes       []int64
	audioMarkers      []int64
	currentSampleTime int64
	expectedAudioTime int64
	mu                sync.Mutex
}

func (p *midiResponsiveProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	return nil
}

func (p *midiResponsiveProcessor) ProcessAudio(ctx *process.Context) {
	// Check if we should generate an audio marker
	p.mu.Lock()
	if p.expectedAudioTime > 0 && p.currentSampleTime >= p.expectedAudioTime {
		p.audioMarkers = append(p.audioMarkers, p.currentSampleTime)
		p.expectedAudioTime = 0
		
		// Generate a click in the audio
		for ch := 0; ch < ctx.NumOutputChannels(); ch++ {
			ctx.Output[ch][0] = 1.0
		}
	}
	p.mu.Unlock()
}

func (p *midiResponsiveProcessor) ProcessMIDI(events []MIDIEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, event := range events {
		if len(event.Data) >= 3 && (event.Data[0]&0xF0) == 0x90 && event.Data[2] > 0 {
			// Note On
			p.noteOnTimes = append(p.noteOnTimes, event.Timestamp)
		}
	}
}

func (p *midiResponsiveProcessor) GetParameters() *param.Registry { return p.params }
func (p *midiResponsiveProcessor) GetBuses() *bus.Configuration   { return p.buses }
func (p *midiResponsiveProcessor) SetActive(active bool) error    { return nil }
func (p *midiResponsiveProcessor) GetLatencySamples() int32       { return 0 }
func (p *midiResponsiveProcessor) GetTailSamples() int32          { return 0 }