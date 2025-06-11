package analysis

import (
	"math"
	"math/cmplx"
)

// FFT performs a Fast Fourier Transform on the input data
type FFT struct {
	size       int
	window     WindowFunc
	windowData []float64
	real       []float64
	imag       []float64
	magnitude  []float64
	phase      []float64
}

// WindowFunc represents a window function type
type WindowFunc int

const (
	RectangularWindow WindowFunc = iota
	HannWindow
	HammingWindow
	BlackmanWindow
	BlackmanHarrisWindow
	KaiserWindow
	FlatTopWindow
)

// NewFFT creates a new FFT processor with the specified size and window function
func NewFFT(size int, window WindowFunc) *FFT {
	fft := &FFT{
		size:       size,
		window:     window,
		windowData: make([]float64, size),
		real:       make([]float64, size),
		imag:       make([]float64, size),
		magnitude:  make([]float64, size/2+1),
		phase:      make([]float64, size/2+1),
	}
	
	// Pre-calculate window coefficients
	fft.calculateWindow()
	
	return fft
}

// calculateWindow pre-calculates the window coefficients
func (f *FFT) calculateWindow() {
	n := float64(f.size)
	
	switch f.window {
	case RectangularWindow:
		for i := 0; i < f.size; i++ {
			f.windowData[i] = 1.0
		}
	
	case HannWindow:
		for i := 0; i < f.size; i++ {
			f.windowData[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/(n-1.0)))
		}
	
	case HammingWindow:
		for i := 0; i < f.size; i++ {
			f.windowData[i] = 0.54 - 0.46*math.Cos(2.0*math.Pi*float64(i)/(n-1.0))
		}
	
	case BlackmanWindow:
		for i := 0; i < f.size; i++ {
			val := 0.42 - 0.5*math.Cos(2.0*math.Pi*float64(i)/(n-1.0)) +
				0.08*math.Cos(4.0*math.Pi*float64(i)/(n-1.0))
			if val < 0 {
				val = 0
			}
			f.windowData[i] = val
		}
	
	case BlackmanHarrisWindow:
		a0, a1, a2, a3 := 0.35875, 0.48829, 0.14128, 0.01168
		for i := 0; i < f.size; i++ {
			f.windowData[i] = a0 - a1*math.Cos(2.0*math.Pi*float64(i)/(n-1.0)) +
				a2*math.Cos(4.0*math.Pi*float64(i)/(n-1.0)) -
				a3*math.Cos(6.0*math.Pi*float64(i)/(n-1.0))
		}
	
	case KaiserWindow:
		// Kaiser window with beta = 8.6 (good sidelobe suppression)
		beta := 8.6
		for i := 0; i < f.size; i++ {
			x := 2.0*float64(i)/(n-1.0) - 1.0
			f.windowData[i] = bessel0(beta*math.Sqrt(1.0-x*x)) / bessel0(beta)
		}
	
	case FlatTopWindow:
		a0, a1, a2, a3, a4 := 0.21557895, 0.41663158, 0.277263158, 0.083578947, 0.006947368
		for i := 0; i < f.size; i++ {
			val := a0 - a1*math.Cos(2.0*math.Pi*float64(i)/(n-1.0)) +
				a2*math.Cos(4.0*math.Pi*float64(i)/(n-1.0)) -
				a3*math.Cos(6.0*math.Pi*float64(i)/(n-1.0)) +
				a4*math.Cos(8.0*math.Pi*float64(i)/(n-1.0))
			if val < 0 {
				val = 0
			}
			f.windowData[i] = val
		}
	}
}

// Forward performs a forward FFT on the input data
// Returns magnitude and phase spectra
func (f *FFT) Forward(input []float64) (magnitude, phase []float64) {
	// Apply window
	for i := 0; i < f.size && i < len(input); i++ {
		f.real[i] = input[i] * f.windowData[i]
		f.imag[i] = 0.0
	}
	
	// Pad with zeros if input is shorter than FFT size
	for i := len(input); i < f.size; i++ {
		f.real[i] = 0.0
		f.imag[i] = 0.0
	}
	
	// Perform FFT
	f.fft(f.real, f.imag)
	
	// Calculate magnitude and phase
	for i := 0; i <= f.size/2; i++ {
		f.magnitude[i] = math.Sqrt(f.real[i]*f.real[i] + f.imag[i]*f.imag[i])
		f.phase[i] = math.Atan2(f.imag[i], f.real[i])
	}
	
	return f.magnitude, f.phase
}

// ForwardComplex performs a forward FFT on complex input data
func (f *FFT) ForwardComplex(input []complex128) []complex128 {
	// Copy input to working arrays
	for i := 0; i < f.size && i < len(input); i++ {
		c := input[i] * complex(f.windowData[i], 0)
		f.real[i] = real(c)
		f.imag[i] = imag(c)
	}
	
	// Pad with zeros if needed
	for i := len(input); i < f.size; i++ {
		f.real[i] = 0.0
		f.imag[i] = 0.0
	}
	
	// Perform FFT
	f.fft(f.real, f.imag)
	
	// Convert back to complex
	result := make([]complex128, f.size)
	for i := 0; i < f.size; i++ {
		result[i] = complex(f.real[i], f.imag[i])
	}
	
	return result
}

