package plugin

import (
	"context"
	"sync"
	"time"

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
	wrapped    Processor
	buffers    []*buffer.WriteAheadBuffer
	numChannels int
	
	// Worker goroutine management
	workerCtx    context.Context
	workerCancel context.CancelFunc
	workerWg     sync.WaitGroup
	
	// MIDI event queue
	midiQueue    chan MIDIEvent
	midiQueueSize int
	pendingMIDI  []MIDIEvent // Events to process in current chunk
	
	// Processing state
	sampleRate       float64
	maxBlockSize     int32
	latencySamples   int32
	isActive         bool
	isActiveMu       sync.RWMutex
	currentSample    int64 // Current sample position for worker thread
	
	// Temporary buffers for processing
	tempInput    [][]float32
	tempOutput   [][]float32
	processCtx   *process.Context
	
	// Statistics
	underruns    uint64
	overruns     uint64
}

// NewBufferedProcessor creates a new buffered processor wrapper
func NewBufferedProcessor(p Processor, numChannels int) *BufferedProcessor {
	bp := &BufferedProcessor{
		wrapped:       p,
		numChannels:   numChannels,
		midiQueueSize: 1024, // Pre-allocate space for MIDI events
	}
	
	return bp
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
	
	// Allocate temporary processing buffers
	bp.tempInput = make([][]float32, bp.numChannels)
	bp.tempOutput = make([][]float32, bp.numChannels)
	for i := 0; i < bp.numChannels; i++ {
		bp.tempInput[i] = make([]float32, maxBlockSize)
		bp.tempOutput[i] = make([]float32, maxBlockSize)
	}
	
	// Create process context for the wrapped processor
	bp.processCtx = process.NewContext(int(maxBlockSize), bp.wrapped.GetParameters())
	
	// Initialize MIDI queue
	bp.midiQueue = make(chan MIDIEvent, bp.midiQueueSize)
	bp.pendingMIDI = make([]MIDIEvent, 0, 128) // Pre-allocate space
	
	// Pre-fill buffers with silence (50ms worth)
	silence := make([]float32, bp.latencySamples)
	for _, buf := range bp.buffers {
		buf.Write(silence)
	}
	
	// Start worker goroutine
	bp.workerCtx, bp.workerCancel = context.WithCancel(context.Background())
	bp.workerWg.Add(1)
	go bp.processingWorker()
	
	return nil
}

// ProcessAudio handles real-time audio processing by reading from buffers
func (bp *BufferedProcessor) ProcessAudio(ctx *process.Context) {
	numSamples := ctx.NumSamples()
	
	// Read from write-ahead buffers
	for ch := 0; ch < bp.numChannels && ch < len(ctx.Output); ch++ {
		n := bp.buffers[ch].Read(ctx.Output[ch][:numSamples])
		
		// If we didn't get enough samples (underrun), fill with silence
		if n < numSamples {
			for i := n; i < numSamples; i++ {
				ctx.Output[ch][i] = 0
			}
		}
	}
	
	// Queue any parameter changes with adjusted timing
	if ctx.HasParameterChanges() {
		changes := ctx.GetParameterChanges()
		for _, change := range changes {
			// Adjust sample offset for latency
			adjustedChange := change
			adjustedChange.SampleOffset += int(bp.latencySamples)
			
			// Queue for worker thread processing
			// Note: In a full implementation, we'd need a parameter change queue
			// similar to the MIDI queue
		}
	}
}

// processingWorker runs in a separate goroutine to process audio ahead of time
func (bp *BufferedProcessor) processingWorker() {
	defer bp.workerWg.Done()
	
	// Process at 5ms intervals (200Hz)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-bp.workerCtx.Done():
			return
			
		case <-ticker.C:
			bp.isActiveMu.RLock()
			active := bp.isActive
			bp.isActiveMu.RUnlock()
			
			if active {
				bp.processAdaptive()
			}
		}
	}
}

// processAdaptive processes audio based on buffer health
func (bp *BufferedProcessor) processAdaptive() {
	// Check buffer health to determine how much to process
	minFill := float32(100.0)
	for _, buf := range bp.buffers {
		stats := buf.GetBufferHealth()
		if stats.FillPercentage < minFill {
			minFill = stats.FillPercentage
		}
	}
	
	// Determine how many chunks to process based on buffer fill
	chunksToProcess := 0
	chunkSize := int(bp.maxBlockSize)
	
	switch {
	case minFill < 30:
		chunksToProcess = 4 // Process aggressively
	case minFill < 50:
		chunksToProcess = 2 // Process normally
	case minFill < 80:
		chunksToProcess = 1 // Process conservatively
	default:
		return // Skip this tick, buffers are full enough
	}
	
	// Process the determined number of chunks
	for i := 0; i < chunksToProcess; i++ {
		bp.processChunk(chunkSize)
	}
}

