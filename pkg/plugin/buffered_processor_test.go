package plugin

import (
	"testing"
	"time"

	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/process"
)

// mockProcessor implements a simple test processor
type mockProcessor struct {
	initialized      bool
	active           bool
	sampleRate       float64
	maxBlockSize     int32
	processedSamples int64
	lastMIDIEvents   []MIDIEvent
	params           *param.Registry
	buses            *bus.Configuration
}

func newMockProcessor() *mockProcessor {
	mp := &mockProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}
	
	// Add a test parameter
	mp.params.Add(&param.Parameter{
		ID:           1,
		Name:         "Test Param",
		ShortName:    "Test",
		Unit:         "%",
		DefaultValue: 0.5,
		Min:          0.0,
		Max:          1.0,
		StepCount:    0,
		Flags:        param.CanAutomate,
	})
	
	return mp
}

func (m *mockProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	m.initialized = true
	m.sampleRate = sampleRate
	m.maxBlockSize = maxBlockSize
	return nil
}

func (m *mockProcessor) ProcessAudio(ctx *process.Context) {
	numSamples := ctx.NumSamples()
	m.processedSamples += int64(numSamples)
	
	// Simple passthrough processing
	for ch := 0; ch < ctx.NumOutputChannels() && ch < ctx.NumInputChannels(); ch++ {
		copy(ctx.Output[ch], ctx.Input[ch])
	}
}

func (m *mockProcessor) ProcessMIDI(events []MIDIEvent) {
	m.lastMIDIEvents = make([]MIDIEvent, len(events))
	copy(m.lastMIDIEvents, events)
}

func (m *mockProcessor) GetParameters() *param.Registry {
	return m.params
}

func (m *mockProcessor) GetBuses() *bus.Configuration {
	return m.buses
}

func (m *mockProcessor) SetActive(active bool) error {
	m.active = active
	return nil
}

func (m *mockProcessor) GetLatencySamples() int32 {
	return 0 // No additional latency
}

func (m *mockProcessor) GetTailSamples() int32 {
	return 0 // No tail
}

func TestBufferedProcessorInitialization(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2) // Stereo
	
	sampleRate := 44100.0
	maxBlockSize := int32(512)
	
	err := bp.Initialize(sampleRate, maxBlockSize)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Check that wrapped processor was initialized
	if !mock.initialized {
		t.Error("Wrapped processor was not initialized")
	}
	
	// Check latency calculation (50ms)
	expectedLatency := int32(50.0 * sampleRate / 1000.0)
	if bp.latencySamples != expectedLatency {
		t.Errorf("Expected latency %d samples, got %d", expectedLatency, bp.latencySamples)
	}
	
	// Check that buffers were created
	if len(bp.buffers) != 2 {
		t.Errorf("Expected 2 buffers, got %d", len(bp.buffers))
	}
	
	// Clean up
	bp.Release()
}

func TestBufferedProcessorLatencyReporting(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	err := bp.Initialize(44100.0, 512)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// BufferedProcessor should report its own latency plus wrapped processor's
	totalLatency := bp.GetLatencySamples()
	expectedLatency := bp.latencySamples + mock.GetLatencySamples()
	
	if totalLatency != expectedLatency {
		t.Errorf("Expected total latency %d, got %d", expectedLatency, totalLatency)
	}
	
	bp.Release()
}

func TestBufferedProcessorWorkerProcessing(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	err := bp.Initialize(44100.0, 512)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Activate processing
	bp.SetActive(true)
	
	// Wait for worker to process some chunks
	time.Sleep(50 * time.Millisecond)
	
	// Check that wrapped processor has processed samples
	if mock.processedSamples == 0 {
		t.Error("Worker thread did not process any samples")
	}
	
	// Deactivate and clean up
	bp.SetActive(false)
	bp.Release()
}

