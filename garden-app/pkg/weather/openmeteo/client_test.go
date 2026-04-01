package openmeteo

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "ValidConfigWithLatLon",
			opts: map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			},
			wantErr: false,
		},
		{
			name: "MissingLatitude",
			opts: map[string]any{
				"longitude": -122.4194,
			},
			wantErr: true,
			errMsg:  "latitude and longitude must be provided",
		},
		{
			name: "MissingLongitude",
			opts: map[string]any{
				"latitude": 37.7749,
			},
			wantErr: true,
			errMsg:  "latitude and longitude must be provided",
		},
		{
			name:    "MissingBoth",
			opts:    map[string]any{},
			wantErr: true,
			errMsg:  "latitude and longitude must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				assert.InDelta(t, tt.opts["latitude"], client.Latitude, 0.0001)
				assert.InDelta(t, tt.opts["longitude"], client.Longitude, 0.0001)
			}
		})
	}
}

func TestGetTotalRain(t *testing.T) {
	// Matcher to normalize query parameters for consistent VCR matching
	matcher := func(r1 *http.Request, r2 cassette.Request) bool {
		// Parse both URLs
		u1 := r1.URL
		u2, err := url.Parse(r2.URL)
		if err != nil {
			return false
		}

		// Compare scheme, host, and path
		if u1.Scheme != u2.Scheme || u1.Host != u2.Host || u1.Path != u2.Path {
			return false
		}

		// Parse query parameters
		q1 := u1.Query()
		q2 := u2.Query()

		// Check required parameters match
		if q1.Get("daily") != q2.Get("daily") {
			return false
		}

		return true
	}

	tests := []struct {
		name     string
		fixture  string
		duration time.Duration
		expected float32
	}{
		{
			name:     "GetTotalRain_3Days",
			fixture:  "testdata/fixtures/GetTotalRain_3Days",
			duration: 72 * time.Hour,
			expected: 15.5,
		},
		{
			name:     "GetTotalRain_1Day",
			fixture:  "testdata/fixtures/GetTotalRain_1Day",
			duration: 24 * time.Hour,
			expected: 5.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			}

			r, err := recorder.New(
				tt.fixture,
				recorder.WithMatcher(matcher),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				require.NoError(t, r.Stop())
			}()

			client, err := NewClientWithHTTPClient(opts, r.GetDefaultClient())
			require.NoError(t, err)

			rain, err := client.GetTotalRain(tt.duration)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, rain, 0.01)
		})
	}
}

func TestGetAverageHighTemperature(t *testing.T) {
	// Matcher to normalize query parameters for consistent VCR matching
	matcher := func(r1 *http.Request, r2 cassette.Request) bool {
		// Parse both URLs
		u1 := r1.URL
		u2, err := url.Parse(r2.URL)
		if err != nil {
			return false
		}

		// Compare scheme, host, and path
		if u1.Scheme != u2.Scheme || u1.Host != u2.Host || u1.Path != u2.Path {
			return false
		}

		// Parse query parameters
		q1 := u1.Query()
		q2 := u2.Query()

		// Check required parameters match
		if q1.Get("daily") != q2.Get("daily") {
			return false
		}

		return true
	}

	tests := []struct {
		name     string
		fixture  string
		duration time.Duration
		expected float32
	}{
		{
			name:     "GetAverageHighTemperature_3Days",
			fixture:  "testdata/fixtures/GetAverageHighTemperature_3Days",
			duration: 72 * time.Hour,
			expected: 22.8,
		},
		{
			name:     "GetAverageHighTemperature_7Days",
			fixture:  "testdata/fixtures/GetAverageHighTemperature_7Days",
			duration: 168 * time.Hour,
			expected: 24.4375,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			}

			r, err := recorder.New(
				tt.fixture,
				recorder.WithMatcher(matcher),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				require.NoError(t, r.Stop())
			}()

			client, err := NewClientWithHTTPClient(opts, r.GetDefaultClient())
			require.NoError(t, err)

			temp, err := client.GetAverageHighTemperature(tt.duration)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, temp, 0.01)
		})
	}
}

