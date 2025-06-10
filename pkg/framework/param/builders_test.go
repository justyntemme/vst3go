package param

import (
	"math"
	"testing"
)

func TestChoice(t *testing.T) {
	options := []ChoiceOption{
		{Value: 0, Name: "Off", Aliases: []string{"disabled", "none"}},
		{Value: 1, Name: "Low", Aliases: []string{"lo", "minimum"}},
		{Value: 2, Name: "Medium", Aliases: []string{"med", "mid", "normal"}},
		{Value: 3, Name: "High", Aliases: []string{"hi", "maximum"}},
	}

	param := Choice(100, "Mode", options).Build()

	t.Run("Formatter", func(t *testing.T) {
		tests := []struct {
			value    float64
			expected string
		}{
			{0, "Off"},
			{1, "Low"},
			{2, "Medium"},
			{3, "High"},
		}

		for _, test := range tests {
			// Need to normalize the value first
			normalized := test.value / 3.0 // 0-3 range
			result := param.FormatValue(normalized)
			if result != test.expected {
				t.Errorf("FormatValue(%f) = %s, want %s", test.value, result, test.expected)
			}
		}
	})

	t.Run("Parser", func(t *testing.T) {
		tests := []struct {
			input         string
			expectedPlain float64
		}{
			{"Off", 0},
			{"disabled", 0},
			{"Low", 1},
			{"lo", 1},
			{"Medium", 2},
			{"med", 2},
			{"High", 3},
			{"hi", 3},
		}

		for _, test := range tests {
			normalized, err := param.ParseValue(test.input)
			if err != nil {
				t.Errorf("ParseValue(%s) error: %v", test.input, err)
				continue
			}
			plain := param.Denormalize(normalized)
			if math.Abs(plain-test.expectedPlain) > 0.001 {
				t.Errorf("ParseValue(%s) = %f (plain), want %f", test.input, plain, test.expectedPlain)
			}
		}
	})
}

func TestGainParameter(t *testing.T) {
	param := GainParameter(200, "Output Gain").Build()

	t.Run("Formatter", func(t *testing.T) {
		tests := []struct {
			plainValue float64
			expected   string
		}{
			{-80, "-∞ dB"},
			{0, "0.0 dB"},
			{6, "6.0 dB"},
			{-6, "-6.0 dB"},
		}

		for _, test := range tests {
			// Format using normalized value
			normalized := param.Normalize(test.plainValue)
			result := param.FormatValue(normalized)
			if result != test.expected {
				t.Errorf("FormatValue(%f dB) = %s, want %s", test.plainValue, result, test.expected)
			}
		}
	})

	t.Run("Parser", func(t *testing.T) {
		normalized, err := param.ParseValue("-inf dB")
		if err != nil {
			t.Errorf("ParseValue error: %v", err)
		}
		plainValue := param.Denormalize(normalized)
		if plainValue != -80 {
			t.Errorf("ParseValue(-inf dB) = %f, want -80", plainValue)
		}
	})
}

func TestMixParameter(t *testing.T) {
	param := MixParameter(300, "Dry/Wet Mix").Build()

	if param.Min != 0 || param.Max != 100 {
		t.Errorf("Mix parameter range should be 0-100, got %f-%f", param.Min, param.Max)
	}

	if param.DefaultValue != 1.0 {
		t.Errorf("Mix parameter default should be 100%% (normalized 1.0), got %f", param.DefaultValue)
	}
}

func TestFrequencyParameter(t *testing.T) {
	param := FrequencyParameter(400, "Cutoff", 20, 20000, 1000).Build()

	// Format using normalized value
	normalized := param.Normalize(1000)
	result := param.FormatValue(normalized)
	if result != "1.00 kHz" {
		t.Errorf("FormatValue(1000 Hz) = %s, want 1.00 kHz", result)
	}
}

