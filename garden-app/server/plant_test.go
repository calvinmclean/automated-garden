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
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func createExamplePlant() *pkg.Plant {
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	pp := uint(0)
	return &pkg.Plant{
		Name:          "test plant",
		ID:            id,
		CreatedAt:     &time,
		PlantPosition: &pp,
		WaterSchedule: &pkg.WaterSchedule{
			Duration:  "1000ms",
			Interval:  "24h",
			StartTime: &time,
		},
	}
}

func TestPlantContextMiddleware(t *testing.T) {
	pr := PlantsResource{
		GardensResource: GardensResource{},
	}
	plant := createExamplePlant()
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		p := r.Context().Value(plantCtxKey).(*pkg.Plant)
		if plant != p {
			t.Errorf("Unexpected Plant saved in request context. Expected %v but got %v", plant, p)
		}
		render.Status(r, http.StatusOK)
	}

	router := chi.NewRouter()
	router.Route(fmt.Sprintf("/plant/{%s}", plantPathParam), func(r chi.Router) {
		r.Use(pr.plantContextMiddleware)
		r.Get("/", testHandler)
	})

	tests := []struct {
		name     string
		plant    *pkg.Plant
		path     string
		code     int
		expected string
	}{
		{
			"Successful",
			plant,
			"/plant/c5cvhpcbcv45e8bp16dg",
			http.StatusOK,
			"",
		},
		{
			"ErrorInvalidID",
			plant,
			"/plant/not-an-xid",
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
		},
		{
			"NotFoundError",
			nil,
			"/plant/c5cvhpcbcv45e8bp16dg",
			http.StatusNotFound,
			`{"status":"Resource not found."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			garden := createExampleGarden()
			garden.Plants[plant.ID] = tt.plant
			ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
			r := httptest.NewRequest("GET", tt.path, nil).WithContext(ctx)
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

func TestPlantRestrictEndDatedMiddleware(t *testing.T) {
	pr := PlantsResource{
		GardensResource: GardensResource{},
	}
	plant := createExamplePlant()
	endDatedPlant := createExamplePlant()
	endDate := time.Now().Add(-1 * time.Minute)
	endDatedPlant.EndDate = &endDate
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
	}

	router := chi.NewRouter()
	router.Route("/plant", func(r chi.Router) {
		r.Use(pr.restrictEndDatedMiddleware)
		r.Get("/", testHandler)
	})

	tests := []struct {
		name     string
		plant    *pkg.Plant
		code     int
		expected string
	}{
		{
			"PlantNotEndDated",
			plant,
			http.StatusOK,
			"",
		},
		{
			"PlantEndDated",
			endDatedPlant,
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"resource not available for end-dated Plant"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), plantCtxKey, tt.plant)
			r := httptest.NewRequest("GET", "/plant", nil).WithContext(ctx)
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

func TestGetPlant(t *testing.T) {
	tests := []struct {
		name      string
		plant     func() *pkg.Plant
		setupMock func(*influxdb.MockClient)
		expected  string
	}{
		{
			"Successful",
			func() *pkg.Plant { return createExamplePlant() },
			func(*influxdb.MockClient) {},
			`{"name":"test plant","id":"c5cvhpcbcv45e8bp16dg","plant_position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/history"}]}`,
		},
		{
			"SuccessfulWithMoisture",
			func() *pkg.Plant {
				plant := createExamplePlant()
				plant.WaterSchedule = &pkg.WaterSchedule{MinimumMoisture: 1}
				return plant
			},
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), nil)
				influxdbClient.On("Close")
			},
			`{"name":"test plant","id":"c5cvhpcbcv45e8bp16dg","plant_position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule":{"duration":"","interval":"","minimum_moisture":1,"start_time":null},"moisture":2,"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/history"}]}`,
		},
		{
			"ErrorGettingMoisture",
			func() *pkg.Plant {
				plant := createExamplePlant()
				plant.WaterSchedule = &pkg.WaterSchedule{MinimumMoisture: 1}
				return plant
			},
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), errors.New("influxdb error"))
				influxdbClient.On("Close")
			},
			`{"name":"test plant","id":"c5cvhpcbcv45e8bp16dg","plant_position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule":{"duration":"","interval":"","minimum_moisture":1,"start_time":null},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/history"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			pr := PlantsResource{
				GardensResource: GardensResource{
					influxdbClient: influxdbClient,
					scheduler:      action.NewScheduler(nil, influxdbClient, nil, logrus.StandardLogger()),
				},
			}
			garden := createExampleGarden()

			plant := tt.plant()
			tt.setupMock(influxdbClient)

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
			r := httptest.NewRequest("GET", "/plant", nil).WithContext(plantCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(pr.getPlant)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
			}

			// plantJSON, _ := json.Marshal(pr.NewPlantResponse(plant, 0))
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestPlantAction(t *testing.T) {
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
			"SuccessfulWaterAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("WateringTopic", "test-garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			`{"water":{"duration":1000}}`,
			"null",
			http.StatusAccepted,
		},
		{
			"ExecuteErrorForWaterAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("WateringTopic", "test-garden").Return("", errors.New("template error"))
			},
			`{"water":{"duration":1000}}`,
			`{"status":"Server Error.","error":"unable to fill MQTT topic template: template error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			tt.setupMock(mqttClient)

			pr := PlantsResource{
				GardensResource: GardensResource{
					mqttClient: mqttClient,
				},
			}
			garden := createExampleGarden()
			plant := createExamplePlant()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
			r := httptest.NewRequest("POST", "/plant", strings.NewReader(tt.body)).WithContext(plantCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(pr.plantAction)

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

func TestUpdatePlant(t *testing.T) {
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
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(nil)
			},
			`{"name":"new name"}`,
			`{"name":"new name","id":"c5cvhpcbcv45e8bp16dg","plant_position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"},"next_watering_time":"0001-01-01T00:00:00Z","links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg/history"}]}`,
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
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			`{"name":"new name"}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			pr := PlantsResource{
				GardensResource: GardensResource{
					storageClient: storageClient,
					scheduler:     action.NewScheduler(nil, nil, nil, logrus.StandardLogger()),
				},
			}
			garden := createExampleGarden()
			plant := createExamplePlant()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
			r := httptest.NewRequest("PATCH", "/plant", strings.NewReader(tt.body)).WithContext(plantCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(pr.updatePlant)

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

func TestEndDatePlant(t *testing.T) {
	now := time.Now()
	endDatedPlant := createExamplePlant()
	endDatedPlant.EndDate = &now

	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		plant          *pkg.Plant
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(nil)
			},
			createExamplePlant(),
			`{"name":"test plant","id":"[0-9a-v]{20}","plant_position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule":{"duration":"1000ms","interval":"24h","start_time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyDeletePlant",
			func(storageClient *storage.MockClient) {
				storageClient.On("DeletePlant", mock.Anything, mock.Anything).Return(nil)
			},
			endDatedPlant,
			"",
			http.StatusNoContent,
		},
		{
			"DeletePlantError",
			func(storageClient *storage.MockClient) {
				storageClient.On("DeletePlant", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			endDatedPlant,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			createExamplePlant(),
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			pr := PlantsResource{
				GardensResource: GardensResource{
					storageClient: storageClient,
					scheduler:     action.NewScheduler(nil, nil, nil, logrus.StandardLogger()),
				},
			}

			garden := createExampleGarden()
			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			plantCtx := context.WithValue(gardenCtx, plantCtxKey, tt.plant)
			r := httptest.NewRequest("DELETE", "/plant", nil).WithContext(plantCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(pr.endDatePlant)

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

func TestGetAllPlants(t *testing.T) {
	pr := PlantsResource{
		GardensResource: GardensResource{
			scheduler: action.NewScheduler(nil, nil, nil, logrus.StandardLogger()),
		},
	}
	garden := createExampleGarden()
	plant := createExamplePlant()
	endDatedPlant := createExamplePlant()
	endDatedPlant.ID = xid.New()
	now := time.Now()
	endDatedPlant.EndDate = &now
	garden.Plants[plant.ID] = plant
	garden.Plants[endDatedPlant.ID] = endDatedPlant

	gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)

	tests := []struct {
		name      string
		targetURL string
		expected  []*pkg.Plant
	}{
		{
			"SuccessfulEndDatedFalse",
			"/plant",
			[]*pkg.Plant{plant},
		},
		{
			"SuccessfulEndDatedTrue",
			"/plant?end_dated=true",
			[]*pkg.Plant{plant, endDatedPlant},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.targetURL, nil).WithContext(gardenCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(pr.getAllPlants)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
			}

			plantJSON, _ := json.Marshal(pr.NewAllPlantsResponse(context.Background(), tt.expected, garden))
			// When the expected result contains more than one Plant, on some occassions it might be out of order
			var reversePlantJSON []byte
			if len(tt.expected) > 1 {
				reversePlantJSON, _ = json.Marshal(pr.NewAllPlantsResponse(context.Background(), []*pkg.Plant{tt.expected[1], tt.expected[0]}, &pkg.Garden{}))
			}
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != string(plantJSON) && actual != string(reversePlantJSON) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(plantJSON))
			}
		})
	}
}

func TestCreatePlant(t *testing.T) {
	gardenWithPlant := createExampleGarden()
	gardenWithPlant.Plants[xid.New()] = &pkg.Plant{}
	gardenWithPlant.Plants[xid.New()] = &pkg.Plant{}
	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		garden         *pkg.Garden
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(nil)
			},
			createExampleGarden(),
			`{"name":"test plant","plant_position":0,"water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			`{"name":"test plant","id":"[0-9a-v]{20}","plant_position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"},"next_watering_time":"0001-01-01T00:00:00Z","links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}/history"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorNegativePlantPosition",
			func(storageClient *storage.MockClient) {},
			createExampleGarden(),
			`{"name":"test plant","plant_position":-1,"water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -1 into Go struct field PlantRequest.plant_position of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorMaxPlantsExceeded",
			func(storageClient *storage.MockClient) {},
			gardenWithPlant,
			`{"name":"test plant","plant_position":0,"water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			`{"status":"Invalid request.","error":"adding a Plant would exceed Garden's max_plants=2"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidPlantPosition",
			func(storageClient *storage.MockClient) {},
			createExampleGarden(),
			`{"name":"test plant","plant_position":2,"water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			`{"status":"Invalid request.","error":"plant_position invalid for Garden with max_plants=2"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestBadJSON",
			func(storageClient *storage.MockClient) {},
			createExampleGarden(),
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			createExampleGarden(),
			`{"name":"test plant","plant_position":0,"water_schedule":{"duration":"1000ms","interval":"24h","start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			pr := PlantsResource{
				GardensResource: GardensResource{
					storageClient: storageClient,
					scheduler:     action.NewScheduler(storageClient, nil, nil, logrus.StandardLogger()),
				},
			}

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, tt.garden)
			r := httptest.NewRequest("POST", "/plant", strings.NewReader(tt.body)).WithContext(gardenCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(pr.createPlant)

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

func TestWateringHistory(t *testing.T) {
	recordTime, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	tests := []struct {
		name        string
		setupMock   func(*influxdb.MockClient)
		queryParams string
		expected    string
		status      int
	}{
		{
			"BadRequestInvalidLimit",
			func(*influxdb.MockClient) {},
			"?limit=-1",
			`{"status":"Invalid request.","error":"strconv.ParseUint: parsing \"-1\": invalid syntax"}`,
			http.StatusBadRequest,
		},
		{
			"BadRequestInvalidTimeRange",
			func(*influxdb.MockClient) {},
			"?range=notTime",
			`{"status":"Invalid request.","error":"time: invalid duration \"notTime\""}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulWaterHistoryEmpty",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWateringHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(0)).Return([]map[string]interface{}{}, nil)
				influxdbClient.On("Close")
			},
			"",
			`{"history":null,"count":0,"average":"0s","total":"0s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistory",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWateringHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(0)).
					Return([]map[string]interface{}{{"Duration": 3000, "RecordTime": recordTime}}, nil)
				influxdbClient.On("Close")
			},
			"",
			`{"history":[{"duration":"3s","record_time":"2021-10-03T11:24:52.891386-07:00"}],"count":1,"average":"3s","total":"3s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistoryWithLimit",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWateringHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(1)).
					Return([]map[string]interface{}{
						{"Duration": 3000, "RecordTime": recordTime},
					}, nil)
				influxdbClient.On("Close")
			},
			"?limit=1",
			`{"history":[{"duration":"3s","record_time":"2021-10-03T11:24:52.891386-07:00"}],"count":1,"average":"3s","total":"3s"}`,
			http.StatusOK,
		},
		{
			"InfluxDBClientError",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWateringHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(0)).
					Return([]map[string]interface{}{}, errors.New("influxdb error"))
				influxdbClient.On("Close")
			},
			"",
			`{"status":"Server Error.","error":"influxdb error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(influxdbClient)

			pr := PlantsResource{
				GardensResource: GardensResource{
					influxdbClient: influxdbClient,
				},
			}
			garden := createExampleGarden()
			plant := createExamplePlant()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
			r := httptest.NewRequest("GET", fmt.Sprintf("/history%s", tt.queryParams), nil).WithContext(plantCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(pr.wateringHistory)

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
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestGetNextWateringTime(t *testing.T) {
	tests := []struct {
		name         string
		expectedDiff time.Duration
	}{
		{"ZeroSkip", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PlantsResource{
				GardensResource: GardensResource{
					scheduler: action.NewScheduler(nil, nil, nil, logrus.StandardLogger()),
				},
			}
			g := createExampleGarden()
			p := createExamplePlant()

			pr.scheduler.ScheduleWateringAction(g, p)
			pr.scheduler.StartAsync()
			defer pr.scheduler.Stop()

			nextWateringTime := pr.scheduler.GetNextWateringTime(p)
			nextWateringTimeWithSkip := pr.scheduler.GetNextWateringTime(p)

			diff := nextWateringTimeWithSkip.Sub(*nextWateringTime)
			if diff != tt.expectedDiff {
				t.Errorf("Unexpected difference between next watering times: expected=%v, actual=%v", tt.expectedDiff, diff)
			}
		})
	}
}
