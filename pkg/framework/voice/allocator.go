package voice

import (
	"github.com/justyntemme/vst3go/pkg/midi"
)

// AllocationMode defines how voices are allocated
type AllocationMode int

const (
	// Poly mode - each note gets its own voice
	ModePoly AllocationMode = iota
	// Mono mode - only one voice active at a time
	ModeMono
	// Legato mode - mono with no retriggering on overlapping notes
	ModeLegato
	// Unison mode - all voices play the same note
	ModeUnison
)

// StealingMode defines how voices are stolen when all are in use
type StealingMode int

const (
	// StealOldest steals the oldest playing voice
	StealOldest StealingMode = iota
	// StealQuietest steals the voice with lowest amplitude
	StealQuietest
	// StealHighest steals the highest pitched voice
	StealHighest
	// StealLowest steals the lowest pitched voice
	StealLowest
	// StealNone doesn't steal - new notes are ignored when full
	StealNone
)

// Voice represents a single voice in the synthesizer
type Voice interface {
	// IsActive returns true if the voice is currently playing
	IsActive() bool
	// GetNote returns the MIDI note number this voice is playing
	GetNote() uint8
	// GetVelocity returns the velocity of the note
	GetVelocity() uint8
	// GetAmplitude returns the current amplitude (for steal quietest)
	GetAmplitude() float64
	// GetAge returns how long this voice has been playing (in samples)
	GetAge() int64
	// TriggerNote starts playing a note
	TriggerNote(note uint8, velocity uint8)
	// ReleaseNote releases the note
	ReleaseNote()
	// Stop immediately stops the voice
	Stop()
	// Process generates audio for this voice
	Process(output []float32)
}

// Allocator manages voice allocation for polyphonic synthesis
type Allocator struct {
	voices        []Voice
	mode          AllocationMode
	stealingMode  StealingMode
	maxVoices     int
	activeVoices  int
	noteToVoice   map[uint8][]int // Maps note number to voice indices
	lastTriggered int              // For round-robin allocation
	sustainPedal  bool
	sustainedNotes map[uint8]bool
	
	// Unison mode settings
	unisonDetune float64
	unisonSpread float64
	
	// Mono/Legato mode state
	currentNote  uint8
	previousNote uint8
	glideTime    float64
	glideActive  bool
}

// NewAllocator creates a new voice allocator
func NewAllocator(voices []Voice) *Allocator {
	return &Allocator{
		voices:         voices,
		mode:           ModePoly,
		stealingMode:   StealOldest,
		maxVoices:      len(voices),
		noteToVoice:    make(map[uint8][]int),
		sustainedNotes: make(map[uint8]bool),
	}
}

// SetMode sets the allocation mode
func (a *Allocator) SetMode(mode AllocationMode) {
	a.mode = mode
	// Reset all voices when changing mode
	a.Reset()
}

// SetStealingMode sets the voice stealing mode
func (a *Allocator) SetStealingMode(mode StealingMode) {
	a.stealingMode = mode
}

// SetMaxVoices sets the maximum number of active voices
func (a *Allocator) SetMaxVoices(max int) {
	if max > len(a.voices) {
		max = len(a.voices)
	}
	if max < 1 {
		max = 1
	}
	a.maxVoices = max
}

// SetUnisonDetune sets the detune amount for unison mode (in cents)
func (a *Allocator) SetUnisonDetune(cents float64) {
	a.unisonDetune = cents
}

// SetGlideTime sets the glide time for mono/legato modes (in seconds)
func (a *Allocator) SetGlideTime(seconds float64) {
	a.glideTime = seconds
}

// ProcessEvent handles a MIDI event
func (a *Allocator) ProcessEvent(event midi.Event) {
	switch e := event.(type) {
	case midi.NoteOnEvent:
		if e.Velocity > 0 {
			a.NoteOn(e.NoteNumber, e.Velocity)
		} else {
			// Note on with velocity 0 is treated as note off
			a.NoteOff(e.NoteNumber, 0)
		}
	case midi.NoteOffEvent:
		a.NoteOff(e.NoteNumber, e.Velocity)
	case midi.ControlChangeEvent:
		if e.Controller == midi.CCSustain {
			a.SetSustainPedal(e.Value >= 64)
		}
	}
}

// NoteOn handles a note on event
func (a *Allocator) NoteOn(note uint8, velocity uint8) {
	switch a.mode {
	case ModePoly:
		a.noteOnPoly(note, velocity)
	case ModeMono:
		a.noteOnMono(note, velocity)
	case ModeLegato:
		a.noteOnLegato(note, velocity)
	case ModeUnison:
		a.noteOnUnison(note, velocity)
	}
}

// NoteOff handles a note off event
func (a *Allocator) NoteOff(note uint8, velocity uint8) {
	if a.sustainPedal {
		// Mark note as sustained instead of releasing
		a.sustainedNotes[note] = true
		return
	}

	switch a.mode {
	case ModePoly:
		a.noteOffPoly(note, velocity)
	case ModeMono, ModeLegato:
		a.noteOffMono(note, velocity)
	case ModeUnison:
		a.noteOffUnison(note, velocity)
	}
}

// SetSustainPedal sets the sustain pedal state
func (a *Allocator) SetSustainPedal(on bool) {
	a.sustainPedal = on
	if !on {
		// Release all sustained notes
		for note := range a.sustainedNotes {
			a.NoteOff(note, 0)
		}
		a.sustainedNotes = make(map[uint8]bool)
	}
}

// Reset stops all voices and clears allocations
func (a *Allocator) Reset() {
	for _, voice := range a.voices {
		voice.Stop()
	}
	a.noteToVoice = make(map[uint8][]int)
	a.sustainedNotes = make(map[uint8]bool)
	a.sustainPedal = false
	a.activeVoices = 0
	a.currentNote = 0
	a.previousNote = 0
	a.glideActive = false
}

