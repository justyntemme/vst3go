package param

import (
	"fmt"
	"strings"
)

// Distortion type constants
const (
	DistortionTypeWaveshaper = iota
	DistortionTypeTube
	DistortionTypeTape
	DistortionTypeBitCrusher
)

// DistortionTypeNames provides display names for distortion types
var DistortionTypeNames = []string{
	"Waveshaper",
	"Tube",
	"Tape",
	"BitCrusher",
}

// DistortionTypeFormatter formats distortion type values
func DistortionTypeFormatter(value float64) string {
	index := int(value)
	if index >= 0 && index < len(DistortionTypeNames) {
		return DistortionTypeNames[index]
	}
	return "Unknown"
}

// DistortionTypeParser parses distortion type strings
func DistortionTypeParser(str string) (float64, error) {
	normalizedStr := strings.ToLower(strings.TrimSpace(str))
	
	// Map variations to standard names
	distortionAliases := map[string]int{
		"waveshaper":  DistortionTypeWaveshaper,
		"wave shaper": DistortionTypeWaveshaper,
		"ws":          DistortionTypeWaveshaper,
		"tube":        DistortionTypeTube,
		"valve":       DistortionTypeTube,
		"tape":        DistortionTypeTape,
		"analog":      DistortionTypeTape,
		"bitcrusher":  DistortionTypeBitCrusher,
		"bit crusher": DistortionTypeBitCrusher,
		"lofi":        DistortionTypeBitCrusher,
		"lo-fi":       DistortionTypeBitCrusher,
	}
	
	if index, ok := distortionAliases[normalizedStr]; ok {
		return float64(index), nil
	}
	
	// Try exact match
	for i, name := range DistortionTypeNames {
		if strings.EqualFold(str, name) {
			return float64(i), nil
		}
	}
	
	return 0, fmt.Errorf("unknown distortion type: %s", str)
}

// Waveshaper curve constants
const (
	WaveshaperCurveHardClip = iota
	WaveshaperCurveSoftClip
	WaveshaperCurveSaturate
	WaveshaperCurveFoldback
	WaveshaperCurveAsymmetric
	WaveshaperCurveSine
	WaveshaperCurveExponential
)

// WaveshaperCurveNames provides display names for waveshaper curves
var WaveshaperCurveNames = []string{
	"Hard Clip",
	"Soft Clip",
	"Saturate",
	"Foldback",
	"Asymmetric",
	"Sine",
	"Exponential",
}

// WaveshaperCurveFormatter formats waveshaper curve values
func WaveshaperCurveFormatter(value float64) string {
	index := int(value)
	if index >= 0 && index < len(WaveshaperCurveNames) {
		return WaveshaperCurveNames[index]
	}
	return "Unknown"
}

// WaveshaperCurveParser parses waveshaper curve strings
func WaveshaperCurveParser(str string) (float64, error) {
	normalizedStr := strings.ToLower(strings.TrimSpace(str))
	
	// Map variations to standard names
	curveAliases := map[string]int{
		"hard clip":    WaveshaperCurveHardClip,
		"hardclip":     WaveshaperCurveHardClip,
		"hard":         WaveshaperCurveHardClip,
		"soft clip":    WaveshaperCurveSoftClip,
		"softclip":     WaveshaperCurveSoftClip,
		"soft":         WaveshaperCurveSoftClip,
		"saturate":     WaveshaperCurveSaturate,
		"saturation":   WaveshaperCurveSaturate,
		"foldback":     WaveshaperCurveFoldback,
		"fold":         WaveshaperCurveFoldback,
		"asymmetric":   WaveshaperCurveAsymmetric,
		"asym":         WaveshaperCurveAsymmetric,
		"sine":         WaveshaperCurveSine,
		"sin":          WaveshaperCurveSine,
		"exponential":  WaveshaperCurveExponential,
		"exp":          WaveshaperCurveExponential,
	}
	
	if index, ok := curveAliases[normalizedStr]; ok {
		return float64(index), nil
	}
	
	// Try exact match
	for i, name := range WaveshaperCurveNames {
		if strings.EqualFold(str, name) {
			return float64(i), nil
		}
	}
	
	return 0, fmt.Errorf("unknown waveshaper curve: %s", str)
}