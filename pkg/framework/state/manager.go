package state

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/justyntemme/vst3go/pkg/framework/param"
)

// Manager handles plugin state saving and loading
type Manager struct {
	version  uint32
	registry *param.Registry
	custom   CustomStateFunc
}

// CustomStateFunc allows plugins to save additional state beyond parameters
type CustomStateFunc func(w io.Writer) error

// NewManager creates a new state manager
func NewManager(registry *param.Registry) *Manager {
	return &Manager{
		version:  1,
		registry: registry,
	}
}

// SetCustomStateFunc sets a function for saving custom state
func (m *Manager) SetCustomStateFunc(fn CustomStateFunc) {
	m.custom = fn
}

// Save writes the plugin state to a writer
func (m *Manager) Save(w io.Writer) error {
	// Write magic header
	if _, err := w.Write([]byte("VST3GO")); err != nil {
		return err
	}

	// Write version
	if err := binary.Write(w, binary.LittleEndian, m.version); err != nil {
		return err
	}

	// Write parameter count
	paramCount := m.registry.Count()
	if err := binary.Write(w, binary.LittleEndian, paramCount); err != nil {
		return err
	}

	// Write each parameter
	for _, param := range m.registry.All() {
		// Write ID
		if err := binary.Write(w, binary.LittleEndian, param.ID); err != nil {
			return err
		}

		// Write value
		value := param.GetValue()
		if err := binary.Write(w, binary.LittleEndian, value); err != nil {
			return err
		}
	}

	// Write custom state if provided
	if m.custom != nil {
		// Mark that custom data follows
		if err := binary.Write(w, binary.LittleEndian, uint32(1)); err != nil {
			return err
		}

		return m.custom(w)
	} else {
		// No custom data
		return binary.Write(w, binary.LittleEndian, uint32(0))
	}
}

// Load reads the plugin state from a reader
func (m *Manager) Load(r io.Reader) error {
	// Read and verify magic header
	header := make([]byte, 6)
	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}
	if string(header) != "VST3GO" {
		return fmt.Errorf("invalid state format")
	}

	// Read version
	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return err
	}

	// Handle version compatibility
	if version > m.version {
		return fmt.Errorf("state version %d is newer than supported version %d", version, m.version)
	}

	// Read parameter count
	var paramCount int32
	if err := binary.Read(r, binary.LittleEndian, &paramCount); err != nil {
		return err
	}

	// Read each parameter
	for i := int32(0); i < paramCount; i++ {
		var id uint32
		if err := binary.Read(r, binary.LittleEndian, &id); err != nil {
			return err
		}

		var value float64
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
			return err
		}

		// Set parameter value if it exists
		if param := m.registry.Get(id); param != nil {
			param.SetValue(value)
		}
		// Ignore unknown parameters for forward compatibility
	}

	// Check for custom data
	var hasCustom uint32
	if err := binary.Read(r, binary.LittleEndian, &hasCustom); err != nil {
		return err
	}

	if hasCustom != 0 && m.custom != nil {
		// Custom load function should be set by the plugin
		// For now, skip custom data
		return fmt.Errorf("custom state loading not implemented")
	}

	return nil
}
