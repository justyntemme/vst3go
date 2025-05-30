package param

// Builder provides a fluent API for creating parameters
type Builder struct {
	param *Parameter
}

// New creates a new parameter builder
func New(id uint32, name string) *Builder {
	return &Builder{
		param: &Parameter{
			ID:           id,
			Name:         name,
			ShortName:    name,
			Min:          0,
			Max:          1,
			DefaultValue: 0,
			Flags:        CanAutomate,
		},
	}
}

// ShortName sets the short name
func (b *Builder) ShortName(name string) *Builder {
	b.param.ShortName = name
	return b
}

// Range sets the min and max values
func (b *Builder) Range(min, max float64) *Builder {
	b.param.Min = min
	b.param.Max = max
	return b
}

// Default sets the default value (in plain range, not normalized)
func (b *Builder) Default(value float64) *Builder {
	// Convert to normalized
	if b.param.Max > b.param.Min {
		normalized := (value - b.param.Min) / (b.param.Max - b.param.Min)
		b.param.DefaultValue = normalized
	}
	return b
}

// Unit sets the unit string
func (b *Builder) Unit(unit string) *Builder {
	b.param.Unit = unit
	return b
}

// Steps sets the number of discrete steps
func (b *Builder) Steps(count int32) *Builder {
	b.param.StepCount = count
	return b
}

// Flags sets parameter flags
func (b *Builder) Flags(flags uint32) *Builder {
	b.param.Flags = flags
	return b
}

// Toggle creates a boolean parameter
func (b *Builder) Toggle() *Builder {
	b.param.Min = 0
	b.param.Max = 1
	b.param.StepCount = 1
	b.param.DefaultValue = 0
	return b
}

// ReadOnly marks the parameter as read-only
func (b *Builder) ReadOnly() *Builder {
	b.param.Flags |= IsReadOnly
	b.param.Flags &^= CanAutomate // Remove automation flag
	return b
}

// Hidden marks the parameter as hidden
func (b *Builder) Hidden() *Builder {
	b.param.Flags |= IsHidden
	return b
}

// Bypass marks this as the bypass parameter
func (b *Builder) Bypass() *Builder {
	b.param.Flags |= IsBypass
	return b
}

// Formatter sets custom value formatting and parsing
func (b *Builder) Formatter(format func(float64) string, parse func(string) (float64, error)) *Builder {
	b.param.formatFunc = format
	b.param.parseFunc = parse
	return b
}

// Build returns the configured parameter
func (b *Builder) Build() *Parameter {
	// Initialize with default value
	b.param.SetValue(b.param.DefaultValue)
	return b.param
}
