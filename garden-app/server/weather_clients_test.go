package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createExampleWeatherClientConfig() *weather.Config {
	return &weather.Config{
		ID:   id,
		Type: "fake",
		Options: map[string]interface{}{
			"rain_mm":              25.4,
			"rain_interval":        "24h",
			"avg_high_temperature": 80,
		},
	}
}

func TestWeatherClientContextMiddleware(t *testing.T) {
	weatherClient := createExampleWeatherClientConfig()

	tests := []struct {
		name          string
		weatherClient *weather.Config
		setupMock     func(*storage.MockClient)
		path          string
		code          int
		expected      string
	}{
		{
			"Successful",
			weatherClient,
			func(mc *storage.MockClient) {
				mc.On("GetWeatherClientConfig", id).Return(weatherClient, nil)
			},
			"/weather_clients/c5cvhpcbcv45e8bp16dg",
			http.StatusOK,
			"",
		},
		{
			"ErrorInvalidID",
			weatherClient,
			func(mc *storage.MockClient) {},
			"/weather_clients/not-an-xid",
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
		},
		{
			"NotFoundError",
			nil,
			func(mc *storage.MockClient) {
				mc.On("GetWeatherClientConfig", id).Return(nil, nil)
			},
			"/weather_clients/c5cvhpcbcv45e8bp16dg",
			http.StatusNotFound,
			`{"status":"Resource not found."}`,
		},
		{
			"InternalError",
			nil,
			func(mc *storage.MockClient) {
				mc.On("GetWeatherClientConfig", id).Return(nil, errors.New("storage error"))
			},
			"/weather_clients/c5cvhpcbcv45e8bp16dg",
			http.StatusInternalServerError,
			`{"status":"Server Error.","error":"storage error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil).WithContext(context.Background())
			w := httptest.NewRecorder()

			storageClient := new(storage.MockClient)
			wcr, _ := NewWeatherClientsResource(storageClient)

			tt.setupMock(storageClient)

			testHandler := func(w http.ResponseWriter, r *http.Request) {
				ws := getWeatherClientFromContext(r.Context())
				assert.Equal(t, weatherClient, ws)
				render.Status(r, http.StatusOK)
			}

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/weather_clients/{%s}", weatherClientPathParam), func(r chi.Router) {
				r.Use(wcr.weatherClientContextMiddleware)
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

func TestGetWeatherClient(t *testing.T) {
	wcr := WeatherClientsResource{}
	weatherClientCtx := context.WithValue(context.Background(), weatherClientCtxKey, createExampleWeatherClientConfig())
	r := httptest.NewRequest("GET", "/weather_clients", nil).WithContext(weatherClientCtx)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(wcr.getWeatherClient)

	h.ServeHTTP(w, r)

	// check HTTP response status code
	if w.Code != http.StatusOK {
		t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
	}

	expected := `{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`
	actual := strings.TrimSpace(w.Body.String())
	if actual != expected {
		t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
	}
}

func TestUpdateWeatherClient(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*storage.MockClient)
		body      string
		expected  string
		status    int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveWeatherClientConfig", mock.Anything, mock.Anything).Return(nil)
			},
			`{"options": {"avg_high_temperature": 81}}`,
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":81,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			func(storageClient *storage.MockClient) {},
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`,
			http.StatusBadRequest,
		},
		{
			"BadRequestInvalidConfigForClient",
			func(storageClient *storage.MockClient) {},
			`{"options": {"rain_interval": "not duration"}}`,
			`{"status":"Invalid request.","error":"time: invalid duration \"not duration\""}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveWeatherClientConfig", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			`{"options": {"key": "value2"}}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			wcr, _ := NewWeatherClientsResource(storageClient)
			weatherClient := createExampleWeatherClientConfig()

			weatherClientCtx := context.WithValue(context.Background(), weatherClientCtxKey, weatherClient)
			r := httptest.NewRequest("PATCH", "/weather_clients", strings.NewReader(tt.body)).WithContext(weatherClientCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wcr.updateWeatherClient)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.status {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.status)
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

func TestDeleteWeatherClient(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*storage.MockClient)
		weatherClient *weather.Config
		expected      string
		code          int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetWaterSchedulesUsingWeatherClient", id).Return(nil, nil)
				storageClient.On("DeleteWeatherClientConfig", mock.Anything, mock.Anything).Return(nil)
			},
			createExampleWeatherClientConfig(),
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`,
			http.StatusOK,
		},
		{
			"UnableToDeleteUsedByWaterSchedules",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetWaterSchedulesUsingWeatherClient", id).Return([]*pkg.WaterSchedule{{}}, nil)
			},
			createExampleWeatherClientConfig(),
			`{"status":"Invalid request.","error":"unable to delete WeatherClient used by 1 WaterSchedules"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetWaterSchedulesUsingWeatherClient", id).Return(nil, nil)
				storageClient.On("DeleteWeatherClientConfig", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			createExampleWeatherClientConfig(),
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			wcr, _ := NewWeatherClientsResource(storageClient)

			weatherClientCtx := context.WithValue(context.Background(), weatherClientCtxKey, tt.weatherClient)
			r := httptest.NewRequest("DELETE", "/weather_clients", nil).WithContext(weatherClientCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wcr.deleteWeatherClient)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			assert.Equal(t, tt.expected, actual)
			storageClient.AssertExpectations(t)
		})
	}
}

func TestGetAllWeatherClients(t *testing.T) {
	weatherClient := createExampleWeatherClientConfig()

	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		expected       string
		expectedStatus int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetWeatherClientConfigs").Return([]*weather.Config{weatherClient}, nil)
			},
			`{"weather_clients":[{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}]}`,
			http.StatusOK,
		},
		{
			"StorageError",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetWeatherClientConfigs").Return(nil, errors.New("storage error"))
			},
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			wcr, _ := NewWeatherClientsResource(storageClient)

			r := httptest.NewRequest("GET", "/weather_clients", nil).WithContext(context.Background())
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wcr.getAllWeatherClients)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.expectedStatus)
			}

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			assert.Equal(t, tt.expected, actual)
			storageClient.AssertExpectations(t)
		})
	}
}

