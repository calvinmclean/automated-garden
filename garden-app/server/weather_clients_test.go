package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
)

func createExampleWeatherClientConfig() *weather.Config {
	return &weather.Config{
		ID:   id,
		Type: "fake",
		Options: map[string]interface{}{
			"rain_mm":              25.4,
			"rain_interval":        "24h",
			"avg_high_temperature": 80.0,
		},
	}
}

func TestWeatherClientContextMiddleware(t *testing.T) {
	weatherClient := createExampleWeatherClientConfig()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.SaveWeatherClientConfig(weatherClient)
	assert.NoError(t, err)

	tests := []struct {
		name          string
		weatherClient *weather.Config
		path          string
		code          int
		expected      string
	}{
		{
			"Successful",
			weatherClient,
			"/weather_clients/c5cvhpcbcv45e8bp16dg",
			http.StatusOK,
			"",
		},
		{
			"ErrorInvalidID",
			weatherClient,
			"/weather_clients/not-an-xid",
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
		},
		{
			"NotFoundError",
			nil,
			"/weather_clients/9m4e2mr0ui3e8a215n4g",
			http.StatusNotFound,
			`{"status":"Resource not found."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil).WithContext(context.Background())
			w := httptest.NewRecorder()

			wcr, _ := NewWeatherClientsResource(storageClient)

			testHandler := func(w http.ResponseWriter, r *http.Request) {
				wc := getWeatherClientFromContext(r.Context())
				assert.Equal(t, weatherClient, wc)
				render.Status(r, http.StatusOK)
			}

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/weather_clients/{%s}", weatherClientPathParam), func(r chi.Router) {
				r.Use(wcr.weatherClientContextMiddleware)
				r.Get("/", testHandler)
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
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
	weatherClient := createExampleWeatherClientConfig()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.SaveWeatherClientConfig(weatherClient)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		body     string
		expected string
		status   int
	}{
		{
			"Successful",
			`{"options": {"avg_high_temperature": 81}}`,
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":81,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`,
			http.StatusBadRequest,
		},
		{
			"BadRequestInvalidConfigForClient",
			`{"options": {"rain_interval": "not duration"}}`,
			`{"status":"Invalid request.","error":"time: invalid duration \"not duration\""}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcr, _ := NewWeatherClientsResource(storageClient)

			r := httptest.NewRequest("PATCH", "/weather_clients/c5cvhpcbcv45e8bp16dg", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/weather_clients/{%s}", weatherClientPathParam), func(r chi.Router) {
				r.Use(wcr.weatherClientContextMiddleware)
				r.Patch("/", wcr.updateWeatherClient)
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestDeleteWeatherClient(t *testing.T) {
	weatherClient := createExampleWeatherClientConfig()

	weatherClientWithWS := createExampleWeatherClientConfig()
	weatherClientWithWS.ID = id2

	ws := createExampleWaterSchedule()
	ws.WeatherControl = &weather.Control{
		Rain: &weather.ScaleControl{
			ClientID: id2,
		},
	}

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.SaveWeatherClientConfig(weatherClient)
	assert.NoError(t, err)
	err = storageClient.SaveWeatherClientConfig(weatherClientWithWS)
	assert.NoError(t, err)
	err = storageClient.SaveWaterSchedule(ws)
	assert.NoError(t, err)

	tests := []struct {
		name          string
		id            string
		weatherClient *weather.Config
		expected      string
		code          int
	}{
		{
			"Successful",
			id.String(),
			createExampleWeatherClientConfig(),
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`,
			http.StatusOK,
		},
		{
			"UnableToDeleteUsedByWaterSchedules",
			id2.String(),
			createExampleWeatherClientConfig(),
			`{"status":"Invalid request.","error":"unable to delete WeatherClient used by 1 WaterSchedules"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcr, _ := NewWeatherClientsResource(storageClient)

			weatherClientCtx := context.WithValue(context.Background(), weatherClientCtxKey, tt.weatherClient)
			r := httptest.NewRequest("DELETE", "/weather_clients/"+tt.id, nil).WithContext(weatherClientCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/weather_clients/{%s}", weatherClientPathParam), func(r chi.Router) {
				r.Use(wcr.weatherClientContextMiddleware)
				r.Delete("/", wcr.deleteWeatherClient)
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestGetAllWeatherClients(t *testing.T) {
	weatherClient := createExampleWeatherClientConfig()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.SaveWeatherClientConfig(weatherClient)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		expected       string
		expectedStatus int
	}{
		{
			"Successful",
			`{"weather_clients":[{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}]}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcr, _ := NewWeatherClientsResource(storageClient)

			r := httptest.NewRequest("GET", "/weather_clients", nil).WithContext(context.Background())
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get("/weather_clients", wcr.getAllWeatherClients)
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestCreateWeatherClient(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	tests := []struct {
		name           string
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}`,
			`{"id":"[0-9a-v]{20}","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":\[{"rel":"self","href":"/weather_clients/[0-9a-v]{20}"}\]}`,
			http.StatusCreated,
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
			wcr, _ := NewWeatherClientsResource(storageClient)

			r := httptest.NewRequest("POST", "/weather_clients", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post("/weather_clients", wcr.createWeatherClient)
			router.ServeHTTP(w, r)

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

func TestTestWeatherClient(t *testing.T) {
	weatherClient := createExampleWeatherClientConfig()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.SaveWeatherClientConfig(weatherClient)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		expected       string
		expectedStatus int
	}{
		{
			"Successful",
			`{"rain":{"mm":76.2,"scale_factor":0},"average_temperature":{"celsius":80,"scale_factor":0}}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcr, _ := NewWeatherClientsResource(storageClient)
			weatherClient := createExampleWeatherClientConfig()

			weatherClientCtx := context.WithValue(context.Background(), weatherClientCtxKey, weatherClient)

			r := httptest.NewRequest("GET", "/weather_clients/c5cvhpcbcv45e8bp16dg", nil).WithContext(weatherClientCtx)
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/weather_clients/{%s}", weatherClientPathParam), func(r chi.Router) {
				r.Use(wcr.weatherClientContextMiddleware)
				r.Get("/", wcr.testWeatherClient)
			})
			router.ServeHTTP(w, r)

			// check HTTP response status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			assert.Equal(t, tt.expected, actual)
		})
	}
}
