package voice

import (
	"testing"

	"github.com/justyntemme/vst3go/pkg/midi"
)

// TestVoice is a simple voice implementation for testing
type TestVoice struct {
	active    bool
	note      uint8
	velocity  uint8
	amplitude float64
	age       int64
}

func (v *TestVoice) IsActive() bool         { return v.active }
func (v *TestVoice) GetNote() uint8         { return v.note }
func (v *TestVoice) GetVelocity() uint8     { return v.velocity }
func (v *TestVoice) GetAmplitude() float64  { return v.amplitude }
func (v *TestVoice) GetAge() int64          { return v.age }
func (v *TestVoice) TriggerNote(note uint8, velocity uint8) {
	v.active = true
	v.note = note
	v.velocity = velocity
	v.age = 0
	v.amplitude = float64(velocity) / 127.0
}
func (v *TestVoice) ReleaseNote() { v.active = false }
func (v *TestVoice) Stop()        { v.active = false; v.note = 0 }
func (v *TestVoice) Process(output []float32) {
	v.age++
	// Simulate amplitude decay
	if v.amplitude > 0.01 {
		v.amplitude *= 0.999
	}
}

func createTestVoices(count int) []Voice {
	voices := make([]Voice, count)
	for i := range voices {
		voices[i] = &TestVoice{}
	}
	return voices
}

func TestAllocatorPolyMode(t *testing.T) {
	voices := createTestVoices(4)
	allocator := NewAllocator(voices)
	allocator.SetMode(ModePoly)

	// Test basic note allocation
	allocator.NoteOn(60, 100)
	allocator.NoteOn(64, 100)
	allocator.NoteOn(67, 100)

	activeCount := allocator.GetActiveVoiceCount()
	if activeCount != 3 {
		t.Errorf("Expected 3 active voices, got %d", activeCount)
	}

	// Test note off
	allocator.NoteOff(64, 0)
	activeCount = allocator.GetActiveVoiceCount()
	if activeCount != 2 {
		t.Errorf("Expected 2 active voices after note off, got %d", activeCount)
	}

	// Test retriggering same note
	allocator.NoteOn(60, 80)
	
	// Find which voice is playing note 60
	var foundVoice *TestVoice
	for _, v := range voices {
		tv := v.(*TestVoice)
		if tv.IsActive() && tv.GetNote() == 60 {
			foundVoice = tv
			break
		}
	}
	
	if foundVoice == nil {
		t.Error("Note 60 should be playing after retrigger")
	} else if foundVoice.velocity != 80 {
		t.Errorf("Expected velocity 80 for retriggered note, got %d", foundVoice.velocity)
	}
}

func TestAllocatorMonoMode(t *testing.T) {
	voices := createTestVoices(4)
	allocator := NewAllocator(voices)
	allocator.SetMode(ModeMono)

	// Play multiple notes - only one should be active
	allocator.NoteOn(60, 100)
	allocator.NoteOn(64, 100)
	allocator.NoteOn(67, 100)

	activeCount := allocator.GetActiveVoiceCount()
	if activeCount != 1 {
		t.Errorf("Expected 1 active voice in mono mode, got %d", activeCount)
	}

	// Check that the last note is the active one
	v0 := voices[0].(*TestVoice)
	if v0.note != 67 {
		t.Errorf("Expected note 67 in mono mode, got %d", v0.note)
	}
}

func TestAllocatorLegatoMode(t *testing.T) {
	voices := createTestVoices(4)
	allocator := NewAllocator(voices)
	allocator.SetMode(ModeLegato)

	// First note should trigger
	allocator.NoteOn(60, 100)
	v0 := voices[0].(*TestVoice)
	if !v0.active || v0.note != 60 {
		t.Error("First note should trigger in legato mode")
	}

	// Second note should change pitch without retriggering
	v0.age = 100 // Simulate some age
	allocator.NoteOn(64, 100)
	
	if !v0.active {
		t.Error("Voice should remain active in legato mode")
	}
	if allocator.currentNote != 64 {
		t.Errorf("Expected current note 64, got %d", allocator.currentNote)
	}
}

func TestAllocatorUnisonMode(t *testing.T) {
	voices := createTestVoices(4)
	allocator := NewAllocator(voices)
	allocator.SetMode(ModeUnison)

	// Play a note - all voices should be active
	allocator.NoteOn(60, 100)

	activeCount := allocator.GetActiveVoiceCount()
	if activeCount != 4 {
		t.Errorf("Expected 4 active voices in unison mode, got %d", activeCount)
	}

	// Check all voices play the same note
	for i, v := range voices {
		tv := v.(*TestVoice)
		if tv.note != 60 {
			t.Errorf("Voice %d: expected note 60, got %d", i, tv.note)
		}
	}
}