func TestCreateWeatherClient(t *testing.T) {
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
				storageClient.On("SaveWeatherClientConfig", mock.Anything, mock.Anything).Return(nil)
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`,
			`{"id":"[0-9a-v]{20}","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":\[{"rel":"self","href":"/weather_clients/[0-9a-v]{20}"}\]}`,
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
				storageClient.On("SaveWeatherClientConfig", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			wcr, _ := NewWeatherClientsResource(storageClient)

			r := httptest.NewRequest("POST", "/weather_clients", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wcr.createWeatherClient)

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

func TestTestWeatherClient(t *testing.T) {
	weatherClient, err := weather.NewClient(createExampleWeatherClientConfig(), func(m map[string]interface{}) error {
		return nil
	})
	assert.NoError(t, err)

	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		expected       string
		expectedStatus int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetWeatherClient", mock.Anything).Return(weatherClient, nil)
			},
			`{"rain":{"mm":76.2,"scale_factor":0},"average_temperature":{"celsius":80,"scale_factor":0}}`,
			http.StatusOK,
		},
		{
			"StorageError",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetWeatherClient", mock.Anything).Return(nil, errors.New("storage error"))
			},
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			wcr, _ := NewWeatherClientsResource(storageClient)
			weatherClient := createExampleWeatherClientConfig()

			weatherClientCtx := context.WithValue(context.Background(), weatherClientCtxKey, weatherClient)

			r := httptest.NewRequest("GET", "/weather_clients", nil).WithContext(weatherClientCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(wcr.testWeatherClient)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.expectedStatus)
			}

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			assert.Equal(t, tt.expected, actual)
			storageClient.AssertExpectations(t)
		})
	}
}
