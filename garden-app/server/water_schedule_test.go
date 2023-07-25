package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var createdAt, _ = time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")

func createExampleWaterSchedule() *pkg.WaterSchedule {
	return &pkg.WaterSchedule{
		ID:        id,
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: &createdAt,
	}
}

func setupWaterScheduleStorage(t *testing.T) *storage.Client {
	t.Helper()

	ws := createExampleWaterSchedule()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.SaveWaterSchedule(ws)
	assert.NoError(t, err)

	return storageClient
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
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386\-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386\-07:00","duration":"1s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"SuccessfulWithRainAndTemperatureData",
			false,
			&pkg.WaterSchedule{
				ID:        id,
				Duration:  &pkg.Duration{Duration: time.Hour},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: &createdAt,
				WeatherControl: &weather.Control{
					Rain: &weather.ScaleControl{
						BaselineValue: float32Pointer(0),
						Factor:        float32Pointer(0),
						Range:         float32Pointer(25.4),
						ClientID:      weatherClientID,
					},
					Temperature: &weather.ScaleControl{
						BaselineValue: float32Pointer(30),
						Factor:        float32Pointer(0.5),
						Range:         float32Pointer(10),
						ClientID:      weatherClientID,
					},
				},
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","weather_control":{"rain_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"c5cvhpcbcv45e8bp16dg"},"temperature_control":{"baseline_value":30,"factor":0.5,"range":10,"client_id":"c5cvhpcbcv45e8bp16dg"}},"weather_data":{"rain":{"mm":25.4,"scale_factor":0},"average_temperature":{"celsius":80,"scale_factor":1.5}},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"0s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"SuccessfulWithRainAndTemperatureDataButWeatherDataExcluded",
			true,
			&pkg.WaterSchedule{
				ID:        id,
				Duration:  &pkg.Duration{Duration: time.Hour},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: &createdAt,
				WeatherControl: &weather.Control{
					Rain: &weather.ScaleControl{
						BaselineValue: float32Pointer(0),
						Factor:        float32Pointer(0),
						Range:         float32Pointer(25.4),
						ClientID:      weatherClientID,
					},
					Temperature: &weather.ScaleControl{
						BaselineValue: float32Pointer(30),
						Factor:        float32Pointer(0.5),
						Range:         float32Pointer(10),
						ClientID:      weatherClientID,
					},
				},
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","weather_control":{"rain_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"c5cvhpcbcv45e8bp16dg"},"temperature_control":{"baseline_value":30,"factor":0.5,"range":10,"client_id":"c5cvhpcbcv45e8bp16dg"}},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1h0m0s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("Close")

			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			err = storageClient.SaveWaterSchedule(tt.waterSchedule)
			assert.NoError(t, err)

			err = storageClient.SaveWeatherClientConfig(createExampleWeatherClientConfig())
			assert.NoError(t, err)

			wsr, err := NewWaterSchedulesResource(storageClient, worker.NewWorker(storageClient, influxdbClient, nil, logrus.New()))
			assert.NoError(t, err)
			wsr.worker.StartAsync()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/water_schedules/{%s}", waterSchedulePathParam), func(r chi.Router) {
				r.Use(wsr.waterScheduleContextMiddleware)
				r.Get("/", wsr.getWaterSchedule)
			})

			r := httptest.NewRequest("GET", fmt.Sprintf("/water_schedules/%s?exclude_weather_data=%t", tt.waterSchedule.ID, tt.excludeWeatherData), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

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

func TestWaterScheduleContextMiddleware(t *testing.T) {
	waterSchedule := createExampleWaterSchedule()

	tests := []struct {
		name          string
		waterSchedule *pkg.WaterSchedule
		path          string
		code          int
		expected      string
	}{
		{
			"Successful",
			waterSchedule,
			"/water_schedules/c5cvhpcbcv45e8bp16dg",
			http.StatusOK,
			"",
		},
		{
			"ErrorInvalidID",
			waterSchedule,
			"/water_schedules/not-an-xid",
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
		},
		{
			"NotFoundError",
			nil,
			"/water_schedules/chkodpg3lcj13q82mq40",
			http.StatusNotFound,
			`{"status":"Resource not found."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil).WithContext(context.Background())
			w := httptest.NewRecorder()

			storageClient := setupWaterScheduleStorage(t)
			wsr := WaterSchedulesResource{
				storageClient: storageClient,
			}

			testHandler := func(w http.ResponseWriter, r *http.Request) {
				ws := getWaterScheduleFromContext(r.Context())
				assert.Equal(t, waterSchedule, ws)
				render.Status(r, http.StatusOK)
			}

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/water_schedules/{%s}", waterSchedulePathParam), func(r chi.Router) {
				r.Use(wsr.waterScheduleContextMiddleware)
				r.Get("/", testHandler)
			})
			router.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
		})
	}
}

func TestWaterScheduleRestrictEndDatedMiddleware(t *testing.T) {
	waterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule := createExampleWaterSchedule()
	endDate := time.Now().Add(-1 * time.Minute)
	endDatedWaterSchedule.EndDate = &endDate
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
	}

	router := chi.NewRouter()
	router.Route("/water_schedules", func(r chi.Router) {
		r.Use(restrictEndDatedMiddleware("WaterSchedule", waterScheduleCtxKey))
		r.Get("/", testHandler)
	})

	tests := []struct {
		name          string
		waterSchedule *pkg.WaterSchedule
		code          int
		expected      string
	}{
		{
			"WaterScheduleNotEndDated",
			waterSchedule,
			http.StatusOK,
			"",
		},
		{
			"WaterScheduleEndDated",
			endDatedWaterSchedule,
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"resource not available for end-dated WaterSchedule"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), waterScheduleCtxKey, tt.waterSchedule)
			r := httptest.NewRequest("GET", "/water_schedules", nil).WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
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
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1h0m0s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
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
			`{"weather_control":{"temperature_control":{"baseline_value":27,"factor":-1,"range":10,"client_id":"c5cvhpcbcv45e8bp16dg"}}}`,
			`{"status":"Invalid request.","error":"error validating temperature_control: factor must be between 0 and 1"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorRainWeatherClientDNE",
			`{"weather_control":{"rain_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"chkodpg3lcj13q82mq40"}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClient for WaterSchedule"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorTemperatureWeatherClientDNE",
			`{"weather_control":{"temperature_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"chkodpg3lcj13q82mq40"}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClient for WaterSchedule"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			err = storageClient.SaveWeatherClientConfig(createExampleWeatherClientConfig())
			assert.NoError(t, err)

			wsr := WaterSchedulesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}
			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			waterSchedule := createExampleWaterSchedule()
			waterScheduleCtx := context.WithValue(context.Background(), waterScheduleCtxKey, waterSchedule)
			r := httptest.NewRequest("PATCH", "/water_schedules", strings.NewReader(tt.body)).WithContext(waterScheduleCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wsr.updateWaterSchedule)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.status {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.status)
			}

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
		})
	}
}

func TestEndDateWaterSchedule(t *testing.T) {
	now := time.Now()
	endDatedWaterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule.EndDate = &now
	endDatedWaterSchedule.ID = id2

	zone := createExampleZone()
	zone.WaterScheduleIDs = append(zone.WaterScheduleIDs, endDatedWaterSchedule.ID)

	tests := []struct {
		name           string
		waterSchedule  *pkg.WaterSchedule
		zone           *pkg.Zone
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			createExampleWaterSchedule(),
			nil,
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","next_water":{},"links":\[{"rel":"self","href":"/water_schedules/[0-9a-v]{20}"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteWaterSchedule",
			endDatedWaterSchedule,
			nil,
			"",
			http.StatusNoContent,
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
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			if tt.zone != nil {
				err = storageClient.SaveGarden(createExampleGarden())
				assert.NoError(t, err)
				err = storageClient.SaveZone(id, zone)
				assert.NoError(t, err)
			}

			wsr := WaterSchedulesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}
			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			waterScheduleCtx := context.WithValue(context.Background(), waterScheduleCtxKey, tt.waterSchedule)
			r := httptest.NewRequest("DELETE", "/water_schedules", nil).WithContext(waterScheduleCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wsr.endDateWaterSchedule)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
		})
	}
}

