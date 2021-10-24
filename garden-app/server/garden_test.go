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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func init() {
	logger = logrus.New()
}

func createExampleGarden() *pkg.Garden {
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Garden{
		Name:      "test-garden",
		ID:        id,
		Plants:    map[xid.ID]*pkg.Plant{},
		CreatedAt: &time,
	}
}

func TestGardenContextMiddleware(t *testing.T) {
	garden := createExampleGarden()

	tests := []struct {
		name      string
		setupMock func(*storage.MockClient)
		path      string
		expected  string
		code      int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetGarden", mock.Anything).Return(garden, nil)
			},
			"/garden/c5cvhpcbcv45e8bp16dg",
			"",
			http.StatusOK,
		},
		{
			"ErrorInvalidID",
			func(storageClient *storage.MockClient) {},
			"/garden/not-an-xid",
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetGarden", garden.ID).Return(nil, errors.New("storage client error"))
			},
			"/garden/c5cvhpcbcv45e8bp16dg",
			`{"status":"Server Error.","error":"storage client error"}`,
			http.StatusInternalServerError,
		},
		{
			"StatusNotFound",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetGarden", garden.ID).Return(nil, nil)
			},
			"/garden/c5cvhpcbcv45e8bp16dg",
			`{"status":"Resource not found."}`,
			http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)
			gr := GardensResource{
				storageClient: storageClient,
				config:        Config{},
			}
			testHandler := func(w http.ResponseWriter, r *http.Request) {
				g := r.Context().Value(gardenCtxKey).(*pkg.Garden)
				if garden != g {
					t.Errorf("Unexpected Garden saved in request context. Expected %v but got %v", garden, g)
				}
				render.Status(r, http.StatusOK)
			}

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/garden/{%s}", gardenPathParam), func(r chi.Router) {
				r.Use(gr.gardenContextMiddleware)
				r.Get("/", testHandler)
			})

			r := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}

			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
			storageClient.AssertExpectations(t)
		})
	}
}

func TestGardenRestrictEndDatedMiddleware(t *testing.T) {
	gr := GardensResource{}
	garden := createExampleGarden()
	endDatedGarden := createExampleGarden()
	endDate := time.Now().Add(-1 * time.Minute)
	endDatedGarden.EndDate = &endDate
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
	}

	router := chi.NewRouter()
	router.Route("/garden", func(r chi.Router) {
		r.Use(gr.restrictEndDatedMiddleware)
		r.Get("/", testHandler)
	})

	tests := []struct {
		name     string
		garden   *pkg.Garden
		code     int
		expected string
	}{
		{
			"GardenNotEndDated",
			garden,
			http.StatusOK,
			"",
		},
		{
			"GardenEndDated",
			endDatedGarden,
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"resource not available for end-dated Garden"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), gardenCtxKey, tt.garden)
			r := httptest.NewRequest("GET", "/garden", nil).WithContext(ctx)
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

func TestCreateGarden(t *testing.T) {
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
				storageClient.On("SaveGarden", mock.Anything).Return(nil)
			},
			`{"name": "test-garden"}`,
			`{"name":"test-garden","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorInvalidRequestBody",
			func(storageClient *storage.MockClient) {},
			"{}",
			`{"status":"Invalid request.","error":"missing required Garden fields"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage client error"))
			},
			`{"name": "test-garden"}`,
			`{"status":"Server Error.","error":"storage client error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)
			gr := GardensResource{
				storageClient: storageClient,
				config:        Config{},
			}

			r := httptest.NewRequest("POST", "/garden", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(gr.createGarden)

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

func TestGetAllGardens(t *testing.T) {
	gardens := []*pkg.Garden{createExampleGarden()}

	tests := []struct {
		name           string
		targetURL      string
		setupMock      func(*storage.MockClient)
		expectedRegexp string
		status         int
	}{
		{
			"SuccessfulEndDatedFalse",
			"/gardens",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetGardens", false).Return(gardens, nil)
			},
			`{"gardens":\[{"name":"test-garden","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfulEndDatedTrue",
			"/gardens?end_dated=true",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetGardens", true).Return(gardens, nil)
			},
			`{"gardens":\[{"name":"test-garden","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}\]}`,
			http.StatusOK,
		},
		{
			"StorageClientError",
			"/gardens",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetGardens", false).Return([]*pkg.Garden{}, errors.New("storage client error"))
			},
			`{"status":"Error rendering response.","error":"storage client error"}`,
			http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			gr := GardensResource{
				storageClient: storageClient,
				config:        Config{},
			}
			tt.setupMock(storageClient)

			r := httptest.NewRequest("GET", tt.targetURL, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(gr.getAllGardens)

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

func TestGetGarden(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()

		ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
		r := httptest.NewRequest("GET", "/garden", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.getGarden)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		gardenJSON, _ := json.Marshal(gr.NewGardenResponse(garden))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(gardenJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(gardenJSON))
		}
		storageClient.AssertExpectations(t)
	})
}

func TestEndDateGarden(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(nil)
			},
			`{"name":"test-garden","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}`,
			http.StatusOK,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage client error"))
			},
			`{"status":"Server Error.","error":"storage client error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)
			gr := GardensResource{
				storageClient: storageClient,
				config:        Config{},
			}
			garden := createExampleGarden()

			ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
			r := httptest.NewRequest("DELETE", "/garden", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(gr.endDateGarden)

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

func TestUpdateGarden(t *testing.T) {
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
				storageClient.On("SaveGarden", mock.Anything).Return(nil)
			},
			`{"name": "new name"}`,
			`{"name":"new name","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}`,
			http.StatusOK,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage client error"))
			},
			`{"name": "new name"}`,
			`{"status":"Server Error.","error":"storage client error"}`,
			http.StatusInternalServerError,
		},
		{
			"ErrorInvalidRequestBody",
			func(storageClient *storage.MockClient) {},
			"{}",
			`{"status":"Invalid request.","error":"missing required Garden fields"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)
			gr := GardensResource{
				storageClient: storageClient,
				config:        Config{},
			}
			garden := createExampleGarden()

			ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
			r := httptest.NewRequest("PATCH", "/garden", strings.NewReader(tt.body)).WithContext(ctx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(gr.updateGarden)

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

func TestGetGardenHealth(t *testing.T) {
	now := time.Now()
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	tests := []struct {
		name           string
		time           time.Time
		err            error
		expectedStatus string
	}{
		{
			"UP",
			now,
			nil,
			"UP",
		},
		{
			"DOWN",
			fiveMinutesAgo,
			nil,
			"DOWN",
		},
		{
			"N/A",
			now,
			errors.New("influxdb error"),
			"N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			influxdbClient := new(influxdb.MockClient)
			gr := GardensResource{
				storageClient:  storageClient,
				influxdbClient: influxdbClient,
				config:         Config{},
			}
			garden := createExampleGarden()
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(tt.time, tt.err)
			influxdbClient.On("Close")

			ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
			r := httptest.NewRequest("GET", "/health", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(gr.getGardenHealth)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
			}

			// check HTTP response body
			var actual GardenHealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &actual)
			if err != nil {
				t.Errorf("Unexpected error unmarshaling GardenHealthResponse: %v", err)
			}
			if actual.Status != tt.expectedStatus {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual.Status, tt.expectedStatus)
			}
			storageClient.AssertExpectations(t)
		})
	}
}