// GetActiveVoiceCount returns the number of active voices
func (a *Allocator) GetActiveVoiceCount() int {
	count := 0
	for _, voice := range a.voices[:a.maxVoices] {
		if voice.IsActive() {
			count++
		}
	}
	return count
}

// noteOnPoly handles poly mode note on
func (a *Allocator) noteOnPoly(note uint8, velocity uint8) {
	// Check if note is already playing
	if voices, exists := a.noteToVoice[note]; exists && len(voices) > 0 {
		// Retrigger the note on existing voice(s)
		for _, idx := range voices {
			a.voices[idx].TriggerNote(note, velocity)
		}
		return
	}

	// Find a free voice
	voiceIdx := a.findFreeVoice()
	if voiceIdx == -1 {
		// No free voice, try stealing
		voiceIdx = a.stealVoice()
		if voiceIdx == -1 {
			// Couldn't steal a voice
			return
		}
	}

	// Allocate the voice
	a.voices[voiceIdx].TriggerNote(note, velocity)
	a.noteToVoice[note] = []int{voiceIdx}
}

// noteOffPoly handles poly mode note off
func (a *Allocator) noteOffPoly(note uint8, velocity uint8) {
	if voices, exists := a.noteToVoice[note]; exists {
		for _, idx := range voices {
			a.voices[idx].ReleaseNote()
		}
		delete(a.noteToVoice, note)
	}
}

// noteOnMono handles mono mode note on
func (a *Allocator) noteOnMono(note uint8, velocity uint8) {
	// Stop all other voices
	for i := 0; i < a.maxVoices && i < 1; i++ {
		if a.voices[i].IsActive() {
			a.voices[i].Stop()
		}
	}
	
	a.previousNote = a.currentNote
	a.currentNote = note
	a.voices[0].TriggerNote(note, velocity)
	a.noteToVoice = map[uint8][]int{note: {0}}
}

// noteOnLegato handles legato mode note on
func (a *Allocator) noteOnLegato(note uint8, velocity uint8) {
	if a.currentNote == 0 {
		// First note, trigger normally
		a.noteOnMono(note, velocity)
	} else {
		// Legato transition - change pitch without retriggering
		a.previousNote = a.currentNote
		a.currentNote = note
		a.glideActive = true
		// Voice implementation should handle the pitch change
		a.noteToVoice = map[uint8][]int{note: {0}}
	}
}

// noteOffMono handles mono/legato mode note off
func (a *Allocator) noteOffMono(note uint8, velocity uint8) {
	if note == a.currentNote {
		a.voices[0].ReleaseNote()
		delete(a.noteToVoice, note)
		a.currentNote = 0
		a.glideActive = false
	}
}

// noteOnUnison handles unison mode note on
func (a *Allocator) noteOnUnison(note uint8, velocity uint8) {
	// Trigger all available voices with the same note
	for i := 0; i < a.maxVoices; i++ {
		a.voices[i].TriggerNote(note, velocity)
	}
	a.noteToVoice[note] = make([]int, a.maxVoices)
	for i := range a.noteToVoice[note] {
		a.noteToVoice[note][i] = i
	}
	a.currentNote = note
}

// noteOffUnison handles unison mode note off
func (a *Allocator) noteOffUnison(note uint8, velocity uint8) {
	if note == a.currentNote {
		for i := 0; i < a.maxVoices; i++ {
			a.voices[i].ReleaseNote()
		}
		delete(a.noteToVoice, note)
		a.currentNote = 0
	}
}

// findFreeVoice finds an inactive voice
func (a *Allocator) findFreeVoice() int {
	// Use round-robin to distribute voices evenly
	start := a.lastTriggered
	for i := 0; i < a.maxVoices; i++ {
		idx := (start + i + 1) % a.maxVoices
		if !a.voices[idx].IsActive() {
			a.lastTriggered = idx
			return idx
		}
	}
	return -1
}

// stealVoice steals a voice based on the stealing mode
func (a *Allocator) stealVoice() int {
	if a.stealingMode == StealNone {
		return -1
	}

	var bestIdx = -1
	var bestValue float64

	for i := 0; i < a.maxVoices; i++ {
		if !a.voices[i].IsActive() {
			continue
		}

		switch a.stealingMode {
		case StealOldest:
			age := float64(a.voices[i].GetAge())
			if bestIdx == -1 || age > bestValue {
				bestIdx = i
				bestValue = age
			}
		case StealQuietest:
			amp := a.voices[i].GetAmplitude()
			if bestIdx == -1 || amp < bestValue {
				bestIdx = i
				bestValue = amp
			}
		case StealHighest:
			note := float64(a.voices[i].GetNote())
			if bestIdx == -1 || note > bestValue {
				bestIdx = i
				bestValue = note
			}
		case StealLowest:
			note := float64(a.voices[i].GetNote())
			if bestIdx == -1 || note < bestValue {
				bestIdx = i
				bestValue = note
			}
		}
	}

	if bestIdx != -1 {
		// Remove the stolen voice from noteToVoice map
		stolenNote := a.voices[bestIdx].GetNote()
		if voices, exists := a.noteToVoice[stolenNote]; exists {
			for i, idx := range voices {
				if idx == bestIdx {
					// Remove this index from the slice
					a.noteToVoice[stolenNote] = append(voices[:i], voices[i+1:]...)
					if len(a.noteToVoice[stolenNote]) == 0 {
						delete(a.noteToVoice, stolenNote)
					}
					break
				}
			}
		}
		a.voices[bestIdx].Stop()
	}

	return bestIdx
}