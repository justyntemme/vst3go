package param

import (
	"fmt"
	"strconv"
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
	
	// Value formatting
	formatFunc func(float64) string
	parseFunc  func(string) (float64, error)
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

// SetFormatter sets custom value formatting
func (p *Parameter) SetFormatter(format func(float64) string, parse func(string) (float64, error)) {
	p.formatFunc = format
	p.parseFunc = parse
}

// FormatValue returns formatted parameter value
func (p *Parameter) FormatValue(normalized float64) string {
	plain := p.Denormalize(normalized)
	
	if p.formatFunc != nil {
		result := p.formatFunc(plain)
		// fmt.Printf("Parameter.FormatValue: id=%d, norm=%.3f, plain=%.3f -> '%s'\n", p.ID, normalized, plain, result)
		return result
	}
	
	// Default formatting
	if p.StepCount > 0 {
		// For discrete parameters, show integer
		return fmt.Sprintf("%.0f", plain)
	}
	return fmt.Sprintf("%.2f", plain)
}

// ParseValue parses string to normalized value
func (p *Parameter) ParseValue(str string) (float64, error) {
	if p.parseFunc != nil {
		plain, err := p.parseFunc(str)
		if err != nil {
			return 0, err
		}
		return p.Normalize(plain), nil
	}
	// Default parsing
	plain, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	return p.Normalize(plain), nil
}

// Normalize converts plain value to normalized (0-1)
func (p *Parameter) Normalize(plain float64) float64 {
	if p.Max <= p.Min {
		return 0
	}
	normalized := (plain - p.Min) / (p.Max - p.Min)
	if normalized < 0 {
		return 0
	}
	if normalized > 1 {
		return 1
	}
	return normalized
}

// Denormalize converts normalized (0-1) to plain value
func (p *Parameter) Denormalize(normalized float64) float64 {
	return p.Min + normalized*(p.Max-p.Min)
}

// Helper functions for float64 <-> uint64 conversion
func float64bits(f float64) uint64 {
	return *(*uint64)(unsafe.Pointer(&f))
}

func float64frombits(b uint64) float64 {
	return *(*float64)(unsafe.Pointer(&b))
}