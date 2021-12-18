package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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
	pr := PlantsResource{
		GardensResource: gr,
	}

	// Initialize watering Jobs for each Plant from the storage client
	allGardens, err := pr.storageClient.GetGardens(false)
	if err != nil {
		return pr, err
	}
	for _, g := range allGardens {
		allPlants, err := pr.storageClient.GetPlants(g.ID, false)
		if err != nil {
			return pr, err
		}
		for _, p := range allPlants {
			if err = pr.scheduleWateringAction(g, p); err != nil {
				err = fmt.Errorf("unable to add watering Job for Plant %v: %v", p.ID, err)
				return pr, err
			}
		}
	}

	return pr, err
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

		// Add new middleware to restrict certain paths to non-end-dated Plants
		r.Route("/", func(r chi.Router) {
			r.Use(pr.restrictEndDatedMiddleware)

			r.Post("/action", pr.plantAction)
			r.Get("/history", pr.wateringHistory)
		})
	})
	return r
}

// backwardCompatibleRoutes is the same as regular routes, but uses a different middleware allowing for compatibility
// with the API before adding Gardens. This does not allow for creating Plants since that requires a Garden
func (pr PlantsResource) backwardCompatibleRoutes() chi.Router {
	r := chi.NewRouter()
	r.Use(pr.backwardCompatibleMiddleware)
	r.Get("/", pr.getAllPlants)
	r.Route(fmt.Sprintf("/{%s}", plantPathParam), func(r chi.Router) {
		r.Use(pr.plantContextMiddleware)

		r.Get("/", pr.getPlant)
		r.Patch("/", pr.updatePlant)
		r.Delete("/", pr.endDatePlant)

		// Add new middleware to restrict certain paths to non-end-dated Plants
		r.Route("/", func(r chi.Router) {
			r.Use(pr.restrictEndDatedMiddleware)

			r.With(pr.backwardsCompatibleActionMiddleware).Post("/action", pr.plantAction)
			r.With(pr.backwardsCompatibleActionMiddleware).Get("/history", pr.wateringHistory)
		})

	})
	return r
}

// restrictEndDatedMiddleware will return a 400 response if the requested Plant is end-dated
func (pr PlantsResource) restrictEndDatedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plant := r.Context().Value(plantCtxKey).(*pkg.Plant)

		if plant.EndDated() {
			render.Render(w, r, ErrInvalidRequest(fmt.Errorf("resource not available for end-dated Plant")))
			return
		}
		next.ServeHTTP(w, r)
	})
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

// backwardCompatibleMiddleware allows REST APIs that are compatible with the V1 which did not include gardens resources.
// Instead of relying on gardenID in the route, this will combine all Gardens into a new one containing all Plants
func (pr PlantsResource) backwardCompatibleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allPlants := map[xid.ID]*pkg.Plant{}

		gardens, err := pr.storageClient.GetGardens(false)
		if err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}

		for _, g := range gardens {
			for id, p := range g.Plants {
				allPlants[id] = p
			}
		}

		garden := &pkg.Garden{
			Name:   "All Gardens Combined",
			Plants: allPlants,
		}

		ctx := context.WithValue(r.Context(), gardenCtxKey, garden)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (pr PlantsResource) backwardsCompatibleActionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
		garden, err := pr.storageClient.GetGarden(plant.GardenID)
		if err != nil {
			logger.Error("Error getting Garden for backwards-compatible action: ", err)
			render.Render(w, r, InternalServerError(err))
			return
		}
		ctx := context.WithValue(r.Context(), gardenCtxKey, garden)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// plantAction reads a PlantAction request and uses it to execute one of the actions
