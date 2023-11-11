package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
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

func setupZonePlantGardenStorage(t *testing.T) *storage.Client {
	t.Helper()

	garden := createExampleGarden()
	plant := createExamplePlant()
	zone := createExampleZone()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.SaveGarden(garden)
	assert.NoError(t, err)

	err = storageClient.SaveZone(garden.ID, zone)
	assert.NoError(t, err)

	err = storageClient.SavePlant(garden.ID, plant)
	assert.NoError(t, err)

	return storageClient
}

func TestGetPlant(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expected       string
	}{
		{
			"Successful",
			"/plants/c5cvhpcbcv45e8bp16dg",
			http.StatusOK,
			`{"name":"test plant","id":"c5cvhpcbcv45e8bp16dg","zone_id":"c5cvhpcbcv45e8bp16dg","created_at":"2021-10-03T11:24:52.891386-07:00","links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"zone","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"}]}`,
		},
		{
			"ErrorInvalidID",
			"/plants/not-an-xid",
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
		},
		{
			"NotFoundError",
			"/plants/chkodpg3lcj13q82mq40",
			http.StatusNotFound,
			`{"status":"Resource not found."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PlantsResource{
				GardensResource: &GardensResource{
					storageClient: setupZonePlantGardenStorage(t),
				},
			}

			r := httptest.NewRequest("GET", "/gardens/c5cvhpcbcv45e8bp16dg"+tt.path, nil).WithContext(context.Background())
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/gardens/{%s}/plants/{%s}", gardenPathParam, plantPathParam), func(r chi.Router) {
				r.Use(pr.gardenContextMiddleware)
				r.Use(pr.plantContextMiddleware)
				r.Get("/", get[*PlantResponse](getPlantFromContext))
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)

			actual := strings.TrimSpace(w.Body.String())
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestUpdatePlant(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
		status   int
	}{
		{
			"Successful",
			`{"name":"new name"}`,
			`{"name":"new name","id":"c5cvhpcbcv45e8bp16dg","zone_id":"c5cvhpcbcv45e8bp16dg","created_at":"2021-10-03T11:24:52.891386-07:00","links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"zone","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"}]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorNonexistentZone",
			`{"zone_id": "chkodpg3lcj13q82mq40"}`,
			`{"status":"Invalid request.","error":"unable to update Plant with non-existent zone: chkodpg3lcj13q82mq40"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PlantsResource{
				GardensResource: &GardensResource{
					storageClient: setupZonePlantGardenStorage(t),
				},
			}

			r := httptest.NewRequest("PATCH", "/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/gardens/{%s}/plants/{%s}", gardenPathParam, plantPathParam), func(r chi.Router) {
				r.Use(pr.gardenContextMiddleware)
				r.Use(pr.plantContextMiddleware)
				r.Patch("/", pr.updatePlant)
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.status, w.Code)

			actual := strings.TrimSpace(w.Body.String())
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestEndDatePlant(t *testing.T) {
	storageClient := setupZonePlantGardenStorage(t)

	tests := []struct {
		name           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"name":"test plant","id":"[0-9a-v]{20}","zone_id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"zone","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyDeletePlant",
			"",
			http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PlantsResource{
				GardensResource: &GardensResource{
					storageClient: storageClient,
				},
			}

			r := httptest.NewRequest("DELETE", "/gardens/c5cvhpcbcv45e8bp16dg/plants/c5cvhpcbcv45e8bp16dg", http.NoBody)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/gardens/{%s}/plants/{%s}", gardenPathParam, plantPathParam), func(r chi.Router) {
				r.Use(pr.gardenContextMiddleware)
				r.Use(pr.plantContextMiddleware)
				r.Delete("/", pr.endDatePlant)
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.code, w.Code)

			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
		})
	}
}

func TestGetAllPlants(t *testing.T) {
	storageClient := setupZonePlantGardenStorage(t)

	endDatedPlant := createExamplePlant()
	endDatedPlant.ID = xid.New()
	now := time.Now().Add(-1 * time.Minute)
	endDatedPlant.EndDate = &now

	err := storageClient.SavePlant(createExampleGarden().ID, endDatedPlant)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		targetURL string
		expected  []*pkg.Plant
	}{
		{
			"SuccessfulEndDatedFalse",
			"/plants",
			[]*pkg.Plant{createExamplePlant()},
		},
		{
			"SuccessfulEndDatedTrue",
			"/plants?end_dated=true",
			[]*pkg.Plant{createExamplePlant(), endDatedPlant},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PlantsResource{
				GardensResource: &GardensResource{
					storageClient: storageClient,
				},
			}

			r := httptest.NewRequest("GET", "/gardens/c5cvhpcbcv45e8bp16dg"+tt.targetURL, http.NoBody)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/gardens/{%s}/plants", gardenPathParam), func(r chi.Router) {
				r.Use(pr.gardenContextMiddleware)
				r.Get("/", pr.getAllPlants)
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, http.StatusOK, w.Code)

			plantJSON, _ := json.Marshal(pr.NewAllPlantsResponse(tt.expected, createExampleGarden()))
			// When the expected result contains more than one Plant, on some occasions it might be out of order
			var reversePlantJSON []byte
			if len(tt.expected) > 1 {
				reversePlantJSON, _ = json.Marshal(pr.NewAllPlantsResponse([]*pkg.Plant{tt.expected[1], tt.expected[0]}, &pkg.Garden{}))
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
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			`{"name":"test plant", "zone_id": "c5cvhpcbcv45e8bp16dg"}`,
			`{"name":"test plant","id":"[0-9a-v]{20}","zone_id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/plants/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"zone","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorBadRequestBadJSON",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorNonexistentZone",
			`{"name":"test plant", "zone_id": "chkodpg3lcj13q82mq40"}`,
			`{"status":"Invalid request.","error":"unable to create Plant with non-existent zone: chkodpg3lcj13q82mq40"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PlantsResource{
				GardensResource: &GardensResource{
					storageClient: setupZonePlantGardenStorage(t),
				},
			}

			r := httptest.NewRequest("POST", "/gardens/c5cvhpcbcv45e8bp16dg/plants", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Route(fmt.Sprintf("/gardens/{%s}/plants", gardenPathParam), func(r chi.Router) {
				r.Use(pr.gardenContextMiddleware)
				r.Post("/", pr.createPlant)
			})
			router.ServeHTTP(w, r)

			assert.Equal(t, tt.code, w.Code)

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
		})
	}
}
