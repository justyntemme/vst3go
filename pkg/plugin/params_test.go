package plugin

import (
	"testing"
	
	"github.com/justyntemme/vst3go/pkg/vst3"
)

func TestParameterManager(t *testing.T) {
	pm := NewParameterManager()
	
	// Test adding parameter
	param := NewParameter(vst3.ParameterInfo{
		ID:           1,
		Title:        "Gain",
		ShortTitle:   "Gain",
		Units:        "dB",
		StepCount:    0,
		DefaultValue: 0.5,
		UnitID:       0,
		Flags:        vst3.ParameterCanAutomate,
	})
	
	pm.AddParameter(param)
	
	// Test parameter count
	if pm.GetParameterCount() != 1 {
		t.Errorf("Expected 1 parameter, got %d", pm.GetParameterCount())
	}
	
	// Test get parameter by index
	p := pm.GetParameterByIndex(0)
	if p == nil {
		t.Error("Expected parameter at index 0, got nil")
	}
	if p.Info.ID != 1 {
		t.Errorf("Expected parameter ID 1, got %d", p.Info.ID)
	}
	
	// Test get parameter by ID
	p = pm.GetParameter(1)
	if p == nil {
		t.Error("Expected parameter with ID 1, got nil")
	}
	
	// Test set/get value
	pm.SetValue(1, 0.75)
	value := pm.GetValue(1)
	if value != 0.75 {
		t.Errorf("Expected value 0.75, got %f", value)
	}
	
	// Test value bounds
	pm.SetValue(1, 1.5) // Should clamp to 1.0
	value = pm.GetValue(1)
	if value != 1.0 {
		t.Errorf("Expected value clamped to 1.0, got %f", value)
	}
	
	pm.SetValue(1, -0.5) // Should clamp to 0.0
	value = pm.GetValue(1)
	if value != 0.0 {
		t.Errorf("Expected value clamped to 0.0, got %f", value)
	}
}

func TestParameter(t *testing.T) {
	param := NewParameter(vst3.ParameterInfo{
		ID:           2,
		Title:        "Volume",
		ShortTitle:   "Vol",
		Units:        "%",
		StepCount:    100,
		DefaultValue: 0.7,
		UnitID:       0,
		Flags:        vst3.ParameterCanAutomate | vst3.ParameterIsReadOnly,
	})
	
	// Test initial value
	if param.GetValue() != 0.7 {
		t.Errorf("Expected initial value 0.7, got %f", param.GetValue())
	}
	
	// Test parameter info
	if param.Info.Title != "Volume" {
		t.Errorf("Expected title 'Volume', got '%s'", param.Info.Title)
	}
	
	// Test set value
	param.SetValue(0.8)
	if param.GetValue() != 0.8 {
		t.Errorf("Expected value 0.8, got %f", param.GetValue())
	}
}

func TestParameterManagerConcurrency(t *testing.T) {
	pm := NewParameterManager()
	
	// Add some parameters
	for i := uint32(0); i < 10; i++ {
		param := NewParameter(vst3.ParameterInfo{
			ID:           i,
			Title:        "Param",
			DefaultValue: 0.5,
		})
		pm.AddParameter(param)
	}
	
	// Test concurrent access
	done := make(chan bool)
	
	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			pm.SetValue(uint32(i%10), float64(i%100)/100.0)
		}
		done <- true
	}()
	
	// Reader goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			_ = pm.GetValue(uint32(i%10))
		}
		done <- true
	}()
	
	// Wait for both goroutines
	<-done
	<-done
	
	// Verify all parameters still exist
	if pm.GetParameterCount() != 10 {
		t.Errorf("Expected 10 parameters, got %d", pm.GetParameterCount())
	}
}