package midi

import (
	"testing"
)

func TestEventQueue(t *testing.T) {
	q := NewEventQueue()

	// Test empty queue
	if !q.IsEmpty() {
		t.Error("Expected queue to be empty")
	}
	if q.Size() != 0 {
		t.Errorf("Expected size 0, got %d", q.Size())
	}

	// Add events
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 100}, NoteNumber: 60, Velocity: 100})
	q.Add(NoteOffEvent{BaseEvent: BaseEvent{Offset: 200}, NoteNumber: 60, Velocity: 0})
	q.Add(ControlChangeEvent{BaseEvent: BaseEvent{Offset: 50}, Controller: CCSustain, Value: 127})

	if q.IsEmpty() {
		t.Error("Expected queue to not be empty")
	}
	if q.Size() != 3 {
		t.Errorf("Expected size 3, got %d", q.Size())
	}
}

func TestEventQueueSorting(t *testing.T) {
	q := NewEventQueue()

	// Add events out of order
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 300}, NoteNumber: 62, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 100}, NoteNumber: 60, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 200}, NoteNumber: 61, Velocity: 100})

	events := q.GetAllEvents()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Check that events are sorted by offset
	offsets := []int32{100, 200, 300}
	for i, event := range events {
		if event.SampleOffset() != offsets[i] {
			t.Errorf("Event %d: expected offset %d, got %d", i, offsets[i], event.SampleOffset())
		}
	}
}

func TestGetEventsInRange(t *testing.T) {
	q := NewEventQueue()

	// Add events
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 0}, NoteNumber: 60, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 50}, NoteNumber: 61, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 100}, NoteNumber: 62, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 150}, NoteNumber: 63, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 200}, NoteNumber: 64, Velocity: 100})

	// Test different ranges
	tests := []struct {
		start    int32
		end      int32
		expected int
	}{
		{0, 100, 2},    // Events at 0 and 50
		{50, 150, 2},   // Events at 50 and 100
		{100, 200, 2},  // Events at 100 and 150
		{0, 250, 5},    // All events
		{250, 300, 0},  // No events
		{-50, 0, 0},    // Before first event
	}

	for _, tt := range tests {
		events := q.GetEventsInRange(tt.start, tt.end)
		if len(events) != tt.expected {
			t.Errorf("Range [%d, %d): expected %d events, got %d", 
				tt.start, tt.end, tt.expected, len(events))
		}
	}
}

func TestRemoveProcessedEvents(t *testing.T) {
	q := NewEventQueue()

	// Add events
	for i := int32(0); i < 5; i++ {
		q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: i * 50}, NoteNumber: 60 + uint8(i), Velocity: 100})
	}

	// Remove events up to sample 125 (should remove first 3 events: 0, 50, 100)
	q.RemoveProcessedEvents(125)

	if q.Size() != 2 {
		t.Errorf("Expected 2 events remaining, got %d", q.Size())
	}

	remaining := q.GetAllEvents()
	if len(remaining) != 2 {
		t.Fatalf("Expected 2 remaining events, got %d", len(remaining))
	}

	// Check that remaining events are at offsets 150 and 200
	if remaining[0].SampleOffset() != 150 {
		t.Errorf("Expected first remaining event at offset 150, got %d", remaining[0].SampleOffset())
	}
	if remaining[1].SampleOffset() != 200 {
		t.Errorf("Expected second remaining event at offset 200, got %d", remaining[1].SampleOffset())
	}
}

func TestOffsetEvents(t *testing.T) {
	q := NewEventQueue()

	// Add various event types
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 100}, NoteNumber: 60, Velocity: 100})
	q.Add(NoteOffEvent{BaseEvent: BaseEvent{Offset: 200}, NoteNumber: 60, Velocity: 0})
	q.Add(ControlChangeEvent{BaseEvent: BaseEvent{Offset: 50}, Controller: CCSustain, Value: 127})
	q.Add(PitchBendEvent{BaseEvent: BaseEvent{Offset: 150}, Value: 1000})

	// Offset all events by 100
	q.OffsetEvents(100)

	events := q.GetAllEvents()
	expectedOffsets := []int32{150, 200, 250, 300} // Sorted order

	for i, event := range events {
		if event.SampleOffset() != expectedOffsets[i] {
			t.Errorf("Event %d: expected offset %d, got %d", 
				i, expectedOffsets[i], event.SampleOffset())
		}
	}
}

func TestEventBuffer(t *testing.T) {
	buffer := NewEventBuffer()

	// Test input events
	buffer.AddInputEvent(NoteOnEvent{BaseEvent: BaseEvent{Offset: 100}, NoteNumber: 60, Velocity: 100})
	buffer.AddInputEvent(NoteOffEvent{BaseEvent: BaseEvent{Offset: 200}, NoteNumber: 60, Velocity: 0})

	inputEvents := buffer.GetInputEvents(0, 300)
	if len(inputEvents) != 2 {
		t.Errorf("Expected 2 input events, got %d", len(inputEvents))
	}

	// Test output events
	buffer.AddOutputEvent(ControlChangeEvent{BaseEvent: BaseEvent{Offset: 50}, Controller: CCVolume, Value: 100})

	outputEvents := buffer.GetOutputEvents()
	if len(outputEvents) != 1 {
		t.Errorf("Expected 1 output event, got %d", len(outputEvents))
	}

	// Test clear functions
	buffer.ClearInput()
	if len(buffer.GetInputEvents(0, 1000)) != 0 {
		t.Error("Expected input queue to be empty after clear")
	}

	buffer.ClearOutput()
	if len(buffer.GetOutputEvents()) != 0 {
		t.Error("Expected output queue to be empty after clear")
	}
}

type testEventProcessor struct {
	processedEvents []Event
}

func (p *testEventProcessor) ProcessEvent(event Event) {
	p.processedEvents = append(p.processedEvents, event)
}

func TestProcessEvents(t *testing.T) {
	q := NewEventQueue()
	processor := &testEventProcessor{}

	// Add events
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 50}, NoteNumber: 60, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 150}, NoteNumber: 61, Velocity: 100})
	q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: 250}, NoteNumber: 62, Velocity: 100})

	// Process events in range [0, 200)
	q.ProcessEvents(processor, 0, 200)

	if len(processor.processedEvents) != 2 {
		t.Fatalf("Expected 2 processed events, got %d", len(processor.processedEvents))
	}

	// Verify correct events were processed
	if processor.processedEvents[0].SampleOffset() != 50 {
		t.Errorf("Expected first event at offset 50, got %d", processor.processedEvents[0].SampleOffset())
	}
	if processor.processedEvents[1].SampleOffset() != 150 {
		t.Errorf("Expected second event at offset 150, got %d", processor.processedEvents[1].SampleOffset())
	}
}

func TestConcurrentAccess(t *testing.T) {
	q := NewEventQueue()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			q.Add(NoteOnEvent{BaseEvent: BaseEvent{Offset: int32(i)}, NoteNumber: 60, Velocity: 100})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = q.GetEventsInRange(0, 100)
			_ = q.Size()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	if q.Size() != 100 {
		t.Errorf("Expected 100 events, got %d", q.Size())
	}
}