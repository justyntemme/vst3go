// Package dsp provides DSP utilities and chain building for the VST3 framework.
package dsp

import (
	"github.com/justyntemme/vst3go/pkg/dsp/dynamics"
	"github.com/justyntemme/vst3go/pkg/dsp/utility"
)

// CompressorAdapter adapts a compressor to the Processor interface.
type CompressorAdapter struct {
	comp    *dynamics.Compressor
	channel int
}

// NewCompressorAdapter creates a new compressor adapter for mono processing.
func NewCompressorAdapter(c *dynamics.Compressor) *CompressorAdapter {
	return &CompressorAdapter{comp: c, channel: 0}
}

func (a *CompressorAdapter) Process(buffer []float32) {
	// Process each sample
	for i := range buffer {
		buffer[i] = a.comp.Process(buffer[i])
	}
}

func (a *CompressorAdapter) Reset() {
	a.comp.Reset()
}

// GateAdapter adapts a gate to the Processor interface.
type GateAdapter struct {
	gate    *dynamics.Gate
	channel int
}

// NewGateAdapter creates a new gate adapter for mono processing.
func NewGateAdapter(g *dynamics.Gate) *GateAdapter {
	return &GateAdapter{gate: g, channel: 0}
}

func (a *GateAdapter) Process(buffer []float32) {
	// Process each sample
	for i := range buffer {
		buffer[i] = a.gate.Process(buffer[i])
	}
}

func (a *GateAdapter) Reset() {
	a.gate.Reset()
}

// DCBlockerAdapter adapts a DC blocker to the Processor interface.
type DCBlockerAdapter struct {
	blocker *utility.SimpleDCBlocker
}

// NewDCBlockerAdapter creates a new DC blocker adapter.
func NewDCBlockerAdapter(sampleRate float64) *DCBlockerAdapter {
	return &DCBlockerAdapter{
		blocker: utility.NewSimpleDCBlocker(sampleRate),
	}
}

func (a *DCBlockerAdapter) Process(buffer []float32) {
	a.blocker.ProcessBuffer(buffer)
}

func (a *DCBlockerAdapter) Reset() {
	a.blocker.Reset()
}

// NoiseAdapter adapts a noise generator to the Processor interface.
type NoiseAdapter struct {
	noise *utility.NoiseGenerator
	mix   float32
}

// NewNoiseAdapter creates a new noise adapter with mix control.
func NewNoiseAdapter(noiseType utility.NoiseType, mix float32) *NoiseAdapter {
	return &NoiseAdapter{
		noise: utility.NewNoiseGenerator(noiseType),
		mix:   mix,
	}
}

func (a *NoiseAdapter) Process(buffer []float32) {
	a.noise.GenerateAdd(buffer, a.mix)
}

func (a *NoiseAdapter) Reset() {
	a.noise.Reset()
}

func (a *NoiseAdapter) SetMix(mix float32) {
	a.mix = mix
}

// Simple helper chains for common use cases

// CreateSimpleChain creates a basic processing chain.
func CreateSimpleChain(sampleRate float64) (*Chain, error) {
	return NewBuilder("Simple Chain").
		WithProcessor(NewDCBlockerAdapter(sampleRate)).
		Build()
}

// CreateDynamicsChain creates a dynamics processing chain.
func CreateDynamicsChain(sampleRate float64) (*Chain, error) {
	gate := dynamics.NewGate(sampleRate)
	gate.SetThreshold(-40)
	gate.SetRange(-60)
	
	comp := dynamics.NewCompressor(sampleRate)
	comp.SetThreshold(-20)
	comp.SetRatio(4)
	comp.SetAttack(10)
	comp.SetRelease(100)
	
	return NewBuilder("Dynamics Chain").
		WithProcessor(NewGateAdapter(gate)).
		WithProcessor(NewCompressorAdapter(comp)).
		Build()
}