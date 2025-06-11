// Package analysis provides audio analysis tools for VST3 plugins.
//
// This package includes a comprehensive set of analysis tools commonly used
// in audio processing and metering:
//
// FFT and Spectral Analysis:
//   - FFT with multiple window functions (Hann, Hamming, Blackman, etc.)
//   - Real-time spectrum analyzer with averaging modes
//   - Octave and third-octave band analysis
//   - Cross-correlation using FFT
//
// Level Metering:
//   - Peak meter with hold and decay
//   - RMS (Root Mean Square) meter
//   - LUFS meter (ITU-R BS.1770-4 compliant)
//   - Momentary, short-term, and integrated loudness
//   - Loudness range (LRA) measurement
//
// Stereo Field Analysis:
//   - Correlation meter for phase relationships
//   - Balance meter for L/R power distribution
//   - Stereo width meter using M/S analysis
//   - Mono compatibility checking
//
// Phase Visualization:
//   - Phase scope with Lissajous display
//   - Goniometer (45° rotated) display
//   - Vector scope with graticule
//   - Polar coordinate display
//
// All analysis tools are designed for real-time operation with minimal
// allocations and thread-safe access.
//
// Example usage:
//
//	// Create a spectrum analyzer
//	sa := analysis.NewSpectrumAnalyzer(2048, 44100, analysis.HannWindow)
//	sa.SetAveraging(analysis.ExponentialAveraging, 10)
//	
//	// Process audio samples
//	if sa.Process(samples) {
//	    spectrum := sa.GetSpectrumDB()
//	    peakFreq, peakMag := sa.GetPeakFrequency()
//	}
//	
//	// Create a LUFS meter
//	lufs := analysis.NewLUFSMeter(48000, 2)
//	lufs.Process(interleavedSamples)
//	
//	momentary := lufs.GetMomentaryLUFS()
//	integrated := lufs.GetIntegratedLUFS()
//	
//	// Create a correlation meter
//	corr := analysis.NewCorrelationMeter(1024, 44100)
//	corr.Process(samplesL, samplesR)
//	
//	correlation := corr.GetCorrelation()
//	monoCompat := corr.GetMonoCompatibility()
package analysis