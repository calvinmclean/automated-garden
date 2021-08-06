package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
)

// PlantsResource encapsulates the structs and dependencies necessary for the "/plants" API
// to function, including storage, scheduling, and caching
type GardenResource struct {
	storageClient  storage.Client
	mqttClient     *mqtt.Client
	scheduler      *gocron.Scheduler
	config         Config
	plantsResource PlantsResource
}

// NewGardenResource creates a new GardenResource
func NewGardenResource(config Config) (gr GardenResource, err error) {
	gr = GardenResource{
		config: config,
	}

	gr.storageClient, err = storage.NewStorageClient(config.StorageConfig)
	if err != nil {
		err = fmt.Errorf("unable to initialize storage client: %v", err)
		return
	}

	gr.mqttClient, err = mqtt.NewMQTTClient(gr.config.MQTTConfig)
	if err != nil {
		err = fmt.Errorf("unable to initialize MQTT client: %v", err)
		return
	}

	gr.scheduler = gocron.NewScheduler(time.Local)

	gr.plantsResource, err = NewPlantsResource(config, gr.storageClient, gr.mqttClient, gr.scheduler)
	if err != nil {
		logger.Error("Error initializing '/plants' endpoint: ", err)
		os.Exit(1)
	}
	// r.Mount("/plants", plantsResource.routes())

	// Initialize watering Jobs for each Plant from the storage client
	for _, p := range gr.storageClient.GetPlants(garden, false) {
		if err = gr.plantsResource.addWateringSchedule(p); err != nil {
			err = fmt.Errorf("unable to add watering Job for Plant %s: %v", p.ID.String(), err)
			return
		}
	}

	gr.scheduler.StartAsync()
	return
}

// routes creates all of the routing that is prefixed by "/plant" for interacting with Plant resources
func (gr GardenResource) routes() chi.Router {
	r := chi.NewRouter()
	// r.Post("/", gr.createGarden)
	// r.Get("/", gr.getAllGardens)
	r.Route("/{gardenName}", func(r chi.Router) {
		r.Use(gr.gardenContextMiddleware)

		// r.Get("/", gr.getGarden)
		// r.Patch("/", gr.updateGarden)
		// r.Delete("/", gr.endDateGarden)

		r.Mount("/plants", gr.plantsResource.routes())
	})
	return r
}

// gardenContextMiddleware middleware is used to load a Garden object from the URL
// parameters passed through as the request. In case the Garden could not be found,
// we stop here and return a 404.
func (gr GardenResource) gardenContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Convert ID string to xid
		gardenName := chi.URLParam(r, "gardenName")

		garden, err := gr.storageClient.GetGarden(gardenName)
		if err != nil {
			render.Render(w, r, ServerError(err))
			return
		}
		if garden == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		ctx := context.WithValue(r.Context(), "garden", garden)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
