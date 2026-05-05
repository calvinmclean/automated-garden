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
			"SuccessfulDecodeDays",
			`{"d": "4d"}`,
			Duration{4 * 24 * time.Hour, ""},
			"",
		},
		{
			"SuccessfulDecodeOneDay",
			`{"d": "1d"}`,
			Duration{24 * time.Hour, ""},
			"",
		},
		{
			"SuccessfulDecodeDaysAndHours",
			`{"d": "5d1h"}`,
			Duration{5*24*time.Hour + 1*time.Hour, ""},
			"",
		},
		{
			"SuccessfulDecodeDaysAndMinutes",
			`{"d": "2d30m"}`,
			Duration{2*24*time.Hour + 30*time.Minute, ""},
			"",
		},
		{
			"SuccessfulDecodeDaysHoursMinutes",
			`{"d": "3d2h15m"}`,
			Duration{3*24*time.Hour + 2*time.Hour + 15*time.Minute, ""},
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
		assert.Equal(t, `"1m"`, string(result))
	})
	t.Run("cron", func(t *testing.T) {
		result, err := json.Marshal(&Duration{0, "*/5 * * * 1"})
		assert.NoError(t, err)
		assert.Equal(t, `"cron:*/5 * * * 1"`, string(result))
	})
	t.Run("days", func(t *testing.T) {
		result, err := json.Marshal(&Duration{4 * 24 * time.Hour, ""})
		assert.NoError(t, err)
		assert.Equal(t, `"4d"`, string(result))
	})
	t.Run("less than one day", func(t *testing.T) {
		result, err := json.Marshal(&Duration{12 * time.Hour, ""})
		assert.NoError(t, err)
		assert.Equal(t, `"12h"`, string(result))
	})
	t.Run("days with hours remainder", func(t *testing.T) {
		result, err := json.Marshal(&Duration{50 * time.Hour, ""})
		assert.NoError(t, err)
		assert.Equal(t, `"2d2h"`, string(result))
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
			"SuccessfulDecodeDays",
			`d: 7d`,
			Duration{7 * 24 * time.Hour, ""},
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
		assert.Equal(t, "1m\n", string(result))
	})
	t.Run("cron", func(t *testing.T) {
		result, err := yaml.Marshal(&Duration{0, "*/5 * * * 1"})
		assert.NoError(t, err)
		assert.Equal(t, "cron:*/5 * * * 1\n", string(result))
	})
	t.Run("days", func(t *testing.T) {
		result, err := yaml.Marshal(&Duration{3 * 24 * time.Hour, ""})
		assert.NoError(t, err)
		assert.Equal(t, "3d\n", string(result))
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
		{
			"DaysString",
			url.Values{
				"Duration": []string{"5d"},
			},
			Duration{Duration: 5 * 24 * time.Hour},
		},
		{
			"OneDay",
			url.Values{
				"Duration": []string{"1d"},
			},
			Duration{Duration: 24 * time.Hour},
		},
		{
			"DaysAndHours",
			url.Values{
				"Duration": []string{"5d1h"},
			},
			Duration{Duration: 5*24*time.Hour + 1*time.Hour},
		},
		{
			"DaysHoursMinutes",
			url.Values{
				"Duration": []string{"2d3h45m"},
			},
			Duration{Duration: 2*24*time.Hour + 3*time.Hour + 45*time.Minute},
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

func TestDurationRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple duration", "1h30m", "1h30m"},
		{"days only", "4d", "4d"},
		{"one day", "1d", "1d"},
		{"days and hours", "3d1h", "3d1h"},
		{"days hours minutes", "2d3h45m", "2d3h45m"},
		{"minutes only", "30m", "30m"},
		{"cron", "cron:*/5 * * * *", "cron:*/5 * * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input string
			var d Duration
			err := d.UnmarshalJSON([]byte(`"` + tt.input + `"`))
			assert.NoError(t, err)

			// Encode back to string
			result := d.String()
			assert.Equal(t, tt.expected, result)

			// Verify it can be parsed again
			var d2 Duration
			err = d2.UnmarshalJSON([]byte(`"` + result + `"`))
			assert.NoError(t, err)
			assert.Equal(t, d, d2)
		})
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds only", 30 * time.Second, "30s"},
		{"minutes only", 45 * time.Minute, "45m"},
		{"minutes and seconds", 5*time.Minute + 30*time.Second, "5m30s"},
		{"hours only", 12 * time.Hour, "12h"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"one day", 24 * time.Hour, "1d"},
		{"days only", 5 * 24 * time.Hour, "5d"},
		{"days and hours", 3*24*time.Hour + 6*time.Hour, "3d6h"},
		{"days hours and minutes", 2*24*time.Hour + 5*time.Hour + 30*time.Minute, "2d5h30m"},
		{"complex", 50*time.Hour + 30*time.Minute, "2d2h30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDurationShort(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
