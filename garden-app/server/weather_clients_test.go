package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createExampleWeatherClientConfig() *weather.Config {
	return &weather.Config{
		ID:   babyapi.ID{ID: id},
		Type: "fake",
		Options: map[string]interface{}{
			"rain_mm":              25.4,
			"rain_interval":        "24h",
			"avg_high_temperature": 80.0,
		},
	}
}

func TestUpdateWeatherClient(t *testing.T) {
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
			"ErrorInvalid",
			`{"type": "other_type"}`,
			`{"status":"Invalid request.","error":"invalid request to update WeatherClient: invalid type 'other_type'"}`,
			http.StatusBadRequest,
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
			`{"status":"Invalid request.","error":"invalid request to update WeatherClient: time: invalid duration \"not duration\""}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			wcr, err := NewWeatherClientsAPI(storageClient)
			require.NoError(t, err)

			err = wcr.storageClient.WeatherClientConfigs.Set(context.Background(), createExampleWeatherClientConfig())
			assert.NoError(t, err)

			r := httptest.NewRequest("PATCH", "/weather_clients/c5cvhpcbcv45e8bp16dg", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")

			w := babytest.TestRequest[*weather.Config](t, wcr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestGetWeatherClient(t *testing.T) {
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
			"NotFoundError",
			id2.String(),
			createExampleWeatherClientConfig(),
			`{"status":"Resource not found."}`,
			http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			wcr, err := NewWeatherClientsAPI(storageClient)
			require.NoError(t, err)

			err = wcr.storageClient.WeatherClientConfigs.Set(context.Background(), createExampleWeatherClientConfig())
			assert.NoError(t, err)

			r := httptest.NewRequest("GET", "/weather_clients/"+tt.id, http.NoBody)
			r.Header.Add("Content-Type", "application/json")

			w := babytest.TestRequest[*weather.Config](t, wcr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestDeleteWeatherClient(t *testing.T) {
	weatherClient := createExampleWeatherClientConfig()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	weatherClientWithWS := createExampleWeatherClientConfig()
	weatherClientWithWS.ID = babyapi.ID{ID: id2}

	ws1 := createExampleWaterSchedule()
	ws1.WeatherControl = &weather.Control{
		Rain: &weather.ScaleControl{
			ClientID: id2,
		},
		Temperature: &weather.ScaleControl{
			ClientID: id2,
		},
	}

	// This water schedule creates the situation where a WaterSchedule has WeatherControl, but doesn't match the ID
	ws2 := createExampleWaterSchedule()
	ws2.ID = babyapi.NewID()
	ws2.WeatherControl = &weather.Control{
		Rain: &weather.ScaleControl{
			ClientID: xid.New(),
		},
		Temperature: &weather.ScaleControl{
			ClientID: xid.New(),
		},
	}

	err = storageClient.WaterSchedules.Set(context.Background(), ws1)
	assert.NoError(t, err)
	err = storageClient.WaterSchedules.Set(context.Background(), ws2)
	assert.NoError(t, err)

	err = storageClient.WeatherClientConfigs.Set(context.Background(), weatherClient)
	assert.NoError(t, err)
	err = storageClient.WeatherClientConfigs.Set(context.Background(), weatherClientWithWS)
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
			``,
			http.StatusNoContent,
		},
		{
			"UnableToDeleteUsedByWaterSchedules",
			id2.String(),
			createExampleWeatherClientConfig(),
			`{"status":"Invalid request.","error":"unable to delete WeatherClient used by 2 WaterSchedules"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcr, err := NewWeatherClientsAPI(storageClient)
			require.NoError(t, err)

			r := httptest.NewRequest("DELETE", "/weather_clients/"+tt.id, http.NoBody)
			r.Header.Add("Content-Type", "application/json")

			w := babytest.TestRequest[*weather.Config](t, wcr.API, r)

			assert.Equal(t, tt.code, w.Code)
		})
	}
}

