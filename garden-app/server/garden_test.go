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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createExampleGarden() *pkg.Garden {
	two := uint(2)
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxZones:    &two,
		ID:          id,
		Plants:      map[xid.ID]*pkg.Plant{},
		Zones:       map[xid.ID]*pkg.Zone{},
		CreatedAt:   &createdAt,
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: "22:00:01-07:00",
		},
	}
}

func TestGetGarden(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		code     int
	}{
		{
			"Successful",
			"/gardens/c5cvhpcbcv45e8bp16dg",
			`{"name":"test-garden","topic_prefix":"test-garden","id":"c5cvhpcbcv45e8bp16dg","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_plants":1,"num_zones":1,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/action"}\]}`,
			http.StatusOK,
		},
		{
			"ErrorInvalidID",
			"/gardens/not-an-xid",
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
			http.StatusBadRequest,
		},
		{
			"StatusNotFound",
			"/gardens/chkodpg3lcj13q82mq40",
			`{"status":"Resource not found."}`,
			http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			storageClient := setupZonePlantGardenStorage(t)
			gr := GardensResource{
				storageClient:  storageClient,
				influxdbClient: influxdbClient,
				worker:         worker.NewWorker(storageClient, nil, nil, logrus.New()),
				config:         Config{},
			}

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/gardens/{%s}", gardenPathParam), func(r chi.Router) {
				r.Use(gr.gardenContextMiddleware)
				r.Get("/", gr.getGarden)
			})

			r := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}

			matcher := regexp.MustCompile(tt.expected)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
		})
	}
}

func TestGardenRestrictEndDatedMiddleware(t *testing.T) {
	garden := createExampleGarden()
	endDatedGarden := createExampleGarden()
	endDate := time.Now().Add(-1 * time.Minute)
	endDatedGarden.EndDate = &endDate
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
	}

	router := chi.NewRouter()
	router.Route("/garden", func(r chi.Router) {
		r.Use(restrictEndDatedMiddleware("Garden", gardenCtxKey))
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
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorNegativeMaxPlants",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones":-2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -2 into Go struct field GardenRequest.max_zones of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidRequestBody",
			"{}",
			`{"status":"Invalid request.","error":"missing required Garden fields"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestInvalidStartTime",
			`{"name":"test-garden", "topic_prefix":"test-garden", "max_zones": 1,"light_schedule": {"duration":"1h","start_time":"NOT A TIME"}}`,
			`{"status":"Invalid request.","error":"invalid time format for light_schedule.start_time: NOT A TIME"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			gr := GardensResource{
				storageClient:  storageClient,
				influxdbClient: influxdbClient,
				config:         Config{},
				worker:         worker.NewWorker(storageClient, nil, nil, logrus.New()),
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
		})
	}
}

func TestGetAllGardens(t *testing.T) {
	gardens := []*pkg.Garden{createExampleGarden()}

	tests := []struct {
		name           string
		targetURL      string
		expectedRegexp string
		status         int
	}{
		{
			"SuccessfulEndDatedFalse",
			"/gardens",
			`{"gardens":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfulEndDatedTrue",
			"/gardens?end_dated=true",
			`{"gardens":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			for _, g := range gardens {
				err = storageClient.SaveGarden(g)
				assert.NoError(t, err)
			}

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			gr := GardensResource{
				storageClient:  storageClient,
				influxdbClient: influxdbClient,
				config:         Config{},
				worker:         worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}

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
		})
	}
}

func TestEndDateGarden(t *testing.T) {
	now := time.Now()
	endDatedGarden := createExampleGarden()
	endDatedGarden.EndDate = &now

	gardenWithZone := createExampleGarden()
	gardenWithZone.Zones[xid.New()] = &pkg.Zone{}

	tests := []struct {
		name           string
		garden         *pkg.Garden
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			createExampleGarden(),
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteGarden",
			endDatedGarden,
			"",
			http.StatusNoContent,
		},
		{
			"ErrorEndDatingGardenWithZones",
			gardenWithZone,
			`{"status":"Invalid request.","error":"unable to end-date Garden with active Zones"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupZonePlantGardenStorage(t)
			gr := GardensResource{
				storageClient: storageClient,
				config:        Config{},
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
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
		body           string
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			createExampleGarden(),
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyRemoveLightSchedule",
			createExampleGarden(),
			`{"name": "new name","light_schedule": {}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyAddLightSchedule",
			gardenWithoutLight,
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_plants":0,"num_zones":0,"plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"zones":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/zones"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"ErrorInvalidRequestBody",
			createExampleGarden(),
			"{}",
			`{"status":"Invalid request.","error":"missing required Garden fields"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorReducingMaxZones",
			gardenWithZone,
			`{"max_zones": 1}`,
			`{"status":"Invalid request.","error":"unable to set max_zones less than current num_zones=2"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			storageClient := setupZonePlantGardenStorage(t)
			gr := GardensResource{
				storageClient:  storageClient,
				influxdbClient: influxdbClient,
				config:         Config{},
				worker:         worker.NewWorker(storageClient, nil, nil, logrus.New()),
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
				worker: worker.NewWorker(setupZonePlantGardenStorage(t), nil, mqttClient, logrus.New()),
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
