package analysis

import (
	"fmt"
	"math"
	"testing"
)

func TestSpectrumAnalyzer(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 1024
	sa := NewSpectrumAnalyzer(fftSize, sampleRate, HannWindow)
	
	// Generate test signal with 440 Hz sine wave
	freq := 440.0
	duration := 0.1 // 100ms
	numSamples := int(duration * sampleRate)
	samples := make([]float64, numSamples)
	
	for i := 0; i < numSamples; i++ {
		samples[i] = math.Sin(2.0 * math.Pi * freq * float64(i) / sampleRate)
	}
	
	// Process samples
	spectrumReady := false
	for i := 0; i < len(samples); i += 256 {
		end := i + 256
		if end > len(samples) {
			end = len(samples)
		}
		if sa.Process(samples[i:end]) {
			spectrumReady = true
		}
	}
	
	if !spectrumReady {
		t.Error("No spectrum was produced")
		return
	}
	
	// Get spectrum and find peak
	spectrum := sa.GetSpectrum()
	peakFreq, peakMag := sa.GetPeakFrequency()
	
	// Check if peak is at expected frequency
	freqTolerance := sampleRate / float64(fftSize) // One bin width
	if math.Abs(peakFreq-freq) > freqTolerance {
		t.Errorf("Peak frequency mismatch: expected %f Hz, got %f Hz", freq, peakFreq)
	}
	
	// Check magnitude is reasonable
	if peakMag < 0.1 {
		t.Errorf("Peak magnitude too low: %f", peakMag)
	}
	
	// Check spectrum length
	expectedLen := fftSize/2 + 1
	if len(spectrum) != expectedLen {
		t.Errorf("Spectrum length mismatch: expected %d, got %d", expectedLen, len(spectrum))
	}
}

func TestSpectrumAveraging(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 512
	
	tests := []struct {
		name string
		mode AveragingMode
	}{
		{"NoAveraging", NoAveraging},
		{"ExponentialAveraging", ExponentialAveraging},
		{"LinearAveraging", LinearAveraging},
		{"PeakHold", PeakHold},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := NewSpectrumAnalyzer(fftSize, sampleRate, RectangularWindow)
			sa.SetAveraging(tt.mode, 10)
			sa.SetSmoothing(0.9)
			
			// Generate test signal
			samples := make([]float64, fftSize*4)
			for i := range samples {
				// Use different amplitude for each frame
				amplitude := 0.5 + 0.1*float64(i/fftSize)
				samples[i] = amplitude * math.Sin(2.0*math.Pi*1000.0*float64(i)/sampleRate)
			}
			
			// Process multiple frames
			frameCount := 0
			for i := 0; i < len(samples); i += fftSize/2 {
				end := i + fftSize/2
				if end > len(samples) {
					end = len(samples)
				}
				if sa.Process(samples[i:end]) {
					frameCount++
				}
			}
			
			if frameCount == 0 {
				t.Error("No frames were processed")
			}
			
			// Get final spectrum
			spectrum := sa.GetSpectrum()
			if len(spectrum) == 0 {
				t.Error("Empty spectrum returned")
			}
			
			// For peak hold, the maximum should be retained
			if tt.mode == PeakHold {
				_, peakMag := sa.GetPeakFrequency()
				if peakMag < 0.7 { // Should have captured the highest amplitude
					t.Errorf("Peak hold didn't retain maximum: %f", peakMag)
				}
			}
		})
	}
}

