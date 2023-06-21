package server

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/stretchr/testify/mock"
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

func TestWaterScheduleContextMiddleware(t *testing.T) {
	waterSchedule := createExampleWaterSchedule()

	tests := []struct {
		name          string
		waterSchedule *pkg.WaterSchedule
		setupMock     func(*storage.MockClient)
		path          string
		code          int
		expected      string
	}{
		{
			"Successful",
			waterSchedule,
			func(mc *storage.MockClient) {
				mc.On("GetWaterSchedule", id).Return(createExampleWaterSchedule(), nil)
			},
			"/water_schedules/c5cvhpcbcv45e8bp16dg",
			http.StatusOK,
			"",
		},
		{
			"ErrorInvalidID",
			waterSchedule,
			func(mc *storage.MockClient) {},
			"/water_schedules/not-an-xid",
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
		},
		{
			"NotFoundError",
			nil,
			func(mc *storage.MockClient) {
				mc.On("GetWaterSchedule", id).Return(nil, nil)
			},
			"/water_schedules/c5cvhpcbcv45e8bp16dg",
			http.StatusNotFound,
			`{"status":"Resource not found."}`,
		},
		{
			"InternalError",
			nil,
			func(mc *storage.MockClient) {
				mc.On("GetWaterSchedule", id).Return(nil, errors.New("storage error"))
			},
			"/water_schedules/c5cvhpcbcv45e8bp16dg",
			http.StatusInternalServerError,
			`{"status":"Server Error.","error":"storage error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil).WithContext(context.Background())
			w := httptest.NewRecorder()

			storageClient := new(storage.MockClient)
			wsr := WaterSchedulesResource{
				storageClient: storageClient,
			}

			tt.setupMock(storageClient)

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

			storageClient.AssertExpectations(t)
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

func TestGetWaterSchedule(t *testing.T) {
	weatherClientID, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")

	tests := []struct {
		name           string
		waterSchedule  *pkg.WaterSchedule
		setupMock      func(*influxdb.MockClient, *weather.MockClient, *storage.MockClient)
		expectedRegexp string
	}{
		{
			"Successful",
			createExampleWaterSchedule(),
			func(*influxdb.MockClient, *weather.MockClient, *storage.MockClient) {},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386\-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386\-07:00","duration":"1s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"SuccessfulWithRainAndTemperatureData",
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
			func(influxdbClient *influxdb.MockClient, weatherClient *weather.MockClient, storageClient *storage.MockClient) {
				storageClient.On("GetWeatherClient", weatherClientID).Return(weatherClient, nil)
				weatherClient.On("GetTotalRain", mock.Anything).Return(float32(12.7), nil)
				weatherClient.On("GetAverageHighTemperature", mock.Anything).Return(float32(35), nil)
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","weather_control":{"rain_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"c5cvhpcbcv45e8bp16dg"},"temperature_control":{"baseline_value":30,"factor":0.5,"range":10,"client_id":"c5cvhpcbcv45e8bp16dg"}},"weather_data":{"rain":{"mm":12.7,"scale_factor":0.5},"average_temperature":{"celsius":35,"scale_factor":1.25}},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"37m30.000039936s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
		{
			"SuccessfulAfterErrorGettingTemperatureWeatherClient",
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
			func(influxdbClient *influxdb.MockClient, weatherClient *weather.MockClient, storageClient *storage.MockClient) {
				storageClient.On("GetWeatherClient", weatherClientID).Return(nil, errors.New("storage error"))
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1h0m0s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","weather_control":{"rain_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"c5cvhpcbcv45e8bp16dg"},"temperature_control":{"baseline_value":30,"factor":0.5,"range":10,"client_id":"c5cvhpcbcv45e8bp16dg"}},"weather_data":{},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1h0m0s","message":"unable to determine water duration scaling"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			weatherClient := new(weather.MockClient)
			storageClient := new(storage.MockClient)
			tt.setupMock(influxdbClient, weatherClient, storageClient)
			storageClient.On("GetWaterSchedules", false).Return([]*pkg.WaterSchedule{tt.waterSchedule}, nil)
			influxdbClient.On("Close")

			wsr, err := NewWaterSchedulesResource(storageClient, worker.NewWorker(storageClient, influxdbClient, nil, logrus.New()))
			assert.NoError(t, err)
			wsr.worker.StartAsync()

			garden := createExampleGarden()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			waterScheduleCtx := context.WithValue(gardenCtx, waterScheduleCtxKey, tt.waterSchedule)
			r := httptest.NewRequest("GET", "/water_schedules", nil).WithContext(waterScheduleCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wsr.getWaterSchedule)

			h.ServeHTTP(w, r)

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
			weatherClient.AssertExpectations(t)
			storageClient.AssertExpectations(t)
		})
	}
}

func TestUpdateWaterSchedule(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		body           string
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveWaterSchedule", mock.Anything, mock.Anything).Return(nil)
			},
			`{"interval":"1h"}`,
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"1h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT22:24:52.891386-07:00","duration":"1s"},"links":\[{"rel":"self","href":"/water_schedules/c5cvhpcbcv45e8bp16dg"}\]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			func(storageClient *storage.MockClient) {},
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"BadRequestInvalidTemperatureControl",
			func(storageClient *storage.MockClient) {},
			`{"weather_control":{"temperature_control":{"baseline_value":27,"factor":-1,"range":10}}}`,
			`{"status":"Invalid request.","error":"error validating temperature_control: factor must be between 0 and 1"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveWaterSchedule", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			`{"interval":"1h"}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			wsr := WaterSchedulesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(nil, nil, nil, logrus.New()),
			}
			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			garden := createExampleGarden()
			waterSchedule := createExampleWaterSchedule()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			waterScheduleCtx := context.WithValue(gardenCtx, waterScheduleCtxKey, waterSchedule)
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
			storageClient.AssertExpectations(t)
		})
	}
}

