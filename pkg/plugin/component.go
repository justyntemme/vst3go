package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include "../../bridge/bridge.h"
import "C"
import (
	"bytes"
	"fmt"
	"sync"
	"unsafe"

	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	"github.com/justyntemme/vst3go/pkg/framework/state"
	"github.com/justyntemme/vst3go/pkg/vst3"
)

// componentImpl wraps a Processor to implement VST3 interfaces
type componentImpl struct {
	processor    Processor
	processCtx   *process.Context
	sampleRate   float64
	maxBlockSize int32
	active       bool
	processing   bool
	mu           sync.RWMutex
	wrapper      *componentWrapper // Reference to wrapper for notifications
}

// newComponent creates a new component implementation
func newComponent(processor Processor) *componentImpl {
	params := processor.GetParameters()
	return &componentImpl{
		processor:    processor,
		processCtx:   process.NewContext(8192, params), // Default max block size
		maxBlockSize: 8192,
	}
}

// IComponent implementation
func (c *componentImpl) Initialize(_ interface{}) error {
	return c.processor.Initialize(48000, c.maxBlockSize) // Default sample rate
}

func (c *componentImpl) Terminate() error {
	return nil
}

func (c *componentImpl) GetControllerClassID() [16]byte {
	// Return same ID - we're a single component
	return [16]byte{}
}

func (c *componentImpl) SetIOMode(_ int32) error {
	return nil
}

func (c *componentImpl) GetBusCount(mediaType, direction int32) int32 {
	buses := c.processor.GetBuses()
	return buses.GetBusCount(bus.MediaType(mediaType), bus.Direction(direction))
}

func (c *componentImpl) GetBusInfo(mediaType, direction, index int32) (*vst3.BusInfo, error) {
	buses := c.processor.GetBuses()
	info := buses.GetBusInfo(bus.MediaType(mediaType), bus.Direction(direction), index)
	if info == nil {
		return nil, vst3.ErrNotImplemented
	}

	flags := uint32(1) // Default active
	if !info.IsActive {
		flags = 0
	}

	return &vst3.BusInfo{
		MediaType:    int32(info.MediaType),
		Direction:    int32(info.Direction),
		ChannelCount: info.ChannelCount,
		Name:         info.Name,
		BusType:      int32(info.BusType),
		Flags:        flags,
	}, nil
}

func (c *componentImpl) GetRoutingInfo(inInfo, outInfo interface{}) error {
	return vst3.ErrNotImplemented
}

func (c *componentImpl) ActivateBus(mediaType, direction, index int32, state bool) error {
	return nil
}

func (c *componentImpl) SetActive(active bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.active = active
	return c.processor.SetActive(active)
}

func (c *componentImpl) SetState(stateData []byte) error {
	if c.processor == nil {
		return fmt.Errorf("no processor available")
	}

	// Get parameter registry from processor
	params := c.processor.GetParameters()
	if params == nil {
		return fmt.Errorf("no parameters available")
	}

	// Create state manager and configure custom state handling
	stateManager := state.NewManager(params)

	// Check if processor implements StatefulProcessor
	if stateful, ok := c.processor.(StatefulProcessor); ok {
		stateManager.SetCustomLoadFunc(stateful.LoadCustomState)
	}

	buf := bytes.NewReader(stateData)
	return stateManager.Load(buf)
}