func TestTimeParameter(t *testing.T) {
	param := TimeParameter(500, "Attack", 0.1, 5000, 10).Build()

	t.Run("Formatter", func(t *testing.T) {
		tests := []struct {
			plainValue float64
			expected   string
		}{
			{10, "10.0 ms"},
			{100, "100.0 ms"},
			{1000, "1.00 s"},
			{2500, "2.50 s"},
		}

		for _, test := range tests {
			normalized := param.Normalize(test.plainValue)
			result := param.FormatValue(normalized)
			if result != test.expected {
				t.Errorf("FormatValue(%f ms) = %s, want %s (min=%f, max=%f)", test.plainValue, result, test.expected, param.Min, param.Max)
			}
		}
	})

	t.Run("Parser", func(t *testing.T) {
		tests := []struct {
			input         string
			expectedPlain float64
		}{
			{"10 ms", 10},
			{"10ms", 10},
			{"1 s", 1000},
			{"1s", 1000},
			{"2.5 s", 2500},
		}

		for _, test := range tests {
			normalized, err := param.ParseValue(test.input)
			if err != nil {
				t.Errorf("ParseValue(%s) error: %v", test.input, err)
				continue
			}
			plain := param.Denormalize(normalized)
			if math.Abs(plain-test.expectedPlain) > 0.1 {
				t.Errorf("ParseValue(%s) = %f ms (plain), want %f ms", test.input, plain, test.expectedPlain)
			}
		}
	})
}

func TestRatioParameter(t *testing.T) {
	param := RatioParameter(600, "Ratio", 1, 100, 4).Build()

	t.Run("Formatter", func(t *testing.T) {
		tests := []struct {
			plainValue float64
			expected   string
		}{
			{1, "1.0:1"},
			{4, "4.0:1"},
			{100, "∞:1"},
		}

		for _, test := range tests {
			normalized := param.Normalize(test.plainValue)
			result := param.FormatValue(normalized)
			if result != test.expected {
				t.Errorf("FormatValue(%f) = %s, want %s", test.plainValue, result, test.expected)
			}
		}
	})

	t.Run("Parser", func(t *testing.T) {
		tests := []struct {
			input         string
			expectedPlain float64
		}{
			{"4:1", 4},
			{"4", 4},
			{"inf:1", 100},
			{"∞:1", 100},
		}

		for _, test := range tests {
			normalized, err := param.ParseValue(test.input)
			if err != nil {
				t.Errorf("ParseValue(%s) error: %v", test.input, err)
				continue
			}
			plain := param.Denormalize(normalized)
			if math.Abs(plain-test.expectedPlain) > 0.1 {
				t.Errorf("ParseValue(%s) = %f, want %f", test.input, plain, test.expectedPlain)
			}
		}
	})
}

func TestPanParameter(t *testing.T) {
	param := PanParameter(700, "Pan").Build()

	t.Run("Formatter", func(t *testing.T) {
		tests := []struct {
			plainValue float64
			expected   string
		}{
			{0, "Center"},
			{-50, "50% L"},
			{50, "50% R"},
			{-100, "100% L"},
			{100, "100% R"},
		}

		for _, test := range tests {
			normalized := param.Normalize(test.plainValue)
			result := param.FormatValue(normalized)
			if result != test.expected {
				t.Errorf("FormatValue(%f) = %s, want %s", test.plainValue, result, test.expected)
			}
		}
	})

	t.Run("Parser", func(t *testing.T) {
		tests := []struct {
			input         string
			expectedPlain float64
		}{
			{"center", 0},
			{"c", 0},
			{"50 l", -50},
			{"50% left", -50},
			{"50 r", 50},
			{"50% right", 50},
			{"75", 75},
		}

		for _, test := range tests {
			normalized, err := param.ParseValue(test.input)
			if err != nil {
				t.Errorf("ParseValue(%s) error: %v", test.input, err)
				continue
			}
			plain := param.Denormalize(normalized)
			if math.Abs(plain-test.expectedPlain) > 0.1 {
				t.Errorf("ParseValue(%s) = %f, want %f", test.input, plain, test.expectedPlain)
			}
		}
	})
}
