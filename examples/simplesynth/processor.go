package main

import (
	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/process"
	"github.com/justyntemme/vst3go/pkg/framework/voice"
)

const (
	// Parameter IDs
	ParamAttack uint32 = iota
	ParamDecay
	ParamSustain
	ParamRelease
	ParamVolume
)

// SimpleSynthProcessor handles the audio processing
type SimpleSynthProcessor struct {
	// Voice management
	voices     []voice.Voice
	voiceAlloc *voice.Allocator
	
	// Parameters
	params *param.Registry
	buses  *bus.Configuration
	
	// Current parameter values
	attack  float64
	decay   float64
	sustain float64
	release float64
	volume  float64
	
	// Processing state
	sampleRate float64
	active     bool
	
	// Pre-allocated buffers
	voiceBuffer []float32
}

// NewSimpleSynthProcessor creates a new instance of the synthesizer processor
func NewSimpleSynthProcessor() *SimpleSynthProcessor {
	p := &SimpleSynthProcessor{
		params:  param.NewRegistry(),
		buses:   bus.NewGenerator(), // Stereo output + MIDI input
		attack:  0.01,  // 10ms
		decay:   0.1,   // 100ms
		sustain: 0.7,   // 70%
		release: 0.3,   // 300ms
		volume:  0.8,   // 80%
	}
	
	// Initialize parameters
	p.initializeParameters()
	
	return p
}

func (p *SimpleSynthProcessor) initializeParameters() {
	// ADSR parameters
	p.params.Add(
		param.New(ParamAttack, "Attack").
			Range(0.001, 2.0).
			Default(p.attack).
			Unit("s").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamDecay, "Decay").
			Range(0.001, 2.0).
			Default(p.decay).
			Unit("s").
			Build(),
	)
	
	p.params.Add(
		param.New(ParamSustain, "Sustain").
			Range(0.0, 1.0).
			Default(p.sustain).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)
	
	p.params.Add(
		param.New(ParamRelease, "Release").
			Range(0.001, 5.0).
			Default(p.release).
			Unit("s").
			Build(),
	)
	
	// Master volume
	p.params.Add(
		param.New(ParamVolume, "Volume").
			Range(0.0, 1.0).
			Default(p.volume).
			Unit("%").
			Formatter(param.PercentFormatter, param.PercentParser).
			Build(),
	)
}

// Initialize is called when the plugin is created
func (p *SimpleSynthProcessor) Initialize(sampleRate float64, maxBlockSize int32) error {
	p.sampleRate = sampleRate
	
	// Pre-allocate voice buffer
	p.voiceBuffer = make([]float32, maxBlockSize)
	
	// Create voices (16 voice polyphony)
	p.voices = createVoices(16, sampleRate)
	
	// Create voice allocator
	p.voiceAlloc = voice.NewAllocator(p.voices)
	p.voiceAlloc.SetMode(voice.ModePoly)
	p.voiceAlloc.SetStealingMode(voice.StealOldest)
	
	// Update voice parameters
	p.updateVoiceParameters()
	
	return nil
}

// ProcessAudio processes audio
func (p *SimpleSynthProcessor) ProcessAudio(ctx *process.Context) {
	
	if !p.active {
		ctx.Clear()
		return
	}
	
	// Update parameters if they've changed
	p.updateParameters(ctx)
	
	// Process MIDI events
	p.processMIDIEvents(ctx)
	
	// Clear output buffers
	ctx.Clear()
	
	// Check if we have output buffers
	if len(ctx.Output) < 2 {
		
		return
	}
	
	// Process each voice
	numSamples := ctx.NumSamples()
	if numSamples == 0 {
		return
	}
	
	// Safety check for voices
	if p.voices == nil || len(p.voices) == 0 {
		
		return
	}
	
	// Use pre-allocated voice buffer
	voiceBuffer := p.voiceBuffer[:numSamples]
	activeVoices := 0
	
	for _, v := range p.voices {
		if v != nil && v.IsActive() {
			activeVoices++
			// Process voice into temporary buffer
			v.Process(voiceBuffer)
			
			// Mix into output (stereo)
			for i := 0; i < numSamples; i++ {
				sample := voiceBuffer[i] * float32(p.volume)
				ctx.Output[0][i] += sample // Left
				ctx.Output[1][i] += sample // Right
			}
		}
	}
	
}

// processMIDIEvents handles incoming MIDI events
func (p *SimpleSynthProcessor) processMIDIEvents(ctx *process.Context) {
	events := ctx.GetAllInputEvents()
	if len(events) > 0 {
		
	}
	
	for _, event := range events {
		
		p.voiceAlloc.ProcessEvent(event)
	}
	
	// Clear processed events
	ctx.ClearInputEvents()
}

// updateParameters checks for parameter changes and updates internal state
func (p *SimpleSynthProcessor) updateParameters(ctx *process.Context) {
	// Check each parameter for changes
	if param := p.params.Get(ParamAttack); param != nil {
		newValue := ctx.ParamPlain(ParamAttack)
		if newValue != p.attack {
			p.attack = newValue
			p.updateVoiceParameters()
		}
	}
	
	if param := p.params.Get(ParamDecay); param != nil {
		newValue := ctx.ParamPlain(ParamDecay)
		if newValue != p.decay {
			p.decay = newValue
			p.updateVoiceParameters()
		}
	}
	
	if param := p.params.Get(ParamSustain); param != nil {
		newValue := ctx.Param(ParamSustain) // Already normalized 0-1
		if newValue != p.sustain {
			p.sustain = newValue
			p.updateVoiceParameters()
		}
	}
	
	if param := p.params.Get(ParamRelease); param != nil {
		newValue := ctx.ParamPlain(ParamRelease)
		if newValue != p.release {
			p.release = newValue
			p.updateVoiceParameters()
		}
	}
	
	if param := p.params.Get(ParamVolume); param != nil {
		p.volume = ctx.Param(ParamVolume) // Normalized 0-1
	}
}

// updateVoiceParameters updates all voice parameters
func (p *SimpleSynthProcessor) updateVoiceParameters() {
	for _, v := range p.voices {
		if sv, ok := v.(*SynthVoice); ok {
			sv.SetADSR(p.attack, p.decay, p.sustain, p.release)
		}
	}
}

// GetParameters returns the parameter registry
func (p *SimpleSynthProcessor) GetParameters() *param.Registry {
	return p.params
}

// GetBuses returns the bus configuration
func (p *SimpleSynthProcessor) GetBuses() *bus.Configuration {
	return p.buses
}

// SetActive is called when processing starts/stops
func (p *SimpleSynthProcessor) SetActive(active bool) error {
	
	p.active = active
	if !active && p.voiceAlloc != nil {
		// Stop all voices when deactivated
		p.voiceAlloc.Reset()
	}
	return nil
}

// GetLatencySamples returns the plugin latency in samples
func (p *SimpleSynthProcessor) GetLatencySamples() int32 {
	return 0
}

// GetTailSamples returns the tail length in samples
func (p *SimpleSynthProcessor) GetTailSamples() int32 {
	// Return maximum release time in samples
	return int32(p.release * p.sampleRate)
}