func (c *componentImpl) GetState() ([]byte, error) {
	if c.processor == nil {
		return nil, fmt.Errorf("no processor available")
	}

	// Get parameter registry from processor
	params := c.processor.GetParameters()
	if params == nil {
		return nil, fmt.Errorf("no parameters available")
	}

	// Create state manager and configure custom state handling
	stateManager := state.NewManager(params)

	// Check if processor implements StatefulProcessor
	if stateful, ok := c.processor.(StatefulProcessor); ok {
		stateManager.SetCustomSaveFunc(stateful.SaveCustomState)
	}

	var buf bytes.Buffer
	if err := stateManager.Save(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// IAudioProcessor implementation
func (c *componentImpl) SetBusArrangements(inputs, outputs []int64) error {
	return nil
}

func (c *componentImpl) GetBusArrangement(direction, index int32) (int64, error) {
	// Return stereo by default
	return int64(3), nil // Left + Right
}

func (c *componentImpl) CanProcessSampleSize(symbolicSampleSize int32) error {
	// We only support 32-bit float
	if symbolicSampleSize == 0 { // kSample32
		return nil
	}
	return vst3.ErrNotImplemented
}

func (c *componentImpl) GetLatencySamples() uint32 {
	return uint32(c.processor.GetLatencySamples())
}

func (c *componentImpl) SetupProcessing(setup *vst3.ProcessSetup) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sampleRate = setup.SampleRate
	if setup.MaxSamplesPerBlock > 0 {
		c.maxBlockSize = setup.MaxSamplesPerBlock
		// Recreate process context with new max block size
		params := c.processor.GetParameters()
		c.processCtx = process.NewContext(int(c.maxBlockSize), params)
	}

	return c.processor.Initialize(c.sampleRate, c.maxBlockSize)
}

func (c *componentImpl) SetProcessing(state bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processing = state
	return nil
}

func (c *componentImpl) Process(data unsafe.Pointer) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.processing {
		return nil
	}

	// Get raw process data struct
	processData := (*C.struct_Steinberg_Vst_ProcessData)(data)

	// Update context with current buffers
	c.processCtx.SampleRate = c.sampleRate

	// Update transport information if available
	if processData.processContext != nil {
		ctx := processData.processContext
		transport := c.processCtx.Transport

		// Transport state
		transport.IsPlaying = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kPlaying) != 0
		transport.IsRecording = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kRecording) != 0
		transport.IsCycling = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kCycleActive) != 0

		// Tempo
		transport.HasTempo = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kTempoValid) != 0
		if transport.HasTempo {
			transport.Tempo = float64(ctx.tempo)
		}

		// Time signature
		transport.HasTimeSignature = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kTimeSigValid) != 0
		if transport.HasTimeSignature {
			transport.TimeSigNumerator = int32(ctx.timeSigNumerator)
			transport.TimeSigDenominator = int32(ctx.timeSigDenominator)
		}

		// Musical time
		transport.HasMusicalTime = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kProjectTimeMusicValid) != 0
		if transport.HasMusicalTime {
			transport.ProjectTimeMusic = float64(ctx.projectTimeMusic)
		}

		// Bar position
		transport.HasBarPosition = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kBarPositionValid) != 0
		if transport.HasBarPosition {
			transport.BarPositionMusic = float64(ctx.barPositionMusic)
		}

		// Cycle points
		transport.HasCycle = (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kCycleValid) != 0
		if transport.HasCycle {
			transport.CycleStartMusic = float64(ctx.cycleStartMusic)
			transport.CycleEndMusic = float64(ctx.cycleEndMusic)
		}

		// Sample positions
		transport.ProjectTimeSamples = int64(ctx.projectTimeSamples)
		transport.ContinuousTimeSamples = int64(ctx.continousTimeSamples)

		// Clock
		if (ctx.state & C.Steinberg_Vst_ProcessContext_StatesAndFlags_kClockValid) != 0 {
			transport.SamplesToNextClock = int32(ctx.samplesToNextClock)
		}
	}

	// Set input/output buffers (slicing pre-allocated arrays, no allocation)
	numSamples := int(processData.numSamples)

	// Clear slices (no allocation, just updating slice headers)
	c.processCtx.Input = c.processCtx.Input[:0]
	c.processCtx.Output = c.processCtx.Output[:0]

	// Map input buffers
	if processData.numInputs > 0 && processData.inputs != nil {
		inputBuses := (*[1]C.struct_Steinberg_Vst_AudioBusBuffers)(unsafe.Pointer(processData.inputs))[:processData.numInputs:processData.numInputs]
		for _, bus := range inputBuses {
			channelBuffers32 := getChannelBuffers32(&bus)
			if bus.numChannels > 0 && channelBuffers32 != nil {
				channels := (*[16]*float32)(unsafe.Pointer(channelBuffers32))[:bus.numChannels:bus.numChannels]
				for _, channel := range channels {
					if channel != nil {
						// Create slice from pointer without allocation
						samples := (*[vst3.MaxArraySize]float32)(unsafe.Pointer(channel))[:numSamples:numSamples]
						c.processCtx.Input = append(c.processCtx.Input, samples)
					}
				}
			}
		}
	}

	// Map output buffers
	if processData.numOutputs > 0 && processData.outputs != nil {
		outputBuses := (*[1]C.struct_Steinberg_Vst_AudioBusBuffers)(unsafe.Pointer(processData.outputs))[:processData.numOutputs:processData.numOutputs]
		for _, bus := range outputBuses {
			channelBuffers32 := getChannelBuffers32(&bus)
			if bus.numChannels > 0 && channelBuffers32 != nil {
				channels := (*[16]*float32)(unsafe.Pointer(channelBuffers32))[:bus.numChannels:bus.numChannels]
				for _, channel := range channels {
					if channel != nil {
						// Create slice from pointer without allocation
						samples := (*[vst3.MaxArraySize]float32)(unsafe.Pointer(channel))[:numSamples:numSamples]
						c.processCtx.Output = append(c.processCtx.Output, samples)
					}
				}
			}
		}
	}

	// Reset parameter changes for this processing block
	c.processCtx.ResetParameterChanges()

	// Collect parameter changes for sample-accurate automation
	if processData.inputParameterChanges != nil {
		// Get parameter count using C helper function
		paramCount := C.getParameterChangeCount(unsafe.Pointer(processData.inputParameterChanges))

		// Process each parameter that has changes
		for i := C.int32_t(0); i < paramCount; i++ {
			paramQueue := C.getParameterData(unsafe.Pointer(processData.inputParameterChanges), i)
			if paramQueue != nil {
				// Get parameter ID
				paramID := C.getParameterId(paramQueue)

				// Get number of automation points
				pointCount := C.getPointCount(paramQueue)

				// Process all automation points for this parameter
				for j := C.int32_t(0); j < pointCount; j++ {
					var sampleOffset C.int32_t
					var value C.double

					// Get the automation point
					result := C.getPoint(paramQueue, j, &sampleOffset, &value)
					if result == 0 { // kResultOk
						// Add parameter change for sample-accurate processing
						c.processCtx.AddParameterChange(uint32(paramID), float64(value), int(sampleOffset))
					}
				}
			}
		}
	}

	// Process audio with sample-accurate parameter automation
	if c.processCtx.HasParameterChanges() {
		// Sort parameter changes by sample offset
		c.processCtx.SortParameterChanges()

		// Process audio in chunks between parameter changes
		c.processSampleAccurate()
	} else {
		// No parameter changes - process entire block
		c.processor.ProcessAudio(c.processCtx)
	}

	return nil
}

