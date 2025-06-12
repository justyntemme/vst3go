// Package dsp provides DSP utilities and chain building for the VST3 framework.
package dsp

import (
	"fmt"
)

// Processor represents a DSP processor that can be chained.
type Processor interface {
	// Process processes audio in-place
	Process(buffer []float32)
	
	// Reset resets the processor state
	Reset()
}

// StereoProcessor represents a stereo DSP processor.
type StereoProcessor interface {
	// ProcessStereo processes stereo audio in-place
	ProcessStereo(left, right []float32)
	
	// Reset resets the processor state
	Reset()
}

// MultiChannelProcessor represents a multi-channel DSP processor.
type MultiChannelProcessor interface {
	// ProcessMultiChannel processes multiple channels
	ProcessMultiChannel(buffers [][]float32)
	
	// Reset resets the processor state
	Reset()
}

// ProcessorFunc allows using a function as a Processor.
type ProcessorFunc func([]float32)

func (f ProcessorFunc) Process(buffer []float32) {
	f(buffer)
}

func (f ProcessorFunc) Reset() {
	// No-op for function processors
}

// Chain represents a chain of DSP processors.
type Chain struct {
	processors []Processor
	name       string
	bypass     bool
}

// NewChain creates a new DSP chain.
func NewChain(name string) *Chain {
	return &Chain{
		name:       name,
		processors: make([]Processor, 0),
	}
}

// Add adds a processor to the chain.
func (c *Chain) Add(processor Processor) *Chain {
	c.processors = append(c.processors, processor)
	return c
}

// AddFunc adds a processing function to the chain.
func (c *Chain) AddFunc(name string, process func([]float32)) *Chain {
	c.processors = append(c.processors, &namedProcessor{
		name:    name,
		process: ProcessorFunc(process),
	})
	return c
}

// Process processes audio through the chain.
func (c *Chain) Process(buffer []float32) {
	if c.bypass {
		return
	}
	
	for _, processor := range c.processors {
		processor.Process(buffer)
	}
}

// Reset resets all processors in the chain.
func (c *Chain) Reset() {
	for _, processor := range c.processors {
		processor.Reset()
	}
}

// SetBypass sets the bypass state of the chain.
func (c *Chain) SetBypass(bypass bool) {
	c.bypass = bypass
}

// IsEmpty returns true if the chain has no processors.
func (c *Chain) IsEmpty() bool {
	return len(c.processors) == 0
}

// Count returns the number of processors in the chain.
func (c *Chain) Count() int {
	return len(c.processors)
}

// namedProcessor wraps a processor with a name for debugging.
type namedProcessor struct {
	name    string
	process Processor
}

func (n *namedProcessor) Process(buffer []float32) {
	n.process.Process(buffer)
}

func (n *namedProcessor) Reset() {
	n.process.Reset()
}

// StereoChain represents a chain of stereo DSP processors.
type StereoChain struct {
	processors []StereoProcessor
	name       string
	bypass     bool
}

// NewStereoChain creates a new stereo DSP chain.
func NewStereoChain(name string) *StereoChain {
	return &StereoChain{
		name:       name,
		processors: make([]StereoProcessor, 0),
	}
}

// Add adds a stereo processor to the chain.
func (c *StereoChain) Add(processor StereoProcessor) *StereoChain {
	c.processors = append(c.processors, processor)
	return c
}

// ProcessStereo processes stereo audio through the chain.
func (c *StereoChain) ProcessStereo(left, right []float32) {
	if c.bypass {
		return
	}
	
	for _, processor := range c.processors {
		processor.ProcessStereo(left, right)
	}
}

// Reset resets all processors in the chain.
func (c *StereoChain) Reset() {
	for _, processor := range c.processors {
		processor.Reset()
	}
}

// SetBypass sets the bypass state of the chain.
func (c *StereoChain) SetBypass(bypass bool) {
	c.bypass = bypass
}

// ParallelChain processes audio through multiple chains in parallel and mixes the results.
type ParallelChain struct {
	chains []Processor
	gains  []float32
	name   string
	bypass bool
}

// NewParallelChain creates a new parallel chain.
func NewParallelChain(name string) *ParallelChain {
	return &ParallelChain{
		name:   name,
		chains: make([]Processor, 0),
		gains:  make([]float32, 0),
	}
}

// Add adds a chain with a gain factor.
func (p *ParallelChain) Add(chain Processor, gain float32) *ParallelChain {
	p.chains = append(p.chains, chain)
	p.gains = append(p.gains, gain)
	return p
}

// Process processes audio through all parallel chains.
func (p *ParallelChain) Process(buffer []float32) {
	if p.bypass || len(p.chains) == 0 {
		return
	}
	
	// Create temporary buffers for each chain
	tempBuffers := make([][]float32, len(p.chains))
	for i := range tempBuffers {
		tempBuffers[i] = make([]float32, len(buffer))
		copy(tempBuffers[i], buffer)
	}
	
	// Process each chain
	for i, chain := range p.chains {
		chain.Process(tempBuffers[i])
	}
	
	// Mix results
	for i := range buffer {
		buffer[i] = 0
		for j, temp := range tempBuffers {
			buffer[i] += temp[i] * p.gains[j]
		}
	}
}

// Reset resets all chains.
func (p *ParallelChain) Reset() {
	for _, chain := range p.chains {
		chain.Reset()
	}
}

// SetBypass sets the bypass state.
func (p *ParallelChain) SetBypass(bypass bool) {
	p.bypass = bypass
}

// Builder provides a fluent API for building DSP chains.
type Builder struct {
	chain  *Chain
	errors []error
}

// NewBuilder creates a new chain builder.
func NewBuilder(name string) *Builder {
	return &Builder{
		chain:  NewChain(name),
		errors: make([]error, 0),
	}
}

// WithProcessor adds a processor to the chain.
func (b *Builder) WithProcessor(processor Processor) *Builder {
	if processor == nil {
		b.errors = append(b.errors, fmt.Errorf("processor cannot be nil"))
		return b
	}
	b.chain.Add(processor)
	return b
}

// WithFunc adds a processing function to the chain.
func (b *Builder) WithFunc(name string, process func([]float32)) *Builder {
	if process == nil {
		b.errors = append(b.errors, fmt.Errorf("process function cannot be nil"))
		return b
	}
	b.chain.AddFunc(name, process)
	return b
}

// Build builds the chain and returns any errors.
func (b *Builder) Build() (*Chain, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("chain build errors: %v", b.errors)
	}
	if b.chain.IsEmpty() {
		return nil, fmt.Errorf("chain is empty")
	}
	return b.chain, nil
}

// StereoBuilder provides a fluent API for building stereo DSP chains.
type StereoBuilder struct {
	chain  *StereoChain
	errors []error
}

// NewStereoBuilder creates a new stereo chain builder.
func NewStereoBuilder(name string) *StereoBuilder {
	return &StereoBuilder{
		chain:  NewStereoChain(name),
		errors: make([]error, 0),
	}
}

// WithProcessor adds a stereo processor to the chain.
func (b *StereoBuilder) WithProcessor(processor StereoProcessor) *StereoBuilder {
	if processor == nil {
		b.errors = append(b.errors, fmt.Errorf("processor cannot be nil"))
		return b
	}
	b.chain.Add(processor)
	return b
}

// Build builds the stereo chain and returns any errors.
func (b *StereoBuilder) Build() (*StereoChain, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("stereo chain build errors: %v", b.errors)
	}
	if len(b.chain.processors) == 0 {
		return nil, fmt.Errorf("stereo chain is empty")
	}
	return b.chain, nil
}