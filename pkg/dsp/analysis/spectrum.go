package analysis

import (
	"math"
	"sync"
)

// SpectrumAnalyzer provides real-time spectral analysis
type SpectrumAnalyzer struct {
	fftSize      int
	sampleRate   float64
	fft          *FFT
	buffer       []float64
	writePos     int
	hopSize      int
	averaging    AveragingMode
	avgBuffer    [][]float64
	avgWritePos  int
	avgCount     int
	smoothing    float64
	minFreq      float64
	maxFreq      float64
	minBin       int
	maxBin       int
	outputBuffer []float64
	mu           sync.Mutex
}

// AveragingMode defines how the spectrum is averaged over time
type AveragingMode int

const (
	NoAveraging AveragingMode = iota
	ExponentialAveraging
	LinearAveraging
	PeakHold
)

// NewSpectrumAnalyzer creates a new spectrum analyzer
func NewSpectrumAnalyzer(fftSize int, sampleRate float64, window WindowFunc) *SpectrumAnalyzer {
	sa := &SpectrumAnalyzer{
		fftSize:      fftSize,
		sampleRate:   sampleRate,
		fft:          NewFFT(fftSize, window),
		buffer:       make([]float64, fftSize),
		hopSize:      fftSize / 2, // 50% overlap by default
		averaging:    NoAveraging,
		smoothing:    0.9,
		minFreq:      20.0,
		maxFreq:      sampleRate / 2.0,
		outputBuffer: make([]float64, fftSize/2+1),
	}
	
	sa.updateFrequencyRange()
	
	return sa
}

// SetHopSize sets the hop size (samples between FFT frames)
func (sa *SpectrumAnalyzer) SetHopSize(hopSize int) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	if hopSize > 0 && hopSize <= sa.fftSize {
		sa.hopSize = hopSize
	}
}

// SetAveraging sets the averaging mode and buffer size
func (sa *SpectrumAnalyzer) SetAveraging(mode AveragingMode, bufferSize int) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	sa.averaging = mode
	if mode != NoAveraging && bufferSize > 0 {
		sa.avgBuffer = make([][]float64, bufferSize)
		for i := range sa.avgBuffer {
			sa.avgBuffer[i] = make([]float64, sa.fftSize/2+1)
		}
		sa.avgWritePos = 0
		sa.avgCount = 0
	}
}

// SetSmoothing sets the smoothing factor for exponential averaging (0-1)
func (sa *SpectrumAnalyzer) SetSmoothing(smoothing float64) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	if smoothing >= 0 && smoothing <= 1 {
		sa.smoothing = smoothing
	}
}

// SetFrequencyRange sets the frequency range to analyze
func (sa *SpectrumAnalyzer) SetFrequencyRange(minFreq, maxFreq float64) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	sa.minFreq = math.Max(0, minFreq)
	sa.maxFreq = math.Min(sa.sampleRate/2.0, maxFreq)
	sa.updateFrequencyRange()
}

// updateFrequencyRange updates the bin range based on frequency limits
func (sa *SpectrumAnalyzer) updateFrequencyRange() {
	binWidth := sa.sampleRate / float64(sa.fftSize)
	sa.minBin = int(sa.minFreq / binWidth)
	sa.maxBin = int(sa.maxFreq/binWidth) + 1
	
	if sa.maxBin > sa.fftSize/2 {
		sa.maxBin = sa.fftSize / 2
	}
}

// Process adds samples to the analyzer and returns true when new spectrum is available
func (sa *SpectrumAnalyzer) Process(samples []float64) bool {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	spectrumReady := false
	
	for _, sample := range samples {
		sa.buffer[sa.writePos] = sample
		sa.writePos++
		
		if sa.writePos >= sa.fftSize {
			// Perform FFT
			magnitude, _ := sa.fft.Forward(sa.buffer)
			
			// Apply averaging
			sa.applyAveraging(magnitude)
			
			// Shift buffer by hop size
			if sa.hopSize < sa.fftSize {
				copy(sa.buffer, sa.buffer[sa.hopSize:])
				sa.writePos = sa.fftSize - sa.hopSize
			} else {
				sa.writePos = 0
			}
			
			spectrumReady = true
		}
	}
	
	return spectrumReady
}

// applyAveraging applies the selected averaging mode
func (sa *SpectrumAnalyzer) applyAveraging(magnitude []float64) {
	switch sa.averaging {
	case NoAveraging:
		copy(sa.outputBuffer, magnitude)
		
	case ExponentialAveraging:
		for i := range magnitude {
			sa.outputBuffer[i] = sa.outputBuffer[i]*sa.smoothing + magnitude[i]*(1-sa.smoothing)
		}
		
	case LinearAveraging:
		if sa.avgBuffer != nil {
			// Store current spectrum
			copy(sa.avgBuffer[sa.avgWritePos], magnitude)
			sa.avgWritePos = (sa.avgWritePos + 1) % len(sa.avgBuffer)
			
			if sa.avgCount < len(sa.avgBuffer) {
				sa.avgCount++
			}
			
			// Average all stored spectra
			for i := range sa.outputBuffer {
				sum := 0.0
				for j := 0; j < sa.avgCount; j++ {
					sum += sa.avgBuffer[j][i]
				}
				sa.outputBuffer[i] = sum / float64(sa.avgCount)
			}
		}
		
	case PeakHold:
		for i := range magnitude {
			if magnitude[i] > sa.outputBuffer[i] {
				sa.outputBuffer[i] = magnitude[i]
			}
		}
	}
}