func TestGetAllWeatherClients(t *testing.T) {
	tests := []struct {
		name           string
		expected       string
		expectedStatus int
	}{
		{
			"Successful",
			`{"items":[{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":[{"rel":"self","href":"/weather_clients/c5cvhpcbcv45e8bp16dg"}]}]}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			wcr, err := NewWeatherClientsAPI(storageClient)
			require.NoError(t, err)

			err = wcr.storageClient.WeatherClientConfigs.Set(context.Background(), createExampleWeatherClientConfig())
			assert.NoError(t, err)

			r := httptest.NewRequest("GET", "/weather_clients", http.NoBody)

			w := babytest.TestRequest[*weather.Config](t, wcr.API, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestCreateWeatherClient(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4}}`,
			`{"id":"[0-9a-v]{20}","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4},"links":\[{"rel":"self","href":"/weather_clients/[0-9a-v]{20}"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorInvalid",
			`{"type": "fake", "options":{"key": "value"}}`,
			`{"status":"Invalid request.","error":"invalid request to update WeatherClient: time: invalid duration \\"\\""}`,
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
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4}}`,
			`{"status":"Invalid request.","error":"unable to manually set ID"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			wcr, err := NewWeatherClientsAPI(storageClient)
			require.NoError(t, err)

			r := httptest.NewRequest("POST", "/weather_clients", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")

			w := babytest.TestRequest[*weather.Config](t, wcr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestUpdateWeatherClientPUT(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"id":"c5cvhpcbcv45e8bp16dg","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4}}`,
			``,
			http.StatusOK,
		},
		{
			"ErrorBadRequestBadJSON",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorMissingID",
			`{"type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4}}`,
			`{"status":"Invalid request.","error":"missing required id field"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWrongID",
			`{"id":"chkodpg3lcj13q82mq40","type":"fake","options":{"avg_high_temperature":80,"rain_interval":"24h","rain_mm":25.4}}`,
			`{"status":"Invalid request.","error":"id must match URL path"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			wc := createExampleWeatherClientConfig()
			err = storageClient.WeatherClientConfigs.Set(context.Background(), wc)
			assert.NoError(t, err)

			wcr, err := NewWeatherClientsAPI(storageClient)
			require.NoError(t, err)

			r := httptest.NewRequest(http.MethodPut, "/weather_clients/"+wc.ID.String(), strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")

			w := babytest.TestRequest[*weather.Config](t, wcr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestTestWeatherClient(t *testing.T) {
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
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			wcr, err := NewWeatherClientsAPI(storageClient)
			require.NoError(t, err)

			err = wcr.storageClient.WeatherClientConfigs.Set(context.Background(), createExampleWeatherClientConfig())
			assert.NoError(t, err)

			r := httptest.NewRequest("GET", "/weather_clients/c5cvhpcbcv45e8bp16dg/test", http.NoBody)
			w := babytest.TestRequest[*weather.Config](t, wcr.API, r)

			// check HTTP response status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestWeatherClientRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *weather.Config
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WeatherClient fields",
		},
		{
			"EmptyTypeError",
			&weather.Config{},
			"missing required type field",
		},
		{
			"EmptyOptionsError",
			&weather.Config{
				Type: "fake",
			},
			"missing required options field",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		req := createExampleWeatherClientConfig()
		req.ID = babyapi.ID{ID: xid.NilID()}
		r := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
		err := req.Bind(r)
		assert.NoError(t, err)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			err := tt.req.Bind(r)
			assert.Error(t, err)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestUpdateWeatherClientRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *weather.Config
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WeatherClient fields",
		},
		{
			"ManualSpecificationOfIDError",
			&weather.Config{
				ID: babyapi.NewID(),
			},
			"updating ID is not allowed",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		wsr := &weather.Config{
			Type: "fake",
		}
		r := httptest.NewRequest(http.MethodPatch, "/", nil)
		err := wsr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading WeatherClientRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPatch, "/", nil)
			err := tt.req.Bind(r)
			if err == nil {
				t.Error("Expected error reading WeatherClientRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
