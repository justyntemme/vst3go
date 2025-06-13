package main

import (
	"github.com/justyntemme/vst3go/pkg/dsp/envelope"
	"github.com/justyntemme/vst3go/pkg/dsp/oscillator"
	"github.com/justyntemme/vst3go/pkg/framework/voice"
	"github.com/justyntemme/vst3go/pkg/midi"
)

// SynthVoice represents a single voice in our synthesizer
type SynthVoice struct {
	// Oscillator
	osc *oscillator.Oscillator
	
	// Envelope
	ampEnv *envelope.ADSR
	
	// Voice state
	note      uint8
	velocity  uint8
	frequency float64
	amplitude float64
	active    bool
	age       int64
	
	// Sample rate
	sampleRate float64
}

// NewSynthVoice creates a new synth voice
func NewSynthVoice(sampleRate float64) *SynthVoice {
	return &SynthVoice{
		osc:        oscillator.New(sampleRate),
		ampEnv:     envelope.New(sampleRate),
		sampleRate: sampleRate,
	}
}

// Voice interface implementation

func (v *SynthVoice) IsActive() bool {
	return v.active
}

func (v *SynthVoice) GetNote() uint8 {
	return v.note
}

func (v *SynthVoice) GetVelocity() uint8 {
	return v.velocity
}

func (v *SynthVoice) GetAmplitude() float64 {
	return v.amplitude
}

func (v *SynthVoice) GetAge() int64 {
	return v.age
}

func (v *SynthVoice) TriggerNote(note uint8, velocity uint8) {
	v.note = note
	v.velocity = velocity
	v.frequency = midi.NoteToFrequency(note, 440.0)
	v.amplitude = float64(velocity) / 127.0
	v.active = true
	v.age = 0
	
	// Set oscillator frequency
	v.osc.SetFrequency(v.frequency)
	
	// Trigger envelope
	v.ampEnv.Trigger()
}

func (v *SynthVoice) ReleaseNote() {
	v.ampEnv.Release()
}

func (v *SynthVoice) Stop() {
	v.active = false
	v.ampEnv.Reset()
	v.osc.Reset()
	v.note = 0
	v.age = 0
}

func (v *SynthVoice) Process(output []float32) {
	if !v.active {
		// Voice is not active, fill with zeros
		for i := range output {
			output[i] = 0
		}
		return
	}
	
	firstSample := true
	maxSample := float32(0.0)
	
	// Generate audio
	for i := range output {
		// Get oscillator sample
		sample := v.osc.Sine()
		
		// Apply envelope
		envValue := v.ampEnv.Next()
		
		// Apply velocity scaling
		sample *= float32(v.amplitude) * envValue
		
		// Write to output
		output[i] = sample
		
		// Track max sample for debugging
		if sample > maxSample {
			maxSample = sample
		}
		
		// Debug first sample
		if firstSample && sample != 0 {
			firstSample = false
		}
		
		// Update age
		v.age++
		
		// Check if envelope has finished
		if v.ampEnv.GetStage() == envelope.StageIdle {
			v.active = false
		}
	}
}

// SetADSR configures the amplitude envelope
func (v *SynthVoice) SetADSR(attack, decay, sustain, release float64) {
	v.ampEnv.SetADSR(attack, decay, sustain, release)
}

// createVoices creates an array of synth voices
func createVoices(count int, sampleRate float64) []voice.Voice {
	voices := make([]voice.Voice, count)
	for i := range voices {
		voices[i] = NewSynthVoice(sampleRate)
	}
	return voices
}