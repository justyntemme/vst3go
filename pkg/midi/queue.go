package midi

import (
	"sort"
	"sync"
)

type EventQueue struct {
	events []Event
	mu     sync.RWMutex
	sorted bool
}

func NewEventQueue() *EventQueue {
	return &EventQueue{
		events: make([]Event, 0, 128),
		sorted: true,
	}
}

func (q *EventQueue) Add(event Event) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.events = append(q.events, event)
	q.sorted = false
}

func (q *EventQueue) AddMultiple(events []Event) {
	if len(events) == 0 {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.events = append(q.events, events...)
	q.sorted = false
}

func (q *EventQueue) GetEventsInRange(startSample, endSample int32) []Event {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.sorted {
		q.mu.RUnlock()
		q.mu.Lock()
		q.sortEvents()
		q.mu.Unlock()
		q.mu.RLock()
	}

	if len(q.events) == 0 {
		return nil
	}

	// Binary search for start position
	startIdx := sort.Search(len(q.events), func(i int) bool {
		return q.events[i].SampleOffset() >= startSample
	})

	if startIdx >= len(q.events) {
		return nil
	}

	// Find end position
	endIdx := startIdx
	for endIdx < len(q.events) && q.events[endIdx].SampleOffset() < endSample {
		endIdx++
	}

	if startIdx == endIdx {
		return nil
	}

	// Return a copy of the relevant events
	result := make([]Event, endIdx-startIdx)
	copy(result, q.events[startIdx:endIdx])
	return result
}

func (q *EventQueue) GetAllEvents() []Event {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.sorted {
		q.mu.RUnlock()
		q.mu.Lock()
		q.sortEvents()
		q.mu.Unlock()
		q.mu.RLock()
	}

	result := make([]Event, len(q.events))
	copy(result, q.events)
	return result
}

func (q *EventQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.events = q.events[:0]
	q.sorted = true
}

func (q *EventQueue) RemoveProcessedEvents(upToSample int32) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.sorted {
		q.sortEvents()
	}

	// Find the first event that should be kept
	keepIdx := sort.Search(len(q.events), func(i int) bool {
		return q.events[i].SampleOffset() > upToSample
	})

	if keepIdx > 0 {
		// Remove processed events
		copy(q.events, q.events[keepIdx:])
		q.events = q.events[:len(q.events)-keepIdx]
	}
}

func (q *EventQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.events)
}

func (q *EventQueue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.events) == 0
}

func (q *EventQueue) sortEvents() {
	sort.SliceStable(q.events, func(i, j int) bool {
		return q.events[i].SampleOffset() < q.events[j].SampleOffset()
	})
	q.sorted = true
}

func (q *EventQueue) OffsetEvents(offset int32) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.events {
		switch e := q.events[i].(type) {
		case NoteOnEvent:
			e.Offset += offset
			q.events[i] = e
		case NoteOffEvent:
			e.Offset += offset
			q.events[i] = e
		case ControlChangeEvent:
			e.Offset += offset
			q.events[i] = e
		case PitchBendEvent:
			e.Offset += offset
			q.events[i] = e
		case PolyPressureEvent:
			e.Offset += offset
			q.events[i] = e
		case ChannelPressureEvent:
			e.Offset += offset
			q.events[i] = e
		case ProgramChangeEvent:
			e.Offset += offset
			q.events[i] = e
		case ClockEvent:
			e.Offset += offset
			q.events[i] = e
		case StartEvent:
			e.Offset += offset
			q.events[i] = e
		case StopEvent:
			e.Offset += offset
			q.events[i] = e
		case ContinueEvent:
			e.Offset += offset
			q.events[i] = e
		}
	}
}

type EventProcessor interface {
	ProcessEvent(event Event)
}

func (q *EventQueue) ProcessEvents(processor EventProcessor, startSample, endSample int32) {
	events := q.GetEventsInRange(startSample, endSample)
	for _, event := range events {
		processor.ProcessEvent(event)
	}
}

type EventBuffer struct {
	inputQueue  *EventQueue
	outputQueue *EventQueue
}

func NewEventBuffer() *EventBuffer {
	return &EventBuffer{
		inputQueue:  NewEventQueue(),
		outputQueue: NewEventQueue(),
	}
}

func (b *EventBuffer) AddInputEvent(event Event) {
	b.inputQueue.Add(event)
}

func (b *EventBuffer) AddOutputEvent(event Event) {
	b.outputQueue.Add(event)
}

func (b *EventBuffer) GetInputEvents(startSample, endSample int32) []Event {
	return b.inputQueue.GetEventsInRange(startSample, endSample)
}

func (b *EventBuffer) GetOutputEvents() []Event {
	return b.outputQueue.GetAllEvents()
}

func (b *EventBuffer) ClearInput() {
	b.inputQueue.Clear()
}

func (b *EventBuffer) ClearOutput() {
	b.outputQueue.Clear()
}

func (b *EventBuffer) ClearAll() {
	b.inputQueue.Clear()
	b.outputQueue.Clear()
}