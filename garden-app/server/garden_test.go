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
		Name:      "test garden",
		ID:        id,
		Plants:    map[xid.ID]*pkg.Plant{},
		CreatedAt: &time,
	}
}

func TestGardenContextMiddleware(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()
		storageClient.On("GetGarden", garden.ID).Return(garden, nil)

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

		r := httptest.NewRequest("GET", "/garden/c5cvhpcbcv45e8bp16dg", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}
	})
	t.Run("ErrorInvalidID", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()

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

		r := httptest.NewRequest("GET", "/garden/not-an-xid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusBadRequest)
		}
		// check HTTP response body
		expected := `{"status":"Invalid request.","error":"xid: invalid ID"}`
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()
		storageClient.On("GetGarden", garden.ID).Return(nil, errors.New("storage client error"))

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

		r := httptest.NewRequest("GET", "/garden/c5cvhpcbcv45e8bp16dg", nil)
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
	t.Run("NotFoundError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()
		storageClient.On("GetGarden", garden.ID).Return(nil, nil)

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

		r := httptest.NewRequest("GET", "/garden/c5cvhpcbcv45e8bp16dg", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusNotFound {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusNotFound)
		}

		// check HTTP response body
		expected := `{"status":"Resource not found."}`
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
}

func TestCreateGarden(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SaveGarden", mock.Anything).Return(nil)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}

		r := httptest.NewRequest("POST", "/garden", strings.NewReader(`{"name": "test garden"}`))
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.createGarden)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusCreated {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusCreated)
		}

		// check HTTP response body
		matcher := regexp.MustCompile(`{"name":"test garden","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}`)
		actual := strings.TrimSpace(w.Body.String())
		if !matcher.MatchString(actual) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("ErrorInvalidRequestBody", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}

		r := httptest.NewRequest("POST", "/garden", strings.NewReader("{}"))
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.createGarden)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusBadRequest)
		}

		// check HTTP response body
		expected := `{"status":"Invalid request.","error":"missing required Garden fields"}`
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage client error"))
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}

		r := httptest.NewRequest("POST", "/garden", strings.NewReader(`{"name": "test garden"}`))
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.createGarden)

		h.ServeHTTP(w, r)

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

func TestGetAllGardens(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		gardens := []*pkg.Garden{createExampleGarden()}
		storageClient.On("GetGardens", false).Return(gardens, nil)

		r := httptest.NewRequest("GET", "/garden", nil)
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.getAllGardens)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		gardenJSON, _ := json.Marshal(gr.NewAllGardensResponse(gardens))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(gardenJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(gardenJSON))
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("SuccessfulWithGetEndDated", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		gardens := []*pkg.Garden{createExampleGarden()}
		storageClient.On("GetGardens", true).Return(gardens, nil)

		r := httptest.NewRequest("GET", "/garden?end_dated=true", nil)
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.getAllGardens)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		gardenJSON, _ := json.Marshal(gr.NewAllGardensResponse(gardens))
		// check HTTP response body
		actual := strings.TrimSpace(w.Body.String())
		if actual != string(gardenJSON) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(gardenJSON))
		}
		storageClient.AssertExpectations(t)
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		storageClient.On("GetGardens", false).Return([]*pkg.Garden{}, errors.New("storage client error"))

		r := httptest.NewRequest("GET", "/garden", nil)
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.getAllGardens)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusUnprocessableEntity)
		}

		// check HTTP response body
		expected := `{"status":"Error rendering response.","error":"storage client error"}`
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
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
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SaveGarden", mock.Anything).Return(nil)
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
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		// check HTTP response body
		matcher := regexp.MustCompile(`{"name":"test garden","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}`)
		actual := strings.TrimSpace(w.Body.String())
		if !matcher.MatchString(actual) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
		}
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage client error"))
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

func TestUpdateGarden(t *testing.T) {
	t.Run("Successful", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SaveGarden", mock.Anything).Return(nil)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()

		ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
		r := httptest.NewRequest("PATCH", "/garden", strings.NewReader(`{"name": "new name"}`)).WithContext(ctx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.updateGarden)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusOK {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
		}

		// check HTTP response body
		matcher := regexp.MustCompile(`{"name":"new name","id":"[0-9a-v]{20}","created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","plants":{"rel":"collection","href":"/gardens/[0-9a-v]{20}/plants"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"plants","href":"/gardens/[0-9a-v]{20}/plants"}\]}`)
		actual := strings.TrimSpace(w.Body.String())
		if !matcher.MatchString(actual) {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
		}
	})
	t.Run("StorageClientError", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		storageClient.On("SaveGarden", mock.Anything).Return(errors.New("storage client error"))
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()

		ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
		r := httptest.NewRequest("PATCH", "/garden", strings.NewReader(`{"name": "new name"}`)).WithContext(ctx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.updateGarden)

		h.ServeHTTP(w, r)

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
	t.Run("ErrorInvalidRequestBody", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			config:        Config{},
		}
		garden := createExampleGarden()

		ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
		r := httptest.NewRequest("PATCH", "/garden", strings.NewReader("{}")).WithContext(ctx)
		r.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h := http.HandlerFunc(gr.updateGarden)

		h.ServeHTTP(w, r)

		// check HTTP response status code
		if w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusBadRequest)
		}

		// check HTTP response body
		expected := `{"status":"Invalid request.","error":"missing required Garden fields"}`
		actual := strings.TrimSpace(w.Body.String())
		if actual != expected {
			t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, expected)
		}
		storageClient.AssertExpectations(t)
	})
}
