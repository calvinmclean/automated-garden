package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	plantBasePath   = "/plants"
	plantPathParam  = "plantID"
	plantCtxKey     = contextKey("plant")
	plantIDLogField = "plant_id"
)

// PlantsResource encapsulates the structs and dependencies necessary for the "/plants" API
// to function, including storage, scheduling, and caching
type PlantsResource struct {
	GardensResource
}

// NewPlantsResource creates a new PlantsResource
func NewPlantsResource(gr GardensResource) (PlantsResource, error) {
	return PlantsResource{
		GardensResource: gr,
	}, nil
}

// routes creates all of the routing that is prefixed by "/plant" for interacting with Plant resources
func (pr PlantsResource) routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", pr.createPlant)
	r.Get("/", pr.getAllPlants)
	r.Route(fmt.Sprintf("/{%s}", plantPathParam), func(r chi.Router) {
		r.Use(pr.plantContextMiddleware)

		r.Get("/", pr.getPlant)
		r.Patch("/", pr.updatePlant)
		r.Delete("/", pr.endDatePlant)
	})
	return r
}

// plantContextMiddleware middleware is used to load a Plant object from the URL
// parameters passed through as the request. In case the Plant could not be found,
// we stop here and return a 404.
func (pr PlantsResource) plantContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plantIDString := chi.URLParam(r, plantPathParam)
		logger := contextLogger(r.Context()).WithField(plantIDLogField, plantIDString)

		garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
		plantID, err := xid.FromString(plantIDString)
		if err != nil {
			logger.WithError(err).Error("unable to parse PlantID")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		plant := garden.Plants[plantID]
		if plant == nil {
			logger.Info("plant not found")
			render.Render(w, r, ErrNotFoundResponse)
			return
		}
		logger.Debugf("found Plant: %+v", plant)

		ctx := context.WithValue(r.Context(), plantCtxKey, plant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getPlant simply returns the Plant requested by the provided ID
func (pr PlantsResource) getPlant(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to get Plant")

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
	logger.Debugf("responding with Plant: %+v", plant)

	plantResponse := pr.NewPlantResponse(r.Context(), garden, plant)
	if err := render.Render(w, r, plantResponse); err != nil {
		logger.WithError(err).Error("unable to render PlantResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// updatePlant will change any specified fields of the Plant and save it
func (pr PlantsResource) updatePlant(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to update Plant")

	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
	request := &UpdatePlantRequest{}
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Read the request body into existing Plant to overwrite fields
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid update Plant request")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Debugf("update request: %+v", request)

	// Don't allow changing ZoneID to non-existent Zone
	if _, ok := garden.Zones[plant.ZoneID]; !ok {
		err := fmt.Errorf("unable to update Plant with non-existent zone: %v", plant.ZoneID)
		logger.WithError(err).Error("unable to update Plant")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	plant.Patch(request.Plant)
	logger.Debugf("plant after patching: %+v", plant)

	// Save the Plant
	logger.Debug("saving updated Plant")
	if err := pr.storageClient.SavePlant(garden.ID, plant); err != nil {
		logger.WithError(err).Error("unable to save Plant")
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		logger.WithError(err).Error("unable to render PlantResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it
func (pr PlantsResource) endDatePlant(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to end-date Plant")

	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	now := time.Now()

	// Permanently delete the Plant if it is already end-dated
	if plant.EndDated() {
		logger.Info("permanently deleting Plant")

		if err := pr.storageClient.DeletePlant(garden.ID, plant.ID); err != nil {
			logger.WithError(err).Error("unable to delete Plant")
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of Plant and save
	plant.EndDate = &now
	logger.Debug("saving end-dated Plant")
	if err := pr.storageClient.SavePlant(garden.ID, plant); err != nil {
		logger.WithError(err).Error("unable to save end-dated Plant")
		render.Render(w, r, InternalServerError(err))
		return
	}
	logger.Debug("saved end-dated Plant")

	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		logger.WithError(err).Error("unable to render PlantResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// getAllPlants will return a list of all Plants
func (pr PlantsResource) getAllPlants(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"

	logger := contextLogger(r.Context()).WithField("include_end_dated", getEndDated)
	logger.Info("received request to get all Plants")

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	plants := []*pkg.Plant{}
	for _, p := range garden.Plants {
		if getEndDated || (p.EndDate == nil || p.EndDate.After(time.Now())) {
			plants = append(plants, p)
		}
	}
	logger.Debugf("found %d Plants", len(plants))

	if err := render.Render(w, r, pr.NewAllPlantsResponse(r.Context(), plants, garden)); err != nil {
		logger.WithError(err).Error("unable to render AllPlantsResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// createPlant will create a new Plant resource
func (pr PlantsResource) createPlant(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to create new Plant")

	request := &PlantRequest{}
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid request to create Plant")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	plant := request.Plant
	logger.Debugf("request to create Plant: %+v", plant)

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Don't allow creating Plant with nonexistent Zone
	if _, ok := garden.Zones[plant.ZoneID]; !ok {
		err := fmt.Errorf("unable to create Plant with non-existent zone: %v", plant.ZoneID)
		logger.WithError(err).Error("unable to create Plant")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Assign values to fields that may not be set in the request
	plant.ID = xid.New()
	if plant.CreatedAt == nil {
		now := time.Now()
		plant.CreatedAt = &now
	}
	logger.Debugf("new plant ID: %v", plant.ID)

	// Save the Plant
	logger.Debug("saving Plant")
	if err := pr.storageClient.SavePlant(garden.ID, plant); err != nil {
		logger.WithError(err).Error("unable to save Plant")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		logger.WithError(err).Error("unable to render PlantResponse")
		render.Render(w, r, ErrRender(err))
	}
}
