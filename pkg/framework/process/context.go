// Package process provides audio processing context and utilities for VST3 audio processing.
package process

import (
	"fmt"
	
	"github.com/justyntemme/vst3go/pkg/framework/param"
)

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
}

// NewContext creates a new process context with pre-allocated buffers
func NewContext(maxBlockSize int, params *param.Registry) *Context {
	return &Context{
		workBuffer: make([]float32, maxBlockSize),
		tempBuffer: make([]float32, maxBlockSize),
		params:     params,
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
// This immediately updates the parameter in the registry for the current processing block
func (c *Context) SetParameterAtOffset(paramID uint32, value float64, sampleOffset int) {
	if param := c.params.Get(paramID); param != nil {
		// For now, apply the change immediately
		// TODO: For true sample-accurate automation, we would need to process 
		// audio in chunks up to each parameter change point
		param.SetValue(value)
		
		// Debug output for parameter changes
		fmt.Printf("[PARAM_AUTOMATION] SetParameterAtOffset: id=%d, value=%.6f, offset=%d, plain=%.1f\n", 
			paramID, value, sampleOffset, param.GetPlainValue())
	}
}
