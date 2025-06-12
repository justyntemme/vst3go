package midi

import (
	"testing"
)

func TestNoteOnEvent(t *testing.T) {
	event := NoteOnEvent{
		BaseEvent: BaseEvent{
			EventChannel: 0,
			Offset:       100,
		},
		NoteNumber: 60, // Middle C
		Velocity:   64,
	}

	if event.Type() != EventTypeNoteOn {
		t.Errorf("Expected type %v, got %v", EventTypeNoteOn, event.Type())
	}

	if event.Channel() != 0 {
		t.Errorf("Expected channel 0, got %d", event.Channel())
	}

	if event.SampleOffset() != 100 {
		t.Errorf("Expected offset 100, got %d", event.SampleOffset())
	}

	expected := "NoteOn{ch:0, note:60, vel:64, offset:100}"
	if event.String() != expected {
		t.Errorf("Expected string %s, got %s", expected, event.String())
	}
}

func TestNoteOffEvent(t *testing.T) {
	event := NoteOffEvent{
		BaseEvent: BaseEvent{
			EventChannel: 1,
			Offset:       200,
		},
		NoteNumber: 72, // C5
		Velocity:   0,
	}

	if event.Type() != EventTypeNoteOff {
		t.Errorf("Expected type %v, got %v", EventTypeNoteOff, event.Type())
	}

	if event.Channel() != 1 {
		t.Errorf("Expected channel 1, got %d", event.Channel())
	}
}

func TestControlChangeEvent(t *testing.T) {
	event := ControlChangeEvent{
		BaseEvent: BaseEvent{
			EventChannel: 0,
			Offset:       50,
		},
		Controller: CCModWheel,
		Value:      100,
	}

	if event.Type() != EventTypeControlChange {
		t.Errorf("Expected type %v, got %v", EventTypeControlChange, event.Type())
	}

	expected := "CC{ch:0, ctrl:1, val:100, offset:50}"
	if event.String() != expected {
		t.Errorf("Expected string %s, got %s", expected, event.String())
	}
}

func TestPitchBendEvent(t *testing.T) {
	tests := []struct {
		value      int16
		normalized float64
	}{
		{0, 0.0},
		{8191, 0.999878}, // Close to 1.0
		{-8192, -1.0},
		{4096, 0.5},
		{-4096, -0.5},
	}

	for _, tt := range tests {
		event := PitchBendEvent{
			BaseEvent: BaseEvent{
				EventChannel: 0,
				Offset:       0,
			},
			Value: tt.value,
		}

		normalized := event.NormalizedValue()
		if diff := normalized - tt.normalized; diff > 0.01 && diff < -0.01 {
			t.Errorf("For value %d, expected normalized %f, got %f", tt.value, tt.normalized, normalized)
		}
	}
}

func TestNoteToFrequency(t *testing.T) {
	tests := []struct {
		note uint8
		freq float64
	}{
		{69, 440.0},    // A4
		{60, 261.63},   // Middle C (C4)
		{57, 220.0},    // A3
		{81, 880.0},    // A5
	}

	for _, tt := range tests {
		freq := NoteToFrequency(tt.note, 440.0)
		if diff := freq - tt.freq; diff > 0.1 && diff < -0.1 {
			t.Errorf("For note %d, expected frequency %f, got %f", tt.note, tt.freq, freq)
		}
	}
}

func TestNoteNumberToName(t *testing.T) {
	tests := []struct {
		note uint8
		name string
	}{
		{60, "C4"},  // Middle C
		{69, "A4"},  // A440
		{0, "C-1"},  // Lowest MIDI note
		{127, "G9"}, // Highest MIDI note
		{61, "C#4"}, // C# above middle C
		{70, "A#4"}, // A# above A4
	}

	for _, tt := range tests {
		name := NoteNumberToName(tt.note)
		if name != tt.name {
			t.Errorf("For note %d, expected name %s, got %s", tt.note, tt.name, name)
		}
	}
}

func TestEventInterface(t *testing.T) {
	events := []Event{
		NoteOnEvent{BaseEvent: BaseEvent{EventChannel: 0, Offset: 0}, NoteNumber: 60, Velocity: 100},
		NoteOffEvent{BaseEvent: BaseEvent{EventChannel: 0, Offset: 100}, NoteNumber: 60, Velocity: 0},
		ControlChangeEvent{BaseEvent: BaseEvent{EventChannel: 0, Offset: 200}, Controller: CCSustain, Value: 127},
		PitchBendEvent{BaseEvent: BaseEvent{EventChannel: 0, Offset: 300}, Value: 0},
	}

	for _, event := range events {
		// Ensure all events implement the interface
		_ = event.Type()
		_ = event.Channel()
		_ = event.SampleOffset()
		_ = event.String()
	}
}