func (c *componentImpl) GetTailSamples() uint32 {
	return uint32(c.processor.GetTailSamples())
}

// IEditController implementation
func (c *componentImpl) SetComponentState(state []byte) error {
	return nil
}

func (c *componentImpl) GetParameterCount() int32 {
	return c.processor.GetParameters().Count()
}

func (c *componentImpl) GetParameterInfo(index int32) (*vst3.ParameterInfo, error) {
	p := c.processor.GetParameters().GetByIndex(index)
	if p == nil {
		return nil, vst3.ErrInvalidArgument
	}

	return &vst3.ParameterInfo{
		ID:           p.ID,
		Title:        p.Name,
		ShortTitle:   p.ShortName,
		Units:        p.Unit,
		StepCount:    p.StepCount,
		DefaultValue: p.DefaultValue,
		UnitID:       p.UnitID,
		Flags:        int32(p.Flags),
	}, nil
}

func (c *componentImpl) GetParamStringByValue(id uint32, value float64) (string, error) {
	if p := c.processor.GetParameters().Get(id); p != nil {
		result := p.FormatValue(value)
		// fmt.Printf("Component.GetParamStringByValue: id=%d, value=%.3f -> '%s'\n", id, value, result)
		return result, nil
	}
	return "", vst3.ErrInvalidArgument
}

func (c *componentImpl) GetParamValueByString(id uint32, str string) (float64, error) {
	if p := c.processor.GetParameters().Get(id); p != nil {
		return p.ParseValue(str)
	}
	return 0, vst3.ErrInvalidArgument
}

func (c *componentImpl) NormalizedParamToPlain(id uint32, normalized float64) float64 {
	if p := c.processor.GetParameters().Get(id); p != nil {
		return p.Min + normalized*(p.Max-p.Min)
	}
	return normalized
}

func (c *componentImpl) PlainParamToNormalized(id uint32, plain float64) float64 {
	if p := c.processor.GetParameters().Get(id); p != nil {
		if p.Max > p.Min {
			return (plain - p.Min) / (p.Max - p.Min)
		}
	}
	return plain
}

func (c *componentImpl) GetParamNormalized(id uint32) float64 {
	if p := c.processor.GetParameters().Get(id); p != nil {
		return p.GetValue()
	}
	return 0
}