// processChunk processes a single chunk of audio
func (bp *BufferedProcessor) processChunk(numSamples int) {
	// Clear temporary buffers
	for ch := 0; ch < bp.numChannels; ch++ {
		for i := 0; i < numSamples; i++ {
			bp.tempInput[ch][i] = 0
			bp.tempOutput[ch][i] = 0
		}
	}
	
	// Set up process context
	bp.processCtx.Input = bp.tempInput
	bp.processCtx.Output = bp.tempOutput
	bp.processCtx.SampleRate = bp.sampleRate
	
	// Process any queued MIDI events that fall within this chunk
	bp.processMIDIEvents(numSamples)
	
	// Pass MIDI events to wrapped processor if it supports MIDI
	if len(bp.pendingMIDI) > 0 {
		if midiProc, ok := bp.wrapped.(MIDIProcessor); ok {
			midiProc.ProcessMIDI(bp.pendingMIDI)
		}
	}
	
	// Call wrapped processor
	bp.wrapped.ProcessAudio(bp.processCtx)
	
	// Write output to buffers
	for ch := 0; ch < bp.numChannels; ch++ {
		if ch < len(bp.tempOutput) {
			err := bp.buffers[ch].Write(bp.tempOutput[ch][:numSamples])
			if err != nil {
				// Buffer overrun - this shouldn't happen with proper adaptive processing
				bp.overruns++
			}
		}
	}
	
	// Update current sample position
	bp.currentSample += int64(numSamples)
}

// processMIDIEvents processes MIDI events for the current chunk
func (bp *BufferedProcessor) processMIDIEvents(numSamples int) {
	// Clear pending MIDI events from previous chunk
	bp.pendingMIDI = bp.pendingMIDI[:0]
	
	// Collect MIDI events that fall within this chunk's time range
	chunkStart := bp.currentSample
	chunkEnd := bp.currentSample + int64(numSamples)
	
	// Non-blocking read of all available MIDI events
	for {
		select {
		case event := <-bp.midiQueue:
			// Check if event falls within this chunk
			if event.Timestamp >= chunkStart && event.Timestamp < chunkEnd {
				// Adjust sample offset to be relative to this chunk
				event.SampleOffset = int32(event.Timestamp - chunkStart)
				bp.pendingMIDI = append(bp.pendingMIDI, event)
			} else if event.Timestamp >= chunkEnd {
				// Event is for a future chunk, put it back
				select {
				case bp.midiQueue <- event:
				default:
					// Queue full, drop event
				}
				return
			}
			// If event.Timestamp < chunkStart, it's too late, drop it
		default:
			// No more events available
			return
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

// Release cleans up resources
func (bp *BufferedProcessor) Release() {
	// Stop worker goroutine
	if bp.workerCancel != nil {
		bp.workerCancel()
		bp.workerWg.Wait()
	}
	
	// Close MIDI queue
	if bp.midiQueue != nil {
		close(bp.midiQueue)
	}
	
	// Drain remaining audio from buffers if needed
	// This ensures we don't cut off audio abruptly
	if bp.isActive {
		// Process remaining buffered audio
		for i := 0; i < 10; i++ { // Process up to 10 more chunks
			allEmpty := true
			for _, buf := range bp.buffers {
				stats := buf.GetBufferHealth()
				if stats.FillPercentage > 1.0 {
					allEmpty = false
					break
				}
			}
			
			if allEmpty {
				break
			}
			
			bp.processChunk(int(bp.maxBlockSize))
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// GetBufferStats returns current buffer statistics for monitoring
func (bp *BufferedProcessor) GetBufferStats() []buffer.BufferStats {
	stats := make([]buffer.BufferStats, len(bp.buffers))
	for i, buf := range bp.buffers {
		stats[i] = buf.GetBufferHealth()
	}
	return stats
}

// QueueMIDIEvent queues a MIDI event with adjusted timing
func (bp *BufferedProcessor) QueueMIDIEvent(event MIDIEvent) {
	// If timestamp is not set, calculate it from sample offset
	if event.Timestamp == 0 && event.SampleOffset >= 0 {
		// Assume the event is relative to the current real-time position
		// This would need to be adjusted based on the actual host callback timing
		event.Timestamp = bp.currentSample + int64(event.SampleOffset)
	}
	
	// Adjust event timing for latency
	event.Timestamp += int64(bp.latencySamples)
	
	select {
	case bp.midiQueue <- event:
		// Event queued successfully
	default:
		// Queue full, drop event (should log this in production)
	}
}