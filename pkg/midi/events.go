package midi

import (
	"fmt"
)

type EventType uint8

const (
	EventTypeNoteOff EventType = iota
	EventTypeNoteOn
	EventTypePolyPressure
	EventTypeControlChange
	EventTypeProgramChange
	EventTypeChannelPressure
	EventTypePitchBend
	EventTypeSystemExclusive
	EventTypeClock
	EventTypeStart
	EventTypeStop
	EventTypeContinue
	EventTypeReset
	EventTypeActiveSensing
)

type Event interface {
	Type() EventType
	Channel() uint8
	SampleOffset() int32
	String() string
}

type BaseEvent struct {
	EventChannel uint8
	Offset       int32
}

func (e BaseEvent) Channel() uint8 {
	return e.EventChannel
}

func (e BaseEvent) SampleOffset() int32 {
	return e.Offset
}

type NoteOnEvent struct {
	BaseEvent
	NoteNumber uint8
	Velocity   uint8
}

func (e NoteOnEvent) Type() EventType {
	return EventTypeNoteOn
}

func (e NoteOnEvent) String() string {
	return fmt.Sprintf("NoteOn{ch:%d, note:%d, vel:%d, offset:%d}", 
		e.EventChannel, e.NoteNumber, e.Velocity, e.Offset)
}

type NoteOffEvent struct {
	BaseEvent
	NoteNumber uint8
	Velocity   uint8
}

func (e NoteOffEvent) Type() EventType {
	return EventTypeNoteOff
}

func (e NoteOffEvent) String() string {
	return fmt.Sprintf("NoteOff{ch:%d, note:%d, vel:%d, offset:%d}", 
		e.EventChannel, e.NoteNumber, e.Velocity, e.Offset)
}

type ControlChangeEvent struct {
	BaseEvent
	Controller uint8
	Value      uint8
}

func (e ControlChangeEvent) Type() EventType {
	return EventTypeControlChange
}

func (e ControlChangeEvent) String() string {
	return fmt.Sprintf("CC{ch:%d, ctrl:%d, val:%d, offset:%d}", 
		e.EventChannel, e.Controller, e.Value, e.Offset)
}

const (
	CCModWheel       uint8 = 1
	CCBreath         uint8 = 2
	CCFoot           uint8 = 4
	CCPortamentoTime uint8 = 5
	CCVolume         uint8 = 7
	CCBalance        uint8 = 8
	CCPan            uint8 = 10
	CCExpression     uint8 = 11
	CCSustain        uint8 = 64
	CCPortamento     uint8 = 65
	CCSostenuto      uint8 = 66
	CCSoft           uint8 = 67
	CCLegato         uint8 = 68
	CCHold2          uint8 = 69
	CCAllSoundOff    uint8 = 120
	CCResetAll       uint8 = 121
	CCLocalControl   uint8 = 122
	CCAllNotesOff    uint8 = 123
)

type PitchBendEvent struct {
	BaseEvent
	Value int16 // -8192 to 8191, 0 is center
}

func (e PitchBendEvent) Type() EventType {
	return EventTypePitchBend
}

func (e PitchBendEvent) String() string {
	return fmt.Sprintf("PitchBend{ch:%d, val:%d, offset:%d}", 
		e.EventChannel, e.Value, e.Offset)
}

func (e PitchBendEvent) NormalizedValue() float64 {
	return float64(e.Value) / 8192.0
}

type PolyPressureEvent struct {
	BaseEvent
	NoteNumber uint8
	Pressure   uint8
}

func (e PolyPressureEvent) Type() EventType {
	return EventTypePolyPressure
}

func (e PolyPressureEvent) String() string {
	return fmt.Sprintf("PolyPressure{ch:%d, note:%d, pressure:%d, offset:%d}", 
		e.EventChannel, e.NoteNumber, e.Pressure, e.Offset)
}

type ChannelPressureEvent struct {
	BaseEvent
	Pressure uint8
}

func (e ChannelPressureEvent) Type() EventType {
	return EventTypeChannelPressure
}

func (e ChannelPressureEvent) String() string {
	return fmt.Sprintf("ChannelPressure{ch:%d, pressure:%d, offset:%d}", 
		e.EventChannel, e.Pressure, e.Offset)
}

type ProgramChangeEvent struct {
	BaseEvent
	Program uint8
}

func (e ProgramChangeEvent) Type() EventType {
	return EventTypeProgramChange
}

func (e ProgramChangeEvent) String() string {
	return fmt.Sprintf("ProgramChange{ch:%d, prog:%d, offset:%d}", 
		e.EventChannel, e.Program, e.Offset)
}

type ClockEvent struct {
	BaseEvent
}

func (e ClockEvent) Type() EventType {
	return EventTypeClock
}

func (e ClockEvent) String() string {
	return fmt.Sprintf("Clock{offset:%d}", e.Offset)
}

type StartEvent struct {
	BaseEvent
}

func (e StartEvent) Type() EventType {
	return EventTypeStart
}

func (e StartEvent) String() string {
	return fmt.Sprintf("Start{offset:%d}", e.Offset)
}

type StopEvent struct {
	BaseEvent
}

func (e StopEvent) Type() EventType {
	return EventTypeStop
}

func (e StopEvent) String() string {
	return fmt.Sprintf("Stop{offset:%d}", e.Offset)
}

type ContinueEvent struct {
	BaseEvent
}

func (e ContinueEvent) Type() EventType {
	return EventTypeContinue
}

func (e ContinueEvent) String() string {
	return fmt.Sprintf("Continue{offset:%d}", e.Offset)
}

func NoteToFrequency(note uint8, tuningA4 float64) float64 {
	if tuningA4 == 0 {
		tuningA4 = 440.0
	}
	return tuningA4 * pow2((float64(note) - 69.0) / 12.0)
}

func pow2(x float64) float64 {
	// Fast approximation of 2^x
	if x >= 0 {
		whole := int(x)
		frac := x - float64(whole)
		// 2^whole * 2^frac
		// Use Taylor series approximation for fractional part
		fracPow := 1.0 + frac*(0.693147 + frac*(0.240227 + frac*0.055504))
		return float64(uint64(1)<<uint(whole)) * fracPow
	} else {
		// For negative x, use 2^x = 1 / 2^(-x)
		return 1.0 / pow2(-x)
	}
}

func FrequencyToNote(freq, tuningA4 float64) uint8 {
	if tuningA4 == 0 {
		tuningA4 = 440.0
	}
	note := 69.0 + 12.0*log2(freq/tuningA4)
	if note < 0 {
		return 0
	}
	if note > 127 {
		return 127
	}
	return uint8(note + 0.5)
}

func log2(x float64) float64 {
	// Fast approximation of log2(x)
	if x <= 0 {
		return -1000.0 // Return a very negative number for invalid input
	}
	
	// Normalize x to [1, 2) range
	exp := 0
	for x >= 2.0 {
		x /= 2.0
		exp++
	}
	for x < 1.0 {
		x *= 2.0
		exp--
	}
	
	// Now x is in [1, 2), use polynomial approximation
	// log2(x) ≈ (x-1) * (1.4427 - 0.7213*(x-1) + 0.4821*(x-1)^2)
	t := x - 1.0
	frac := t * (1.4427 - t*(0.7213 - t*0.4821))
	
	return float64(exp) + frac
}

func NoteNumberToName(note uint8) string {
	noteNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	octave := int(note / 12) - 1
	noteName := noteNames[note % 12]
	return fmt.Sprintf("%s%d", noteName, octave)
}