func TestGetAllWaterSchedules(t *testing.T) {
	waterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule.ID = xid.New()
	now := time.Now()
	endDatedWaterSchedule.EndDate = &now

	tests := []struct {
		name      string
		targetURL string
		expected  []*pkg.WaterSchedule
	}{
		{
			"SuccessfulEndDatedFalse",
			"/water_schedules",
			[]*pkg.WaterSchedule{waterSchedule},
		},
		{
			"SuccessfulEndDatedTrue",
			"/water_schedules?end_dated=true",
			[]*pkg.WaterSchedule{waterSchedule, endDatedWaterSchedule},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)
			err = storageClient.SaveWaterSchedule(waterSchedule)
			assert.NoError(t, err)
			err = storageClient.SaveWaterSchedule(endDatedWaterSchedule)
			assert.NoError(t, err)

			wsr := WaterSchedulesResource{
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
				storageClient: storageClient,
			}

			r := httptest.NewRequest("GET", tt.targetURL, nil).WithContext(context.Background())
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wsr.getAllWaterSchedules)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
			}

			// check HTTP response body
			var actual AllWaterSchedulesResponse
			err = json.NewDecoder(w.Body).Decode(&actual)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(actual.WaterSchedules))
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
			`{"duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00"}`,
			`{"id":"[0-9a-v]{20}","duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1s"},"links":\[{"rel":"self","href":"/water_schedules/[0-9a-v]{20}"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorRainWeatherClientDNE",
			`{"duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00", "weather_control":{"rain_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"c5cvhpcbcv45e8bp16dg"}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClient for WaterSchedule"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorTemperatureWeatherClientDNE",
			`{"duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00", "weather_control":{"temperature_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"c5cvhpcbcv45e8bp16dg"}}}`,
			`{"status":"Invalid request.","error":"unable to get WeatherClient for WaterSchedule"}`,
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
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			wsr := WaterSchedulesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}
			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			r := httptest.NewRequest("POST", "/water_schedules", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wsr.createWaterSchedule)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
		})
	}
}
