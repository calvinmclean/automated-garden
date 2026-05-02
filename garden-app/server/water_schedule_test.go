package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var createdAt, _ = time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")

func float64Ptr(f float64) *float64 { return &f }

func createExampleWaterSchedule() *pkg.WaterSchedule {
	return &pkg.WaterSchedule{
		ID:        babyapi.ID{ID: id},
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: pkg.NewStartTime(createdAt),
		StartDate: &createdAt,
	}
}

func TestGetWaterSchedule(t *testing.T) {
	weatherClientID, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")

	tests := []struct {
		name               string
		excludeWeatherData bool
		waterSchedule      *pkg.WaterSchedule
		expectedRegexp     string
	}{
		{
			"Successful",
			false,
			createExampleWaterSchedule(),
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","start_time":"11:24:52\-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52\-07:00","duration":"1s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"SuccessfulWithRainAndTemperatureData",
			false,
			&pkg.WaterSchedule{
				ID:        babyapi.ID{ID: id},
				Duration:  &pkg.Duration{Duration: time.Hour},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: pkg.NewStartTime(createdAt),
				StartDate: &createdAt,
				WeatherControl: &weather.Control{
					Rain: &weather.WeatherScaler{
						ClientID:      weatherClientID,
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(0),
						InputMax:      float64Ptr(30),
						FactorMin:     float64Ptr(1.0),
						FactorMax:     float64Ptr(0.0),
					},
					Temperature: &weather.WeatherScaler{
						ClientID:      weatherClientID,
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						InputMax:      float64Ptr(40),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","start_time":"11:24:52-07:00","weather_control":{"rain_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":0,"input_max":30,"factor_min":1,"factor_max":0},"temperature_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":20,"input_max":40,"factor_min":0.5,"factor_max":1.5}},"weather_data":{"rain":{"mm":25.4,"inches":1},"temperature":{"celsius":80,"fahrenheit":176}},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"13m48\.[0-9]+s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"SuccessfulWithRainAndTemperatureDataButWeatherDataExcluded",
			true,
			&pkg.WaterSchedule{
				ID:        babyapi.ID{ID: id},
				Duration:  &pkg.Duration{Duration: time.Hour},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: pkg.NewStartTime(createdAt),
				StartDate: &createdAt,
				WeatherControl: &weather.Control{
					Rain: &weather.WeatherScaler{
						ClientID:      weatherClientID,
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(0),
						InputMax:      float64Ptr(25.4),
						FactorMin:     float64Ptr(1.0),
						FactorMax:     float64Ptr(1.0),
					},
					Temperature: &weather.WeatherScaler{
						ClientID:      weatherClientID,
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						InputMax:      float64Ptr(40),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","start_time":"11:24:52-07:00","weather_control":{"rain_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":0,"input_max":25.4,"factor_min":1,"factor_max":1},"temperature_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":20,"input_max":40,"factor_min":0.5,"factor_max":1.5}},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"1h0m0s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"ErrorRainWeatherClientDNE",
			false,
			&pkg.WaterSchedule{
				ID:        babyapi.ID{ID: id},
				Duration:  &pkg.Duration{Duration: time.Hour},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: pkg.NewStartTime(createdAt),
				StartDate: &createdAt,
				WeatherControl: &weather.Control{
					Rain: &weather.WeatherScaler{
						ClientID:      id2,
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(0),
						InputMax:      float64Ptr(25.4),
						FactorMin:     float64Ptr(1.0),
						FactorMax:     float64Ptr(0.0),
					},
				},
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","start_time":"11:24:52-07:00","weather_control":{"rain_control":{"client_id":"chkodpg3lcj13q82mq40","interpolation":"linear","input_min":0,"input_max":25.4,"factor_min":1,"factor_max":0}},"weather_data":{},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"1h0m0s","message":"error impacted duration scaling"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"ErrorTemperatureWeatherClientDNE",
			false,
			&pkg.WaterSchedule{
				ID:        babyapi.ID{ID: id},
				Duration:  &pkg.Duration{Duration: time.Hour},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: pkg.NewStartTime(createdAt),
				StartDate: &createdAt,
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						ClientID:      id2,
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						InputMax:      float64Ptr(40),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","start_time":"11:24:52-07:00","weather_control":{"temperature_control":{"client_id":"chkodpg3lcj13q82mq40","interpolation":"linear","input_min":20,"input_max":40,"factor_min":0\.5,"factor_max":1\.5}},"weather_data":{},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"[0-9a-zµs.]+","message":"error impacted duration scaling"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
	}

	t.Run("ErrNotFound", func(t *testing.T) {
		storageClient, err := storage.NewClient(storage.Config{
			ConnectionString: ":memory:",
		})
		assert.NoError(t, err)

		wsr := NewWaterSchedulesAPI()
		err = wsr.setup(storageClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
		require.NoError(t, err)
		wsr.worker.StartAsync()

		r := httptest.NewRequest(http.MethodGet, "/water_schedules/"+id2.String(), http.NoBody)
		r.Header.Add("Content-Type", "application/json")

		w := babytest.TestRequest[*pkg.WaterSchedule](t, wsr.API, r)

		// check HTTP response status code
		if w.Code != http.StatusNotFound {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, `{"status":"Resource not found."}`, strings.TrimSpace(w.Body.String()))

		wsr.worker.Stop()
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("Close")

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			err = storageClient.WaterSchedules.Set(context.Background(), tt.waterSchedule)
			assert.NoError(t, err)

			err = storageClient.WeatherClientConfigs.Set(context.Background(), createExampleWeatherClientConfig())
			assert.NoError(t, err)

			wsr := NewWaterSchedulesAPI()
			err = wsr.setup(storageClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			require.NoError(t, err)
			wsr.worker.StartAsync()

			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/water_schedules/%s?exclude_weather_data=%t", tt.waterSchedule.ID, tt.excludeWeatherData), http.NoBody)
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestRequest[*pkg.WaterSchedule](t, wsr.API, r)

			// check HTTP response status code
			if w.Code != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
			}

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}

			wsr.worker.Stop()
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestUpdateWaterSchedule(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			`{"duration":"1h"}`,
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","start_time":"11:24:52-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"1h0m0s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"BadRequestInvalidTemperatureControl",
			`{"weather_control":{"temperature_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":20,"input_max":40,"factor_min":-1,"factor_max":1.5}}}`,
			`{"status":"Invalid request.","error":"invalid WaterSchedule.WeatherControl after patching: error validating temperature_control: factors must be non-negative"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorRainWeatherClientDNE",
			`{"weather_control":{"rain_control":{"client_id":"chkodpg3lcj13q82mq40","interpolation":"linear","input_min":0,"input_max":25.4,"factor_min":0,"factor_max":0}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClients for WaterSchedule: error getting client for RainControl: error getting WeatherClient with ID \\"chkodpg3lcj13q82mq40\\": resource not found"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorTemperatureWeatherClientDNE",
			`{"weather_control":{"temperature_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"chkodpg3lcj13q82mq40"}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClients for WaterSchedule: error getting client for TemperatureControl: error getting WeatherClient with ID \\"chkodpg3lcj13q82mq40\\": resource not found"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			err = storageClient.WeatherClientConfigs.Set(context.Background(), createExampleWeatherClientConfig())
			assert.NoError(t, err)

			err = storageClient.WaterSchedules.Set(context.Background(), createExampleWaterSchedule())
			assert.NoError(t, err)

			wsr := NewWaterSchedulesAPI()
			err = wsr.setup(storageClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			require.NoError(t, err)

			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			r := httptest.NewRequest(http.MethodPatch, "/water_schedules/"+createExampleWaterSchedule().GetID(), strings.NewReader(tt.body))
			r.Header.Set("X-TZ-Offset", "420")
			r.Header.Set("Content-Type", "application/json")
			w := babytest.TestRequest[*pkg.WaterSchedule](t, wsr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestEndDateWaterSchedule(t *testing.T) {
	now := clock.Now()
	endDatedWaterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule.EndDate = &now
	endDatedWaterSchedule.ID = babyapi.ID{ID: id2}

	zone := createExampleZone()
	zone.WaterScheduleIDs = append(zone.WaterScheduleIDs, endDatedWaterSchedule.ID.ID)

	tests := []struct {
		name             string
		waterSchedule    *pkg.WaterSchedule
		zone             *pkg.Zone
		expectedResponse string
		code             int
	}{
		{
			"Successful",
			createExampleWaterSchedule(),
			nil,
			"",
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteWaterSchedule",
			endDatedWaterSchedule,
			nil,
			"",
			http.StatusOK,
		},
		{
			"UnableToDeleteUsedByZones",
			endDatedWaterSchedule,
			zone,
			`{"status":"Invalid request.","error":"unable to end-date WaterSchedule with 1 Zones"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			if tt.zone != nil {
				err = storageClient.Gardens.Set(context.Background(), createExampleGarden())
				assert.NoError(t, err)
				err = storageClient.Zones.Set(context.Background(), zone)
				assert.NoError(t, err)
			}

			err = storageClient.WaterSchedules.Set(context.Background(), tt.waterSchedule)
			assert.NoError(t, err)

			wsr := NewWaterSchedulesAPI()
			err = wsr.setup(storageClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			require.NoError(t, err)

			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			r := httptest.NewRequest(http.MethodDelete, "/water_schedules/"+tt.waterSchedule.GetID(), http.NoBody)
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestRequest[*pkg.WaterSchedule](t, wsr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Equal(t, tt.expectedResponse, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestGetAllWaterSchedules(t *testing.T) {
	waterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule.ID = babyapi.NewID()
	now := clock.Now()
	endDatedWaterSchedule.EndDate = &now

	tests := []struct {
		name        string
		targetURL   string
		expectedIDs []string
	}{
		{
			"SuccessfulEndDatedFalse",
			"/water_schedules",
			[]string{waterSchedule.GetID()},
		},
		{
			"SuccessfulEndDatedTrue",
			"/water_schedules?end_dated=true",
			[]string{waterSchedule.GetID(), endDatedWaterSchedule.GetID()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			err = storageClient.WaterSchedules.Set(context.Background(), waterSchedule)
			assert.NoError(t, err)
			err = storageClient.WaterSchedules.Set(context.Background(), endDatedWaterSchedule)
			assert.NoError(t, err)

			wsr := NewWaterSchedulesAPI()
			err = wsr.setup(storageClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			require.NoError(t, err)

			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			r := httptest.NewRequest(http.MethodGet, tt.targetURL, nil)
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestRequest[*pkg.WaterSchedule](t, wsr.API, r)

			var actual babyapi.ResourceList[*pkg.WaterSchedule]
			err = json.NewDecoder(w.Body).Decode(&actual)
			assert.NoError(t, err)

			actualIDs := []string{}
			for _, ws := range actual.Items {
				actualIDs = append(actualIDs, ws.GetID())
			}

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, len(tt.expectedIDs), len(actualIDs))
			assert.ElementsMatch(t, tt.expectedIDs, actualIDs)
		})
	}
}

func TestCreateWaterSchedule(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00"}`,
			`{"id":"[0-9a-v]{20}","duration":"1s","interval":"24h0m0s","start_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","start_time":"11:24:52-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"1s"},"links":\[{"rel":"self","href":"/water_schedules/[0-9a-v]{20}"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorDurationZero",
			`{"duration":"0s","interval":"24h0m0s","start_time":"11:24:52-07:00"}`,
			`{"status":"Invalid request.","error":"duration must not be 0"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorIntervalZero",
			`{"duration":"10s","interval":0,"start_time":"11:24:52-07:00"}`,
			`{"status":"Invalid request.","error":"interval must not be 0"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorRainWeatherClientDNE",
			`{"duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00", "weather_control":{"rain_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":0,"input_max":25.4,"factor_min":0,"factor_max":0}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClients for WaterSchedule: error getting client for RainControl: error getting WeatherClient with ID \\"c5cvhpcbcv45e8bp16dg\\": resource not found"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorTemperatureWeatherClientDNE",
			`{"duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00", "weather_control":{"temperature_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":0,"input_max":25.4,"factor_min":0,"factor_max":0}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClients for WaterSchedule: error getting client for TemperatureControl: error getting WeatherClient with ID \\"c5cvhpcbcv45e8bp16dg\\": resource not found"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidStartTime",
			`{"duration":"1s","interval":"24h0m0s","start_time":"invalid"}`,
			`{"status":"Invalid request.","error":"error parsing start time: parsing time \\"invalid\\" as \\"15:04:05Z07:00\\": cannot parse \\"invalid\\" as \\"15\\""}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestBadJSON",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorCannotSetID",
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00"}`,
			`{"status":"Invalid request.","error":"unable to manually set ID"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			wsr := NewWaterSchedulesAPI()
			err = wsr.setup(storageClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			require.NoError(t, err)

			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			r := httptest.NewRequest(http.MethodPost, "/water_schedules", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestRequest[*pkg.WaterSchedule](t, wsr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, regexp.MustCompile(tt.expectedRegexp), strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestUpdateWaterSchedulePUT(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00"}`,
			``,
			http.StatusOK,
		},
		{
			"ErrorMissingID",
			`{"duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00"}`,
			`{"status":"Invalid request.","error":"missing required id field"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWrongID",
			`{"id":"chkodpg3lcj13q82mq40","duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00"}`,
			`{"status":"Invalid request.","error":"id must match URL path"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorRainWeatherClientDNE",
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00", "weather_control":{"rain_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":0,"input_max":25.4,"factor_min":0,"factor_max":0}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClients for WaterSchedule: error getting client for RainControl: error getting WeatherClient with ID \\"c5cvhpcbcv45e8bp16dg\\": resource not found"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorTemperatureWeatherClientDNE",
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"11:24:52-07:00", "weather_control":{"temperature_control":{"client_id":"c5cvhpcbcv45e8bp16dg","interpolation":"linear","input_min":0,"input_max":25.4,"factor_min":0,"factor_max":0}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClients for WaterSchedule: error getting client for TemperatureControl: error getting WeatherClient with ID \\"c5cvhpcbcv45e8bp16dg\\": resource not found"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestBadJSON",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			ws := createExampleWaterSchedule()
			err = storageClient.WaterSchedules.Set(context.Background(), ws)
			assert.NoError(t, err)

			wsr := NewWaterSchedulesAPI()
			err = wsr.setup(storageClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			require.NoError(t, err)

			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			r := httptest.NewRequest(http.MethodPut, "/water_schedules/"+ws.GetID(), strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestRequest[*pkg.WaterSchedule](t, wsr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, regexp.MustCompile(tt.expectedRegexp), strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestWaterScheduleRequest(t *testing.T) {
	now := clock.Now()
	tests := []struct {
		name string
		pr   *pkg.WaterSchedule
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WaterSchedule fields",
		},
		{
			"EmptyIntervalError",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			"missing required interval field",
		},
		{
			"EmptyDurationError",
			&pkg.WaterSchedule{
				Interval: &pkg.Duration{Duration: time.Hour * 24},
			},
			"missing required duration field",
		},
		{
			"EmptyStartTimeError",
			&pkg.WaterSchedule{
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				Duration: &pkg.Duration{Duration: time.Second},
			},
			"missing required start_time field",
		},
		{
			"EmptyWeatherControlClientID",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						InputMax:      float64Ptr(40),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: client_id",
		},
		{
			"EmptyWeatherControlInputMin",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						ClientID:      xid.New(),
						Interpolation: weather.Linear,
						InputMax:      float64Ptr(40),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: input_min",
		},
		{
			"EmptyWeatherControlInputMax",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						ClientID:      xid.New(),
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: input_max",
		},
		{
			"EmptyWeatherControlFactorMin",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						ClientID:      xid.New(),
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						InputMax:      float64Ptr(40),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: factor_min",
		},
		{
			"EmptyWeatherControlFactorMax",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						ClientID:      xid.New(),
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						InputMax:      float64Ptr(40),
						FactorMin:     float64Ptr(0.5),
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: factor_max",
		},
		{
			"WeatherControlInvalidFactorMin",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						ClientID:      xid.New(),
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(20),
						InputMax:      float64Ptr(40),
						FactorMin:     float64Ptr(-1),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			"error validating weather_control: error validating temperature_control: factors must be non-negative",
		},
		{
			"WeatherControlInvalidInputRange",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Temperature: &weather.WeatherScaler{
						ClientID:      xid.New(),
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(40),
						InputMax:      float64Ptr(20),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.5),
					},
				},
			},
			"error validating weather_control: error validating temperature_control: input_max must be greater than input_min",
		},
		{
			"WeatherControlRainInvalidInputRange",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				WeatherControl: &weather.Control{
					Rain: &weather.WeatherScaler{
						ClientID:      xid.New(),
						Interpolation: weather.Linear,
						InputMin:      float64Ptr(50),
						InputMax:      float64Ptr(20),
						FactorMin:     float64Ptr(0.5),
						FactorMax:     float64Ptr(1.0),
					},
				},
			},
			"error validating weather_control: error validating rain_control: input_max must be greater than input_min",
		},
		{
			"ActivePeriodInvalid",
			&pkg.WaterSchedule{
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Second},
				StartTime: pkg.NewStartTime(now),
				ActivePeriod: &pkg.ActivePeriod{
					StartMonth: "not a month",
				},
			},
			"error validating active_period: invalid StartMonth: parsing time \"not a month\" as \"January\": cannot parse \"not a month\" as \"January\"",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &pkg.WaterSchedule{
			Duration:  &pkg.Duration{Duration: time.Second},
			Interval:  &pkg.Duration{Duration: time.Hour * 24},
			StartTime: pkg.NewStartTime(now),
			WeatherControl: &weather.Control{
				Temperature: &weather.WeatherScaler{
					ClientID:      xid.New(),
					Interpolation: weather.Linear,
					InputMin:      float64Ptr(20),
					InputMax:      float64Ptr(40),
					FactorMin:     float64Ptr(0.5),
					FactorMax:     float64Ptr(1.5),
				},
			},
		}
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		err := pr.Bind(r)
		assert.NoError(t, err)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			err := tt.pr.Bind(r)
			assert.Error(t, err)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestUpdateWaterScheduleRequest(t *testing.T) {
	now := clock.Now()
	tests := []struct {
		name string
		pr   *pkg.WaterSchedule
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WaterSchedule fields",
		},
		{
			"ManualSpecificationOfIDError",
			&pkg.WaterSchedule{
				ID: babyapi.ID{ID: id},
			},
			"updating ID is not allowed",
		},
		{
			"EndDateError",
			&pkg.WaterSchedule{
				EndDate: &now,
			},
			"to end-date a WaterSchedule, please use the DELETE endpoint",
		},
		{
			"InvalidActivePeriod",
			&pkg.WaterSchedule{
				ActivePeriod: &pkg.ActivePeriod{
					StartMonth: "not a month",
				},
			},
			"error validating active_period: invalid StartMonth: parsing time \"not a month\" as \"January\": cannot parse \"not a month\" as \"January\"",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		wsr := &pkg.WaterSchedule{
			Interval: &pkg.Duration{Duration: time.Hour},
		}
		r := httptest.NewRequest(http.MethodPatch, "/", nil)
		err := wsr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading WaterScheduleRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPatch, "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading WaterScheduleRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