func (c *componentImpl) SetParamNormalized(id uint32, value float64) error {
	if p := c.processor.GetParameters().Get(id); p != nil {
		// Debug parameter changes
		fmt.Printf("[PARAM_CHANGE] SetParamNormalized: id=%d, value=%.3f, plain=%.1f\n",
			id, value, p.Min+value*(p.Max-p.Min))
		p.SetValue(value)
		return nil
	}
	return vst3.ErrInvalidArgument
}

func (c *componentImpl) SetComponentHandler(handler interface{}) error {
	return nil
}

func (c *componentImpl) CreateView(name string) (interface{}, error) {
	return nil, vst3.ErrNotImplemented
}

// SetParamNormalizedWithNotification sets a parameter value and notifies the host
// This should be used when the plugin changes a parameter value internally
func (c *componentImpl) SetParamNormalizedWithNotification(id uint32, value float64) error {
	if p := c.processor.GetParameters().Get(id); p != nil {
		// Notify host of parameter change
		if c.wrapper != nil {
			c.wrapper.notifyParamBeginEdit(id)
			p.SetValue(value)
			c.wrapper.notifyParamPerformEdit(id, value)
			c.wrapper.notifyParamEndEdit(id)
		} else {
			// Fallback if no wrapper available
			p.SetValue(value)
		}
		return nil
	}
	return vst3.ErrInvalidArgument
}

// processSampleAccurate processes audio with sample-accurate parameter automation
func (c *componentImpl) processSampleAccurate() {
	changes := c.processCtx.GetParameterChanges()
	numSamples := c.processCtx.NumSamples()
	lastOffset := 0

	// Store original buffers
	origInput := c.processCtx.Input
	origOutput := c.processCtx.Output

	// Process each chunk between parameter changes
	for _, change := range changes {
		if change.SampleOffset > lastOffset {
			// Process chunk from lastOffset to change.SampleOffset
			chunkSize := change.SampleOffset - lastOffset

			// Temporarily update context buffers to point to sub-slices (no allocation)
			c.processCtx.Input = nil
			c.processCtx.Output = nil

			// Set up input sub-slices
			for ch := 0; ch < len(origInput); ch++ {
				if lastOffset < len(origInput[ch]) {
					endOffset := lastOffset + chunkSize
					if endOffset > len(origInput[ch]) {
						endOffset = len(origInput[ch])
					}
					c.processCtx.Input = append(c.processCtx.Input, origInput[ch][lastOffset:endOffset])
				}
			}

			// Set up output sub-slices
			for ch := 0; ch < len(origOutput); ch++ {
				if lastOffset < len(origOutput[ch]) {
					endOffset := lastOffset + chunkSize
					if endOffset > len(origOutput[ch]) {
						endOffset = len(origOutput[ch])
					}
					c.processCtx.Output = append(c.processCtx.Output, origOutput[ch][lastOffset:endOffset])
				}
			}

			// Process this chunk
			c.processor.ProcessAudio(c.processCtx)

			lastOffset = change.SampleOffset
		}

		// Apply the parameter change
		c.processCtx.ApplyParameterChange(change)

		// Debug output
		if p := c.processor.GetParameters().Get(change.ParamID); p != nil {
			fmt.Printf("[SAMPLE_ACCURATE] Applied param %d change at sample %d: value=%.6f, plain=%.1f\n",
				change.ParamID, change.SampleOffset, change.Value, p.GetPlainValue())
		}
	}

	// Process final chunk if there are samples remaining
	if lastOffset < numSamples {
		// Temporarily update context buffers for final chunk
		c.processCtx.Input = nil
		c.processCtx.Output = nil

		// Set up input sub-slices for final chunk
		for ch := 0; ch < len(origInput); ch++ {
			if lastOffset < len(origInput[ch]) {
				c.processCtx.Input = append(c.processCtx.Input, origInput[ch][lastOffset:])
			}
		}

		// Set up output sub-slices for final chunk
		for ch := 0; ch < len(origOutput); ch++ {
			if lastOffset < len(origOutput[ch]) {
				c.processCtx.Output = append(c.processCtx.Output, origOutput[ch][lastOffset:])
			}
		}

		// Process final chunk
		c.processor.ProcessAudio(c.processCtx)
	}

	// Restore original buffers
	c.processCtx.Input = origInput
	c.processCtx.Output = origOutput
}
