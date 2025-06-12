package param

import (
	"testing"
)

func TestAutoRegistry(t *testing.T) {
	t.Run("AutomaticID", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		// Register parameters without IDs
		p1 := GainParameter(0, "Volume").Build()
		p2 := BypassParameter(0, "Bypass").Build()
		p3 := FrequencyParameter(0, "Cutoff", 20, 20000, 1000).Build()
		
		err := reg.Register(p1, p2, p3)
		if err != nil {
			t.Fatalf("Registration failed: %v", err)
		}
		
		// Check IDs were assigned
		if p1.ID != 0 {
			t.Errorf("Expected ID 0, got %d", p1.ID)
		}
		if p2.ID != 1 {
			t.Errorf("Expected ID 1, got %d", p2.ID)
		}
		if p3.ID != 2 {
			t.Errorf("Expected ID 2, got %d", p3.ID)
		}
		
		// Verify retrieval
		if reg.Get(0) != p1 {
			t.Error("Failed to retrieve parameter by ID")
		}
	})
	
	t.Run("ManualID", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		// Register with specific ID
		p1 := GainParameter(0, "Volume").Build()
		err := reg.RegisterWithID(10, p1)
		if err != nil {
			t.Fatalf("Registration failed: %v", err)
		}
		
		// Auto ID should continue from there
		p2 := BypassParameter(0, "Bypass").Build()
		err = reg.Register(p2)
		if err != nil {
			t.Fatalf("Registration failed: %v", err)
		}
		
		if p1.ID != 10 {
			t.Errorf("Expected ID 10, got %d", p1.ID)
		}
		if p2.ID != 11 {
			t.Errorf("Expected ID 11, got %d", p2.ID)
		}
	})
	
	t.Run("GetByName", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		p1 := GainParameter(0, "Master Volume").Build()
		reg.Register(p1)
		
		// Retrieve by name
		found := reg.GetByName("Master Volume")
		if found != p1 {
			t.Error("Failed to retrieve by name")
		}
		
		// Non-existent
		if reg.GetByName("NonExistent") != nil {
			t.Error("Should return nil for non-existent parameter")
		}
	})
	
	t.Run("DuplicateNames", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		p1 := GainParameter(0, "Volume").Build()
		p1.DefaultValue = 0.5
		p2 := GainParameter(0, "Volume").Build()
		p2.DefaultValue = 0.7
		
		reg.Register(p1)
		reg.Register(p2)
		
		// Second registration should update the existing parameter
		found := reg.GetByName("Volume")
		if found.DefaultValue != 0.7 {
			t.Error("Should update existing parameter with same name")
		}
		
		// Should not create duplicate
		if reg.Count() != 1 {
			t.Errorf("Expected 1 parameter, got %d", reg.Count())
		}
	})
	
	t.Run("IDConflict", func(t *testing.T) {
		reg := NewAutoRegistry()
		reg.EnableAutoID(false) // Disable auto ID
		
		p1 := &Parameter{ID: 5, Name: "Param1"}
		p2 := &Parameter{ID: 5, Name: "Param2"}
		
		err := reg.Register(p1)
		if err != nil {
			t.Fatalf("First registration failed: %v", err)
		}
		
		err = reg.Register(p2)
		if err == nil {
			t.Error("Should fail with ID conflict")
		}
	})
	
	t.Run("Reserve", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		// Reserve first 10 IDs
		firstID := reg.Reserve(10)
		if firstID != 0 {
			t.Errorf("Expected first ID 0, got %d", firstID)
		}
		
		// Next auto ID should be 10
		p := GainParameter(0, "Test").Build()
		reg.Register(p)
		
		if p.ID != 10 {
			t.Errorf("Expected ID 10 after reservation, got %d", p.ID)
		}
	})
	
	t.Run("Clear", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		reg.Register(GainParameter(0, "P1").Build())
		reg.Register(GainParameter(0, "P2").Build())
		
		reg.Clear()
		
		if reg.Count() != 0 {
			t.Error("Registry not cleared")
		}
		
		// ID counter should reset
		p := GainParameter(0, "P3").Build()
		reg.Register(p)
		if p.ID != 0 {
			t.Errorf("ID counter not reset, got %d", p.ID)
		}
	})
}

func TestRegistryBuilder(t *testing.T) {
	t.Run("FluentAPI", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		err := NewRegistryBuilder(reg).
			Add(GainParameter(0, "Gain").Build()).
			Add(BypassParameter(0, "Bypass").Build()).
			AddWithID(10, FrequencyParameter(0, "Cutoff", 20, 20000, 1000).Build()).
			Build()
		
		if err != nil {
			t.Fatalf("Builder failed: %v", err)
		}
		
		if reg.Count() != 3 {
			t.Errorf("Expected 3 parameters, got %d", reg.Count())
		}
		
		// Check specific IDs
		if id, exists := reg.GetID("Cutoff"); !exists || id != 10 {
			t.Error("Cutoff should have ID 10")
		}
	})
	
	t.Run("BuilderErrors", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		// Create ID conflict
		reg.RegisterWithID(5, GainParameter(0, "P1").Build())
		
		err := NewRegistryBuilder(reg).
			Add(GainParameter(0, "P2").Build()).
			AddWithID(5, GainParameter(0, "P3").Build()). // Conflict
			Build()
		
		if err == nil {
			t.Error("Builder should return error")
		}
	})
}

func TestStandardControls(t *testing.T) {
	t.Run("StandardControls", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		err := reg.RegisterStandardControls()
		if err != nil {
			t.Fatalf("Failed to register standard controls: %v", err)
		}
		
		// Check all standard controls exist
		expectedParams := []string{"Bypass", "Mix", "Input Gain", "Output Gain"}
		
		for _, name := range expectedParams {
			if reg.GetByName(name) == nil {
				t.Errorf("Missing standard control: %s", name)
			}
		}
	})
	
	t.Run("CompressorControls", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		err := reg.RegisterCompressorControls()
		if err != nil {
			t.Fatalf("Failed to register compressor controls: %v", err)
		}
		
		// Check threshold exists and has correct range
		threshold := reg.GetByName("Threshold")
		if threshold == nil {
			t.Fatal("Missing Threshold parameter")
		}
		
		if threshold.Min != -60 || threshold.Max != 0 {
			t.Error("Threshold has wrong range")
		}
	})
	
	t.Run("EQBand", func(t *testing.T) {
		reg := NewAutoRegistry()
		
		// Register 3 EQ bands
		for i := 1; i <= 3; i++ {
			err := reg.RegisterEQBand(i)
			if err != nil {
				t.Fatalf("Failed to register EQ band %d: %v", i, err)
			}
		}
		
		// Check band 2 parameters
		freq := reg.GetByName("Band 2 Frequency")
		if freq == nil {
			t.Error("Missing Band 2 Frequency")
		}
		
		// Should have 5 params per band × 3 bands = 15 total
		if reg.Count() != 15 {
			t.Errorf("Expected 15 parameters, got %d", reg.Count())
		}
	})
}

func BenchmarkAutoRegistry(b *testing.B) {
	b.Run("AutoRegister", func(b *testing.B) {
		reg := NewAutoRegistry()
		params := make([]*Parameter, 100)
		for i := range params {
			params[i] = New(0, "Param").Build()
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reg.Clear()
			for _, p := range params {
				reg.Register(p)
			}
		}
	})
	
	b.Run("GetByName", func(b *testing.B) {
		reg := NewAutoRegistry()
		for i := 0; i < 100; i++ {
			reg.Register(New(0, string(rune('A'+i))).Build())
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = reg.GetByName("M")
		}
	})
}