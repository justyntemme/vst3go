// Package bus provides VST3 audio bus configuration and management.
package bus

import "fmt"

// Common bus configuration templates for different plugin types

// NewEffectStereo creates a standard stereo effect configuration (1 stereo in, 1 stereo out)
func NewEffectStereo() *Configuration {
	return NewBuilder().
		WithStereoInput("Stereo In").
		WithStereoOutput("Stereo Out").
		MustBuild()
}

// NewEffectMono creates a mono effect configuration (1 mono in, 1 mono out)
func NewEffectMono() *Configuration {
	return NewBuilder().
		WithMonoInput("Mono In").
		WithMonoOutput("Mono Out").
		MustBuild()
}

// NewEffectStereoSidechain creates a stereo effect with sidechain input
// Main: stereo in/out, Aux: stereo sidechain in
func NewEffectStereoSidechain() *Configuration {
	return NewBuilder().
		WithStereoInput("Stereo In").
		WithStereoOutput("Stereo Out").
		WithSidechain("Sidechain In").
		MustBuild()
}

// NewMonoToStereo creates a mono-to-stereo effect configuration
func NewMonoToStereo() *Configuration {
	return NewBuilder().
		WithMonoInput("Mono In").
		WithStereoOutput("Stereo Out").
		MustBuild()
}

// NewStereoToMono creates a stereo-to-mono effect configuration
func NewStereoToMono() *Configuration {
	return NewBuilder().
		WithStereoInput("Stereo In").
		WithMonoOutput("Mono Out").
		MustBuild()
}

// NewDualMono creates a dual mono configuration (2 mono in, 2 mono out)
func NewDualMono() *Configuration {
	return NewBuilder().
		WithMonoInput("Left In").
		WithMonoInput("Right In").
		WithMonoOutput("Left Out").
		WithMonoOutput("Right Out").
		MustBuild()
}

// NewMidSideProcessor creates a configuration for mid-side processing
func NewMidSideProcessor() *Configuration {
	return NewBuilder().
		WithStereoInput("Stereo In").
		WithStereoOutput("Stereo Out").
		WithAuxOutput("Mid", 1).
		WithAuxOutput("Side", 1).
		MustBuild()
}

// NewSurroundPanner creates a configuration for surround panning
// Mono/stereo input to 5.1 output
func NewSurroundPanner() *Configuration {
	return NewBuilder().
		WithStereoInput("Stereo In").
		With5_1Output("5.1 Out").
		MustBuild()
}

// NewSurround5_1Effect creates a 5.1 surround effect configuration
func NewSurround5_1Effect() *Configuration {
	return NewBuilder().
		With5_1Input("5.1 In").
		With5_1Output("5.1 Out").
		MustBuild()
}

// NewSurround7_1Effect creates a 7.1 surround effect configuration
func NewSurround7_1Effect() *Configuration {
	return NewBuilder().
		With7_1Input("7.1 In").
		With7_1Output("7.1 Out").
		MustBuild()
}

// NewMixerChannel creates a mixer channel configuration
// Stereo in, multiple outputs (main + sends)
func NewMixerChannel(numSends int) *Configuration {
	b := NewBuilder().
		WithStereoInput("Channel In").
		WithStereoOutput("Main Out")

	// Add send outputs
	for i := 0; i < numSends; i++ {
		b = b.WithAuxOutput(fmt.Sprintf("Send %d", i+1), 2)
	}

	return b.MustBuild()
}

// NewAnalyzer creates an analyzer configuration
// Stereo in, stereo thru, no processing on output
func NewAnalyzer() *Configuration {
	return NewBuilder().
		WithStereoInput("Analysis In").
		WithStereoOutput("Thru Out").
		MustBuild()
}

// NewGenerator creates a generator/instrument configuration
// No audio input, stereo output, MIDI input
func NewGenerator() *Configuration {
	return NewBuilder().
		WithStereoOutput("Stereo Out").
		WithEventInput("MIDI In").
		MustBuild()
}

// NewMIDIEffect creates a MIDI effect configuration
// MIDI in/out, no audio
func NewMIDIEffect() *Configuration {
	return NewBuilder().
		WithEventInput("MIDI In").
		WithEventOutput("MIDI Out").
		MustBuild()
}

// NewVocoder creates a vocoder configuration
// Main input (voice), sidechain input (carrier)
func NewVocoder() *Configuration {
	return NewBuilder().
		WithMonoInput("Voice In").
		WithMonoOutput("Vocoded Out").
		WithSidechain("Carrier In").
		MustBuild()
}

// NewMultiChannelEffect creates a flexible multi-channel effect
func NewMultiChannelEffect(numChannels int32) *Configuration {
	return NewBuilder().
		WithAudioInput("Multi In", numChannels).
		WithAudioOutput("Multi Out", numChannels).
		MustBuild()
}

// NewCrossover creates a crossover/band splitter configuration
// Stereo in, multiple band outputs
func NewCrossover(numBands int) *Configuration {
	b := NewBuilder().
		WithStereoInput("Stereo In")

	// Add band outputs
	for i := 0; i < numBands; i++ {
		b = b.WithAuxOutput(fmt.Sprintf("Band %d", i+1), 2)
	}

	// Also add a main stereo output for the summed result
	b = b.WithStereoOutput("Main Out")

	return b.MustBuild()
}

// NewSplitter creates a signal splitter configuration
// One input, multiple identical outputs
func NewSplitter(numOutputs int) *Configuration {
	b := NewBuilder().
		WithStereoInput("Input")

	for i := 0; i < numOutputs; i++ {
		b = b.WithStereoOutput(fmt.Sprintf("Output %d", i+1))
	}

	return b.MustBuild()
}