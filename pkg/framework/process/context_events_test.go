package process

import (
	"testing"

	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/midi"
)

func TestContextEventProcessing(t *testing.T) {
	registry := param.NewRegistry()
	ctx := NewContext(512, registry)

	// Set up input/output buffers first
	ctx.Input = [][]float32{make([]float32, 512)}
	ctx.Output = [][]float32{make([]float32, 512)}

	// Test adding input events
	noteOn := midi.NoteOnEvent{
		BaseEvent:  midi.BaseEvent{EventChannel: 0, Offset: 100},
		NoteNumber: 60,
		Velocity:   100,
	}
	noteOff := midi.NoteOffEvent{
		BaseEvent:  midi.BaseEvent{EventChannel: 0, Offset: 200},
		NoteNumber: 60,
		Velocity:   0,
	}

	ctx.AddInputEvent(noteOn)
	ctx.AddInputEvent(noteOff)

	// Test getting all input events
	events := ctx.GetAllInputEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 input events, got %d", len(events))
	}

	// Test HasInputEvents
	if !ctx.HasInputEvents() {
		t.Error("Expected HasInputEvents to return true")
	}

	// Test getting events in range
	events = ctx.GetInputEvents(0, 150)
	if len(events) != 1 {
		t.Errorf("Expected 1 event in range [0, 150), got %d", len(events))
	}

	events = ctx.GetInputEvents(150, 300)
	if len(events) != 1 {
		t.Errorf("Expected 1 event in range [150, 300), got %d", len(events))
	}

	// Test clearing input events
	ctx.ClearInputEvents()
	if ctx.HasInputEvents() {
		t.Error("Expected no input events after clear")
	}
}

func TestContextOutputEvents(t *testing.T) {
	registry := param.NewRegistry()
	ctx := NewContext(512, registry)

	// Add output events
	cc := midi.ControlChangeEvent{
		BaseEvent:  midi.BaseEvent{EventChannel: 0, Offset: 50},
		Controller: midi.CCVolume,
		Value:      100,
	}
	pitchBend := midi.PitchBendEvent{
		BaseEvent: midi.BaseEvent{EventChannel: 0, Offset: 75},
		Value:     1000,
	}

	ctx.AddOutputEvent(cc)
	ctx.AddOutputEvent(pitchBend)

	// Get output events
	outputEvents := ctx.GetOutputEvents()
	if len(outputEvents) != 2 {
		t.Errorf("Expected 2 output events, got %d", len(outputEvents))
	}

	// Clear output events
	ctx.ClearOutputEvents()
	outputEvents = ctx.GetOutputEvents()
	if len(outputEvents) != 0 {
		t.Error("Expected no output events after clear")
	}
}

type testProcessor struct {
	receivedEvents []midi.Event
}

func (p *testProcessor) ProcessEvent(event midi.Event) {
	p.receivedEvents = append(p.receivedEvents, event)
}

func TestContextProcessEvents(t *testing.T) {
	registry := param.NewRegistry()
	ctx := NewContext(512, registry)
	ctx.Input = [][]float32{make([]float32, 512)}

	// Add events at different offsets
	events := []midi.Event{
		midi.NoteOnEvent{BaseEvent: midi.BaseEvent{Offset: 50}, NoteNumber: 60, Velocity: 100},
		midi.NoteOnEvent{BaseEvent: midi.BaseEvent{Offset: 150}, NoteNumber: 61, Velocity: 100},
		midi.NoteOnEvent{BaseEvent: midi.BaseEvent{Offset: 250}, NoteNumber: 62, Velocity: 100},
		midi.NoteOnEvent{BaseEvent: midi.BaseEvent{Offset: 350}, NoteNumber: 63, Velocity: 100},
	}

	for _, e := range events {
		ctx.AddInputEvent(e)
	}

	// Process events in chunks
	processor := &testProcessor{}
	
	// Process first chunk [0, 200)
	ctx.ProcessEvents(processor, 0, 200)
	if len(processor.receivedEvents) != 2 {
		t.Errorf("Expected 2 events in first chunk, got %d", len(processor.receivedEvents))
	}

	// Process second chunk [200, 400)
	processor.receivedEvents = nil
	ctx.ProcessEvents(processor, 200, 400)
	if len(processor.receivedEvents) != 2 {
		t.Errorf("Expected 2 events in second chunk, got %d", len(processor.receivedEvents))
	}
}

func TestContextClearAllEvents(t *testing.T) {
	registry := param.NewRegistry()
	ctx := NewContext(512, registry)

	// Add both input and output events
	ctx.AddInputEvent(midi.NoteOnEvent{BaseEvent: midi.BaseEvent{Offset: 100}, NoteNumber: 60, Velocity: 100})
	ctx.AddOutputEvent(midi.ControlChangeEvent{BaseEvent: midi.BaseEvent{Offset: 50}, Controller: midi.CCVolume, Value: 100})

	// Clear all events
	ctx.ClearAllEvents()

	if ctx.HasInputEvents() {
		t.Error("Expected no input events after ClearAllEvents")
	}

	if len(ctx.GetOutputEvents()) != 0 {
		t.Error("Expected no output events after ClearAllEvents")
	}
}