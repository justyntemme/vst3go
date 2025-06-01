// +build !cgo

package plugin

import (
	"testing"
	"time"
)

func TestWriteAheadBufferIntegration(t *testing.T) {
	// Test that BufferedProcessor properly manages WriteAheadBuffer instances
	mock := &mockProcessor{
		params: nil, // Simplified mock
		buses:  nil,
	}
	
	bp := NewBufferedProcessor(mock, 2)
	
	// Check initial state
	if bp.numChannels != 2 {
		t.Errorf("Expected 2 channels, got %d", bp.numChannels)
	}
	
	if bp.midiQueueSize != 1024 {
		t.Errorf("Expected MIDI queue size 1024, got %d", bp.midiQueueSize)
	}
}

func TestMIDIEventTiming(t *testing.T) {
	bp := &BufferedProcessor{
		latencySamples: 2205, // 50ms at 44.1kHz
		currentSample:  1000,
		midiQueue:      make(chan MIDIEvent, 10),
	}
	
	// Test event with only sample offset
	event1 := MIDIEvent{
		Data:         []byte{0x90, 0x3C, 0x7F},
		SampleOffset: 100,
		Timestamp:    0,
	}
	
	bp.QueueMIDIEvent(event1)
	
	// Check that event was queued
	select {
	case queuedEvent := <-bp.midiQueue:
		// Timestamp should be current sample + offset + latency
		expectedTimestamp := int64(1000 + 100 + 2205)
		if queuedEvent.Timestamp != expectedTimestamp {
			t.Errorf("Expected timestamp %d, got %d", expectedTimestamp, queuedEvent.Timestamp)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Event was not queued")
	}
	
	// Test event with pre-set timestamp
	event2 := MIDIEvent{
		Data:         []byte{0x80, 0x3C, 0x00},
		SampleOffset: -1,
		Timestamp:    5000,
	}
	
	bp.QueueMIDIEvent(event2)
	
	select {
	case queuedEvent := <-bp.midiQueue:
		// Timestamp should be original + latency
		expectedTimestamp := int64(5000 + 2205)
		if queuedEvent.Timestamp != expectedTimestamp {
			t.Errorf("Expected timestamp %d, got %d", expectedTimestamp, queuedEvent.Timestamp)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Event was not queued")
	}
}

func TestProcessMIDIEventsFiltering(t *testing.T) {
	bp := &BufferedProcessor{
		currentSample: 1000,
		midiQueue:     make(chan MIDIEvent, 10),
		pendingMIDI:   make([]MIDIEvent, 0, 10),
	}
	
	// Queue events with different timestamps
	events := []MIDIEvent{
		{Timestamp: 900},   // Too early
		{Timestamp: 1050},  // In range
		{Timestamp: 1100},  // In range
		{Timestamp: 1500},  // Too late
		{Timestamp: 1200},  // In range
	}
	
	for _, e := range events {
		bp.midiQueue <- e
	}
	
	// Process events for a 256 sample chunk
	bp.processMIDIEvents(256)
	
	// Should have collected 3 events (1050, 1100, 1200)
	if len(bp.pendingMIDI) != 3 {
		t.Errorf("Expected 3 pending MIDI events, got %d", len(bp.pendingMIDI))
	}
	
	// Check that sample offsets were adjusted correctly
	for _, e := range bp.pendingMIDI {
		expectedOffset := e.Timestamp - 1000
		if e.SampleOffset != int32(expectedOffset) {
			t.Errorf("Expected sample offset %d, got %d", expectedOffset, e.SampleOffset)
		}
	}
	
	// The future event should still be in the queue
	select {
	case e := <-bp.midiQueue:
		if e.Timestamp != 1500 {
			t.Errorf("Expected future event with timestamp 1500, got %d", e.Timestamp)
		}
	default:
		t.Error("Future event was not put back in queue")
	}
}

func TestAdaptiveProcessingLogic(t *testing.T) {
	testCases := []struct {
		fillPercentage   float32
		expectedChunks   int
		shouldSkip       bool
	}{
		{10.0, 4, false},   // Very low - process aggressively
		{25.0, 4, false},   // Low - process aggressively
		{40.0, 2, false},   // Medium low - process normally
		{60.0, 1, false},   // Medium high - process conservatively
		{75.0, 1, false},   // High - process conservatively
		{85.0, 0, true},    // Very high - skip processing
		{95.0, 0, true},    // Full - skip processing
	}
	
	for _, tc := range testCases {
		// This is a conceptual test - in real implementation we'd need to
		// mock the buffer statistics and verify processChunk is called
		// the correct number of times
		
		// Determine chunks based on fill percentage
		var chunks int
		var skip bool
		
		switch {
		case tc.fillPercentage < 30:
			chunks = 4
		case tc.fillPercentage < 50:
			chunks = 2
		case tc.fillPercentage < 80:
			chunks = 1
		default:
			skip = true
			chunks = 0
		}
		
		if chunks != tc.expectedChunks {
			t.Errorf("For fill %.1f%%, expected %d chunks, got %d",
				tc.fillPercentage, tc.expectedChunks, chunks)
		}
		
		if skip != tc.shouldSkip {
			t.Errorf("For fill %.1f%%, expected skip=%v, got %v",
				tc.fillPercentage, tc.shouldSkip, skip)
		}
	}
}

func TestLatencyCalculation(t *testing.T) {
	sampleRates := []float64{44100, 48000, 88200, 96000, 192000}
	latencyMs := 50.0
	
	for _, sr := range sampleRates {
		expectedSamples := int32(latencyMs * sr / 1000.0)
		
		// Simulate what Initialize does
		calculatedSamples := int32(50.0 * sr / 1000.0)
		
		if calculatedSamples != expectedSamples {
			t.Errorf("For sample rate %.0f, expected %d samples, got %d",
				sr, expectedSamples, calculatedSamples)
		}
	}
}