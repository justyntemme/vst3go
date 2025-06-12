// Package bus provides VST3 audio bus configuration and management.
package bus

import "fmt"

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

// SetBusActive activates or deactivates a specific bus
func (c *Configuration) SetBusActive(mediaType MediaType, direction Direction, index int32, active bool) error {
	buses := c.audioBuses
	if mediaType == MediaTypeEvent {
		buses = c.eventBuses
	}

	busIndex := int32(0)
	for i := range buses {
		if buses[i].Direction == direction {
			if busIndex == index {
				buses[i].IsActive = active
				return nil
			}
			busIndex++
		}
	}

	return fmt.Errorf("bus not found: mediaType=%d, direction=%d, index=%d", mediaType, direction, index)
}

// GetActiveInputChannelCount returns the total number of active input channels
func (c *Configuration) GetActiveInputChannelCount() int32 {
	count := int32(0)
	for _, bus := range c.audioBuses {
		if bus.Direction == DirectionInput && bus.IsActive {
			count += bus.ChannelCount
		}
	}
	return count
}

// GetActiveOutputChannelCount returns the total number of active output channels
func (c *Configuration) GetActiveOutputChannelCount() int32 {
	count := int32(0)
	for _, bus := range c.audioBuses {
		if bus.Direction == DirectionOutput && bus.IsActive {
			count += bus.ChannelCount
		}
	}
	return count
}

// GetActiveBuses returns all active buses of a given type and direction
func (c *Configuration) GetActiveBuses(mediaType MediaType, direction Direction) []Info {
	buses := c.audioBuses
	if mediaType == MediaTypeEvent {
		buses = c.eventBuses
	}

	var active []Info
	for _, bus := range buses {
		if bus.Direction == direction && bus.IsActive {
			active = append(active, bus)
		}
	}
	return active
}

// HasSidechain returns true if the configuration has a sidechain input
func (c *Configuration) HasSidechain() bool {
	for _, bus := range c.audioBuses {
		if bus.Direction == DirectionInput && bus.BusType == TypeAux {
			return true
		}
	}
	return false
}

// GetSidechainBus returns the first sidechain (aux input) bus if it exists
func (c *Configuration) GetSidechainBus() *Info {
	for i, bus := range c.audioBuses {
		if bus.Direction == DirectionInput && bus.BusType == TypeAux {
			return &c.audioBuses[i]
		}
	}
	return nil
}
