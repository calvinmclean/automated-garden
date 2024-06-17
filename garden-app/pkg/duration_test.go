package pkg

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/ajg/form"
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
			Duration{1 * time.Minute, ""},
			"",
		},
		{
			"SuccessfulDecodeInt",
			`{"d": 60000000000}`,
			Duration{1 * time.Minute, ""},
			"",
		},
		{
			"SuccessfulDecodeCron",
			`{"d": "cron:*/5 * * * 1"}`,
			Duration{0, "*/5 * * * 1"},
			"",
		},
		{
			"ErrorInvalidDurationString",
			`{"d": "60000000000"}`,
			Duration{},
			`invalid json input for Duration: invalid format for time.Duration: time: missing unit in duration "60000000000"`,
		},
		{
			"ErrorInvalidCronString",
			`{"d": "cron:abc"}`,
			Duration{},
			`invalid json input for Duration: invalid cron expression: expected exactly 5 fields, found 1: [abc]`,
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
	t.Run("time.Duration", func(t *testing.T) {
		result, err := json.Marshal(&Duration{1 * time.Minute, ""})
		assert.NoError(t, err)
		assert.Equal(t, `"1m0s"`, string(result))
	})
	t.Run("cron", func(t *testing.T) {
		result, err := json.Marshal(&Duration{0, "*/5 * * * 1"})
		assert.NoError(t, err)
		assert.Equal(t, `"cron:*/5 * * * 1"`, string(result))
	})
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
			Duration{1 * time.Minute, ""},
			"",
		},
		{
			"SuccessfulDecodeInt",
			`d: 60000000000`,
			Duration{1 * time.Minute, ""},
			"",
		},
		{
			"SuccessfulDecodeCron",
			`d: cron:*/5 * * * 1`,
			Duration{0, "*/5 * * * 1"},
			"",
		},
		{
			"ErrorInvalidDurationString",
			`{"d": "60000000000"}`,
			Duration{},
			`invalid yaml input for Duration: invalid format for time.Duration: time: missing unit in duration "60000000000"`,
		},
		{
			"ErrorInvalidCronString",
			`{"d": "cron:abc"}`,
			Duration{},
			"invalid yaml input for Duration: invalid cron expression: expected exactly 5 fields, found 1: [abc]",
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
	t.Run("time.Duration", func(t *testing.T) {
		result, err := yaml.Marshal(&Duration{1 * time.Minute, ""})
		assert.NoError(t, err)
		assert.Equal(t, "1m0s\n", string(result))
	})
	t.Run("cron", func(t *testing.T) {
		result, err := yaml.Marshal(&Duration{0, "*/5 * * * 1"})
		assert.NoError(t, err)
		assert.Equal(t, "cron:*/5 * * * 1\n", string(result))
	})
}

func TestDurationUnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    url.Values
		expected Duration
	}{
		{
			"DurationString",
			url.Values{
				"Duration": []string{"1m0s"},
			},
			Duration{Duration: 1 * time.Minute},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Duration Duration
			}
			err := form.DecodeString(&result, tt.input.Encode())
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.Duration)

			var formResult struct {
				Duration Duration
			}
			err = form.DecodeValues(&formResult, tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, formResult.Duration)
		})
	}
}
