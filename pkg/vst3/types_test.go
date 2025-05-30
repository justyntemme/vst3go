package vst3

import (
	"testing"
)

func TestBusInfo(t *testing.T) {
	bus := BusInfo{
		MediaType:    MediaTypeAudio,
		Direction:    BusDirectionInput,
		ChannelCount: 2,
		Name:         "Main Input",
		BusType:      BusTypeMain,
		Flags:        1,
	}
	
	if bus.MediaType != MediaTypeAudio {
		t.Errorf("Expected MediaTypeAudio, got %d", bus.MediaType)
	}
	
	if bus.Direction != BusDirectionInput {
		t.Errorf("Expected BusDirectionInput, got %d", bus.Direction)
	}
	
	if bus.ChannelCount != 2 {
		t.Errorf("Expected 2 channels, got %d", bus.ChannelCount)
	}
}

func TestParameterInfo(t *testing.T) {
	param := ParameterInfo{
		ID:           1,
		Title:        "Volume",
		ShortTitle:   "Vol",
		Units:        "dB",
		StepCount:    0,
		DefaultValue: 0.7,
		UnitID:       0,
		Flags:        ParameterCanAutomate,
	}
	
	if param.ID != 1 {
		t.Errorf("Expected ID 1, got %d", param.ID)
	}
	
	if param.Title != "Volume" {
		t.Errorf("Expected title 'Volume', got '%s'", param.Title)
	}
	
	if param.DefaultValue != 0.7 {
		t.Errorf("Expected default value 0.7, got %f", param.DefaultValue)
	}
	
	if param.Flags != ParameterCanAutomate {
		t.Errorf("Expected ParameterCanAutomate flag, got %d", param.Flags)
	}
}

func TestProcessSetup(t *testing.T) {
	setup := ProcessSetup{
		ProcessMode:        0, // Realtime
		SymbolicSampleSize: 0, // 32-bit
		MaxSamplesPerBlock: 1024,
		SampleRate:         48000.0,
	}
	
	if setup.ProcessMode != 0 {
		t.Errorf("Expected process mode 0, got %d", setup.ProcessMode)
	}
	
	if setup.SampleRate != 48000.0 {
		t.Errorf("Expected sample rate 48000, got %f", setup.SampleRate)
	}
}