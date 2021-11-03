package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
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
	moistureCache map[xid.ID]float64
}

// NewPlantsResource creates a new PlantsResource
func NewPlantsResource(gr GardensResource) (PlantsResource, error) {
	pr := PlantsResource{
		GardensResource: gr,
		moistureCache:   map[xid.ID]float64{},
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
			if err = pr.addWateringSchedule(g, p); err != nil {
				err = fmt.Errorf("unable to add watering Job for Plant %v: %v", p.ID, err)
				return pr, err
			}
		}
	}

	pr.scheduler.StartAsync()
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

	// Save the Plant in case anything was changed (watering a plant might change the skip_count field)
	// TODO: consider giving the action the ability to use the storage client
	if err := pr.storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
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

	moisture, cached := pr.moistureCache[plant.ID]
	plantResponse := pr.NewPlantResponse(plant, moisture)
	if err := render.Render(w, r, plantResponse); err != nil {
		render.Render(w, r, ErrRender(err))
	}

	// If moisture was not already cached (and plant has moisture sensor), get it and cache it
	// Otherwise, clear cache
	if !cached && plant.WateringStrategy.MinimumMoisture > 0 {
		// I was doing this with a goroutine, but that made the call untestable. I don't think there was any benefit to
		// using the goroutine because the result is already rendered
		pr.getAndCacheMoisture(garden, plant)
	} else {
		delete(pr.moistureCache, plant.ID)
	}
}

// updatePlant will change any specified fields of the Plant and save it
func (pr PlantsResource) updatePlant(w http.ResponseWriter, r *http.Request) {
	request := &PlantRequest{r.Context().Value(plantCtxKey).(*pkg.Plant)}
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Read the request body into existing plant to overwrite fields
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	plant := request.Plant

	// Update the watering schedule for the Plant
	if err := pr.resetWateringSchedule(garden, plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Save the Plant
	if err := pr.storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it
func (pr PlantsResource) endDatePlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)

	// Set end date of Plant and save
	now := time.Now()
	plant.EndDate = &now
	if err := pr.storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Remove scheduled watering Job
	if err := pr.removeWateringSchedule(plant); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		logger.Errorf("Unable to remove watering Job for Plant %s: %v", plant.ID.String(), err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(plant, 0)); err != nil {
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
	if err := render.Render(w, r, pr.NewAllPlantsResponse(plants)); err != nil {
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

	// Check that water time is valid
	_, err := time.Parse(pkg.WaterTimeFormat, plant.WateringStrategy.StartTime)
	if err != nil {
		logger.Errorf("Invalid time format for WateringStrategy.StartTime: %s", plant.WateringStrategy.StartTime)
		render.Render(w, r, ErrInvalidRequest(err))
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
	if err := pr.addWateringSchedule(garden, plant); err != nil {
		logger.Errorf("Unable to add watering Job for Plant %v: %v", plant.ID, err)
	}

	// Save the Plant
	if err := pr.storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, pr.NewPlantResponse(plant, 0)); err != nil {
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

	history, err := pr.getWateringHistory(plant, garden, timeRange, int(limit))
	if err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}
	if err := render.Render(w, r, NewPlantWateringHistoryResponse(history)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (pr PlantsResource) getAndCacheMoisture(g *pkg.Garden, p *pkg.Plant) {
	defer pr.influxdbClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
	defer cancel()

	moisture, err := pr.influxdbClient.GetMoisture(ctx, p.PlantPosition, g.Name)
	if err != nil {
		logger.Errorf("unable to get moisture of Plant %v: %v", p.ID, err)
	}
	pr.moistureCache[p.ID] = moisture
}

// getWateringHistory gets previous WateringEvents for this Plant from InfluxDB
func (pr PlantsResource) getWateringHistory(plant *pkg.Plant, garden *pkg.Garden, timeRange time.Duration, limit int) (result []pkg.WateringHistory, err error) {
	defer pr.influxdbClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
	defer cancel()

	history, err := pr.influxdbClient.GetWateringHistory(ctx, plant.PlantPosition, garden.Name, timeRange)
	if err != nil {
		return
	}

	for _, h := range history {
		if len(result) >= limit {
			break
		}
		result = append(result, pkg.WateringHistory{
			WateringAmount: h["WateringAmount"].(int),
			RecordTime:     h["RecordTime"].(time.Time),
		})
	}
	return
}
