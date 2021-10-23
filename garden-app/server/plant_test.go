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

func TestBackwardsCompatibleMiddleware(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
				config:        Config{},
			},
		}
		gardenOne := createExampleGarden()
		gardenTwo := createExampleGarden()
		gardenOne.Plants[xid.New()] = createExamplePlant()
		gardenTwo.Plants[xid.New()] = createExamplePlant()

		storageClient.On("GetGardens", false).Return([]*pkg.Garden{gardenOne, gardenTwo}, nil)

		testHandler := func(w http.ResponseWriter, r *http.Request) {
			g := r.Context().Value(gardenCtxKey).(*pkg.Garden)
			if g.Name != "All Gardens Combined" {
				t.Errorf("Unexpected name for combined Garden: %s", g.Name)
			}
			if len(g.Plants) != 2 {
				t.Errorf("Unexpected number of Plants in combined Garden: %d", len(g.Plants))
			}
			if g.ID != xid.NilID() {
				t.Errorf("Expected Garden with nil ID but got: %v", g.ID)
			}
			render.Status(r, http.StatusOK)
		}

		router := chi.NewRouter()
		router.Route("/plants", func(r chi.Router) {
			r.Use(pr.backwardCompatibleMiddleware)
			r.Get("/", testHandler)
		})

		r := httptest.NewRequest("GET", "/plants", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
				config:        Config{},
			},
		}
		gardenOne := createExampleGarden()
		gardenTwo := createExampleGarden()
		gardenOne.Plants[xid.New()] = createExamplePlant()
		gardenTwo.Plants[xid.New()] = createExamplePlant()

		storageClient.On("GetGardens", false).Return([]*pkg.Garden{}, errors.New("storage client error"))

		testHandler := func(w http.ResponseWriter, r *http.Request) {
			render.Status(r, http.StatusOK)
		}

		router := chi.NewRouter()
		router.Route("/plants", func(r chi.Router) {
			r.Use(pr.backwardCompatibleMiddleware)
			r.Get("/", testHandler)
		})

		r := httptest.NewRequest("GET", "/plants", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		// check HTTP response body
		expected := `{"status":"Server Error.","error":"storage client error"}`
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
}

func TestBackwardsCompatibleActionMiddleware(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
				config:        Config{},
			},
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.GardenID = garden.ID

		storageClient.On("GetGarden", garden.ID).Return(garden, nil)

		testHandler := func(w http.ResponseWriter, r *http.Request) {
			g := r.Context().Value(gardenCtxKey).(*pkg.Garden)
			if g != garden {
				t.Errorf("Unexpected Garden. Expected %v, got %v", garden, g)
			}
			render.Status(r, http.StatusOK)
		}

		router := chi.NewRouter()
		router.Route("/plants", func(r chi.Router) {
			r.Use(pr.backwardsCompatibleActionMiddleware)
			r.Get("/", testHandler)
		})

		ctx := context.WithValue(context.Background(), plantCtxKey, plant)
		r := httptest.NewRequest("GET", "/plants", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		pr := PlantsResource{
			GardensResource: GardensResource{
				storageClient: storageClient,
				config:        Config{},
			},
		}
		garden := createExampleGarden()
		plant := createExamplePlant()
		plant.GardenID = garden.ID

		storageClient.On("GetGarden", garden.ID).Return(nil, errors.New("storage client error"))

		testHandler := func(w http.ResponseWriter, r *http.Request) {
			render.Status(r, http.StatusOK)
		}

		router := chi.NewRouter()
		router.Route("/plants", func(r chi.Router) {
			r.Use(pr.backwardsCompatibleActionMiddleware)
			r.Get("/", testHandler)
		})

		ctx := context.WithValue(context.Background(), plantCtxKey, plant)
		r := httptest.NewRequest("GET", "/plants", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusInternalServerError)
		}

		// check HTTP response body
		expected := `{"status":"Server Error.","error":"storage client error"}`
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
}

