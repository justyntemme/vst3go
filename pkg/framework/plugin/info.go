package plugin

// Info contains plugin metadata
type Info struct {
	ID       string // Unique plugin identifier (e.g., "com.example.myplugin")
	Name     string // Display name
	Version  string // Semantic version (e.g., "1.0.0")
	Vendor   string // Company/developer name
	Category string // Plugin category (e.g., "Fx", "Instrument")
}

// UID converts the string ID to a 16-byte array for VST3
func (i Info) UID() [16]byte {
	// For now, use the same UIDs as before for compatibility
	// TODO: Implement proper UUID generation from string ID

	// Hardcoded for gain example
	if i.ID == "com.vst3go.examples.gain" {
		return [16]byte{
			0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0,
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		}
	}

	// Hardcoded for delay example
	if i.ID == "com.vst3go.examples.delay" {
		return [16]byte{
			0x22, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0,
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x99,
		}
	}

	// Default
	var uid [16]byte
	idBytes := []byte(i.ID)
	for j := 0; j < 16 && j < len(idBytes); j++ {
		uid[j] = idBytes[j]
	}
	return uid
}
