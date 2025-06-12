// Package process provides audio processing context and utilities for VST3 audio processing.
package process

import (
	"fmt"
	"sort"

	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/midi"
)

// ParameterChange represents a parameter change at a specific sample offset
type ParameterChange struct {
	ParamID      uint32
	Value        float64
	SampleOffset int
}

// TransportInfo provides musical timing and transport state information
type TransportInfo struct {
	// Transport state
	IsPlaying   bool // Transport is playing
	IsRecording bool // Transport is recording
	IsCycling   bool // Loop/cycle is active

	// Tempo and timing
	Tempo              float64 // Current tempo in BPM (0 if not available)
	TimeSigNumerator   int32   // Time signature numerator (e.g., 4 for 4/4)
	TimeSigDenominator int32   // Time signature denominator (e.g., 4 for 4/4)

	// Musical position
	ProjectTimeMusic float64 // Musical position in quarter notes
	BarPositionMusic float64 // Bar position in quarter notes

	// Cycle/loop points
	CycleStartMusic float64 // Cycle start in quarter notes
	CycleEndMusic   float64 // Cycle end in quarter notes

	// Sample position
	ProjectTimeSamples    int64 // Project time in samples
	ContinuousTimeSamples int64 // Continuous time in samples (doesn't reset on loop)

	// Clock information
	SamplesToNextClock int32 // Samples until next clock/beat

	// Validity flags
	HasTempo         bool // Tempo field is valid
	HasTimeSignature bool // Time signature fields are valid
	HasMusicalTime   bool // Musical time fields are valid
	HasBarPosition   bool // Bar position is valid
	HasCycle         bool // Cycle points are valid
}

// GetBarsBeats returns the current position in bars and beats
func (t *TransportInfo) GetBarsBeats() (bars int, beats float64) {
	if !t.HasMusicalTime || !t.HasTimeSignature || t.TimeSigDenominator == 0 {
		return 0, 0
	}

	// Convert quarter notes to beats based on time signature
	beatsPerBar := float64(t.TimeSigNumerator)
	quarterNotesPerBeat := 4.0 / float64(t.TimeSigDenominator)
	totalBeats := t.ProjectTimeMusic / quarterNotesPerBeat

	bars = int(totalBeats / beatsPerBar)
	beats = totalBeats - float64(bars)*beatsPerBar

	return bars, beats
}

// GetSamplesPerBeat returns the number of samples per beat at current tempo
func (t *TransportInfo) GetSamplesPerBeat(sampleRate float64) float64 {
	if !t.HasTempo || t.Tempo <= 0 {
		return 0
	}

	// 60 seconds per minute / tempo = seconds per beat
	// seconds per beat * sample rate = samples per beat
	return (60.0 / t.Tempo) * sampleRate
}

// GetBeatPosition returns the current position within the current beat (0-1)
func (t *TransportInfo) GetBeatPosition() float64 {
	if !t.HasMusicalTime || !t.HasBarPosition {
		return 0
	}

	// Calculate position within current bar in quarter notes
	positionInBar := t.ProjectTimeMusic - t.BarPositionMusic

	// Convert to beat position based on time signature
	if t.HasTimeSignature && t.TimeSigDenominator > 0 {
		quarterNotesPerBeat := 4.0 / float64(t.TimeSigDenominator)
		beatPosition := (positionInBar / quarterNotesPerBeat)
		return beatPosition - float64(int(beatPosition)) // Get fractional part
	}

	// Default to quarter note = beat
	return positionInBar - float64(int(positionInBar))
}

// IsOnBeat returns true if the current position is at the start of a beat
func (t *TransportInfo) IsOnBeat(threshold float64) bool {
	beatPos := t.GetBeatPosition()
	return beatPos < threshold || beatPos > (1.0-threshold)
}

// Context provides a clean API for audio processing with zero allocations
type Context struct {
	Input      [][]float32
	Output     [][]float32
	SampleRate float64

	// Pre-allocated work buffers
	workBuffer []float32
	tempBuffer []float32

	// Parameter access
	params *param.Registry

	// Sample-accurate automation
	paramChanges []ParameterChange // Pre-allocated slice for parameter changes
	changeCount  int               // Number of active parameter changes

	// Transport and timing information
	Transport *TransportInfo

	// MIDI event processing
	eventBuffer *midi.EventBuffer
}

// NewContext creates a new process context with pre-allocated buffers
func NewContext(maxBlockSize int, params *param.Registry) *Context {
	return &Context{
		workBuffer:   make([]float32, maxBlockSize),
		tempBuffer:   make([]float32, maxBlockSize),
		params:       params,
		paramChanges: make([]ParameterChange, 128), // Pre-allocate space for parameter changes
		changeCount:  0,
		Transport:    &TransportInfo{}, // Initialize transport info
		eventBuffer:  midi.NewEventBuffer(),
	}
}

// Param returns the current value of a parameter (0-1 normalized)
func (c *Context) Param(id uint32) float64 {
	if p := c.params.Get(id); p != nil {
		return p.GetValue()
	}
	return 0
}

// ParamPlain returns the current plain value of a parameter
func (c *Context) ParamPlain(id uint32) float64 {
	if p := c.params.Get(id); p != nil {
		return p.GetPlainValue()
	}
	return 0
}

