package bus

import (
	"testing"
)

func TestBuilder(t *testing.T) {
	t.Run("BasicStereo", func(t *testing.T) {
		config, err := NewBuilder().
			WithStereoInput("In").
			WithStereoOutput("Out").
			Build()

		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if config.GetBusCount(MediaTypeAudio, DirectionInput) != 1 {
			t.Error("Expected 1 input bus")
		}
		if config.GetBusCount(MediaTypeAudio, DirectionOutput) != 1 {
			t.Error("Expected 1 output bus")
		}
	})

	t.Run("MultiChannel", func(t *testing.T) {
		config, err := NewBuilder().
			WithMonoInput("Mono").
			WithStereoInput("Stereo").
			WithQuadOutput("Quad").
			With5_1Output("Surround").
			Build()

		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Check channel counts
		mono := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
		if mono.ChannelCount != 1 {
			t.Errorf("Expected 1 channel for mono, got %d", mono.ChannelCount)
		}

		stereo := config.GetBusInfo(MediaTypeAudio, DirectionInput, 1)
		if stereo.ChannelCount != 2 {
			t.Errorf("Expected 2 channels for stereo, got %d", stereo.ChannelCount)
		}

		quad := config.GetBusInfo(MediaTypeAudio, DirectionOutput, 0)
		if quad.ChannelCount != 4 {
			t.Errorf("Expected 4 channels for quad, got %d", quad.ChannelCount)
		}

		surround := config.GetBusInfo(MediaTypeAudio, DirectionOutput, 1)
		if surround.ChannelCount != 6 {
			t.Errorf("Expected 6 channels for 5.1, got %d", surround.ChannelCount)
		}
	})

	t.Run("WithSidechain", func(t *testing.T) {
		config, err := NewBuilder().
			WithStereoInput("Main").
			WithStereoOutput("Out").
			WithSidechain("SC").
			Build()

		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if !config.HasSidechain() {
			t.Error("Expected configuration to have sidechain")
		}

		sc := config.GetSidechainBus()
		if sc == nil {
			t.Fatal("Expected sidechain bus to exist")
		}
		if sc.BusType != TypeAux {
			t.Error("Expected sidechain to be auxiliary bus")
		}
		if sc.IsActive {
			t.Error("Expected sidechain to start inactive")
		}
	})

	t.Run("WithEvents", func(t *testing.T) {
		config, err := NewBuilder().
			WithStereoOutput("Audio").
			WithEventInput("MIDI In").
			WithEventOutput("MIDI Thru").
			Build()

		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if config.GetBusCount(MediaTypeEvent, DirectionInput) != 1 {
			t.Error("Expected 1 event input")
		}
		if config.GetBusCount(MediaTypeEvent, DirectionOutput) != 1 {
			t.Error("Expected 1 event output")
		}
	})

	t.Run("ValidationNoOutput", func(t *testing.T) {
		_, err := NewBuilder().
			WithStereoInput("In").
			Build()

		if err == nil {
			t.Error("Expected validation error for missing output")
		}
	})

	t.Run("SetBusActiveInBuilder", func(t *testing.T) {
		config, err := NewBuilder().
			WithStereoInput("In1").
			WithStereoInput("In2").
			WithStereoOutput("Out").
			SetBusActive(MediaTypeAudio, DirectionInput, 1, false).
			Build()

		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		bus1 := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
		if !bus1.IsActive {
			t.Error("Expected first bus to be active")
		}

		bus2 := config.GetBusInfo(MediaTypeAudio, DirectionInput, 1)
		if bus2.IsActive {
			t.Error("Expected second bus to be inactive")
		}
	})

	t.Run("MustBuildPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic from MustBuild with invalid config")
			}
		}()

		// This should panic due to no output
		NewBuilder().
			WithStereoInput("In").
			MustBuild()
	})
}

func TestBuilderConvenienceMethods(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *Builder
		expected int32
	}{
		{
			name: "7.1 Surround",
			builder: func() *Builder {
				return NewBuilder().
					With7_1Input("7.1 In").
					With7_1Output("7.1 Out")
			},
			expected: 8,
		},
		{
			name: "Aux Buses",
			builder: func() *Builder {
				return NewBuilder().
					WithAuxInput("Aux In", 4).
					WithAuxOutput("Aux Out", 4).
					WithStereoOutput("Main")
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.builder().Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			bus := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
			if bus != nil && bus.ChannelCount != tt.expected {
				t.Errorf("Expected %d channels, got %d", tt.expected, bus.ChannelCount)
			}
		})
	}
}