// GetSpectrum returns the current magnitude spectrum
func (sa *SpectrumAnalyzer) GetSpectrum() []float64 {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	result := make([]float64, len(sa.outputBuffer))
	copy(result, sa.outputBuffer)
	return result
}

// GetSpectrumDB returns the spectrum in decibels
func (sa *SpectrumAnalyzer) GetSpectrumDB() []float64 {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	db := make([]float64, len(sa.outputBuffer))
	for i, mag := range sa.outputBuffer {
		if mag > 0 {
			db[i] = 20.0 * math.Log10(mag)
		} else {
			db[i] = -120.0 // Floor at -120 dB
		}
	}
	return db
}

// GetSpectrumInRange returns the spectrum only for the configured frequency range
func (sa *SpectrumAnalyzer) GetSpectrumInRange() []float64 {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	if sa.minBin >= sa.maxBin {
		return []float64{}
	}
	
	rangeSize := sa.maxBin - sa.minBin
	result := make([]float64, rangeSize)
	copy(result, sa.outputBuffer[sa.minBin:sa.maxBin])
	return result
}

// GetSpectrumDBInRange returns the spectrum in dB for the configured frequency range
func (sa *SpectrumAnalyzer) GetSpectrumDBInRange() []float64 {
	spectrum := sa.GetSpectrumInRange()
	db := make([]float64, len(spectrum))
	
	for i, mag := range spectrum {
		if mag > 0 {
			db[i] = 20.0 * math.Log10(mag)
		} else {
			db[i] = -120.0
		}
	}
	return db
}

// GetFrequencyForBin returns the frequency corresponding to a bin index
func (sa *SpectrumAnalyzer) GetFrequencyForBin(bin int) float64 {
	return float64(bin) * sa.sampleRate / float64(sa.fftSize)
}

// GetBinForFrequency returns the bin index for a given frequency
func (sa *SpectrumAnalyzer) GetBinForFrequency(freq float64) int {
	return int(freq * float64(sa.fftSize) / sa.sampleRate)
}

// Reset clears all buffers and averaging history
func (sa *SpectrumAnalyzer) Reset() {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	// Clear buffers
	for i := range sa.buffer {
		sa.buffer[i] = 0
	}
	for i := range sa.outputBuffer {
		sa.outputBuffer[i] = 0
	}
	
	// Reset positions
	sa.writePos = 0
	sa.avgWritePos = 0
	sa.avgCount = 0
	
	// Clear averaging buffers
	if sa.avgBuffer != nil {
		for i := range sa.avgBuffer {
			for j := range sa.avgBuffer[i] {
				sa.avgBuffer[i][j] = 0
			}
		}
	}
}

// GetPeakFrequency finds the frequency with the highest magnitude
func (sa *SpectrumAnalyzer) GetPeakFrequency() (float64, float64) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	maxMag := 0.0
	maxBin := 0
	
	for i := sa.minBin; i < sa.maxBin && i < len(sa.outputBuffer); i++ {
		if sa.outputBuffer[i] > maxMag {
			maxMag = sa.outputBuffer[i]
			maxBin = i
		}
	}
	
	freq := sa.GetFrequencyForBin(maxBin)
	return freq, maxMag
}

// GetBandEnergy calculates the energy in a frequency band
func (sa *SpectrumAnalyzer) GetBandEnergy(minFreq, maxFreq float64) float64 {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	minBin := sa.GetBinForFrequency(minFreq)
	maxBin := sa.GetBinForFrequency(maxFreq)
	
	if minBin < 0 {
		minBin = 0
	}
	if maxBin >= len(sa.outputBuffer) {
		maxBin = len(sa.outputBuffer) - 1
	}
	
	energy := 0.0
	for i := minBin; i <= maxBin; i++ {
		energy += sa.outputBuffer[i] * sa.outputBuffer[i]
	}
	
	return energy
}

// GetOctaveBands returns spectrum data grouped into octave bands
func (sa *SpectrumAnalyzer) GetOctaveBands(centerFreqs []float64) []float64 {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	bands := make([]float64, len(centerFreqs))
	
	for i, centerFreq := range centerFreqs {
		// Calculate octave band limits
		lowerFreq := centerFreq / math.Sqrt(2)
		upperFreq := centerFreq * math.Sqrt(2)
		
		// Sum energy in band
		energy := 0.0
		count := 0
		
		lowerBin := sa.GetBinForFrequency(lowerFreq)
		upperBin := sa.GetBinForFrequency(upperFreq)
		
		for bin := lowerBin; bin <= upperBin && bin < len(sa.outputBuffer); bin++ {
			if bin >= 0 {
				energy += sa.outputBuffer[bin] * sa.outputBuffer[bin]
				count++
			}
		}
		
		if count > 0 {
			bands[i] = math.Sqrt(energy / float64(count))
		}
	}
	
	return bands
}

// StandardOctaveBands returns standard octave band center frequencies
func StandardOctaveBands() []float64 {
	return []float64{31.5, 63, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}
}

// StandardThirdOctaveBands returns standard 1/3 octave band center frequencies
func StandardThirdOctaveBands() []float64 {
	bands := []float64{}
	baseFreq := 1000.0
	
	// Generate bands from 20 Hz to 20 kHz
	for i := -16; i <= 13; i++ {
		freq := baseFreq * math.Pow(2, float64(i)/3.0)
		if freq >= 20 && freq <= 20000 {
			bands = append(bands, freq)
		}
	}
	
	return bands
}