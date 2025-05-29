package main

import (
	"fmt"
	"log"
	
	"github.com/justyntemme/vst3go/pkg/plugin"
	"github.com/justyntemme/vst3go/pkg/vst3"
)

type DebugPlugin struct{}

func (d *DebugPlugin) GetPluginName() string {
	return "Debug Plugin"
}

func (d *DebugPlugin) GetVendorName() string {
	return "Debug"
}

func (d *DebugPlugin) GetPluginVersion() string {
	return "1.0.0"
}

func (d *DebugPlugin) GetUID() [16]byte {
	return [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
}

func (d *DebugPlugin) CreateComponent() plugin.Component {
	log.Println("CreateComponent called!")
	return nil
}

func main() {
	fmt.Println("Testing plugin creation...")
	
	// Test UID matching
	uid := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	fmt.Printf("Plugin UID: %x\n", uid)
	
	// Test interface IDs
	fmt.Printf("IComponent IID: %x\n", vst3.IIDIComponent)
	fmt.Printf("IAudioProcessor IID: %x\n", vst3.IIDIAudioProcessor)
}