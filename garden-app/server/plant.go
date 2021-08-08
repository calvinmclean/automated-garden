package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
	"github.com/rs/xid"
)

const (
	plantCtxKey = contextKey("plant")
)

// PlantsResource encapsulates the structs and dependencies necessary for the "/plants" API
// to function, including storage, scheduling, and caching
type PlantsResource struct {
	GardenResource
	mqttClient    *mqtt.Client
	moistureCache map[xid.ID]float64
	scheduler     *gocron.Scheduler
}

const (
	plantBasePath  = "/plants"
	plantPathParam = "plantID"
	plantCtxKey    = "plant"
)

var garden = "garden"

// NewPlantsResource creates a new PlantsResource
func NewPlantsResource(gr GardenResource) (PlantsResource, error) {
	pr := PlantsResource{
		GardenResource: gr,
		moistureCache:  map[xid.ID]float64{},
		scheduler:      gocron.NewScheduler(time.Local),
	}

	// Initialize MQTT Client
	var err error
	pr.mqttClient, err = mqtt.NewMQTTClient(gr.config.MQTTConfig)
	if err != nil {
		err = fmt.Errorf("unable to initialize MQTT client: %v", err)
		return pr, err
	}

	// Initialize watering Jobs for each Plant from the storage client
	allGardens, err := pr.storageClient.GetGardens(false)
	if err != nil {
		return pr, err
	}
	for _, g := range allGardens {
		allPlants, err := pr.storageClient.GetPlants(g.Name, false)
		if err != nil {
			return pr, err
		}
		for _, p := range allPlants {
			if err = pr.addWateringSchedule(p); err != nil {
				err = fmt.Errorf("unable to add watering Job for Plant %s: %v", p.ID.String(), err)
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

		r.Post("/action", pr.plantAction)
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
		// Convert ID string to xid
		id, err := xid.FromString(chi.URLParam(r, plantPathParam))
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		gardenName := chi.URLParam(r, gardenPathParam)

		plant, err := pr.storageClient.GetPlant(gardenName, id)
		if err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
		if plant == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		ctx := context.WithValue(r.Context(), plantCtxKey, plant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// plantAction reads an AggregateAction request and uses it to execute one of the actions
// that is available to run against a Plant. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (pr PlantsResource) plantAction(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value(plantCtxKey).(*pkg.Plant)

	action := &AggregateActionRequest{}
	if err := render.Bind(r, action); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Received request to perform action on Plant %s\n", plant.ID)
	if err := action.Execute(plant, pr.mqttClient, pr.config.InfluxDBConfig); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Save the Plant in case anything was changed (watering a plant might change the skip_count field)
	// TODO: consider giving the action the ability to use the storage client
	if err := pr.storageClient.SavePlant(garden, plant); err != nil {
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
	moisture, cached := pr.moistureCache[plant.ID]
	plantResponse := pr.NewPlantResponse(plant, moisture)
	if err := render.Render(w, r, plantResponse); err != nil {
		render.Render(w, r, ErrRender(err))
	}

	// If moisture was not already cached (and plant has moisture sensor), asynchronously get it and cache it
	// Otherwise, clear cache
	if !cached && plant.WateringStrategy.MinimumMoisture > 0 {
		go pr.getAndCacheMoisture(plant)
	} else {
		delete(pr.moistureCache, plant.ID)
	}
}

// updatePlant will change any specified fields of the Plant and save it
func (pr PlantsResource) updatePlant(w http.ResponseWriter, r *http.Request) {
	request := &PlantRequest{r.Context().Value(plantCtxKey).(*pkg.Plant)}

	// Read the request body into existing plant to overwrite fields
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	plant := request.Plant

	// Update the watering schedule for the Plant
	if err := pr.resetWateringSchedule(plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Save the Plant
	if err := pr.storageClient.SavePlant(garden, plant); err != nil {
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
	if err := pr.storageClient.SavePlant(garden, plant); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Remove scheduled watering Job
	if err := pr.removeWateringSchedule(plant); err != nil {
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
	plants, err := pr.storageClient.GetPlants(garden, getEndDated)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
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

	// Check that water time is valid
	_, err := time.Parse(pkg.WaterTimeFormat, plant.WateringStrategy.StartTime)
	if err != nil {
		logger.Errorf("Invalid time format for WateringStrategy.StartTime: %s", plant.WateringStrategy.StartTime)
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Assign new unique ID and CreatedAt to plant
	plant.ID = xid.New()
	if plant.CreatedAt == nil {
		now := time.Now()
		plant.CreatedAt = &now
	}

	// Start watering schedule
	if err := pr.addWateringSchedule(plant); err != nil {
		logger.Errorf("Unable to add watering Job for Plant %s: %v", plant.ID.String(), err)
	}

	// Save the Plant
	if err := pr.storageClient.SavePlant(garden, plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, pr.NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (pr PlantsResource) getAndCacheMoisture(p *pkg.Plant) {
	influxdbClient := influxdb.NewClient(pr.config.InfluxDBConfig)
	defer influxdbClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
	defer cancel()

	moisture, err := influxdbClient.GetMoisture(ctx, p.PlantPosition, p.Garden)
	if err != nil {
		logger.Errorf("error getting Plant's moisture data: %v", err)
	}

	if err != nil {
		logger.Errorf("unable to get moisture of Plant %v: %v", p.ID, err)
	}
	pr.moistureCache[p.ID] = moisture
}