func TestSpectrumFrequencyRange(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 1024
	sa := NewSpectrumAnalyzer(fftSize, sampleRate, HannWindow)
	
	// Set frequency range
	minFreq := 100.0
	maxFreq := 2000.0
	sa.SetFrequencyRange(minFreq, maxFreq)
	
	// Generate test signal with multiple frequencies
	samples := make([]float64, fftSize*2)
	freqs := []float64{50, 200, 1000, 3000} // Some inside, some outside range
	
	for i := range samples {
		for _, freq := range freqs {
			samples[i] += 0.25 * math.Sin(2.0*math.Pi*freq*float64(i)/sampleRate)
		}
	}
	
	// Process
	sa.Process(samples[:fftSize])
	
	// Get spectrum in range
	spectrumInRange := sa.GetSpectrumInRange()
	
	// Calculate expected number of bins in range
	binWidth := sampleRate / float64(fftSize)
	expectedBins := int((maxFreq-minFreq)/binWidth) + 1
	
	// Allow some tolerance
	if math.Abs(float64(len(spectrumInRange)-expectedBins)) > 2 {
		t.Errorf("Spectrum range bins mismatch: expected ~%d, got %d", 
			expectedBins, len(spectrumInRange))
	}
}

func TestSpectrumDB(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 512
	sa := NewSpectrumAnalyzer(fftSize, sampleRate, RectangularWindow)
	
	// Generate unity amplitude signal
	samples := make([]float64, fftSize)
	for i := range samples {
		samples[i] = 1.0 // DC signal
	}
	
	sa.Process(samples)
	
	// Get spectrum in dB
	spectrumDB := sa.GetSpectrumDB()
	
	// DC bin should have significant magnitude
	if spectrumDB[0] < 20.0 {
		t.Errorf("DC magnitude in dB too low: %f", spectrumDB[0])
	}
	
	// Other bins should be very low
	for i := 10; i < len(spectrumDB); i++ {
		if spectrumDB[i] > -60.0 {
			t.Errorf("Non-DC bin %d has unexpected magnitude: %f dB", i, spectrumDB[i])
			break
		}
	}
}

func TestBandEnergy(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 1024
	sa := NewSpectrumAnalyzer(fftSize, sampleRate, HannWindow)
	
	// Generate test signal with specific frequency
	freq := 1000.0
	samples := make([]float64, fftSize)
	for i := range samples {
		samples[i] = math.Sin(2.0 * math.Pi * freq * float64(i) / sampleRate)
	}
	
	sa.Process(samples)
	
	// Calculate band energy around the signal frequency
	bandMin := 900.0
	bandMax := 1100.0
	bandEnergy := sa.GetBandEnergy(bandMin, bandMax)
	
	// Calculate energy outside the band
	outsideEnergy := sa.GetBandEnergy(2000.0, 3000.0)
	
	// Energy in band should be much higher than outside
	if bandEnergy < outsideEnergy*10 {
		t.Errorf("Band energy detection failed: in-band %f, out-of-band %f", 
			bandEnergy, outsideEnergy)
	}
}

func TestOctaveBands(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 2048
	sa := NewSpectrumAnalyzer(fftSize, sampleRate, HannWindow)
	
	// Generate test signal with 1 kHz
	freq := 1000.0
	samples := make([]float64, fftSize)
	for i := range samples {
		samples[i] = math.Sin(2.0 * math.Pi * freq * float64(i) / sampleRate)
	}
	
	sa.Process(samples)
	
	// Get octave bands
	centerFreqs := StandardOctaveBands()
	bands := sa.GetOctaveBands(centerFreqs)
	
	// Find which band contains 1 kHz
	maxBandIdx := 0
	maxBandValue := 0.0
	for i, value := range bands {
		if value > maxBandValue {
			maxBandValue = value
			maxBandIdx = i
		}
	}
	
	// 1 kHz should be in the 1 kHz octave band
	expectedIdx := -1
	for i, cf := range centerFreqs {
		if cf == 1000.0 {
			expectedIdx = i
			break
		}
	}
	
	if expectedIdx < 0 {
		t.Error("1 kHz center frequency not found in standard octave bands")
		return
	}
	
	if maxBandIdx != expectedIdx {
		t.Errorf("Peak octave band mismatch: expected band %d (%.1f Hz), got band %d (%.1f Hz)",
			expectedIdx, centerFreqs[expectedIdx], maxBandIdx, centerFreqs[maxBandIdx])
	}
}