// Inverse performs an inverse FFT
func (f *FFT) Inverse(real, imag []float64) []float64 {
	// Copy input
	copy(f.real, real)
	copy(f.imag, imag)
	
	// Conjugate
	for i := 0; i < f.size; i++ {
		f.imag[i] = -f.imag[i]
	}
	
	// Forward FFT
	f.fft(f.real, f.imag)
	
	// Conjugate and scale
	result := make([]float64, f.size)
	scale := 1.0 / float64(f.size)
	for i := 0; i < f.size; i++ {
		result[i] = f.real[i] * scale
	}
	
	return result
}

// fft performs the actual FFT using Cooley-Tukey algorithm
func (f *FFT) fft(real, imag []float64) {
	n := f.size
	
	// Bit reversal
	j := 0
	for i := 0; i < n; i++ {
		if i < j {
			real[i], real[j] = real[j], real[i]
			imag[i], imag[j] = imag[j], imag[i]
		}
		m := n >> 1
		for m >= 1 && j >= m {
			j -= m
			m >>= 1
		}
		j += m
	}
	
	// Cooley-Tukey FFT
	for stage := 2; stage <= n; stage <<= 1 {
		theta := -2.0 * math.Pi / float64(stage)
		wReal := math.Cos(theta)
		wImag := math.Sin(theta)
		
		for k := 0; k < n; k += stage {
			wTempReal := 1.0
			wTempImag := 0.0
			
			for j := 0; j < stage/2; j++ {
				i1 := k + j
				i2 := i1 + stage/2
				
				tempReal := wTempReal*real[i2] - wTempImag*imag[i2]
				tempImag := wTempReal*imag[i2] + wTempImag*real[i2]
				
				real[i2] = real[i1] - tempReal
				imag[i2] = imag[i1] - tempImag
				
				real[i1] += tempReal
				imag[i1] += tempImag
				
				// Update twiddle factor
				oldWReal := wTempReal
				wTempReal = oldWReal*wReal - wTempImag*wImag
				wTempImag = oldWReal*wImag + wTempImag*wReal
			}
		}
	}
}

// GetMagnitudeDB returns the magnitude spectrum in decibels
func (f *FFT) GetMagnitudeDB() []float64 {
	db := make([]float64, len(f.magnitude))
	for i, mag := range f.magnitude {
		if mag > 0 {
			db[i] = 20.0 * math.Log10(mag)
		} else {
			db[i] = -120.0 // Floor at -120 dB
		}
	}
	return db
}

// GetFrequencyBin returns the frequency corresponding to a given FFT bin
func (f *FFT) GetFrequencyBin(bin int, sampleRate float64) float64 {
	return float64(bin) * sampleRate / float64(f.size)
}

// bessel0 computes the modified Bessel function of the first kind, order 0
func bessel0(x float64) float64 {
	if x == 0.0 {
		return 1.0
	}
	
	ax := math.Abs(x)
	
	if ax < 3.75 {
		y := x / 3.75
		y *= y
		return 1.0 + y*(3.5156229+y*(3.0899424+y*(1.2067492+
			y*(0.2659732+y*(0.360768e-1+y*0.45813e-2)))))
	}
	
	y := 3.75 / ax
	return (math.Exp(ax) / math.Sqrt(ax)) * (0.39894228 + y*(0.1328592e-1+
		y*(0.225319e-2+y*(-0.157565e-2+y*(0.916281e-2+
		y*(-0.2057706e-1+y*(0.2635537e-1+y*(-0.1647633e-1+
		y*0.392377e-2))))))))
}

// PowerSpectrum calculates the power spectrum from magnitude spectrum
func PowerSpectrum(magnitude []float64) []float64 {
	power := make([]float64, len(magnitude))
	for i, mag := range magnitude {
		power[i] = mag * mag
	}
	return power
}

// ApplyHannWindow applies a Hann window to the input data
func ApplyHannWindow(data []float64) []float64 {
	n := len(data)
	windowed := make([]float64, n)
	for i := 0; i < n; i++ {
		window := 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(n-1)))
		windowed[i] = data[i] * window
	}
	return windowed
}

// CrossCorrelation computes the cross-correlation between two signals using FFT
func CrossCorrelation(a, b []float64) []float64 {
	n := len(a)
	if len(b) != n {
		panic("signals must have the same length")
	}
	
	// Pad to next power of 2 for FFT efficiency
	size := 1
	for size < 2*n {
		size <<= 1
	}
	
	fft := NewFFT(size, RectangularWindow)
	
	// Zero-pad inputs
	paddedA := make([]float64, size)
	paddedB := make([]float64, size)
	copy(paddedA, a)
	copy(paddedB, b)
	
	// Forward FFT
	magA, phaseA := fft.Forward(paddedA)
	magB, phaseB := fft.Forward(paddedB)
	
	// Multiply A with conjugate of B
	realResult := make([]float64, size)
	imagResult := make([]float64, size)
	for i := 0; i < size; i++ {
		// Convert to complex, multiply, then back to real/imag
		aComplex := cmplx.Rect(magA[i%len(magA)], phaseA[i%len(phaseA)])
		bComplex := cmplx.Rect(magB[i%len(magB)], -phaseB[i%len(phaseB)]) // conjugate
		result := aComplex * bComplex
		realResult[i] = real(result)
		imagResult[i] = imag(result)
	}
	
	// Inverse FFT
	correlation := fft.Inverse(realResult, imagResult)
	
	// Extract valid correlation values
	result := make([]float64, 2*n-1)
	// Negative lags
	for i := 0; i < n-1; i++ {
		result[i] = correlation[size-n+1+i]
	}
	// Zero and positive lags
	for i := 0; i < n; i++ {
		result[n-1+i] = correlation[i]
	}
	
	return result
}