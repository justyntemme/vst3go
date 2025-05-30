package param

import (
	"sync/atomic"
	"unsafe"
)

// Parameter represents a plugin parameter
type Parameter struct {
	ID           uint32
	Name         string
	ShortName    string
	Unit         string
	Min          float64
	Max          float64
	DefaultValue float64
	StepCount    int32
	Flags        uint32
	UnitID       int32
	
	// Atomic value for lock-free access in audio thread
	value uint64 // Store as uint64 for atomic operations
}

// Flags for parameters
const (
	CanAutomate     uint32 = 1 << 0
	IsReadOnly      uint32 = 1 << 1
	IsWrapAround    uint32 = 1 << 2
	IsList          uint32 = 1 << 3
	IsHidden        uint32 = 1 << 4
	IsProgramChange uint32 = 1 << 15
	IsBypass        uint32 = 1 << 16
)

// GetValue returns the current normalized value (0-1)
func (p *Parameter) GetValue() float64 {
	bits := atomic.LoadUint64(&p.value)
	return float64frombits(bits)
}

// SetValue sets the normalized value (0-1)
func (p *Parameter) SetValue(value float64) {
	// Clamp to 0-1
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}
	
	atomic.StoreUint64(&p.value, float64bits(value))
}

// GetPlainValue converts normalized to plain value
func (p *Parameter) GetPlainValue() float64 {
	normalized := p.GetValue()
	return p.Min + normalized*(p.Max-p.Min)
}

// SetPlainValue converts plain to normalized value
func (p *Parameter) SetPlainValue(plain float64) {
	if p.Max <= p.Min {
		p.SetValue(0)
		return
	}
	
	normalized := (plain - p.Min) / (p.Max - p.Min)
	p.SetValue(normalized)
}

// Helper functions for float64 <-> uint64 conversion
func float64bits(f float64) uint64 {
	return *(*uint64)(unsafe.Pointer(&f))
}

func float64frombits(b uint64) float64 {
	return *(*float64)(unsafe.Pointer(&b))
}