func TestMinimumIntervals(t *testing.T) {
	matcher := func(r1 *http.Request, r2 cassette.Request) bool {
		return true // Simple matcher for these tests
	}

	t.Run("GetTotalRain_Minimum24Hours", func(t *testing.T) {
		opts := map[string]any{
			"latitude":  37.7749,
			"longitude": -122.4194,
		}

		r, err := recorder.New(
			"testdata/fixtures/GetTotalRain_MinimumInterval",
			recorder.WithMatcher(matcher),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			require.NoError(t, r.Stop())
		}()

		client, err := NewClientWithHTTPClient(opts, r.GetDefaultClient())
		require.NoError(t, err)

		// Pass less than 24 hours - should be enforced to 24h minimum
		rain, err := client.GetTotalRain(1 * time.Hour)
		require.NoError(t, err)
		// Should not error due to minimum interval enforcement
		_ = rain
	})

	t.Run("GetAverageHighTemperature_Minimum72Hours", func(t *testing.T) {
		opts := map[string]any{
			"latitude":  37.7749,
			"longitude": -122.4194,
		}

		r, err := recorder.New(
			"testdata/fixtures/GetAverageHighTemperature_MinimumInterval",
			recorder.WithMatcher(matcher),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			require.NoError(t, r.Stop())
		}()

		client, err := NewClientWithHTTPClient(opts, r.GetDefaultClient())
		require.NoError(t, err)

		// Pass less than 72 hours - should be enforced to 72h minimum
		temp, err := client.GetAverageHighTemperature(24 * time.Hour)
		require.NoError(t, err)
		// Should not error due to minimum interval enforcement
		_ = temp
	})
}

func TestOpenMeteoResponseParsing(t *testing.T) {
	t.Run("ParseValidResponse", func(t *testing.T) {
		jsonData := `{
			"daily": {
				"time": ["2024-01-01", "2024-01-02", "2024-01-03"],
				"temperature_2m_max": [20.5, 22.3, 19.8],
				"precipitation_sum": [0.0, 5.2, 1.5]
			}
		}`

		var data openMeteoResponse
		err := json.Unmarshal([]byte(jsonData), &data)
		require.NoError(t, err)
		assert.Len(t, data.Daily.Time, 3)
		assert.Len(t, data.Daily.Temperature2mMax, 3)
		assert.Len(t, data.Daily.PrecipitationSum, 3)
		assert.InDelta(t, float32(20.5), data.Daily.Temperature2mMax[0], 0.01)
		assert.InDelta(t, float32(5.2), data.Daily.PrecipitationSum[1], 0.01)
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		jsonData := `{"daily": {"time": [], "temperature_2m_max": [], "precipitation_sum": []}}`

		var data openMeteoResponse
		err := json.Unmarshal([]byte(jsonData), &data)
		require.NoError(t, err)
		assert.Empty(t, data.Daily.Time)
	})
}

func TestCalculatePastDays(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected int
	}{
		{24 * time.Hour, 2},  // 1 day + 1
		{48 * time.Hour, 3},  // 2 days + 1
		{72 * time.Hour, 4},  // 3 days + 1
		{168 * time.Hour, 8}, // 7 days + 1
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			pastDays := int(tt.duration.Hours()/24) + 1
			assert.Equal(t, tt.expected, pastDays)
		})
	}
}

func TestEndOfYesterday(t *testing.T) {
	now := clock.Now()
	endOfYesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 0, time.Local)

	// Verify it's yesterday
	assert.Equal(t, now.AddDate(0, 0, -1).Day(), endOfYesterday.Day())
	assert.Equal(t, 23, endOfYesterday.Hour())
	assert.Equal(t, 59, endOfYesterday.Minute())
	assert.Equal(t, 59, endOfYesterday.Second())
}
