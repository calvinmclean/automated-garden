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
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func createExamplePlant() *pkg.Plant {
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Plant{
		Name:      "test plant",
		ID:        id,
		ZoneID:    id,
		CreatedAt: &time,
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
			`{"name":"test plant","id":"c5cvhpcbcv45e8bp16dg","zone_id":"c5cvhpcbcv45e8bp16dg","created_at":"2021-10-03T11:24:52.891386-07:00","links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"zone","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			pr := PlantsResource{
				GardensResource: GardensResource{
					influxdbClient: influxdbClient,
					worker:         worker.NewWorker(nil, influxdbClient, nil, nil, logrus.New()),
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

func TestUpdatePlant(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*storage.MockClient)
		body      string
		garden    *pkg.Garden
		expected  string
		status    int
	}{
		{
			"Successful",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(nil)
			},
			`{"name":"new name"}`,
			createExampleGardenWithZone(),
			`{"name":"new name","id":"c5cvhpcbcv45e8bp16dg","zone_id":"c5cvhpcbcv45e8bp16dg","created_at":"2021-10-03T11:24:52.891386-07:00","links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"zone","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"}]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			func(storageClient *storage.MockClient) {},
			"this is not json",
			createExampleGardenWithZone(),
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			`{"name":"new name"}`,
			createExampleGardenWithZone(),
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
		{
			"ErrorNonexistentZone",
			func(storageClient *storage.MockClient) {},
			`{"zone_id": "c5cvhpcbcv45e8bp16dg"}`,
			createExampleGarden(),
			`{"status":"Invalid request.","error":"unable to update Plant with non-existent zone: c5cvhpcbcv45e8bp16dg"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			pr := PlantsResource{
				GardensResource: GardensResource{
					storageClient: storageClient,
					worker:        worker.NewWorker(nil, nil, nil, nil, logrus.New()),
				},
			}
			plant := createExamplePlant()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, tt.garden)
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
			`{"name":"test plant","id":"[0-9a-v]{20}","zone_id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"zone","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"}\]}`,
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
					worker:        worker.NewWorker(nil, nil, nil, nil, logrus.New()),
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
			worker: worker.NewWorker(nil, nil, nil, nil, logrus.New()),
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
			createExampleGardenWithZone(),
			`{"name":"test plant", "zone_id": "c5cvhpcbcv45e8bp16dg"}`,
			`{"name":"test plant","id":"[0-9a-v]{20}","zone_id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"zone","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorBadRequestBadJSON",
			func(storageClient *storage.MockClient) {},
			createExampleGardenWithZone(),
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"StorageClientError",
			func(storageClient *storage.MockClient) {
				storageClient.On("SavePlant", mock.Anything, mock.Anything).Return(errors.New("storage error"))
			},
			createExampleGardenWithZone(),
			`{"name":"test plant", "zone_id": "c5cvhpcbcv45e8bp16dg"}`,
			`{"status":"Server Error.","error":"storage error"}`,
			http.StatusInternalServerError,
		},
		{
			"ErrorNonexistentZone",
			func(storageClient *storage.MockClient) {},
			createExampleGarden(),
			`{"name":"test plant", "zone_id": "c5cvhpcbcv45e8bp16dg"}`,
			`{"status":"Invalid request.","error":"unable to create Plant with non-existent zone: c5cvhpcbcv45e8bp16dg"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			tt.setupMock(storageClient)

			pr := PlantsResource{
				GardensResource: GardensResource{
					storageClient: storageClient,
					worker:        worker.NewWorker(storageClient, nil, nil, nil, logrus.New()),
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
