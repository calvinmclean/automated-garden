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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
	"github.com/rs/xid"
	"github.com/stretchr/testify/mock"
)

func createExamplePlant() *pkg.Plant {
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Plant{
		Name:      "test plant",
		ID:        id,
		CreatedAt: &time,
		WateringStrategy: pkg.WateringStrategy{
			WateringAmount: 1000,
			Interval:       "24h",
			StartTime:      "22:00:01-07:00",
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
		name  string
		plant *pkg.Plant
		path  string
		code  int
		body  string
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
			if actual != tt.body {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.body)
			}
		})
	}
}

func TestGetPlant(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		pr := PlantsResource{
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

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

		plantJSON, _ := json.Marshal(pr.NewPlantResponse(plant, 0))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(plantJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(plantJSON))
		}
	})
	t.Run("SuccessfulWithMoisture", func(t *testing.T) {
		influxdbClient := new(influxdb.MockClient)
		pr := PlantsResource{
			influxdbClient: influxdbClient,
			moistureCache:  map[xid.ID]float64{},
			scheduler:      gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.WateringStrategy = pkg.WateringStrategy{MinimumMoisture: 1}

		influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), nil)
		influxdbClient.On("Close")

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

		plantJSON, _ := json.Marshal(pr.NewPlantResponse(plant, 0))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(plantJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(plantJSON))
		}
		influxdbClient.AssertExpectations(t)
	})
	t.Run("ErrorGettingMoisture", func(t *testing.T) {
		influxdbClient := new(influxdb.MockClient)
		pr := PlantsResource{
			influxdbClient: influxdbClient,
			moistureCache:  map[xid.ID]float64{},
			scheduler:      gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.WateringStrategy = pkg.WateringStrategy{MinimumMoisture: 1}

		influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), errors.New("influxdb error"))
		influxdbClient.On("Close")

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

		plantJSON, _ := json.Marshal(pr.NewPlantResponse(plant, 0))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(plantJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(plantJSON))
		}
		influxdbClient.AssertExpectations(t)
	})
}

