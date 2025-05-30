// Package bus provides VST3 audio bus configuration and management.
package bus

// MediaType represents the type of bus
type MediaType int32

const (
	// MediaTypeAudio represents audio bus type
	MediaTypeAudio MediaType = 0
	// MediaTypeEvent represents event/MIDI bus type
	MediaTypeEvent MediaType = 1
)

// Direction represents the bus direction
type Direction int32

const (
	// DirectionInput represents input bus
	DirectionInput Direction = 0
	// DirectionOutput represents output bus
	DirectionOutput Direction = 1
)

// Type represents the bus type
type Type int32

const (
	// TypeMain represents main bus
	TypeMain Type = 0
	// TypeAux represents auxiliary bus
	TypeAux Type = 1
)

// Info contains bus configuration
type Info struct {
	MediaType    MediaType
	Direction    Direction
	ChannelCount int32
	Name         string
	BusType      Type
	IsActive     bool
}

// Configuration manages audio and event buses
type Configuration struct {
	audioBuses []Info
	eventBuses []Info
}

// NewStereoConfiguration creates a standard stereo I/O configuration
func NewStereoConfiguration() *Configuration {
	return &Configuration{
		audioBuses: []Info{
			{
				MediaType:    MediaTypeAudio,
				Direction:    DirectionInput,
				ChannelCount: 2,
				Name:         "Stereo In",
				BusType:      TypeMain,
				IsActive:     true,
			},
			{
				MediaType:    MediaTypeAudio,
				Direction:    DirectionOutput,
				ChannelCount: 2,
				Name:         "Stereo Out",
				BusType:      TypeMain,
				IsActive:     true,
			},
		},
	}
}

// NewMonoConfiguration creates a mono I/O configuration
func NewMonoConfiguration() *Configuration {
	return &Configuration{
		audioBuses: []Info{
			{
				MediaType:    MediaTypeAudio,
				Direction:    DirectionInput,
				ChannelCount: 1,
				Name:         "Mono In",
				BusType:      TypeMain,
				IsActive:     true,
			},
			{
				MediaType:    MediaTypeAudio,
				Direction:    DirectionOutput,
				ChannelCount: 1,
				Name:         "Mono Out",
				BusType:      TypeMain,
				IsActive:     true,
			},
		},
	}
}

// GetBusCount returns the number of buses for a given type and direction
func (c *Configuration) GetBusCount(mediaType MediaType, direction Direction) int32 {
	count := int32(0)

	buses := c.audioBuses
	if mediaType == MediaTypeEvent {
		buses = c.eventBuses
	}

	for _, bus := range buses {
		if bus.Direction == direction {
			count++
		}
	}

	return count
}

// GetBusInfo returns information about a specific bus
func (c *Configuration) GetBusInfo(mediaType MediaType, direction Direction, index int32) *Info {
	buses := c.audioBuses
	if mediaType == MediaTypeEvent {
		buses = c.eventBuses
	}

	busIndex := int32(0)
	for i := range buses {
		if buses[i].Direction == direction {
			if busIndex == index {
				return &buses[i]
			}
			busIndex++
		}
	}

	return nil
}

// AddEventBus adds an event bus (for MIDI input)
func (c *Configuration) AddEventBus(direction Direction, name string) {
	c.eventBuses = append(c.eventBuses, Info{
		MediaType:    MediaTypeEvent,
		Direction:    direction,
		ChannelCount: 1,
		Name:         name,
		BusType:      TypeMain,
		IsActive:     true,
	})
}
