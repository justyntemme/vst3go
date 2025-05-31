package plugin

import (
	"crypto/md5" // #nosec G501 - MD5 is appropriate for deterministic UUID generation
	"fmt"
)

const (
	// UUID v4 version and variant constants per RFC 4122
	uuidVersion4    = 0x40
	uuidVariant     = 0x80
	uuidVersionMask = 0x0f
	uuidVariantMask = 0x3f
)

// Info contains plugin metadata
type Info struct {
	ID       string // Unique plugin identifier (e.g., "com.example.myplugin")
	Name     string // Display name
	Version  string // Semantic version (e.g., "1.0.0")
	Vendor   string // Company/developer name
	Category string // Plugin category (e.g., "Fx", "Instrument")
}

// UID converts the string ID to a 16-byte array for VST3
func (i *Info) UID() [16]byte {
	// Maintain backward compatibility for existing examples
	switch i.ID {
	case "com.vst3go.examples.gain":
		return [16]byte{
			0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0,
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		}
	case "com.vst3go.examples.filter":
		return [16]byte{
			0x87, 0x65, 0x43, 0x21, 0xFE, 0xDC, 0xBA, 0x98,
			0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11,
		}
	case "com.vst3go.examples.delay":
		return [16]byte{
			0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11,
			0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22,
		}
	default:
		// Generate deterministic UUID for new plugins
		return i.generateDeterministicUID()
	}
}

// generateDeterministicUID creates a deterministic UUID v4 from the plugin ID
func (i *Info) generateDeterministicUID() [16]byte {
	// Generate deterministic UUID from plugin ID string
	// This ensures the same plugin ID always generates the same UUID
	hash := md5.Sum([]byte(i.ID)) // #nosec G401 - MD5 is appropriate for deterministic UUID generation

	// Ensure it's a valid UUID v4 format
	// Set version (4) and variant bits according to RFC 4122
	hash[6] = (hash[6] & uuidVersionMask) | uuidVersion4 // Version 4
	hash[8] = (hash[8] & uuidVariantMask) | uuidVariant  // Variant 10

	return hash
}

// ValidateUID checks if the generated UID is unique and valid
func (i *Info) ValidateUID() error {
	uid := i.UID()

	// Check against known UIDs to prevent collisions
	knownUIDs := map[string][16]byte{
		"com.vst3go.examples.gain": {
			0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0,
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		},
		"com.vst3go.examples.filter": {
			0x87, 0x65, 0x43, 0x21, 0xFE, 0xDC, 0xBA, 0x98,
			0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11,
		},
		"com.vst3go.examples.delay": {
			0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11,
			0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22,
		},
	}

	// Check for collisions with known UIDs
	for id, knownUID := range knownUIDs {
		if id != i.ID && uid == knownUID {
			return fmt.Errorf("UID collision detected with plugin %s", id)
		}
	}

	// Validate that ID is not empty
	if i.ID == "" {
		return fmt.Errorf("plugin ID cannot be empty")
	}

	return nil
}
