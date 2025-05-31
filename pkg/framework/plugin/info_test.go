package plugin

import (
	"testing"
)

func TestUIDGeneration(t *testing.T) {
	tests := []struct {
		name     string
		pluginID string
		wantSame bool
	}{
		{
			name:     "Existing gain plugin uses hardcoded UID",
			pluginID: "com.vst3go.examples.gain",
			wantSame: false, // Should use hardcoded, not generated
		},
		{
			name:     "New plugin generates deterministic UID",
			pluginID: "com.mycompany.newplugin",
			wantSame: true, // Should generate same UID each time
		},
		{
			name:     "Another new plugin generates different UID",
			pluginID: "com.mycompany.anotherplugin",
			wantSame: true, // Should generate same UID each time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{ID: tt.pluginID}

			// Generate UID twice
			uid1 := info.UID()
			uid2 := info.UID()

			// Should always be deterministic
			if uid1 != uid2 {
				t.Errorf("UID generation is not deterministic for %s", tt.pluginID)
			}

			// Validate UID
			if err := info.ValidateUID(); err != nil {
				t.Errorf("UID validation failed for %s: %v", tt.pluginID, err)
			}
		})
	}
}

func TestUIDUniqueness(t *testing.T) {
	// Test that different plugin IDs generate different UIDs
	plugins := []string{
		"com.company1.plugin1",
		"com.company1.plugin2",
		"com.company2.plugin1",
		"com.different.name",
	}

	uids := make(map[[16]byte]string)

	for _, pluginID := range plugins {
		info := &Info{ID: pluginID}
		uid := info.UID()

		if existingID, exists := uids[uid]; exists {
			t.Errorf("UID collision between %s and %s", pluginID, existingID)
		}

		uids[uid] = pluginID
	}
}

func TestUIDValidation(t *testing.T) {
	tests := []struct {
		name    string
		info    Info
		wantErr bool
	}{
		{
			name:    "Valid plugin ID",
			info:    Info{ID: "com.example.plugin"},
			wantErr: false,
		},
		{
			name:    "Empty plugin ID",
			info:    Info{ID: ""},
			wantErr: true,
		},
		{
			name:    "Known gain plugin",
			info:    Info{ID: "com.vst3go.examples.gain"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.info.ValidateUID()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that existing plugins maintain their hardcoded UIDs
	expectedUIDs := map[string][16]byte{
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

	for pluginID, expectedUID := range expectedUIDs {
		t.Run(pluginID, func(t *testing.T) {
			info := &Info{ID: pluginID}
			actualUID := info.UID()

			if actualUID != expectedUID {
				t.Errorf("Backward compatibility broken for %s\nExpected: %x\nActual:   %x",
					pluginID, expectedUID, actualUID)
			}
		})
	}
}
