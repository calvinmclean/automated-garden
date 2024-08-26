package netatmo

import (
	"net/http"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func TestNewClientUsingDeviceName(t *testing.T) {
	r, err := recorder.New("testdata/fixtures/GetDeviceIDs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		require.NoError(t, r.Stop())
	}()

	if r.Mode() != recorder.ModeRecordOnce {
		t.Fatal("Recorder should be in ModeRecordOnce")
	}

	DefaultClient = r.GetDefaultClient()

	opts := map[string]any{
		"authentication": map[string]any{
			"access_token":    "ACCESS_TOKEN",
			"refresh_token":   "REFRESH_TOKEN",
			"expiration_date": clock.Now().Add(1 * time.Minute).Format(time.RFC3339Nano),
		},
		"client_id":           "CLIENT_ID",
		"client_secret":       "CLIENT_SECRET",
		"outdoor_module_name": "Outdoor Module",
		"rain_module_name":    "Smart Rain Gauge",
		"station_name":        "Weather Station",
	}
	client, err := NewClient(opts, func(m map[string]interface{}) error {
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Equal(t, "STATION_ID", client.Config.StationID)
	require.Equal(t, "OUTDOOR_MODULE_ID", client.Config.OutdoorModuleID)
	require.Equal(t, "RAIN_MODULE_ID", client.Config.RainModuleID)
}

func TestWeatherRequestMethods(t *testing.T) {
	// Modify request from garden-app to use placeholder for date_begin query param
	matcher := func(r1 *http.Request, r2 cassette.Request) bool {
		query := r1.URL.Query()
		if query.Get("date_begin") != "" {
			query.Set("date_begin", "DATE_BEGIN")
			r1.URL.RawQuery = query.Encode()
		}
		if query.Get("date_end") != "" {
			query.Set("date_end", "DATE_END")
			r1.URL.RawQuery = query.Encode()
		}

		return cassette.DefaultMatcher(r1, r2)
	}

	tests := []struct {
		name            string
		fixture         string
		tokenExpiration time.Time
		exec            func(t *testing.T, client *Client)
	}{
		{
			"GetTotalRain_NoRefresh",
			"testdata/fixtures/GetTotalRain_NoRefresh",
			clock.Now().Add(1 * time.Minute),
			func(t *testing.T, client *Client) {
				rain, err := client.GetTotalRain(72 * time.Hour)
				require.NoError(t, err)
				require.Equal(t, float32(0), rain)
			},
		},
		{
			"GetTotalRain_Refresh",
			"testdata/fixtures/GetTotalRain_Refresh",
			clock.Now().Add(-1 * time.Minute),
			func(t *testing.T, client *Client) {
				rain, err := client.GetTotalRain(72 * time.Hour)
				require.NoError(t, err)
				require.Equal(t, float32(0), rain)
				require.Equal(t, "NEW_REFRESH_TOKEN", client.Config.Authentication.RefreshToken)
			},
		},
		{
			"GetAverageHighTemperature_NoRefresh",
			"testdata/fixtures/GetAverageHighTemperature_NoRefresh",
			clock.Now().Add(1 * time.Minute),
			func(t *testing.T, client *Client) {
				temp, err := client.GetAverageHighTemperature(72 * time.Hour)
				require.NoError(t, err)
				require.Equal(t, float32(48.066666), temp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := map[string]any{
				"authentication": map[string]any{
					"access_token":    "ACCESS_TOKEN",
					"refresh_token":   "REFRESH_TOKEN",
					"expiration_date": tt.tokenExpiration.Format(time.RFC3339Nano),
				},
				"client_id":         "CLIENT_ID",
				"client_secret":     "CLIENT_SECRET",
				"outdoor_module_id": "OUTDOOR_MODULE_ID",
				"rain_module_id":    "RAIN_MODULE_ID",
				"station_id":        "STATION_ID",
			}
			client, err := NewClient(opts, func(newOpts map[string]interface{}) error { return nil })
			require.NoError(t, err)

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

			if r.Mode() != recorder.ModeRecordOnce {
				t.Fatal("Recorder should be in ModeRecordOnce")
			}

			client.Client = r.GetDefaultClient()

			tt.exec(t, client)
		})
	}
}
