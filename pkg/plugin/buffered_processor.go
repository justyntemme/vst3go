package plugin

import (
	"sync"
	"sync/atomic"

	"github.com/justyntemme/vst3go/pkg/dsp/buffer"
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/process"
)

// MIDIEvent represents a MIDI event with timing information
type MIDIEvent struct {
	Data         []byte  // MIDI data
	SampleOffset int32   // Sample offset within the processing block
	Timestamp    int64   // Absolute timestamp in samples
}

// MIDIProcessor is an optional interface that processors can implement to handle MIDI
type MIDIProcessor interface {
	ProcessMIDI(events []MIDIEvent)
}

// BufferedProcessor wraps a Processor with write-ahead buffering for GC protection
type BufferedProcessor struct {
	wrapped      Processor
	buffers      []*buffer.WriteAheadBuffer // Output buffers
	numChannels  int
	
	// Processing state
	sampleRate     float64
	maxBlockSize   int32
	latencySamples int32
	isActive       bool
	isActiveMu     sync.RWMutex
	
	// Statistics
	underruns uint64
	overruns  uint64
}

// NewBufferedProcessor creates a new buffered processor wrapper
func NewBufferedProcessor(p Processor, numChannels int) *BufferedProcessor {
	return &BufferedProcessor{
		wrapped:     p,
		numChannels: numChannels,
	}
}

// Initialize sets up the buffered processor
func (bp *BufferedProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	bp.sampleRate = sampleRate
	bp.maxBlockSize = maxBlockSize
	
	// Initialize wrapped processor first
	if err := bp.wrapped.Initialize(sampleRate, maxBlockSize); err != nil {
		return err
	}
	
	// Calculate latency (50ms)
	latencyMs := 50.0
	bp.latencySamples = int32(latencyMs * sampleRate / 1000.0)
	
	// Create write-ahead buffers for each channel
	bp.buffers = make([]*buffer.WriteAheadBuffer, bp.numChannels)
	for i := 0; i < bp.numChannels; i++ {
		bp.buffers[i] = buffer.NewWriteAheadBuffer(sampleRate, 1) // 1 channel per buffer
	}
	
	return nil
}

// ProcessAudio handles real-time audio processing
func (bp *BufferedProcessor) ProcessAudio(ctx *process.Context) {
	numSamples := ctx.NumSamples()
	
	// First, process the audio with the wrapped processor
	// This happens in real-time with no delay
	bp.wrapped.ProcessAudio(ctx)
	
	// Now write the processed output to our delay buffers
	for ch := 0; ch < bp.numChannels && ch < len(ctx.Output); ch++ {
		err := bp.buffers[ch].Write(ctx.Output[ch][:numSamples])
		if err != nil {
			// Buffer full - this shouldn't happen with proper sizing
			atomic.AddUint64(&bp.overruns, 1)
		}
	}
	
	// Finally, read from the buffers with the 50ms delay
	// This gives us our latency-compensated output
	for ch := 0; ch < bp.numChannels && ch < len(ctx.Output); ch++ {
		n := bp.buffers[ch].Read(ctx.Output[ch][:numSamples])
		
		// If we didn't get enough samples (during startup), fill with silence
		if n < numSamples {
			for i := n; i < numSamples; i++ {
				ctx.Output[ch][i] = 0
			}
			atomic.AddUint64(&bp.underruns, 1)
		}
	}
}

// GetParameters returns the wrapped processor's parameters
func (bp *BufferedProcessor) GetParameters() *param.Registry {
	return bp.wrapped.GetParameters()
}

// GetBuses returns the wrapped processor's bus configuration
func (bp *BufferedProcessor) GetBuses() *bus.Configuration {
	return bp.wrapped.GetBuses()
}

// SetActive is called when processing starts/stops
func (bp *BufferedProcessor) SetActive(active bool) error {
	bp.isActiveMu.Lock()
	bp.isActive = active
	bp.isActiveMu.Unlock()
	
	// Forward to wrapped processor
	return bp.wrapped.SetActive(active)
}

// GetLatencySamples returns the buffer latency
func (bp *BufferedProcessor) GetLatencySamples() int32 {
	// Return our buffer latency plus any latency from the wrapped processor
	wrappedLatency := bp.wrapped.GetLatencySamples()
	return bp.latencySamples + wrappedLatency
}

// GetTailSamples returns the wrapped processor's tail length
func (bp *BufferedProcessor) GetTailSamples() int32 {
	return bp.wrapped.GetTailSamples()
}

// GetBufferHealth returns buffer statistics for monitoring
func (bp *BufferedProcessor) GetBufferHealth() (underruns, overruns uint64) {
	return atomic.LoadUint64(&bp.underruns), atomic.LoadUint64(&bp.overruns)
}

// QueueMIDIEvent queues a MIDI event for processing
// Note: MIDI handling is not yet implemented in the simplified buffered processor
func (bp *BufferedProcessor) QueueMIDIEvent(event MIDIEvent) {
	// TODO: Implement MIDI buffering if needed
	// For now, pass through directly if the processor supports MIDI
	if midiProc, ok := bp.wrapped.(MIDIProcessor); ok {
		// Adjust timing for latency
		event.SampleOffset += int32(bp.latencySamples)
		midiProc.ProcessMIDI([]MIDIEvent{event})
	}
}