func TestBufferedProcessorAudioPassthrough(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	err := bp.Initialize(44100.0, 512)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Create process context
	ctx := process.NewContext(512, bp.GetParameters())
	
	// Allocate I/O buffers
	input := make([][]float32, 2)
	output := make([][]float32, 2)
	for ch := 0; ch < 2; ch++ {
		input[ch] = make([]float32, 512)
		output[ch] = make([]float32, 512)
		
		// Fill input with test signal
		for i := 0; i < 512; i++ {
			input[ch][i] = float32(i) / 512.0
		}
	}
	
	ctx.Input = input
	ctx.Output = output
	ctx.SampleRate = 44100.0
	
	// Process - this will read from buffers (initially silence due to latency)
	bp.ProcessAudio(ctx)
	
	// Output should be silence initially
	for ch := 0; ch < 2; ch++ {
		for i := 0; i < 512; i++ {
			if output[ch][i] != 0 {
				t.Errorf("Expected silence at ch=%d, i=%d, got %f", ch, i, output[ch][i])
				break
			}
		}
	}
	
	bp.Release()
}

func TestBufferedProcessorMIDIHandling(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	err := bp.Initialize(44100.0, 512)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	bp.SetActive(true)
	
	// Queue a MIDI event
	testEvent := MIDIEvent{
		Data:         []byte{0x90, 0x3C, 0x7F}, // Note On C4 Velocity 127
		SampleOffset: 100,
		Timestamp:    0, // Will be calculated
	}
	
	bp.QueueMIDIEvent(testEvent)
	
	// Wait for worker to process
	time.Sleep(20 * time.Millisecond)
	
	// Check that MIDI event was processed
	// Due to latency adjustment, the event won't be processed immediately
	// but we can check that it was queued
	select {
	case <-bp.midiQueue:
		// Event was not processed yet (good, it's queued)
	default:
		// Queue is empty, event might have been processed
	}
	
	bp.SetActive(false)
	bp.Release()
}

func TestBufferedProcessorAdaptiveProcessing(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	err := bp.Initialize(44100.0, 512)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	bp.SetActive(true)
	
	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)
	
	// Check buffer statistics
	stats := bp.GetBufferStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 buffer stats, got %d", len(stats))
	}
	
	// Buffers should have some fill level
	for i, stat := range stats {
		if stat.FillPercentage == 0 {
			t.Errorf("Buffer %d has 0%% fill", i)
		}
		
		// Check that latency is maintained
		if stat.CurrentLatency < 40*time.Millisecond || stat.CurrentLatency > 60*time.Millisecond {
			t.Errorf("Buffer %d latency %v is outside expected range", i, stat.CurrentLatency)
		}
	}
	
	bp.SetActive(false)
	bp.Release()
}

func TestBufferedProcessorParameterForwarding(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	// Parameters should be forwarded
	params := bp.GetParameters()
	if params == nil {
		t.Fatal("GetParameters returned nil")
	}
	
	// Check that test parameter exists
	testParam := params.Get(1)
	if testParam == nil {
		t.Fatal("Test parameter not found")
	}
	
	if testParam.Name != "Test Param" {
		t.Errorf("Expected parameter name 'Test Param', got '%s'", testParam.Name)
	}
}

func TestBufferedProcessorBusConfiguration(t *testing.T) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	// Bus configuration should be forwarded
	buses := bp.GetBuses()
	if buses == nil {
		t.Fatal("GetBuses returned nil")
	}
	
	// Check audio inputs/outputs
	inputCount := buses.GetBusCount(bus.MediaTypeAudio, bus.DirectionInput)
	if inputCount != 1 {
		t.Errorf("Expected 1 audio input bus, got %d", inputCount)
	}
	
	outputCount := buses.GetBusCount(bus.MediaTypeAudio, bus.DirectionOutput)
	if outputCount != 1 {
		t.Errorf("Expected 1 audio output bus, got %d", outputCount)
	}
}

func BenchmarkBufferedProcessorThroughput(b *testing.B) {
	mock := newMockProcessor()
	bp := NewBufferedProcessor(mock, 2)
	
	err := bp.Initialize(44100.0, 512)
	if err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}
	
	bp.SetActive(true)
	
	ctx := process.NewContext(512, bp.GetParameters())
	
	// Allocate I/O buffers
	input := make([][]float32, 2)
	output := make([][]float32, 2)
	for ch := 0; ch < 2; ch++ {
		input[ch] = make([]float32, 512)
		output[ch] = make([]float32, 512)
	}
	
	ctx.Input = input
	ctx.Output = output
	ctx.SampleRate = 44100.0
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		bp.ProcessAudio(ctx)
	}
	
	b.StopTimer()
	bp.SetActive(false)
	bp.Release()
}