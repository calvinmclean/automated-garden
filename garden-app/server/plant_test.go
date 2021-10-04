package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
	"github.com/rs/xid"
)

func createExamplePlant() *pkg.Plant {
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Plant{
		Name:      "test plant",
		ID:        id,
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
		storageClient.AssertExpectations(t)
	})
	t.Run("SuccessfulWithMoisture", func(t *testing.T) {
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
		plant.WateringStrategy = pkg.WateringStrategy{MinimumMoisture: 1}

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
		storageClient.AssertExpectations(t)
	})
}
