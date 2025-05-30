package plugin

import (
	"testing"
	
	"github.com/justyntemme/vst3go/pkg/vst3"
)

func TestParameterBasic(t *testing.T) {
	param := &Parameter{
		Info: vst3.ParameterInfo{
			ID:           1,
			Title:        "Test",
			DefaultValue: 0.5,
		},
	}
	param.value.Store(0.5)
	
	// Test get value
	if param.GetValue() != 0.5 {
		t.Errorf("Expected value 0.5, got %f", param.GetValue())
	}
	
	// Test set value
	param.SetValue(0.75)
	if param.GetValue() != 0.75 {
		t.Errorf("Expected value 0.75, got %f", param.GetValue())
	}
	
	// Test clamping
	param.SetValue(1.5)
	if param.GetValue() != 1.0 {
		t.Errorf("Expected value clamped to 1.0, got %f", param.GetValue())
	}
	
	param.SetValue(-0.5)
	if param.GetValue() != 0.0 {
		t.Errorf("Expected value clamped to 0.0, got %f", param.GetValue())
	}
}

func TestParameterManagerBasic(t *testing.T) {
	pm := &ParameterManager{
		params: make(map[uint32]*Parameter),
		order:  make([]uint32, 0),
	}
	
	// Create and add parameter
	param := &Parameter{
		Info: vst3.ParameterInfo{
			ID:           1,
			Title:        "Gain",
			DefaultValue: 0.5,
		},
	}
	param.value.Store(0.5)
	
	pm.params[1] = param
	pm.order = append(pm.order, 1)
	
	// Test get parameter count
	count := pm.GetParameterCount()
	if count != 1 {
		t.Errorf("Expected 1 parameter, got %d", count)
	}
	
	// Test get parameter by index
	p := pm.GetParameterByIndex(0)
	if p == nil {
		t.Error("Expected parameter at index 0, got nil")
	}
	
	// Test get value
	value := pm.GetValue(1)
	if value != 0.5 {
		t.Errorf("Expected value 0.5, got %f", value)
	}
	
	// Test set value
	pm.SetValue(1, 0.8)
	value = pm.GetValue(1)
	if value != 0.8 {
		t.Errorf("Expected value 0.8, got %f", value)
	}
}