// that is available to run against a Plant. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (pr PlantsResource) plantAction(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)

	action := &PlantActionRequest{}
	if err := render.Bind(r, action); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Received request to perform action on Plant %s\n", plant.ID)
	if err := action.Execute(garden, plant, pr.mqttClient, pr.influxdbClient); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusAccepted)
	render.DefaultResponder(w, r, nil)
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
	plant.Patch(request.Plant)

	// Save the Plant
	if err := pr.storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Update the watering schedule for the Plant if it was changed or EndDate is removed
	if request.Plant.WaterSchedule != nil || request.Plant.EndDate == nil {
		if err := pr.resetWateringSchedule(garden, plant); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it
func (pr PlantsResource) endDatePlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)
	now := time.Now()

	// Permanently delete the Plant if it is already end-dated
	if plant.EndDated() {
		if err := pr.storageClient.DeletePlant(plant.GardenID, plant.ID); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of Plant and save
	plant.EndDate = &now
	if err := pr.storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Remove scheduled watering Job
	if err := pr.removeJobsByID(plant.ID); err != nil {
		logger.Errorf("Unable to remove watering Job for Plant %s: %v", plant.ID.String(), err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), nil, plant)); err != nil {
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

	// Validate that adding a Plant does not exceed Garden.MaxPlants
	if garden.NumPlants()+1 > *garden.MaxPlants {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("adding a Plant would exceed Garden's max_plants=%d", *garden.MaxPlants)))
		return
	}
	// Validate that PlantPosition works for a Garden with MaxPlants (remember PlantPosition is zero-indexed)
	if *plant.PlantPosition >= *garden.MaxPlants {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("plant_position invalid for Garden with max_plants=%d", *garden.MaxPlants)))
		return
	}

	// Assign values to fields that may not be set in the request
	plant.ID = xid.New()
	if plant.CreatedAt == nil {
		now := time.Now()
		plant.CreatedAt = &now
	}
	plant.GardenID = garden.ID

	// Start watering schedule
	if err := pr.scheduleWateringAction(garden, plant); err != nil {
		logger.Errorf("Unable to add watering Job for Plant %v: %v", plant.ID, err)
	}

	// Save the Plant
	if err := pr.storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, pr.NewPlantResponse(r.Context(), garden, plant)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// wateringHistory responds with the Plant's recent watering events read from InfluxDB
func (pr PlantsResource) wateringHistory(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)

	// Read query parameters and set default values
	timeRangeString := r.URL.Query().Get("range")
	if len(timeRangeString) == 0 {
		timeRangeString = "72h"
	}
	limitString := r.URL.Query().Get("limit")
	if len(limitString) == 0 {
		limitString = "5"
	}

	// Parse query parameter strings into correct types
	timeRange, err := time.ParseDuration(timeRangeString)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	limit, err := strconv.ParseUint(limitString, 0, 64)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	history, err := pr.getWateringHistory(r.Context(), plant, garden, timeRange, limit)
	if err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}
	if err := render.Render(w, r, NewPlantWateringHistoryResponse(history)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (pr PlantsResource) getMoisture(ctx context.Context, g *pkg.Garden, p *pkg.Plant) (float64, error) {
	defer pr.influxdbClient.Close()

	moisture, err := pr.influxdbClient.GetMoisture(ctx, *p.PlantPosition, g.TopicPrefix)
	if err != nil {
		return 0, err
	}
	return moisture, err
}

// getWateringHistory gets previous WateringEvents for this Plant from InfluxDB
func (pr PlantsResource) getWateringHistory(ctx context.Context, plant *pkg.Plant, garden *pkg.Garden, timeRange time.Duration, limit uint64) (result []pkg.WateringHistory, err error) {
	defer pr.influxdbClient.Close()

	history, err := pr.influxdbClient.GetWateringHistory(ctx, *plant.PlantPosition, garden.TopicPrefix, timeRange, limit)
	if err != nil {
		return
	}

	for _, h := range history {
		result = append(result, pkg.WateringHistory{
			Duration:   (time.Duration(h["Duration"].(int)) * time.Millisecond).String(),
			RecordTime: h["RecordTime"].(time.Time),
		})
	}
	return
}
