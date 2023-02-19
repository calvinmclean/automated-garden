package pkg

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Duration
		expectedErr string
	}{
		{
			"SuccessfulDecodeString",
			`{"d": "1m"}`,
			Duration{1 * time.Minute},
			"",
		},
		{
			"SuccessfulDecodeInt",
			`{"d": 60000000000}`,
			Duration{1 * time.Minute},
			"",
		},
		{
			"ErrorInvalidDurationString",
			`{"d": "60000000000"}`,
			Duration{},
			`invalid format for Duration: time: missing unit in duration "60000000000"`,
		},
		{
			"ErrorDecodingOtherType",
			`{"d": true}`,
			Duration{},
			"unexpected type bool, must be string or number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				D Duration `json:"d"`
			}
			err := json.Unmarshal([]byte(tt.input), &result)
			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result.D)
		})
	}
}

func TestDurationJSONMarshal(t *testing.T) {
	result, err := json.Marshal(Duration{1 * time.Minute})
	assert.NoError(t, err)
	assert.Equal(t, `"1m0s"`, string(result))
}

func TestDurationUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Duration
		expectedErr string
	}{
		{
			"SuccessfulDecodeString",
			`d: 1m`,
			Duration{1 * time.Minute},
			"",
		},
		{
			"SuccessfulDecodeInt",
			`d: 60000000000`,
			Duration{1 * time.Minute},
			"",
		},
		{
			"ErrorInvalidDurationString",
			`{"d": "60000000000"}`,
			Duration{},
			`invalid format for Duration: time: missing unit in duration "60000000000"`,
		},
		{
			"ErrorDecodingOtherType",
			`d: true`,
			Duration{},
			"unexpected type !!bool, must be string or number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				D Duration `json:"d"`
			}
			err := yaml.Unmarshal([]byte(tt.input), &result)
			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result.D)
		})
	}
}

func TestDurationYAMLMarshal(t *testing.T) {
	result, err := yaml.Marshal(Duration{1 * time.Minute})
	assert.NoError(t, err)
	assert.Equal(t, "duration: 1m0s\n", string(result))
}
