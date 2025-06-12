package main

import (
	"github.com/justyntemme/vst3go/pkg/framework/plugin"
	vst3plugin "github.com/justyntemme/vst3go/pkg/plugin"
)

// SimpleSynthPlugin implements the Plugin interface
type SimpleSynthPlugin struct{}

func (s *SimpleSynthPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		ID:       "com.vst3go.examples.simplesynth",
		Name:     "Simple Synth",
		Version:  "1.0.0",
		Vendor:   "VST3Go Examples",
		Category: "Instrument|Synth",
	}
}

func (s *SimpleSynthPlugin) CreateProcessor() vst3plugin.Processor {
	return NewSimpleSynthProcessor()
}