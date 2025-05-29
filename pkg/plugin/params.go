package plugin

import (
	"sync"
	"sync/atomic"
	
	"github.com/justyntemme/vst3go/pkg/vst3"
)

// Parameter represents a plugin parameter
type Parameter struct {
	Info  vst3.ParameterInfo
	value atomic.Value // stores float64
	mu    sync.RWMutex
}

// NewParameter creates a new parameter
func NewParameter(info vst3.ParameterInfo) *Parameter {
	p := &Parameter{
		Info: info,
	}
	p.value.Store(info.DefaultValue)
	return p
}

// GetValue returns the current normalized value (0.0 to 1.0)
func (p *Parameter) GetValue() float64 {
	return p.value.Load().(float64)
}

// SetValue sets the normalized value (0.0 to 1.0)
func (p *Parameter) SetValue(value float64) {
	// Clamp to valid range
	if value < 0.0 {
		value = 0.0
	} else if value > 1.0 {
		value = 1.0
	}
	p.value.Store(value)
}

// GetPlainValue converts normalized to plain value
func (p *Parameter) GetPlainValue() float64 {
	normalized := p.GetValue()
	// Simple linear conversion for now
	// Could be logarithmic, exponential, etc. based on parameter type
	return normalized
}

// SetPlainValue converts plain to normalized value
func (p *Parameter) SetPlainValue(plain float64) {
	// Simple linear conversion for now
	p.SetValue(plain)
}

// ParameterManager manages all plugin parameters
type ParameterManager struct {
	params map[uint32]*Parameter
	order  []uint32 // Maintain parameter order
	mu     sync.RWMutex
}

// NewParameterManager creates a new parameter manager
func NewParameterManager() *ParameterManager {
	return &ParameterManager{
		params: make(map[uint32]*Parameter),
		order:  make([]uint32, 0),
	}
}

// AddParameter adds a parameter to the manager
func (m *ParameterManager) AddParameter(param *Parameter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.params[param.Info.ID] = param
	m.order = append(m.order, param.Info.ID)
}

// GetParameter returns a parameter by ID
func (m *ParameterManager) GetParameter(id uint32) *Parameter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.params[id]
}

// GetParameterByIndex returns a parameter by index
func (m *ParameterManager) GetParameterByIndex(index int32) *Parameter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if index < 0 || index >= int32(len(m.order)) {
		return nil
	}
	
	id := m.order[index]
	return m.params[id]
}

// GetParameterCount returns the number of parameters
func (m *ParameterManager) GetParameterCount() int32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return int32(len(m.order))
}

// GetValue returns a parameter's normalized value
func (m *ParameterManager) GetValue(id uint32) float64 {
	param := m.GetParameter(id)
	if param == nil {
		return 0.0
	}
	return param.GetValue()
}

// SetValue sets a parameter's normalized value
func (m *ParameterManager) SetValue(id uint32, value float64) {
	param := m.GetParameter(id)
	if param != nil {
		param.SetValue(value)
	}
}

// Common parameter IDs
const (
	ParamIDGain   uint32 = 0
	ParamIDVolume uint32 = 1
	ParamIDPan    uint32 = 2
	ParamIDBypass uint32 = 1000
)