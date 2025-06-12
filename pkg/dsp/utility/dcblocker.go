// Package utility provides common DSP utility functions and processors.
package utility

// DCBlocker removes DC offset from audio signals.
// Uses a high-pass filter with a very low cutoff frequency.
type DCBlocker struct {
	// State variables for each channel
	x1 []float32 // Previous input
	y1 []float32 // Previous output
	
	// Filter coefficient
	coefficient float32
}

// NewDCBlocker creates a new DC blocker for the specified number of channels.
// The cutoff frequency is typically around 5-20 Hz.
func NewDCBlocker(channels int, cutoffHz float32, sampleRate float64) *DCBlocker {
	// Calculate coefficient for first-order high-pass filter
	// y[n] = x[n] - x[n-1] + R * y[n-1]
	// where R = 1 - (2 * PI * cutoff / sampleRate)
	R := float32(1.0 - (2.0 * 3.14159265359 * float64(cutoffHz) / sampleRate))
	
	// Clamp R to ensure stability
	if R < 0.9 {
		R = 0.9
	}
	if R > 0.999 {
		R = 0.999
	}
	
	return &DCBlocker{
		x1:          make([]float32, channels),
		y1:          make([]float32, channels),
		coefficient: R,
	}
}

// Process removes DC offset from a single sample on a single channel.
func (dc *DCBlocker) Process(input float32, channel int) float32 {
	if channel >= len(dc.x1) {
		return input // Safety check
	}
	
	// First-order high-pass filter
	output := input - dc.x1[channel] + dc.coefficient*dc.y1[channel]
	
	// Update state
	dc.x1[channel] = input
	dc.y1[channel] = output
	
	return output
}

// ProcessBuffer removes DC offset from a buffer in-place.
func (dc *DCBlocker) ProcessBuffer(buffer []float32, channel int) {
	if channel >= len(dc.x1) {
		return // Safety check
	}
	
	for i := range buffer {
		buffer[i] = dc.Process(buffer[i], channel)
	}
}

// ProcessStereo processes stereo buffers in-place.
func (dc *DCBlocker) ProcessStereo(left, right []float32) {
	if len(dc.x1) < 2 {
		return // Not enough channels
	}
	
	dc.ProcessBuffer(left, 0)
	dc.ProcessBuffer(right, 1)
}

// ProcessMultiChannel processes multiple channel buffers.
func (dc *DCBlocker) ProcessMultiChannel(buffers [][]float32) {
	for ch, buffer := range buffers {
		if ch < len(dc.x1) {
			dc.ProcessBuffer(buffer, ch)
		}
	}
}

// Reset clears the DC blocker state.
func (dc *DCBlocker) Reset() {
	for i := range dc.x1 {
		dc.x1[i] = 0
		dc.y1[i] = 0
	}
}

// SetCutoff updates the cutoff frequency.
func (dc *DCBlocker) SetCutoff(cutoffHz float32, sampleRate float64) {
	R := float32(1.0 - (2.0 * 3.14159265359 * float64(cutoffHz) / sampleRate))
	
	// Clamp R to ensure stability
	if R < 0.9 {
		R = 0.9
	}
	if R > 0.999 {
		R = 0.999
	}
	
	dc.coefficient = R
}

// SimpleDCBlocker provides a single-channel DC blocker with minimal state.
type SimpleDCBlocker struct {
	x1, y1      float32
	coefficient float32
}

// NewSimpleDCBlocker creates a single-channel DC blocker.
func NewSimpleDCBlocker(sampleRate float64) *SimpleDCBlocker {
	// Default to 10 Hz cutoff
	cutoffHz := 10.0
	R := float32(1.0 - (2.0 * 3.14159265359 * cutoffHz / sampleRate))
	
	return &SimpleDCBlocker{
		coefficient: R,
	}
}

// Process removes DC from a single sample.
func (dc *SimpleDCBlocker) Process(input float32) float32 {
	output := input - dc.x1 + dc.coefficient*dc.y1
	dc.x1 = input
	dc.y1 = output
	return output
}

// ProcessBuffer processes a buffer in-place.
func (dc *SimpleDCBlocker) ProcessBuffer(buffer []float32) {
	for i := range buffer {
		buffer[i] = dc.Process(buffer[i])
	}
}

// Reset clears the state.
func (dc *SimpleDCBlocker) Reset() {
	dc.x1 = 0
	dc.y1 = 0
}