// NumSamples returns the number of samples to process
func (c *Context) NumSamples() int {
	if len(c.Input) > 0 && len(c.Input[0]) > 0 {
		return len(c.Input[0])
	}
	if len(c.Output) > 0 && len(c.Output[0]) > 0 {
		return len(c.Output[0])
	}
	return 0
}

// NumInputChannels returns the number of input channels
func (c *Context) NumInputChannels() int {
	return len(c.Input)
}

// NumOutputChannels returns the number of output channels
func (c *Context) NumOutputChannels() int {
	return len(c.Output)
}

// WorkBuffer returns a slice of the pre-allocated work buffer
// sized to the current block size - no allocation!
func (c *Context) WorkBuffer() []float32 {
	return c.workBuffer[:c.NumSamples()]
}

// TempBuffer returns a slice of the pre-allocated temp buffer
// sized to the current block size - no allocation!
func (c *Context) TempBuffer() []float32 {
	return c.tempBuffer[:c.NumSamples()]
}

// PassThrough copies input to output (for bypass)
func (c *Context) PassThrough() {
	numChannels := c.NumInputChannels()
	if c.NumOutputChannels() < numChannels {
		numChannels = c.NumOutputChannels()
	}

	for ch := 0; ch < numChannels; ch++ {
		copy(c.Output[ch], c.Input[ch])
	}
}

// Clear zeros the output buffers
func (c *Context) Clear() {
	for ch := range c.Output {
		for i := range c.Output[ch] {
			c.Output[ch][i] = 0
		}
	}
}

// SetParameterAtOffset sets a parameter value at a specific sample offset within the current block
// Deprecated: Use AddParameterChange for sample-accurate automation
func (c *Context) SetParameterAtOffset(paramID uint32, value float64, sampleOffset int) {
	if param := c.params.Get(paramID); param != nil {
		// For backward compatibility, apply the change immediately
		param.SetValue(value)

		// Debug output for parameter changes
		fmt.Printf("[PARAM_AUTOMATION] SetParameterAtOffset: id=%d, value=%.6f, offset=%d, plain=%.1f\n",
			paramID, value, sampleOffset, param.GetPlainValue())
	}
}

// AddParameterChange adds a parameter change for sample-accurate processing
// This method is used during the parameter change collection phase
func (c *Context) AddParameterChange(paramID uint32, value float64, sampleOffset int) {
	// Ensure we don't exceed pre-allocated space
	if c.changeCount < len(c.paramChanges) {
		c.paramChanges[c.changeCount] = ParameterChange{
			ParamID:      paramID,
			Value:        value,
			SampleOffset: sampleOffset,
		}
		c.changeCount++
	}
}

// ResetParameterChanges clears the parameter change list for the next processing block
func (c *Context) ResetParameterChanges() {
	c.changeCount = 0
}

// SortParameterChanges sorts parameter changes by sample offset for processing
func (c *Context) SortParameterChanges() {
	if c.changeCount > 1 {
		// Sort only the active portion of the slice
		sort.Slice(c.paramChanges[:c.changeCount], func(i, j int) bool {
			return c.paramChanges[i].SampleOffset < c.paramChanges[j].SampleOffset
		})
	}
}

// GetParameterChanges returns the active parameter changes for this block
func (c *Context) GetParameterChanges() []ParameterChange {
	return c.paramChanges[:c.changeCount]
}

// HasParameterChanges returns true if there are parameter changes in this block
func (c *Context) HasParameterChanges() bool {
	return c.changeCount > 0
}

// ApplyParameterChange applies a parameter change immediately
func (c *Context) ApplyParameterChange(change ParameterChange) {
	if param := c.params.Get(change.ParamID); param != nil {
		param.SetValue(change.Value)
	}
}

// Event processing methods

// AddInputEvent adds a MIDI event to the input queue
func (c *Context) AddInputEvent(event midi.Event) {
	c.eventBuffer.AddInputEvent(event)
}

// AddOutputEvent adds a MIDI event to the output queue
func (c *Context) AddOutputEvent(event midi.Event) {
	c.eventBuffer.AddOutputEvent(event)
}

// GetInputEvents returns input events in the specified sample range
func (c *Context) GetInputEvents(startSample, endSample int32) []midi.Event {
	return c.eventBuffer.GetInputEvents(startSample, endSample)
}

// GetAllInputEvents returns all input events for the current block
func (c *Context) GetAllInputEvents() []midi.Event {
	return c.eventBuffer.GetInputEvents(0, int32(c.NumSamples()))
}

// GetOutputEvents returns all output events generated during processing
func (c *Context) GetOutputEvents() []midi.Event {
	return c.eventBuffer.GetOutputEvents()
}

// ClearInputEvents clears the input event queue
func (c *Context) ClearInputEvents() {
	c.eventBuffer.ClearInput()
}

// ClearOutputEvents clears the output event queue
func (c *Context) ClearOutputEvents() {
	c.eventBuffer.ClearOutput()
}

// ClearAllEvents clears both input and output event queues
func (c *Context) ClearAllEvents() {
	c.eventBuffer.ClearAll()
}

// ProcessEvents processes events through an event processor for sample-accurate processing
func (c *Context) ProcessEvents(processor midi.EventProcessor, startSample, endSample int32) {
	events := c.GetInputEvents(startSample, endSample)
	for _, event := range events {
		processor.ProcessEvent(event)
	}
}

// HasInputEvents returns true if there are input events in the current block
func (c *Context) HasInputEvents() bool {
	events := c.GetAllInputEvents()
	return len(events) > 0
}
