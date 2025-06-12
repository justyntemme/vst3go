package param

import (
	"fmt"
	"sync/atomic"
)

// AutoRegistry extends Registry with automatic ID management
type AutoRegistry struct {
	*Registry
	nextID      atomic.Uint32
	nameToID    map[string]uint32
	autoEnabled bool
}

// NewAutoRegistry creates a registry with automatic ID management
func NewAutoRegistry() *AutoRegistry {
	return &AutoRegistry{
		Registry:    NewRegistry(),
		nameToID:    make(map[string]uint32),
		autoEnabled: true,
	}
}

// EnableAutoID enables or disables automatic ID assignment
func (r *AutoRegistry) EnableAutoID(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.autoEnabled = enabled
}

// Register adds a parameter with automatic ID assignment
func (r *AutoRegistry) Register(params ...*Parameter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for _, p := range params {
		// Check if we already have this parameter by name
		if existingID, exists := r.nameToID[p.Name]; exists {
			// Update existing parameter
			p.ID = existingID
			r.params[existingID] = p
			continue
		}
		
		// Assign new ID if auto-enabled and ID is 0
		if r.autoEnabled && p.ID == 0 {
			p.ID = r.nextID.Add(1) - 1
		}
		
		// Check for ID conflicts
		if _, exists := r.params[p.ID]; exists {
			return fmt.Errorf("parameter ID %d already exists", p.ID)
		}
		
		// Register the parameter
		r.params[p.ID] = p
		r.order = append(r.order, p.ID)
		r.nameToID[p.Name] = p.ID
	}
	
	return nil
}

// RegisterWithID adds a parameter with a specific ID
func (r *AutoRegistry) RegisterWithID(id uint32, param *Parameter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Force the ID
	param.ID = id
	
	// Check for conflicts
	if existing, exists := r.params[id]; exists {
		return fmt.Errorf("parameter ID %d already used by '%s'", id, existing.Name)
	}
	
	// Update next ID if necessary
	if id >= r.nextID.Load() {
		r.nextID.Store(id + 1)
	}
	
	// Register
	r.params[id] = param
	r.order = append(r.order, id)
	r.nameToID[param.Name] = id
	
	return nil
}

// GetByName retrieves a parameter by name
func (r *AutoRegistry) GetByName(name string) *Parameter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	id, exists := r.nameToID[name]
	if !exists {
		return nil
	}
	
	return r.params[id]
}

// GetID returns the ID for a parameter name
func (r *AutoRegistry) GetID(name string) (uint32, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	id, exists := r.nameToID[name]
	return id, exists
}

// Clear removes all parameters and resets the ID counter
func (r *AutoRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.params = make(map[uint32]*Parameter)
	r.order = make([]uint32, 0)
	r.nameToID = make(map[string]uint32)
	r.nextID.Store(0)
}

// Reserve reserves a range of IDs for manual assignment
func (r *AutoRegistry) Reserve(count uint32) uint32 {
	return r.nextID.Add(count) - count
}

// Builder for fluent parameter registration
type RegistryBuilder struct {
	registry *AutoRegistry
	errors   []error
}

// NewRegistryBuilder creates a builder for fluent registration
func NewRegistryBuilder(registry *AutoRegistry) *RegistryBuilder {
	return &RegistryBuilder{
		registry: registry,
		errors:   make([]error, 0),
	}
}

// Add registers a parameter
func (b *RegistryBuilder) Add(param *Parameter) *RegistryBuilder {
	if err := b.registry.Register(param); err != nil {
		b.errors = append(b.errors, err)
	}
	return b
}

// AddWithID registers a parameter with a specific ID
func (b *RegistryBuilder) AddWithID(id uint32, param *Parameter) *RegistryBuilder {
	if err := b.registry.RegisterWithID(id, param); err != nil {
		b.errors = append(b.errors, err)
	}
	return b
}

// Build finalizes the registration and returns any errors
func (b *RegistryBuilder) Build() error {
	if len(b.errors) > 0 {
		return fmt.Errorf("registration errors: %v", b.errors)
	}
	return nil
}

// Helper functions for common parameter groups

// RegisterStandardControls registers common audio effect controls
func (r *AutoRegistry) RegisterStandardControls() error {
	return NewRegistryBuilder(r).
		Add(BypassParameter(0, "Bypass").Build()).
		Add(MixParameter(0, "Mix").Build()).
		Add(GainParameter(0, "Input Gain").Build()).
		Add(GainParameter(0, "Output Gain").Build()).
		Build()
}

// RegisterCompressorControls registers standard compressor parameters
func (r *AutoRegistry) RegisterCompressorControls() error {
	return NewRegistryBuilder(r).
		Add(ThresholdParameter(0, "Threshold", -60, 0, -20).Build()).
		Add(RatioParameter(0, "Ratio", 1, 20, 4).Build()).
		Add(AttackParameter(0, "Attack", 100).Build()).
		Add(ReleaseParameter(0, "Release", 1000).Build()).
		Add(New(0, "Knee").
			Range(0, 10).
			Default(2).
			Formatter(func(v float64) string { return fmt.Sprintf("%.1f dB", v) }, nil).
			Build()).
		Add(BypassParameter(0, "Auto Gain").Build()).
		Build()
}

// RegisterEQBand registers a parametric EQ band
func (r *AutoRegistry) RegisterEQBand(bandNumber int) error {
	prefix := fmt.Sprintf("Band %d", bandNumber)
	
	filterTypes := []ChoiceOption{
		{Value: 0, Name: "Bell"},
		{Value: 1, Name: "Low Shelf"},
		{Value: 2, Name: "High Shelf"},
		{Value: 3, Name: "Low Pass"},
		{Value: 4, Name: "High Pass"},
		{Value: 5, Name: "Notch"},
	}
	
	return NewRegistryBuilder(r).
		Add(BypassParameter(0, prefix+" Enable").Build()).
		Add(FrequencyParameter(0, prefix+" Frequency", 20, 20000, 1000).Build()).
		Add(GainParameter(0, prefix+" Gain").Build()).
		Add(QParameter(0, prefix+" Q", 0.1, 10, 0.7).Build()).
		Add(Choice(0, prefix+" Type", filterTypes).Build()).
		Build()
}