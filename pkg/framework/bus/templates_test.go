package bus

import (
	"testing"
)

func TestTemplates(t *testing.T) {
	tests := []struct {
		name           string
		config         *Configuration
		expectInputs   int32
		expectOutputs  int32
		expectSidechain bool
		expectEvents   int
	}{
		{
			name:          "EffectStereo",
			config:        NewEffectStereo(),
			expectInputs:  1,
			expectOutputs: 1,
		},
		{
			name:          "EffectMono",
			config:        NewEffectMono(),
			expectInputs:  1,
			expectOutputs: 1,
		},
		{
			name:            "EffectStereoSidechain",
			config:          NewEffectStereoSidechain(),
			expectInputs:    2, // main + sidechain
			expectOutputs:   1,
			expectSidechain: true,
		},
		{
			name:          "MonoToStereo",
			config:        NewMonoToStereo(),
			expectInputs:  1,
			expectOutputs: 1,
		},
		{
			name:          "DualMono",
			config:        NewDualMono(),
			expectInputs:  2,
			expectOutputs: 2,
		},
		{
			name:          "Generator",
			config:        NewGenerator(),
			expectInputs:  0,
			expectOutputs: 1,
			expectEvents:  1,
		},
		{
			name:         "MIDIEffect",
			config:       NewMIDIEffect(),
			expectInputs: 0,
			expectOutputs: 0,
			expectEvents: 2, // in and out
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCount := tt.config.GetBusCount(MediaTypeAudio, DirectionInput)
			if inputCount != tt.expectInputs {
				t.Errorf("Expected %d audio inputs, got %d", tt.expectInputs, inputCount)
			}

			outputCount := tt.config.GetBusCount(MediaTypeAudio, DirectionOutput)
			if outputCount != tt.expectOutputs {
				t.Errorf("Expected %d audio outputs, got %d", tt.expectOutputs, outputCount)
			}

			if tt.config.HasSidechain() != tt.expectSidechain {
				t.Errorf("Expected HasSidechain=%v", tt.expectSidechain)
			}

			eventCount := tt.config.GetBusCount(MediaTypeEvent, DirectionInput) +
				tt.config.GetBusCount(MediaTypeEvent, DirectionOutput)
			if int(eventCount) != tt.expectEvents {
				t.Errorf("Expected %d event buses, got %d", tt.expectEvents, eventCount)
			}
		})
	}
}

func TestMixerChannel(t *testing.T) {
	config := NewMixerChannel(4)

	// Should have 1 input, 5 outputs (main + 4 sends)
	if config.GetBusCount(MediaTypeAudio, DirectionInput) != 1 {
		t.Error("Expected 1 input")
	}

	if config.GetBusCount(MediaTypeAudio, DirectionOutput) != 5 {
		t.Error("Expected 5 outputs (main + 4 sends)")
	}

	// Check main output
	mainOut := config.GetBusInfo(MediaTypeAudio, DirectionOutput, 0)
	if mainOut.Name != "Main Out" {
		t.Errorf("Expected 'Main Out', got %s", mainOut.Name)
	}
	if mainOut.BusType != TypeMain {
		t.Error("Expected main output to be TypeMain")
	}

	// Check sends
	for i := 1; i <= 4; i++ {
		send := config.GetBusInfo(MediaTypeAudio, DirectionOutput, int32(i))
		if send.BusType != TypeAux {
			t.Errorf("Expected send %d to be auxiliary", i)
		}
		if send.IsActive {
			t.Errorf("Expected send %d to start inactive", i)
		}
	}
}

func TestSurroundConfigs(t *testing.T) {
	t.Run("5.1 Effect", func(t *testing.T) {
		config := NewSurround5_1Effect()
		
		in := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
		if in.ChannelCount != 6 {
			t.Errorf("Expected 6 input channels for 5.1, got %d", in.ChannelCount)
		}

		out := config.GetBusInfo(MediaTypeAudio, DirectionOutput, 0)
		if out.ChannelCount != 6 {
			t.Errorf("Expected 6 output channels for 5.1, got %d", out.ChannelCount)
		}
	})

	t.Run("7.1 Effect", func(t *testing.T) {
		config := NewSurround7_1Effect()
		
		in := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
		if in.ChannelCount != 8 {
			t.Errorf("Expected 8 input channels for 7.1, got %d", in.ChannelCount)
		}
	})

	t.Run("Surround Panner", func(t *testing.T) {
		config := NewSurroundPanner()
		
		in := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
		if in.ChannelCount != 2 {
			t.Errorf("Expected stereo input, got %d channels", in.ChannelCount)
		}

		out := config.GetBusInfo(MediaTypeAudio, DirectionOutput, 0)
		if out.ChannelCount != 6 {
			t.Errorf("Expected 5.1 output, got %d channels", out.ChannelCount)
		}
	})
}

func TestSpecialTemplates(t *testing.T) {
	t.Run("Crossover", func(t *testing.T) {
		config := NewCrossover(3)
		
		// 1 input, 4 outputs (3 bands + main)
		if config.GetBusCount(MediaTypeAudio, DirectionOutput) != 4 {
			t.Error("Expected 4 outputs for 3-band crossover")
		}

		// Check band outputs are aux
		for i := 0; i < 3; i++ {
			band := config.GetBusInfo(MediaTypeAudio, DirectionOutput, int32(i))
			if band.BusType != TypeAux {
				t.Errorf("Expected band %d to be auxiliary", i+1)
			}
		}

		// Main output should be last
		main := config.GetBusInfo(MediaTypeAudio, DirectionOutput, 3)
		if main.Name != "Main Out" {
			t.Error("Expected last output to be main")
		}
	})

	t.Run("Splitter", func(t *testing.T) {
		config := NewSplitter(3)
		
		if config.GetBusCount(MediaTypeAudio, DirectionInput) != 1 {
			t.Error("Expected 1 input")
		}

		if config.GetBusCount(MediaTypeAudio, DirectionOutput) != 3 {
			t.Error("Expected 3 outputs")
		}

		// All outputs should be main type
		for i := 0; i < 3; i++ {
			out := config.GetBusInfo(MediaTypeAudio, DirectionOutput, int32(i))
			if out.BusType != TypeMain {
				t.Errorf("Expected output %d to be main type", i+1)
			}
		}
	})

	t.Run("Vocoder", func(t *testing.T) {
		config := NewVocoder()
		
		// Check voice input
		voice := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
		if voice.ChannelCount != 1 {
			t.Error("Expected mono voice input")
		}

		// Check carrier (sidechain)
		if !config.HasSidechain() {
			t.Error("Expected vocoder to have sidechain")
		}

		carrier := config.GetSidechainBus()
		if carrier.Name != "Carrier In" {
			t.Errorf("Expected 'Carrier In', got %s", carrier.Name)
		}
	})

	t.Run("MultiChannelEffect", func(t *testing.T) {
		config := NewMultiChannelEffect(12)
		
		in := config.GetBusInfo(MediaTypeAudio, DirectionInput, 0)
		out := config.GetBusInfo(MediaTypeAudio, DirectionOutput, 0)
		
		if in.ChannelCount != 12 {
			t.Errorf("Expected 12 input channels, got %d", in.ChannelCount)
		}
		if out.ChannelCount != 12 {
			t.Errorf("Expected 12 output channels, got %d", out.ChannelCount)
		}
	})
}