package param

import (
	"fmt"
	"strings"
)

// Filter type constants
const (
	FilterTypeLowpass = iota
	FilterTypeHighpass
	FilterTypeBandpass
	FilterTypeNotch
	FilterTypeAllpass
	FilterTypePeaking
	FilterTypeLowShelf
	FilterTypeHighShelf
)

// FilterTypeNames provides display names for filter types
var FilterTypeNames = []string{
	"Lowpass",
	"Highpass",
	"Bandpass",
	"Notch",
	"Allpass",
	"Peaking EQ",
	"Low Shelf",
	"High Shelf",
}

// FilterTypeFormatter formats filter type values
func FilterTypeFormatter(value float64) string {
	index := int(value)
	if index >= 0 && index < len(FilterTypeNames) {
		return FilterTypeNames[index]
	}
	return "Unknown"
}

// FilterTypeParser parses filter type strings
func FilterTypeParser(str string) (float64, error) {
	// Handle common variations
	normalizedStr := strings.ToLower(strings.TrimSpace(str))

	// Map variations to standard names
	filterAliases := map[string]int{
		"lowpass":     FilterTypeLowpass,
		"low pass":    FilterTypeLowpass,
		"lpf":         FilterTypeLowpass,
		"lp":          FilterTypeLowpass,
		"highpass":    FilterTypeHighpass,
		"high pass":   FilterTypeHighpass,
		"hpf":         FilterTypeHighpass,
		"hp":          FilterTypeHighpass,
		"bandpass":    FilterTypeBandpass,
		"band pass":   FilterTypeBandpass,
		"bpf":         FilterTypeBandpass,
		"bp":          FilterTypeBandpass,
		"notch":       FilterTypeNotch,
		"band reject": FilterTypeNotch,
		"band stop":   FilterTypeNotch,
		"br":          FilterTypeNotch,
		"bs":          FilterTypeNotch,
		"allpass":     FilterTypeAllpass,
		"all pass":    FilterTypeAllpass,
		"apf":         FilterTypeAllpass,
		"ap":          FilterTypeAllpass,
		"peaking":     FilterTypePeaking,
		"peaking eq":  FilterTypePeaking,
		"peak":        FilterTypePeaking,
		"bell":        FilterTypePeaking,
		"parametric":  FilterTypePeaking,
		"low shelf":   FilterTypeLowShelf,
		"lowshelf":    FilterTypeLowShelf,
		"ls":          FilterTypeLowShelf,
		"bass":        FilterTypeLowShelf,
		"high shelf":  FilterTypeHighShelf,
		"highshelf":   FilterTypeHighShelf,
		"hs":          FilterTypeHighShelf,
		"treble":      FilterTypeHighShelf,
	}

	if index, ok := filterAliases[normalizedStr]; ok {
		return float64(index), nil
	}

	// Try exact match
	for i, name := range FilterTypeNames {
		if strings.EqualFold(str, name) {
			return float64(i), nil
		}
	}

	return 0, fmt.Errorf("unknown filter type: %s", str)
}

// Gate type constants
const (
	GateTypeHard = iota
	GateTypeSoft
	GateTypeExpander
	GateTypeDucker
)

// GateTypeNames provides display names for gate types
var GateTypeNames = []string{
	"Hard Gate",
	"Soft Gate",
	"Expander",
	"Ducker",
}

// GateTypeFormatter formats gate type values
func GateTypeFormatter(value float64) string {
	index := int(value)
	if index >= 0 && index < len(GateTypeNames) {
		return GateTypeNames[index]
	}
	return "Unknown"
}

// GateTypeParser parses gate type strings
func GateTypeParser(str string) (float64, error) {
	normalizedStr := strings.ToLower(strings.TrimSpace(str))

	gateAliases := map[string]int{
		"hard":      GateTypeHard,
		"hard gate": GateTypeHard,
		"gate":      GateTypeHard,
		"soft":      GateTypeSoft,
		"soft gate": GateTypeSoft,
		"expander":  GateTypeExpander,
		"expand":    GateTypeExpander,
		"expansion": GateTypeExpander,
		"ducker":    GateTypeDucker,
		"duck":      GateTypeDucker,
		"ducking":   GateTypeDucker,
	}

	if index, ok := gateAliases[normalizedStr]; ok {
		return float64(index), nil
	}

	for i, name := range GateTypeNames {
		if strings.EqualFold(str, name) {
			return float64(i), nil
		}
	}

	return 0, fmt.Errorf("unknown gate type: %s", str)
}
