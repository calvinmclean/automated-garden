package pkg

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/ajg/form"
	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/stretchr/testify/assert"
)

func TestTimeLocationFromOffset(t *testing.T) {
	tests := []struct {
		name        string
		offset      string
		expectedLoc string
	}{
		{
			"MST",
			"420",
			"MST",
		},
		{
			"UTC",
			"0",
			"UTC",
		},
		{
			"GMT",
			"0",
			"GMT",
		},
	}

	now := clock.Now()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedLoc, _ := time.LoadLocation(tt.expectedLoc)

			loc, err := TimeLocationFromOffset(tt.offset)
			assert.NoError(t, err)

			assert.Equal(t, now.In(expectedLoc).UnixNano(), now.In(loc).UnixNano())
		})
	}

	t.Run("InvalidInput", func(t *testing.T) {
		loc, err := TimeLocationFromOffset("f")
		assert.Error(t, err)
		assert.Nil(t, loc)
	})
}

func TestStartTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *StartTime
		expectedErr string
	}{
		{
			"SuccessfulDecodeString",
			`{"start_time": "15:04:05-07:00"}`,
			NewStartTime(time.Date(0, 1, 1, 15, 4, 5, 0, time.FixedZone("", -7*3600))),
			"",
		},
		{
			"SuccessfulDecodeStringUTC",
			`{"start_time": "15:04:05Z"}`,
			NewStartTime(time.Date(0, 1, 1, 15, 4, 5, 0, time.UTC)),
			"",
		},
		{
			"SuccessfulDecodeStringZeroOffset",
			`{"start_time": "15:04:05-00:00"}`,
			NewStartTime(time.Date(0, 1, 1, 15, 4, 5, 0, time.FixedZone("", 0))),
			"",
		},
		{
			"SuccessfulDecodeSplit",
			`{"start_time": {"hour": 15, "minute": 4, "TZ": "-07:00"}}`,
			NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.FixedZone("", -7*3600))),
			"",
		},
		{
			"SuccessfulDecodeSplitUTC",
			`{"start_time": {"hour": 15, "minute": 4, "TZ": "Z"}}`,
			NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.UTC)),
			"",
		},
		{
			"SuccessfulDecodeSplitZeroOffset",
			`{"start_time": {"hour": 15, "minute": 4, "TZ": "+00:00"}}`,
			NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.FixedZone("", 0))),
			"",
		},
		{
			"ErrorDecodingOtherType",
			`{"start_time": true}`,
			&StartTime{},
			"unexpected type bool, must be string or object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				StartTime StartTime `json:"start_time"`
			}
			err := json.Unmarshal([]byte(tt.input), &result)
			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected.Time.Unix(), result.StartTime.Time.Unix())
		})
	}
}

func TestStartTimeJSONMarshal(t *testing.T) {
	st := NewStartTime(time.Date(0, 1, 1, 15, 4, 5, 0, time.FixedZone("", -7*3600)))
	result, err := json.Marshal(&st)
	assert.NoError(t, err)
	assert.Equal(t, `"15:04:05-07:00"`, string(result))
}

func TestStartTimeUnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    url.Values
		expected *StartTime
	}{
		{
			"-07:00",
			url.Values{
				"StartTime.Hour":   []string{"15"},
				"StartTime.Minute": []string{"4"},
				"StartTime.TZ":     []string{"-07:00"},
			},
			NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.FixedZone("", -7*3600))),
		},
		{
			"UTC",
			url.Values{
				"StartTime.Hour":   []string{"15"},
				"StartTime.Minute": []string{"4"},
				"StartTime.TZ":     []string{"Z"},
			},
			NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.UTC)),
		},
		{
			"NoOffset",
			url.Values{
				"StartTime.Hour":   []string{"15"},
				"StartTime.Minute": []string{"4"},
				"StartTime.TZ":     []string{"+00:00"},
			},
			NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.FixedZone("", 0))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				StartTime *StartTime
			}
			err := form.DecodeString(&result, tt.input.Encode())
			assert.NoError(t, err)
			assert.NoError(t, result.StartTime.Validate())
			assert.Equal(t, tt.expected.Time.Unix(), result.StartTime.Time.Unix())
		})
	}
}
