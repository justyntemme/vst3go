// Package process provides audio processing context and utilities for VST3 audio processing.
package process

import (
	"github.com/justyntemme/vst3go/pkg/framework/bus"
)

// BusBuffers represents audio buffers for a single bus
type BusBuffers struct {
	Channels [][]float32
	BusInfo  *bus.Info
}

// MultiBusContext extends Context with multi-bus support
type MultiBusContext struct {
	*Context
	
	// Multi-bus audio buffers
	InputBuses  []BusBuffers
	OutputBuses []BusBuffers
	
	// Bus configuration
	BusConfig *bus.Configuration
}

// NewMultiBusContext creates a new multi-bus process context
func NewMultiBusContext(ctx *Context, busConfig *bus.Configuration) *MultiBusContext {
	return &MultiBusContext{
		Context:     ctx,
		InputBuses:  make([]BusBuffers, 0),
		OutputBuses: make([]BusBuffers, 0),
		BusConfig:   busConfig,
	}
}

// GetMainInput returns the main input bus buffers
func (m *MultiBusContext) GetMainInput() [][]float32 {
	for i, busBuffers := range m.InputBuses {
		if busBuffers.BusInfo.BusType == bus.TypeMain {
			return m.InputBuses[i].Channels
		}
	}
	return nil
}

// GetMainOutput returns the main output bus buffers
func (m *MultiBusContext) GetMainOutput() [][]float32 {
	for i, busBuffers := range m.OutputBuses {
		if busBuffers.BusInfo.BusType == bus.TypeMain {
			return m.OutputBuses[i].Channels
		}
	}
	return nil
}

// GetSidechainInput returns the sidechain (first aux) input if available
func (m *MultiBusContext) GetSidechainInput() [][]float32 {
	for i, busBuffers := range m.InputBuses {
		if busBuffers.BusInfo.BusType == bus.TypeAux {
			return m.InputBuses[i].Channels
		}
	}
	return nil
}

// GetInputBus returns a specific input bus by index
func (m *MultiBusContext) GetInputBus(index int) [][]float32 {
	if index >= 0 && index < len(m.InputBuses) {
		return m.InputBuses[index].Channels
	}
	return nil
}

// GetOutputBus returns a specific output bus by index
func (m *MultiBusContext) GetOutputBus(index int) [][]float32 {
	if index >= 0 && index < len(m.OutputBuses) {
		return m.OutputBuses[index].Channels
	}
	return nil
}

// GetInputBusInfo returns information about a specific input bus
func (m *MultiBusContext) GetInputBusInfo(index int) *bus.Info {
	if index >= 0 && index < len(m.InputBuses) {
		return m.InputBuses[index].BusInfo
	}
	return nil
}

// GetOutputBusInfo returns information about a specific output bus
func (m *MultiBusContext) GetOutputBusInfo(index int) *bus.Info {
	if index >= 0 && index < len(m.OutputBuses) {
		return m.OutputBuses[index].BusInfo
	}
	return nil
}

// NumInputBuses returns the number of input buses
func (m *MultiBusContext) NumInputBuses() int {
	return len(m.InputBuses)
}

// NumOutputBuses returns the number of output buses
func (m *MultiBusContext) NumOutputBuses() int {
	return len(m.OutputBuses)
}

// ProcessInputBuses iterates through all active input buses
func (m *MultiBusContext) ProcessInputBuses(fn func(busIndex int, channels [][]float32, info *bus.Info)) {
	for i, bus := range m.InputBuses {
		if bus.BusInfo.IsActive {
			fn(i, bus.Channels, bus.BusInfo)
		}
	}
}

// ProcessOutputBuses iterates through all active output buses
func (m *MultiBusContext) ProcessOutputBuses(fn func(busIndex int, channels [][]float32, info *bus.Info)) {
	for i, bus := range m.OutputBuses {
		if bus.BusInfo.IsActive {
			fn(i, bus.Channels, bus.BusInfo)
		}
	}
}

// ProcessMainBuses processes only the main buses
func (m *MultiBusContext) ProcessMainBuses(fn func(input, output [][]float32)) {
	mainIn := m.GetMainInput()
	mainOut := m.GetMainOutput()
	if mainIn != nil && mainOut != nil {
		fn(mainIn, mainOut)
	}
}

// ProcessWithSidechain processes main I/O with sidechain
func (m *MultiBusContext) ProcessWithSidechain(fn func(main, sidechain, output [][]float32)) {
	mainIn := m.GetMainInput()
	sidechain := m.GetSidechainInput()
	mainOut := m.GetMainOutput()
	
	if mainIn != nil && mainOut != nil {
		// If no sidechain, pass nil
		fn(mainIn, sidechain, mainOut)
	}
}

// ClearAllOutputs clears all output buses
func (m *MultiBusContext) ClearAllOutputs() {
	for _, bus := range m.OutputBuses {
		for ch := range bus.Channels {
			for i := range bus.Channels[ch] {
				bus.Channels[ch][i] = 0
			}
		}
	}
}

// PassThroughAll copies all inputs to corresponding outputs
func (m *MultiBusContext) PassThroughAll() {
	minBuses := len(m.InputBuses)
	if len(m.OutputBuses) < minBuses {
		minBuses = len(m.OutputBuses)
	}
	
	for busIdx := 0; busIdx < minBuses; busIdx++ {
		inChannels := m.InputBuses[busIdx].Channels
		outChannels := m.OutputBuses[busIdx].Channels
		
		minChannels := len(inChannels)
		if len(outChannels) < minChannels {
			minChannels = len(outChannels)
		}
		
		for ch := 0; ch < minChannels; ch++ {
			copy(outChannels[ch], inChannels[ch])
		}
	}
}