func TestSpectrumReset(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 512
	sa := NewSpectrumAnalyzer(fftSize, sampleRate, HannWindow)
	sa.SetAveraging(PeakHold, 0)
	
	// Process some signal
	samples := make([]float64, fftSize)
	for i := range samples {
		samples[i] = math.Sin(2.0 * math.Pi * 1000.0 * float64(i) / sampleRate)
	}
	sa.Process(samples)
	
	// Get spectrum before reset
	spectrumBefore := sa.GetSpectrum()
	maxBefore := 0.0
	for _, val := range spectrumBefore {
		if val > maxBefore {
			maxBefore = val
		}
	}
	
	// Reset
	sa.Reset()
	
	// Get spectrum after reset
	spectrumAfter := sa.GetSpectrum()
	maxAfter := 0.0
	for _, val := range spectrumAfter {
		if val > maxAfter {
			maxAfter = val
		}
	}
	
	// After reset, spectrum should be zero
	if maxAfter > 1e-10 {
		t.Errorf("Spectrum not cleared after reset: max value %f", maxAfter)
	}
	
	// Before reset should have had signal
	if maxBefore < 0.1 {
		t.Errorf("No signal detected before reset: max value %f", maxBefore)
	}
}

func TestHopSize(t *testing.T) {
	sampleRate := 44100.0
	fftSize := 1024
	sa := NewSpectrumAnalyzer(fftSize, sampleRate, HannWindow)
	
	// Test different hop sizes
	hopSizes := []int{256, 512, 1024}
	
	for _, hopSize := range hopSizes {
		sa.SetHopSize(hopSize)
		sa.Reset()
		
		// Generate longer signal
		numSamples := fftSize * 4
		samples := make([]float64, numSamples)
		for i := range samples {
			samples[i] = math.Sin(2.0 * math.Pi * 440.0 * float64(i) / sampleRate)
		}
		
		// Count how many spectra are produced
		spectraCount := 0
		for i := 0; i < len(samples); i++ {
			if sa.Process(samples[i : i+1]) {
				spectraCount++
			}
		}
		
		// With smaller hop size, we should get more spectra
		if hopSize < fftSize && spectraCount < 2 {
			t.Errorf("Not enough spectra produced with hop size %d: got %d", 
				hopSize, spectraCount)
		}
	}
}

func TestStandardBands(t *testing.T) {
	// Test standard octave bands
	octaveBands := StandardOctaveBands()
	if len(octaveBands) != 10 {
		t.Errorf("Expected 10 standard octave bands, got %d", len(octaveBands))
	}
	
	// Check some expected values
	expectedOctaves := []float64{31.5, 63, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}
	for i, expected := range expectedOctaves {
		if i < len(octaveBands) && math.Abs(octaveBands[i]-expected) > 0.1 {
			t.Errorf("Octave band %d mismatch: expected %f, got %f", 
				i, expected, octaveBands[i])
		}
	}
	
	// Test third octave bands
	thirdOctaveBands := StandardThirdOctaveBands()
	if len(thirdOctaveBands) < 29 || len(thirdOctaveBands) > 31 {
		t.Errorf("Expected ~30 third octave bands, got %d", len(thirdOctaveBands))
	}
	
	// Check that 1000 Hz is included
	found1000 := false
	for _, freq := range thirdOctaveBands {
		if math.Abs(freq-1000.0) < 1.0 {
			found1000 = true
			break
		}
	}
	if !found1000 {
		t.Error("1000 Hz not found in third octave bands")
	}
}

func BenchmarkSpectrumAnalyzer(b *testing.B) {
	sampleRate := 44100.0
	fftSizes := []int{512, 1024, 2048, 4096}
	
	for _, fftSize := range fftSizes {
		b.Run(fmt.Sprintf("FFTSize%d", fftSize), func(b *testing.B) {
			sa := NewSpectrumAnalyzer(fftSize, sampleRate, HannWindow)
			samples := make([]float64, fftSize)
			
			// Generate test signal
			for i := range samples {
				samples[i] = math.Sin(2.0*math.Pi*440.0*float64(i)/sampleRate)
			}
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				sa.Process(samples)
				sa.GetSpectrum()
			}
		})
	}
}