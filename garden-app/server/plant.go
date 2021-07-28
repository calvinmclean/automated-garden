package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/calvinmclean/automated-garden/garden-app/api/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/api/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/api/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
	"github.com/rs/xid"
)

// PlantsResource encapsulates the structs and dependencies necessary for the "/plants" API
// to function, including storage, scheduling, and caching
type PlantsResource struct {
	storageClient storage.Client
	mqttClient    *mqtt.Client
	scheduler     *gocron.Scheduler
	moistureCache map[xid.ID]float64
	config        Config
}

// NewPlantsResource creates a new PlantsResource
func NewPlantsResource(config Config) (pr PlantsResource, err error) {
	pr = PlantsResource{
		moistureCache: map[xid.ID]float64{},
		config:        config,
	}

	pr.storageClient, err = storage.NewStorageClient(config.StorageConfig)
	if err != nil {
		err = fmt.Errorf("unable to initialize storage client: %v", err)
		return
	}

	pr.mqttClient, err = mqtt.NewMQTTClient(pr.config.MQTTConfig)
	if err != nil {
		err = fmt.Errorf("unable to initialize MQTT client: %v", err)
		return
	}

	pr.scheduler = gocron.NewScheduler(time.Local)

	// Initialize watering Jobs for each Plant from the storage client
	for _, p := range pr.storageClient.GetPlants(false) {
		if err = pr.addWateringSchedule(p); err != nil {
			err = fmt.Errorf("unable to add watering Job for Plant %s: %v", p.ID.String(), err)
			return
		}
	}

	pr.scheduler.StartAsync()
	return
}

// routes creates all of the routing that is prefixed by "/plant" for interacting with Plant resources
func (pr PlantsResource) routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", pr.createPlant)
	r.Get("/", pr.getAllPlants)
	r.Route("/{plantID}", func(r chi.Router) {
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
		id, err := xid.FromString(chi.URLParam(r, "plantID"))
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		plant, err := pr.storageClient.GetPlant(id)
		if err != nil {
			render.Render(w, r, ServerError(err))
			return
		}
		if plant == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		ctx := context.WithValue(r.Context(), "plant", plant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// plantAction reads an AggregateAction request and uses it to execute one of the actions
// that is available to run against a Plant. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (pr PlantsResource) plantAction(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)

	action := &AggregateActionRequest{}
	if err := render.Bind(r, action); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Received request to perform action on Plant %s\n", plant.ID)
	if err := action.Execute(plant, pr.mqttClient, pr.config.InfluxDBConfig); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	// Save the Plant in case anything was changed (watering a plant might change the skip_count field)
	// TODO: consider giving the action the ability to use the storage client
	if err := pr.storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, ServerError(err))
		return
	}

	render.Status(r, http.StatusAccepted)
	render.DefaultResponder(w, r, nil)
}

// getPlant simply returns the Plant requested by the provided ID
func (pr PlantsResource) getPlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)
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
	request := &PlantRequest{r.Context().Value("plant").(*api.Plant)}

	// Read the request body into existing plant to overwrite fields
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	plant := request.Plant

	// Update the watering schedule for the Plant
	if err := pr.resetWateringSchedule(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	// Save the Plant
	if err := pr.storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it
func (pr PlantsResource) endDatePlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)

	// Set end date of Plant and save
	now := time.Now()
	plant.EndDate = &now
	if err := pr.storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	// Remove scheduled watering Job
	if err := pr.removeWateringSchedule(plant); err != nil {
		logger.Errorf("Unable to remove watering Job for Plant %s: %v", plant.ID.String(), err)
		render.Render(w, r, ServerError(err))
		return
	}

	if err := render.Render(w, r, pr.NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllPlants will return a list of all Plants
func (pr PlantsResource) getAllPlants(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	plants := pr.storageClient.GetPlants(getEndDated)
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
	_, err := time.Parse(api.WaterTimeFormat, plant.WateringStrategy.StartTime)
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
	if err := pr.storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, ServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, pr.NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func (pr PlantsResource) getAndCacheMoisture(p *api.Plant) {
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
