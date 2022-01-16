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
	plantBasePath  = "/plants"
	plantPathParam = "plantID"
	plantCtxKey    = contextKey("plant")
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
		garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

		plantID, err := xid.FromString(chi.URLParam(r, plantPathParam))
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		plant := garden.Plants[plantID]
		if plant == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		// t := context.WithValue(r.Context(), gardenCtxKey, garden)
		ctx := context.WithValue(r.Context(), plantCtxKey, plant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getPlant simply returns the Plant requested by the provided ID
func (pr PlantsResource) getPlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	plantResponse := pr.NewPlantResponse(r.Context(), garden, plant)
	if err := render.Render(w, r, plantResponse); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// updatePlant will change any specified fields of the Plant and save it
func (pr PlantsResource) updatePlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
	request := &UpdatePlantRequest{}
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Read the request body into existing plant to overwrite fields
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Don't allow changing ZoneID to non-existent Zone
	if _, ok := garden.Zones[plant.ZoneID]; !ok {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("unable to update Plant with non-existent zone: %v", plant.ZoneID)))
		return
	}

	plant.Patch(request.Plant)

	// Save the Plant
	if err := pr.storageClient.SavePlant(garden.ID, plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it
func (pr PlantsResource) endDatePlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	now := time.Now()

	// Permanently delete the Plant if it is already end-dated
	if plant.EndDated() {
		if err := pr.storageClient.DeletePlant(garden.ID, plant.ID); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of Plant and save
	plant.EndDate = &now
	if err := pr.storageClient.SavePlant(garden.ID, plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Remove scheduled watering Job
	if err := pr.scheduler.RemoveJobsByID(plant.ID); err != nil {
		logger.Errorf("Unable to remove watering Job for Plant %s: %v", plant.ID.String(), err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllPlants will return a list of all Plants
func (pr PlantsResource) getAllPlants(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	plants := []*pkg.Plant{}
	for _, p := range garden.Plants {
		if getEndDated || (p.EndDate == nil || p.EndDate.After(time.Now())) {
			plants = append(plants, p)
		}
	}
	if err := render.Render(w, r, pr.NewAllPlantsResponse(r.Context(), plants, garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// createPlant will create a new Plant resource
func (pr PlantsResource) createPlant(w http.ResponseWriter, r *http.Request) {
	request := &PlantRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	plant := request.Plant
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Don't allow creating Plant with nonexistent Zone
	if _, ok := garden.Zones[plant.ZoneID]; !ok {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("unable to create Plant with non-existent zone: %v", plant.ZoneID)))
		return
	}

	// Assign values to fields that may not be set in the request
	plant.ID = xid.New()
	if plant.CreatedAt == nil {
		now := time.Now()
		plant.CreatedAt = &now
	}

	// Save the Plant
	if err := pr.storageClient.SavePlant(garden.ID, plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}
