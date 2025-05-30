package param

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Common parameter formatters and parsers

// FrequencyFormatter formats frequency values with Hz/kHz
func FrequencyFormatter(hz float64) string {
	if hz >= 1000 {
		return fmt.Sprintf("%.2f kHz", hz/1000)
	}
	return fmt.Sprintf("%.1f Hz", hz)
}

// FrequencyParser parses frequency strings
func FrequencyParser(str string) (float64, error) {
	str = strings.TrimSpace(str)
	
	// Handle kHz
	if strings.HasSuffix(str, "kHz") || strings.HasSuffix(str, "khz") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(str, "kHz"), "khz")
		numStr = strings.TrimSpace(numStr)
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, err
		}
		return val * 1000, nil
	}
	
	// Handle Hz
	str = strings.TrimSuffix(strings.TrimSuffix(str, "Hz"), "hz")
	str = strings.TrimSpace(str)
	return strconv.ParseFloat(str, 64)
}

// DecibelFormatter formats dB values
func DecibelFormatter(db float64) string {
	if db <= -60 {
		return "-∞ dB"
	}
	return fmt.Sprintf("%.1f dB", db)
}

// DecibelParser parses dB strings
func DecibelParser(str string) (float64, error) {
	if strings.Contains(str, "∞") || strings.Contains(str, "inf") {
		return -96.0, nil // Practical minimum
	}
	str = strings.TrimSuffix(strings.TrimSpace(str), "dB")
	str = strings.TrimSuffix(strings.TrimSpace(str), "db")
	return strconv.ParseFloat(strings.TrimSpace(str), 64)
}

// PercentFormatter formats percentage values
func PercentFormatter(value float64) string {
	return fmt.Sprintf("%.0f%%", value)
}

// PercentParser parses percentage strings
func PercentParser(str string) (float64, error) {
	str = strings.TrimSuffix(strings.TrimSpace(str), "%")
	return strconv.ParseFloat(str, 64)
}

// TimeFormatter formats time values with appropriate units
func TimeFormatter(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%.2f µs", ms*1000)
	} else if ms < 1000 {
		return fmt.Sprintf("%.1f ms", ms)
	}
	return fmt.Sprintf("%.2f s", ms/1000)
}

// TimeParser parses time strings
func TimeParser(str string) (float64, error) {
	str = strings.TrimSpace(str)
	
	// Handle microseconds
	if strings.HasSuffix(str, "µs") || strings.HasSuffix(str, "us") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(str, "µs"), "us")
		val, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
		if err != nil {
			return 0, err
		}
		return val / 1000, nil // Convert to ms
	}
	
	// Handle seconds
	if strings.HasSuffix(str, "s") && !strings.HasSuffix(str, "ms") {
		numStr := strings.TrimSuffix(str, "s")
		val, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
		if err != nil {
			return 0, err
		}
		return val * 1000, nil // Convert to ms
	}
	
	// Handle milliseconds (default)
	str = strings.TrimSuffix(str, "ms")
	return strconv.ParseFloat(strings.TrimSpace(str), 64)
}

// RatioFormatter formats ratio values
func RatioFormatter(value float64) string {
	return fmt.Sprintf("%.1f:1", value)
}

// RatioParser parses ratio strings
func RatioParser(str string) (float64, error) {
	str = strings.TrimSpace(str)
	str = strings.TrimSuffix(str, ":1")
	return strconv.ParseFloat(strings.TrimSpace(str), 64)
}

// PanFormatter formats pan position
func PanFormatter(pan float64) string {
	if math.Abs(pan) < 0.01 {
		return "C"
	} else if pan < 0 {
		return fmt.Sprintf("%.0fL", -pan*100)
	}
	return fmt.Sprintf("%.0fR", pan*100)
}

// PanParser parses pan position strings
func PanParser(str string) (float64, error) {
	str = strings.ToUpper(strings.TrimSpace(str))
	
	if str == "C" || str == "CENTER" {
		return 0, nil
	}
	
	if strings.HasSuffix(str, "L") {
		numStr := strings.TrimSuffix(str, "L")
		val, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
		if err != nil {
			return 0, err
		}
		return -val / 100, nil
	}
	
	if strings.HasSuffix(str, "R") {
		numStr := strings.TrimSuffix(str, "R")
		val, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
		if err != nil {
			return 0, err
		}
		return val / 100, nil
	}
	
	// Try to parse as plain number (-1 to 1)
	return strconv.ParseFloat(str, 64)
}

// NoteFormatter formats MIDI note numbers
func NoteFormatter(noteNumber float64) string {
	notes := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	note := int(noteNumber) % 12
	octave := int(noteNumber)/12 - 1
	return fmt.Sprintf("%s%d", notes[note], octave)
}

// NoteParser parses note names to MIDI numbers
func NoteParser(str string) (float64, error) {
	str = strings.ToUpper(strings.TrimSpace(str))
	
	noteMap := map[string]int{
		"C": 0, "B#": 0,
		"C#": 1, "DB": 1,
		"D": 2,
		"D#": 3, "EB": 3,
		"E": 4, "FB": 4,
		"F": 5, "E#": 5,
		"F#": 6, "GB": 6,
		"G": 7,
		"G#": 8, "AB": 8,
		"A": 9,
		"A#": 10, "BB": 10,
		"B": 11, "CB": 11,
	}
	
	// Find where the octave number starts
	octaveStart := -1
	for i, ch := range str {
		if ch >= '0' && ch <= '9' || ch == '-' {
			octaveStart = i
			break
		}
	}
	
	if octaveStart == -1 {
		return 0, fmt.Errorf("no octave number found in note: %s", str)
	}
	
	noteName := str[:octaveStart]
	octaveStr := str[octaveStart:]
	
	noteOffset, ok := noteMap[noteName]
	if !ok {
		return 0, fmt.Errorf("unknown note name: %s", noteName)
	}
	
	octave, err := strconv.Atoi(octaveStr)
	if err != nil {
		return 0, fmt.Errorf("invalid octave number: %s", octaveStr)
	}
	
	return float64((octave+1)*12 + noteOffset), nil
}

// OnOffFormatter formats boolean as On/Off
func OnOffFormatter(value float64) string {
	if value > 0.5 {
		return "On"
	}
	return "Off"
}

// OnOffParser parses On/Off strings
func OnOffParser(str string) (float64, error) {
	str = strings.ToLower(strings.TrimSpace(str))
	switch str {
	case "on", "yes", "true", "1":
		return 1, nil
	case "off", "no", "false", "0":
		return 0, nil
	default:
		return 0, fmt.Errorf("expected 'on' or 'off', got: %s", str)
	}
}