func TestPlantAction(t *testing.T) {
	t.Run("BadRequest", func(t *testing.T) {
		pr := PlantsResource{
			moistureCache: map[xid.ID]float64{},
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader(`bad request`)).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.plantAction)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusBadRequest)
		}

		expected := `{"status":"Invalid request.","error":"invalid character 'b' looking for beginning of value"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
	})
	t.Run("SuccessfulWaterAction", func(t *testing.T) {
		mqttClient := new(mqtt.MockClient)
		mqttClient.On("WateringTopic", "test garden").Return("garden/action/water", nil)
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(nil)

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			mqttClient:    mqttClient,
			moistureCache: map[xid.ID]float64{},
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader(`{"water":{"duration":1000}}`)).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.plantAction)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusAccepted {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusAccepted)
		}

		expected := "null"
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
		mqttClient.AssertExpectations(t)
	})
	t.Run("ExecuteErrorForWaterAction", func(t *testing.T) {
		mqttClient := new(mqtt.MockClient)
		mqttClient.On("WateringTopic", "test garden").Return("", errors.New("template error"))

		pr := PlantsResource{
			mqttClient:    mqttClient,
			moistureCache: map[xid.ID]float64{},
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader(`{"water":{"duration":1000}}`)).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.plantAction)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		expected := `{"status":"Server Error.","error":"unable to fill MQTT topic template: template error"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		mqttClient.AssertExpectations(t)
	})
	t.Run("StorageClientErrorForWaterAction", func(t *testing.T) {
		mqttClient := new(mqtt.MockClient)
		mqttClient.On("WateringTopic", "test garden").Return("garden/action/water", nil)
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			mqttClient:    mqttClient,
			moistureCache: map[xid.ID]float64{},
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader(`{"water":{"duration":1000}}`)).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.plantAction)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		expected := `{"status":"Server Error.","error":"storage error"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
		mqttClient.AssertExpectations(t)
	})
}

func TestUpdatePlant(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(nil)

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("PATCH", "/plant", strings.NewReader(`{"name":"new name"}`)).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.updatePlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		plant.Name = "new name"
		expected, _ := json.Marshal(pr.NewPlantResponse(plant, 0))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(expected) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(expected))
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("BadRequest", func(t *testing.T) {
		storageClient := new(storage.MockClient)

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("PATCH", "/plant", strings.NewReader("this is not json")).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.updatePlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusBadRequest)
		}

		expected := `{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("PATCH", "/plant", strings.NewReader(`{"name":"new name"}`)).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.updatePlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		expected := `{"status":"Server Error.","error":"storage error"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
}

func TestEndDatePlant(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(nil)

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		plant := createExamplePlant()

		plantCtx := context.WithValue(context.Background(), plantCtxKey, plant)
		r := httptest.NewRequest("DELETE", "/plant", nil).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.endDatePlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		expected, _ := json.Marshal(pr.NewPlantResponse(plant, 0))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(expected) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(expected))
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		plant := createExamplePlant()

		plantCtx := context.WithValue(context.Background(), plantCtxKey, plant)
		r := httptest.NewRequest("DELETE", "/plant", nil).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.endDatePlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		expected := `{"status":"Server Error.","error":"storage error"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
}

func TestGetAllPlants(t *testing.T) {
	t.Run("SuccessfulEndDatedFalse", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plants := []*pkg.Plant{createExamplePlant()}

		storageClient.On("GetPlants", garden.ID, false).Return(plants, nil)

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		r := httptest.NewRequest("GET", "/plant", nil).WithContext(gardenCtx)
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.getAllPlants)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		plantJSON, _ := json.Marshal(pr.NewAllPlantsResponse(plants))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(plantJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(plantJSON))
		}
	})
	t.Run("SuccessfulEndDatedTrue", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plants := []*pkg.Plant{createExamplePlant()}

		storageClient.On("GetPlants", garden.ID, true).Return(plants, nil)

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		r := httptest.NewRequest("GET", "/plant?end_dated=true", nil).WithContext(gardenCtx)
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.getAllPlants)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		plantJSON, _ := json.Marshal(pr.NewAllPlantsResponse(plants))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(plantJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(plantJSON))
		}
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plants := []*pkg.Plant{createExamplePlant()}

		storageClient.On("GetPlants", garden.ID, false).Return(plants, errors.New("storage error"))

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		r := httptest.NewRequest("GET", "/plant", nil).WithContext(gardenCtx)
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.getAllPlants)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		expected := `{"status":"Server Error.","error":"storage error"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
	})
}

func TestCreatePlant(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(nil)

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.CreatedAt = nil

		plantJSON, _ := json.Marshal(plant)

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader(string(plantJSON))).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.createPlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusCreated {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusCreated)
		}

		// check HTTP response body
		matcher := regexp.MustCompile(`{"name":"test plant","id":"[0-9a-v]{20}","garden_id":"[0-9a-v]{20}","plant_position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","watering_strategy":{"watering_amount":1000,"interval":"24h","start_time":"22:00:01-07:00"},"next_watering_time":"0001-01-01T00:00:00Z","links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"actions","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}/actions"}\]}`)
		actual := strings.TrimSpace(w.Body.String())
		if !matcher.MatchString(actual) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
		}

		storageClient.AssertExpectations(t)
	})
	t.Run("ErrorBadRequestBadJSON", func(t *testing.T) {
		storageClient := new(storage.MockClient)

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.CreatedAt = nil

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader("this is not json")).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.createPlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusBadRequest)
		}

		expected := `{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("ErrorBadRequestInvalidStartTime", func(t *testing.T) {
		storageClient := new(storage.MockClient)

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.CreatedAt = nil
		plant.WateringStrategy.StartTime = "NOT A TIME"

		plantJSON, _ := json.Marshal(plant)

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader(string(plantJSON))).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.createPlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusBadRequest)
		}

		expected := `{"status":"Invalid request.","error":"parsing time \"NOT A TIME\" as \"15:04:05-07:00\": cannot parse \"NOT A TIME\" as \"15\""}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))

		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
			},
			moistureCache: map[xid.ID]float64{},
			scheduler:     gocron.NewScheduler(time.Local),
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.CreatedAt = nil

		plantJSON, _ := json.Marshal(plant)

		gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
		plantCtx := context.WithValue(gardenCtx, plantCtxKey, plant)
		r := httptest.NewRequest("POST", "/plant", strings.NewReader(string(plantJSON))).WithContext(plantCtx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(pr.createPlant)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		expected := `{"status":"Server Error.","error":"storage error"}`
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
}