func TestGetPlant(t *testing.T) {
	tests := []struct {
		name      string
		plant     func() *pkg.Plant
		setupMock func(*influxdb.MockClient)
	}{
		{
			"Successful",
			func() *pkg.Plant { return createExamplePlant() },
			func(*influxdb.MockClient) {},
		},
		{
			"SuccessfulWithMoisture",
			func() *pkg.Plant {
				plant := createExamplePlant()
				plant.WateringStrategy = pkg.WateringStrategy{MinimumMoisture: 1}
				return plant
			},
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), nil)
				influxdbClient.On("Close")
			},
		},
		{
			"ErrorGettingMoisture",
			func() *pkg.Plant {
				plant := createExamplePlant()
				plant.WateringStrategy = pkg.WateringStrategy{MinimumMoisture: 1}
				return plant
			},
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), errors.New("influxdb error"))
				influxdbClient.On("Close")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			pr := PlantsResource{
				GardensResource: GardensResource{
					influxdbClient: influxdbClient,
				},
				moistureCache: map[xid.ID]float64{},
				scheduler:     gocron.NewScheduler(time.Local),
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

			plantJSON, _ := json.Marshal(pr.NewPlantResponse(plant, 0))
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != string(plantJSON) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(plantJSON))
			}
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestPlantAction(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mqtt.MockClient, *storage.MockClient)
		body      string
		expected  string
		status    int
	}{
		{
			"BadRequest",
			func(mqttClient *mqtt.MockClient, storageClient *storage.MockClient) {},
			"bad request",
			`{"status":"Invalid request.","error":"invalid character 'b' looking for beginning of value"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulWaterAction",
			func(mqttClient *mqtt.MockClient, storageClient *storage.MockClient) {
				mqttClient.On("WateringTopic", "test garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
				storageClient.On("SavePlant", mock.Anything).Return(nil)
			},
			`{"water":{"duration":1000}}`,
			"null",
			http.StatusAccepted,
		},
		{
			"ExecuteErrorForWaterAction",
			func(mqttClient *mqtt.MockClient, storageClient *storage.MockClient) {
				mqttClient.On("WateringTopic", "test garden").Return("", errors.New("template error"))
			},
			`{"water":{"duration":1000}}`,
			`{"status":"Server Error.","error":"unable to fill MQTT topic template: template error"}`,
			http.StatusInternalServerError,
		},
		{
			"StorageClientErrorForWaterAction",
			func(mqttClient *mqtt.MockClient, storageClient *storage.MockClient) {
				mqttClient.On("WateringTopic", "test garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
				storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))
			},
			`{"water":{"duration":1000}}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			storageClient := new(storage.MockClient)
			tt.setupMock(mqttClient, storageClient)

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
			storageClient.AssertExpectations(t)
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
				storageClient.On("SavePlant", mock.Anything).Return(nil)
			},
			`{"name":"new name"}`,
			`{"name":"new name","id":"c5cvhpcbcv45e8bp16dg","garden_id":null,"plant_position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","watering_strategy":{"watering_amount":1000,"interval":"24h","start_time":"22:00:01-07:00"},"next_watering_time":"0001-01-01T00:00:00Z","links":[{"rel":"self","href":"/gardens/00000000000000000000/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/00000000000000000000"},{"rel":"actions","href":"/gardens/00000000000000000000/plants/c5cvhpcbcv45e8bp16dg/actions"},{"rel":"history","href":"/gardens/00000000000000000000/plants/c5cvhpcbcv45e8bp16dg/history"}]}`,
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
				storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))
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
				},
				moistureCache: map[xid.ID]float64{},
				scheduler:     gocron.NewScheduler(time.Local),
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
	tests := []struct {
		name           string
		setupMock      func(*storage.MockClient)
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything).Return(nil)
			},
			`{"name":"test plant","id":"[0-9a-v]{20}","garden_id":null,"plant_position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","watering_strategy":{"watering_amount":1000,"interval":"24h","start_time":"22:00:01-07:00"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"actions","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}/actions"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}/history"}\]}`,
			http.StatusOK,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))
			},
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
		moistureCache: map[xid.ID]float64{},
		scheduler:     gocron.NewScheduler(time.Local),
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

			plantJSON, _ := json.Marshal(pr.NewAllPlantsResponse(tt.expected))
			// When the expected result contains more than one Plant, on some occassions it might be out of order
			var reversePlantJSON []byte
			if len(tt.expected) > 1 {
				reversePlantJSON, _ = json.Marshal(pr.NewAllPlantsResponse([]*pkg.Plant{tt.expected[1], tt.expected[0]}))
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
				storageClient.On("SavePlant", mock.Anything).Return(nil)
			},
			`{"name":"test plant","watering_strategy":{"watering_amount":1000,"interval":"24h","start_time":"22:00:01-07:00"}}`,
			`{"name":"test plant","id":"[0-9a-v]{20}","garden_id":"[0-9a-v]{20}","plant_position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","watering_strategy":{"watering_amount":1000,"interval":"24h","start_time":"22:00:01-07:00"},"next_watering_time":"0001-01-01T00:00:00Z","links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"actions","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}/actions"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}/history"}\]}`,
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
			"ErrorBadRequestInvalidStartTime",
			func(storageClient *storage.MockClient) {},
			`{"name":"test plant","watering_strategy":{"watering_amount":1000,"interval":"24h","start_time":"NOT A TIME"}}`,
			`{"status":"Invalid request.","error":"parsing time \\"NOT A TIME\\" as \\"15:04:05-07:00\\": cannot parse \\"NOT A TIME\\" as \\"15\\""}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything).Return(errors.New("storage error"))
			},
			`{"name":"test plant","watering_strategy":{"watering_amount":1000,"interval":"24h","start_time":"22:00:01-07:00"}}`,
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
				},
				moistureCache: map[xid.ID]float64{},
				scheduler:     gocron.NewScheduler(time.Local),
			}
			garden := createExampleGarden()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
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
