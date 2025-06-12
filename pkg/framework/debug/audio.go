// Package debug provides debugging utilities for VST3 plugin development.
package debug

import (
	"fmt"
	"math"
	"strings"
)

// AudioAnalyzer provides utilities for analyzing audio buffers.
type AudioAnalyzer struct {
	detectClipping   bool
	detectDC         bool
	detectSilence    bool
	detectNaN        bool
	clippingThreshold float32
	dcThreshold      float32
	silenceThreshold float32
}

// NewAudioAnalyzer creates a new audio analyzer with default settings.
func NewAudioAnalyzer() *AudioAnalyzer {
	return &AudioAnalyzer{
		detectClipping:    true,
		detectDC:          true,
		detectSilence:     true,
		detectNaN:         true,
		clippingThreshold: 0.99,
		dcThreshold:       0.01,
		silenceThreshold:  0.0001,
	}
}

// AnalysisResult contains the results of audio buffer analysis.
type AnalysisResult struct {
	Peak          float32
	RMS           float32
	DC            float32
	Clipping      bool
	ClippedSamples int
	Silent        bool
	HasNaN        bool
	NaNCount      int
	ZeroCrossings int
}

// Analyze performs comprehensive analysis on an audio buffer.
func (a *AudioAnalyzer) Analyze(buffer []float32) AnalysisResult {
	result := AnalysisResult{}
	
	if len(buffer) == 0 {
		return result
	}
	
	var sum, sumSquares, dcSum float64
	var lastSample float32
	
	for i, sample := range buffer {
		// Check for NaN
		if a.detectNaN && math.IsNaN(float64(sample)) {
			result.HasNaN = true
			result.NaNCount++
			continue
		}
		
		// Absolute value for peak detection
		absSample := sample
		if absSample < 0 {
			absSample = -absSample
		}
		
		// Update peak
		if absSample > result.Peak {
			result.Peak = absSample
		}
		
		// Check for clipping
		if a.detectClipping && absSample >= a.clippingThreshold {
			result.Clipping = true
			result.ClippedSamples++
		}
		
		// Accumulate for RMS and DC
		sum += float64(sample)
		sumSquares += float64(sample * sample)
		dcSum += float64(absSample)
		
		// Count zero crossings
		if i > 0 && ((lastSample < 0 && sample >= 0) || (lastSample >= 0 && sample < 0)) {
			result.ZeroCrossings++
		}
		lastSample = sample
	}
	
	// Calculate RMS
	result.RMS = float32(math.Sqrt(sumSquares / float64(len(buffer))))
	
	// Calculate DC offset
	result.DC = float32(sum / float64(len(buffer)))
	
	// Check if silent
	if a.detectSilence && result.RMS < a.silenceThreshold {
		result.Silent = true
	}
	
	return result
}

// PrintBuffer prints a visual representation of an audio buffer.
func PrintBuffer(buffer []float32, width int) string {
	if len(buffer) == 0 {
		return "Empty buffer"
	}
	
	if width <= 0 {
		width = 80
	}
	
	// Find peak for normalization
	peak := float32(0)
	for _, sample := range buffer {
		absSample := sample
		if absSample < 0 {
			absSample = -absSample
		}
		if absSample > peak {
			peak = absSample
		}
	}
	
	if peak == 0 {
		return "Silent buffer (all zeros)"
	}
	
	// Create visual representation
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Audio Buffer Visualization (peak: %.3f):\n", peak))
	
	// Calculate samples per character
	samplesPerChar := len(buffer) / width
	if samplesPerChar < 1 {
		samplesPerChar = 1
	}
	
	// Generate waveform
	centerLine := 40 // Height of visualization
	halfHeight := centerLine / 2
	
	for i := 0; i < width && i*samplesPerChar < len(buffer); i++ {
		// Average samples for this position
		sum := float32(0)
		count := 0
		for j := 0; j < samplesPerChar && i*samplesPerChar+j < len(buffer); j++ {
			sum += buffer[i*samplesPerChar+j]
			count++
		}
		avg := sum / float32(count)
		
		// Normalize and scale
		normalized := avg / peak
		height := int(normalized * float32(halfHeight))
		
		// Draw column
		for y := halfHeight; y >= -halfHeight; y-- {
			if y == 0 {
				sb.WriteRune('-') // Center line
			} else if (height > 0 && y > 0 && y <= height) || (height < 0 && y < 0 && y >= height) {
				sb.WriteRune('█')
			} else {
				sb.WriteRune(' ')
			}
		}
		sb.WriteRune('\n')
	}
	
	return sb.String()
}

