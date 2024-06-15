package pkg

import (
	"encoding/json"
	"testing"
)

func TestLightStateString(t *testing.T) {
	tests := []struct {
		name     string
		input    LightState
		expected string
	}{
		{
			"ON",
			LightStateOn,
			"ON",
		},
		{
			"OFF",
			LightStateOff,
			"OFF",
		},
		{
			"Toggle",
			LightStateToggle,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.String() != tt.expected {
				t.Errorf("Expected %v, but got: %v", tt.expected, tt.input)
			}
		})
	}
}

func TestLightStateUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LightState
	}{
		{
			"ON",
			`"ON"`,
			LightStateOn,
		},
		{
			"on",
			`"on"`,
			LightStateOn,
		},
		{
			"OFF",
			`"OFF"`,
			LightStateOff,
		},
		{
			"off",
			`"off"`,
			LightStateOff,
		},
		{
			"Toggle",
			`""`,
			LightStateToggle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l LightState
			err := json.Unmarshal([]byte(tt.input), &l)
			if err != nil {
				t.Errorf("Unexpected error when unmarshaling JSON: %v", err)
			}
			if l != tt.expected {
				t.Errorf("Expected %v, but got: %v", tt.expected, l.String())
			}
		})
	}

	t.Run("InvalidInput", func(t *testing.T) {
		var l LightState
		err := json.Unmarshal([]byte(`"invalid"`), &l)
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != `cannot unmarshal "invalid" into Go value of type *pkg.LightState` {
			t.Errorf("Unexpected error string: %v", err)
		}
	})
}

func TestLightStateMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    LightState
		expected string
	}{
		{
			"ON",
			LightStateOn,
			`"ON"`,
		},
		{
			"OFF",
			LightStateOff,
			`"OFF"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			if err != nil {
				t.Errorf("Unexpected error when marshaling JSON: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Expected %v, but got: %s", tt.expected, string(result))
			}
		})
	}

	t.Run("InvalidLightState", func(t *testing.T) {
		result, err := json.Marshal(LightState(3))
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != `json: error calling MarshalJSON for type pkg.LightState: cannot convert 3 to pkg.LightState` {
			t.Errorf("Unexpected error string: %v", err)
		}
		if string(result) != "" {
			t.Errorf("Expected empty string but got: %v", string(result))
		}
	})
}
