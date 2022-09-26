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
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func createExampleGarden() *pkg.Garden {
	two := uint(2)
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxZones:    &two,
		ID:          id,
		Plants:      map[xid.ID]*pkg.Plant{},
		Zones:       map[xid.ID]*pkg.Zone{},
		CreatedAt:   &time,
		LightSchedule: &pkg.LightSchedule{
			Duration:  "15h",
			StartTime: "22:00:01-07:00",
		},
	}
}

func createExampleGardenWithZone() *pkg.Garden {
	garden := createExampleGarden()
	zone := createExampleZone()
	garden.Zones[zone.ID] = zone
	return garden
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
				g := getGardenFromContext(r.Context())
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
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorNegativeMaxPlants",
			func(storageClient *storage.MockClient) {},
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones":-2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -2 into Go struct field GardenRequest.max_zones of type uint"}`,
			http.StatusBadRequest,
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
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2}`,
			`{"status":"Server Error.","error":"storage client error"}`,
			http.StatusInternalServerError,
		},
		{
			"ErrorBadRequestInvalidStartTime",
			func(storageClient *storage.MockClient) {},
			`{"name":"test-garden", "topic_prefix":"test-garden", "max_zones": 1,"light_schedule": {"duration":"1h","start_time":"NOT A TIME"}}`,
			`{"status":"Invalid request.","error":"invalid time format for light_schedule.start_time: NOT A TIME"}`,
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
				worker:        worker.NewWorker(storageClient, nil, nil, nil, logrus.New()),
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
			`{"gardens":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfulEndDatedTrue",
			"/gardens?end_dated=true",
			func(storageClient *storage.MockClient) {
				storageClient.On("GetGardens", true).Return(gardens, nil)
			},
			`{"gardens":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
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
				worker:        worker.NewWorker(storageClient, nil, nil, nil, logrus.New()),
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
			worker:        worker.NewWorker(storageClient, nil, nil, nil, logrus.New()),
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

		gardenJSON, _ := json.Marshal(gr.NewGardenResponse(ctx, garden))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(gardenJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(gardenJSON))
		}
		storageClient.AssertExpectations(t)
	})
}

func TestEndDateGarden(t *testing.T) {
	now := time.Now()
	endDatedGarden := createExampleGarden()
	endDatedGarden.EndDate = &now

	gardenWithZone := createExampleGarden()
	gardenWithZone.Zones[xid.New()] = &pkg.Zone{}

	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		garden         *pkg.Garden
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(nil)
			},
			createExampleGarden(),
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteGarden",
			func(storageClient *storage.MockClient) {
				storageClient.On("DeleteGarden", mock.Anything).Return(nil)
			},
			endDatedGarden,
			"",
			http.StatusNoContent,
		},
		{
			"ErrorEndDatingGardenWithZones",
			func(storageClient *storage.MockClient) {},
			gardenWithZone,
			`{"status":"Invalid request.","error":"unable to end-date Garden with active Zones"}`,
			http.StatusBadRequest,
		},
		{
			"DeleteGardenError",
			func(storageClient *storage.MockClient) {
				storageClient.On("DeleteGarden", mock.Anything).Return(errors.New("storage error"))
			},
			endDatedGarden,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage error"))
			},
			createExampleGarden(),
			`{"status":"Server Error.","error":"storage error"}`,
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
				worker:        worker.NewWorker(storageClient, nil, nil, nil, logrus.New()),
			}

			ctx := context.WithValue(context.Background(), gardenCtxKey, tt.garden)
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
	gardenWithoutLight := createExampleGarden()
	gardenWithoutLight.LightSchedule = nil

	gardenWithZone := createExampleGarden()
	gardenWithZone.Zones[xid.New()] = &pkg.Zone{}
	gardenWithZone.Zones[xid.New()] = &pkg.Zone{}

	tests := []struct {
		name           string
		garden         *pkg.Garden
		setupMock      func(*storage.MockClient)
		body           string
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			createExampleGarden(),
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(nil)
			},
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyRemoveLightSchedule",
			createExampleGarden(),
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(nil)
			},
			`{"name": "new name","light_schedule": {}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyAddLightSchedule",
			gardenWithoutLight,
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(nil)
			},
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"health","href":"/gardens/[0-9a-v]{20}/health"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"StorageClientError",
			createExampleGarden(),
			func(storageClient *storage.MockClient) {
				storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage client error"))
			},
			`{"name": "new name"}`,
			`{"status":"Server Error.","error":"storage client error"}`,
			http.StatusInternalServerError,
		},
		{
			"ErrorInvalidRequestBody",
			createExampleGarden(),
			func(storageClient *storage.MockClient) {},
			"{}",
			`{"status":"Invalid request.","error":"missing required Garden fields"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorReducingMaxZones",
			gardenWithZone,
			func(storageClient *storage.MockClient) {},
			`{"max_zones": 1}`,
			`{"status":"Invalid request.","error":"unable to set max_zones less than current num_zones=2"}`,
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
				worker:        worker.NewWorker(storageClient, nil, nil, nil, logrus.New()),
			}

			ctx := context.WithValue(context.Background(), gardenCtxKey, tt.garden)
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

func TestGardenAction(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mqtt.MockClient)
		body      string
		expected  string
		status    int
	}{
		{
			"BadRequest",
			func(mqttClient *mqtt.MockClient) {},
			"bad request",
			`{"status":"Invalid request.","error":"invalid character 'b' looking for beginning of value"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulLightAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("LightTopic", "test-garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
			},
			`{"light":{"state":"on"}}`,
			"null",
			http.StatusAccepted,
		},
		{
			"ExecuteErrorForLightAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("LightTopic", "test-garden").Return("", errors.New("template error"))
			},
			`{"light":{"state":"on"}}`,
			`{"status":"Server Error.","error":"unable to execute LightAction: unable to fill MQTT topic template: template error"}`,
			http.StatusInternalServerError,
		},
		{
			"ErrorInvalidLightState",
			func(mqttClient *mqtt.MockClient) {},
			`{"light":{"state":"BAD"}}`,
			`{"status":"Invalid request.","error":"cannot unmarshal \"BAD\" into Go value of type *pkg.LightState"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			tt.setupMock(mqttClient)

			gr := GardensResource{
				worker: worker.NewWorker(nil, nil, mqttClient, nil, logrus.New()),
			}
			garden := createExampleGarden()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			r := httptest.NewRequest("POST", "/garden", strings.NewReader(tt.body)).WithContext(gardenCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(gr.gardenAction)

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
			mqttClient.AssertExpectations(t)
		})
	}
}