func TestVoiceStealing(t *testing.T) {
	voices := createTestVoices(2)
	allocator := NewAllocator(voices)
	allocator.SetMode(ModePoly)

	// Fill all voices
	allocator.NoteOn(60, 100)
	allocator.NoteOn(64, 100)

	// Age the voices differently
	voices[0].(*TestVoice).age = 100
	voices[1].(*TestVoice).age = 50

	// Test steal oldest
	allocator.SetStealingMode(StealOldest)
	allocator.NoteOn(67, 100)

	// Voice 0 should have been stolen (oldest)
	v0 := voices[0].(*TestVoice)
	if v0.note != 67 {
		t.Errorf("Expected voice 0 to be stolen and play note 67, got %d", v0.note)
	}

	// Test steal quietest
	voices[0].(*TestVoice).amplitude = 0.1
	voices[1].(*TestVoice).amplitude = 0.5
	allocator.SetStealingMode(StealQuietest)
	allocator.NoteOn(70, 100)

	// Voice 0 should have been stolen (quietest)
	if v0.note != 70 {
		t.Errorf("Expected voice 0 to be stolen and play note 70, got %d", v0.note)
	}

	// Test steal none
	allocator.SetStealingMode(StealNone)
	allocator.NoteOn(72, 100)
	allocator.NoteOn(74, 100)

	// No new notes should have been allocated
	activeNotes := make(map[uint8]bool)
	for _, v := range voices {
		if v.IsActive() {
			activeNotes[v.GetNote()] = true
		}
	}
	if activeNotes[72] || activeNotes[74] {
		t.Error("StealNone mode should not allocate new notes when full")
	}
}

func TestSustainPedal(t *testing.T) {
	voices := createTestVoices(4)
	allocator := NewAllocator(voices)
	allocator.SetMode(ModePoly)

	// Play and hold some notes
	allocator.NoteOn(60, 100)
	allocator.NoteOn(64, 100)

	// Press sustain pedal
	allocator.SetSustainPedal(true)

	// Release notes - they should stay active
	allocator.NoteOff(60, 0)
	allocator.NoteOff(64, 0)

	activeCount := allocator.GetActiveVoiceCount()
	if activeCount != 2 {
		t.Errorf("Expected 2 active voices with sustain pedal, got %d", activeCount)
	}

	// Release sustain pedal - notes should stop
	allocator.SetSustainPedal(false)

	activeCount = allocator.GetActiveVoiceCount()
	if activeCount != 0 {
		t.Errorf("Expected 0 active voices after releasing sustain, got %d", activeCount)
	}
}

func TestProcessEvent(t *testing.T) {
	voices := createTestVoices(4)
	allocator := NewAllocator(voices)

	// Test note on event
	noteOn := midi.NoteOnEvent{
		BaseEvent:  midi.BaseEvent{EventChannel: 0, Offset: 0},
		NoteNumber: 60,
		Velocity:   100,
	}
	allocator.ProcessEvent(noteOn)

	if allocator.GetActiveVoiceCount() != 1 {
		t.Error("ProcessEvent should handle note on")
	}

	// Test note off event
	noteOff := midi.NoteOffEvent{
		BaseEvent:  midi.BaseEvent{EventChannel: 0, Offset: 100},
		NoteNumber: 60,
		Velocity:   0,
	}
	allocator.ProcessEvent(noteOff)

	if allocator.GetActiveVoiceCount() != 0 {
		t.Error("ProcessEvent should handle note off")
	}

	// Test sustain pedal
	sustainOn := midi.ControlChangeEvent{
		BaseEvent:  midi.BaseEvent{EventChannel: 0, Offset: 0},
		Controller: midi.CCSustain,
		Value:      127,
	}
	allocator.ProcessEvent(sustainOn)

	if !allocator.sustainPedal {
		t.Error("ProcessEvent should handle sustain pedal")
	}
}

func TestMaxVoices(t *testing.T) {
	voices := createTestVoices(8)
	allocator := NewAllocator(voices)
	allocator.SetMaxVoices(4)

	// Try to play more than max voices
	for i := uint8(60); i < 68; i++ {
		allocator.NoteOn(i, 100)
	}

	activeCount := allocator.GetActiveVoiceCount()
	if activeCount > 4 {
		t.Errorf("Should not exceed max voices (4), got %d", activeCount)
	}
}

func TestReset(t *testing.T) {
	voices := createTestVoices(4)
	allocator := NewAllocator(voices)

	// Play some notes
	allocator.NoteOn(60, 100)
	allocator.NoteOn(64, 100)
	allocator.SetSustainPedal(true)

	// Reset
	allocator.Reset()

	if allocator.GetActiveVoiceCount() != 0 {
		t.Error("Reset should stop all voices")
	}
	if allocator.sustainPedal {
		t.Error("Reset should clear sustain pedal")
	}
	if len(allocator.noteToVoice) != 0 {
		t.Error("Reset should clear note mappings")
	}
}