func TestEndDateWaterSchedule(t *testing.T) {
	now := time.Now()
	endDatedWaterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule.EndDate = &now

	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		waterSchedule  *pkg.WaterSchedule
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetZonesUsingWaterSchedule", id).Return(nil, nil)
				storageClient.On("SaveWaterSchedule", mock.Anything, mock.Anything).Return(nil)
			},
			createExampleWaterSchedule(),
			`{"id":"c5cvhpcbcv45e8bp16dg","duration":"1s","interval":"24h0m0s","start_time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","next_water":{},"links":\[{"rel":"self","href":"/water_schedules/[0-9a-v]{20}"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteWaterSchedule",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetZonesUsingWaterSchedule", id).Return(nil, nil)
				storageClient.On("DeleteWaterSchedule", mock.Anything, mock.Anything).Return(nil)
			},
			endDatedWaterSchedule,
			"",
			http.StatusNoContent,
		},
		{
			"UnableToDeleteUsedByZones",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetZonesUsingWaterSchedule", id).Return([]*pkg.ZoneAndGarden{{}}, nil)
			},
			endDatedWaterSchedule,
			`{"status":"Invalid request.","error":"unable to end-date WaterSchedule with 1 Zones"}`,
			http.StatusBadRequest,
		},
		{
			"DeleteWaterScheduleError",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetZonesUsingWaterSchedule", id).Return(nil, nil)
				storageClient.On("DeleteWaterSchedule", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			endDatedWaterSchedule,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetZonesUsingWaterSchedule", id).Return(nil, nil)
				storageClient.On("SaveWaterSchedule", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			createExampleWaterSchedule(),
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			wsr := WaterSchedulesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(nil, nil, nil, logrus.New()),
			}
			wsr.worker.StartAsync()
			defer wsr.worker.Stop()

			garden := createExampleGarden()
			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			waterScheduleCtx := context.WithValue(gardenCtx, waterScheduleCtxKey, tt.waterSchedule)
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
			storageClient.AssertExpectations(t)
		})
	}
}

func TestGetAllWaterSchedules(t *testing.T) {
	storageClient := new(storage.MockClient)
	wsr := WaterSchedulesResource{
		worker:        worker.NewWorker(nil, nil, nil, logrus.New()),
		storageClient: storageClient,
	}
	waterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule := createExampleWaterSchedule()
	endDatedWaterSchedule.ID = xid.New()
	now := time.Now()
	endDatedWaterSchedule.EndDate = &now

	storageClient.On("GetWaterSchedules", false).Return([]*pkg.WaterSchedule{waterSchedule}, nil)
	storageClient.On("GetWaterSchedules", true).Return([]*pkg.WaterSchedule{waterSchedule, endDatedWaterSchedule}, nil)

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
			err := json.NewDecoder(w.Body).Decode(&actual)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(actual.WaterSchedules))
		})
	}
}

func TestCreateWaterSchedule(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveWaterSchedule", mock.Anything, mock.Anything).Return(nil)
			},
			`{"duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00"}`,
			`{"id":"[0-9a-v]{20}","duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00","next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1s"},"links":\[{"rel":"self","href":"/water_schedules/[0-9a-v]{20}"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorBadRequestBadJSON",
			func(storageClient *storage.MockClient) {},
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveWaterSchedule", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			`{"duration":"1s","interval":"24h0m0s","start_time":"2021-10-03T11:24:52.891386-07:00"}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

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
			storageClient.AssertExpectations(t)
		})
	}
}