// CompareBuffers compares two audio buffers and reports differences.
func CompareBuffers(a, b []float32, tolerance float32) string {
	if len(a) != len(b) {
		return fmt.Sprintf("Buffer length mismatch: %d vs %d", len(a), len(b))
	}
	
	var maxDiff float32
	var maxDiffIndex int
	var totalDiff float64
	var diffCount int
	
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		if diff < 0 {
			diff = -diff
		}
		
		if diff > tolerance {
			diffCount++
			totalDiff += float64(diff)
			
			if diff > maxDiff {
				maxDiff = diff
				maxDiffIndex = i
			}
		}
	}
	
	if diffCount == 0 {
		return "Buffers are identical within tolerance"
	}
	
	avgDiff := totalDiff / float64(diffCount)
	
	return fmt.Sprintf("Buffer differences:\n"+
		"  Samples different: %d / %d (%.1f%%)\n"+
		"  Max difference: %.6f at sample %d\n"+
		"  Average difference: %.6f\n"+
		"  Tolerance: %.6f",
		diffCount, len(a), float64(diffCount)/float64(len(a))*100,
		maxDiff, maxDiffIndex,
		avgDiff,
		tolerance)
}

// CheckBuffer performs basic sanity checks on an audio buffer.
func CheckBuffer(buffer []float32, name string) []string {
	var issues []string
	
	analyzer := NewAudioAnalyzer()
	result := analyzer.Analyze(buffer)
	
	if result.HasNaN {
		issues = append(issues, fmt.Sprintf("%s: Contains %d NaN values", name, result.NaNCount))
	}
	
	if result.Clipping {
		issues = append(issues, fmt.Sprintf("%s: Clipping detected (%d samples)", name, result.ClippedSamples))
	}
	
	if math.Abs(float64(result.DC)) > float64(analyzer.dcThreshold) {
		issues = append(issues, fmt.Sprintf("%s: DC offset detected (%.3f)", name, result.DC))
	}
	
	if result.Peak > 1.0 {
		issues = append(issues, fmt.Sprintf("%s: Peak exceeds 1.0 (%.3f)", name, result.Peak))
	}
	
	return issues
}

// DumpBuffer creates a detailed dump of an audio buffer for debugging.
func DumpBuffer(buffer []float32, maxSamples int) string {
	if len(buffer) == 0 {
		return "Empty buffer"
	}
	
	if maxSamples <= 0 || maxSamples > len(buffer) {
		maxSamples = len(buffer)
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Audio Buffer Dump (%d samples, showing first %d):\n", len(buffer), maxSamples))
	sb.WriteString("Index | Value      | Hex        | Bar\n")
	sb.WriteString("------|------------|------------|--------------------\n")
	
	for i := 0; i < maxSamples; i++ {
		sample := buffer[i]
		
		// Create a simple bar visualization
		barWidth := 20
		normalized := sample
		if normalized > 1.0 {
			normalized = 1.0
		} else if normalized < -1.0 {
			normalized = -1.0
		}
		
		barPos := int((normalized + 1.0) * float32(barWidth) / 2.0)
		bar := strings.Repeat(" ", barWidth)
		if barPos >= 0 && barPos < barWidth {
			bar = bar[:barPos] + "|" + bar[barPos+1:]
		}
		
		sb.WriteString(fmt.Sprintf("%5d | %+.6f | 0x%08X | %s\n", 
			i, sample, math.Float32bits(sample), bar))
	}
	
	if maxSamples < len(buffer) {
		sb.WriteString(fmt.Sprintf("... %d more samples ...\n", len(buffer)-maxSamples))
	}
	
	return sb.String()
}

// Global audio debugging functions

var defaultAnalyzer = NewAudioAnalyzer()

// AnalyzeBuffer performs analysis on a buffer using the default analyzer.
func AnalyzeBuffer(buffer []float32) AnalysisResult {
	return defaultAnalyzer.Analyze(buffer)
}

// CheckAudioBuffer performs sanity checks using the default analyzer.
func CheckAudioBuffer(buffer []float32, name string) {
	issues := CheckBuffer(buffer, name)
	for _, issue := range issues {
		Warn("%s", issue)
	}
}

// LogBufferStats logs statistics about an audio buffer.
func LogBufferStats(buffer []float32, name string) {
	result := defaultAnalyzer.Analyze(buffer)
	
	Info("Audio buffer '%s' stats:", name)
	Info("  Samples: %d", len(buffer))
	Info("  Peak: %.3f", result.Peak)
	Info("  RMS: %.3f", result.RMS)
	Info("  DC: %.6f", result.DC)
	
	if result.Clipping {
		Warn("  Clipping: %d samples", result.ClippedSamples)
	}
	if result.Silent {
		Info("  Status: Silent")
	}
	if result.HasNaN {
		Error("  NaN values: %d", result.NaNCount)
	}
}