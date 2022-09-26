package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

const (
	gardenBasePath   = "/gardens"
	gardenPathParam  = "gardenID"
	gardenCtxKey     = contextKey("garden")
	gardenIDLogField = "garden_id"
)

// GardensResource encapsulates the structs and dependencies necessary for the "/gardens" API
// to function, including storage and configurating
type GardensResource struct {
	storageClient  storage.Client
	influxdbClient influxdb.Client
	worker         *worker.Worker
	config         Config
}

// NewGardenResource creates a new GardenResource
func NewGardenResource(config Config, logger *logrus.Logger) (GardensResource, error) {
	gr := GardensResource{
		config: config,
	}

	// Initialize MQTT Client
	logger.WithFields(logrus.Fields{
		"client_id": config.MQTTConfig.ClientID,
		"broker":    config.MQTTConfig.Broker,
		"port":      config.MQTTConfig.Port,
	}).Info("initializing MQTT client")
	mqttClient, err := mqtt.NewClient(gr.config.MQTTConfig, nil)
	if err != nil {
		return gr, fmt.Errorf("unable to initialize MQTT client: %v", err)
	}

	// Initialize Storage Client
	logger.WithField("type", config.StorageConfig.Type).Info("initializing storage client")
	gr.storageClient, err = storage.NewClient(config.StorageConfig)
	if err != nil {
		return gr, fmt.Errorf("unable to initialize storage client: %v", err)
	}

	// Initialize InfluxDB Client
	logger.WithFields(logrus.Fields{
		"address": config.InfluxDBConfig.Address,
		"org":     config.InfluxDBConfig.Org,
		"bucket":  config.InfluxDBConfig.Bucket,
	}).Info("initializing InfluxDB client")
	gr.influxdbClient = influxdb.NewClient(gr.config.InfluxDBConfig)

	// Initialize weather Client
	weatherClient, err := weather.NewClient(gr.config.WeatherConfig)
	if err != nil {
		return gr, fmt.Errorf("unable to initialize weather Client: %v", err)
	}

	// Initialize Scheduler
	logger.Info("initializing scheduler")
	gr.worker = worker.NewWorker(gr.storageClient, gr.influxdbClient, mqttClient, weatherClient, logger)
	gr.worker.StartAsync()

	// Initialize light schedules for all Gardens
	logger.Info("setting up LightSchedules for Gardens")
	allGardens, err := gr.storageClient.GetGardens(false)
	if err != nil {
		return gr, err
	}
	for _, g := range allGardens {
		gardenLogger := logger.WithField(gardenIDLogField, g.ID)
		gardenLogger.Debugf("scheduling LightAction for: %+v", g.LightSchedule)
		if g.LightSchedule != nil {
			if err = gr.worker.ScheduleLightActions(g); err != nil {
				return gr, fmt.Errorf("unable to schedule LightAction for Garden %v: %v", g.ID, err)
			}
		}
	}

	return gr, nil
}

// routes creates all of the routing that is prefixed by "/plant" for interacting with Plant resources
func (gr GardensResource) routes(pr PlantsResource, zr ZonesResource) chi.Router {
	r := chi.NewRouter()
	r.Post("/", gr.createGarden)
	r.Get("/", gr.getAllGardens)
	r.Route(fmt.Sprintf("/{%s}", gardenPathParam), func(r chi.Router) {
		r.Use(gr.gardenContextMiddleware)

		r.Get("/", gr.getGarden)
		r.Patch("/", gr.updateGarden)
		r.Delete("/", gr.endDateGarden)

		// Add new middleware to restrict certain paths to non-end-dated Gardens
		r.Route("/", func(r chi.Router) {
			r.Use(gr.restrictEndDatedMiddleware)

			r.Post("/action", gr.gardenAction)
			r.Get("/health", gr.getGardenHealth)
			r.Mount(plantBasePath, pr.routes())
			r.Mount(zoneBasePath, zr.routes())
		})
	})
	return r
}

// restrictEndDatedMiddleware will return a 400 response if the requested Garden is end-dated
func (gr GardensResource) restrictEndDatedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
		logger := contextLogger(r.Context())

		if garden.EndDated() {
			err := fmt.Errorf("resource not available for end-dated Garden")
			logger.WithError(err).Error("unable to complete request")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// gardenContextMiddleware middleware is used to load a Garden object from the URL
// parameters passed through as the request. In case the Garden could not be found,
// we stop here and return a 404.
func (gr GardensResource) gardenContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		gardenIDString := chi.URLParam(r, gardenPathParam)
		logger := contextLogger(ctx).WithField(gardenIDLogField, gardenIDString)
		gardenID, err := xid.FromString(gardenIDString)
		if err != nil {
			logger.WithError(err).Error("unable to parse Garden ID")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		garden, err := gr.storageClient.GetGarden(gardenID)
		if err != nil {
			logger.WithError(err).Error("unable to get Garden")
			render.Render(w, r, InternalServerError(err))
			return
		}
		if garden == nil {
			logger.Info("garden not found")
			render.Render(w, r, ErrNotFoundResponse)
			return
		}
		logger.Debugf("found Garden: %+v", garden)

		ctx = context.WithValue(ctx, gardenCtxKey, garden)
		ctx = context.WithValue(ctx, loggerCtxKey, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (gr GardensResource) createGarden(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to create new Garden")

	request := &GardenRequest{}
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid request to create Garden")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	garden := request.Garden
	logger.Debugf("request to create Garden: %+v", garden)

	// Assign new unique ID and CreatedAt to garden
	garden.ID = xid.New()
	if garden.CreatedAt == nil {
		now := time.Now()
		garden.CreatedAt = &now
	}
	logger.Debugf("new garden ID: %v", garden.ID)

	// Start light schedule (if applicable)
	if garden.LightSchedule != nil {
		if err := gr.worker.ScheduleLightActions(garden); err != nil {
			logger.WithError(err).Error("unable to schedule LightAction")
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	// Save the Garden
	logger.Debug("saving Garden")
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		logger.WithError(err).Error("unable to save Garden")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, gr.NewGardenResponse(r.Context(), garden)); err != nil {
		logger.WithError(err).Error("unable to render GardenResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// getAllGardens will return a list of all Gardens
func (gr GardensResource) getAllGardens(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"

	logger := contextLogger(r.Context()).WithField("include_end_dated", getEndDated)
	logger.Info("received request to get all Gardens")

	gardens, err := gr.storageClient.GetGardens(getEndDated)
	if err != nil {
		logger.WithError(err).Error("unable to get all Gardens")
		render.Render(w, r, ErrRender(err))
		return
	}
	logger.Debugf("found %d Gardens", len(gardens))

	if err := render.Render(w, r, gr.NewAllGardensResponse(r.Context(), gardens)); err != nil {
		logger.WithError(err).Error("unable to render AllGardensResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// getGarden will return a garden by ID/name
func (gr GardensResource) getGarden(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to get Garden")

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	logger.Debugf("responding with Garden: %+v", garden)

	gardenResponse := gr.NewGardenResponse(r.Context(), garden)
	if err := render.Render(w, r, gardenResponse); err != nil {
		logger.WithError(err).Error("unable to render GardenResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it. If the Garden is already
// end-dated, it will permanently delete it
func (gr GardensResource) endDateGarden(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to end-date Garden")

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	now := time.Now()

	// Don't allow end-dating a Garden with active Zones
	if garden.NumZones() > 0 {
		err := errors.New("unable to end-date Garden with active Zones")
		logger.WithError(err).Error("unable to end-date Garden")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Permanently delete the Garden if it is already end-dated
	if garden.EndDated() {
		logger.Info("permanently deleting Garden")

		if err := gr.storageClient.DeleteGarden(garden.ID); err != nil {
			logger.WithError(err).Error("unable to delete Garden")
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of Garden and save
	garden.EndDate = &now
	logger.Debug("saving end-dated Garden")
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		logger.WithError(err).Error("unable to save end-dated Garden")
		render.Render(w, r, InternalServerError(err))
		return
	}
	logger.Debug("saved end-dated Garden")

	// Remove scheduled light actions
	logger.Info("removing scheduled LightActions for Garden")
	if err := gr.worker.RemoveJobsByID(garden.ID); err != nil {
		logger.WithError(err).Error("unable to remove scheduled LightActions")
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, gr.NewGardenResponse(r.Context(), garden)); err != nil {
		logger.WithError(err).Error("unable to render GardenResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// updateGarden updates any fields in the existing Garden from the request
func (gr GardensResource) updateGarden(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to update Garden")

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	request := &UpdateGardenRequest{}

	// Read the request body into existing garden to overwrite fields
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid update Garden request")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Debugf("update request: %+v", request)

	// Validate that new MaxPlants (if defined) is not less than NumZones
	if request.Garden.MaxZones != nil && *request.Garden.MaxZones < garden.NumZones() {
		err := fmt.Errorf("unable to set max_zones less than current num_zones=%d", garden.NumZones())
		logger.WithError(err).Error("unable to update Garden")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	garden.Patch(request.Garden)
	logger.Debugf("garden after patching: %+v", garden)

	// Save the Garden
	logger.Debug("saving updated Garden")
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		logger.WithError(err).Error("unable to save updated Garden")
		render.Render(w, r, InternalServerError(err))
		return
	}

	// If LightSchedule is empty, remove the scheduled Job
	if garden.LightSchedule == nil {
		logger.Info("removing LightSchedule")
		if err := gr.worker.RemoveJobsByID(garden.ID); err != nil {
			logger.WithError(err).Error("unable to remove LightSchedule for Garden")
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	// Update the light schedule for the Garden (if it exists)
	if garden.LightSchedule != nil {
		logger.Info("updating/resetting LightSchedule for Garden")
		if err := gr.worker.ResetLightSchedule(garden); err != nil {
			logger.WithError(err).Errorf("unable to update/reset LightSchedule: %+v", garden.LightSchedule)
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	if err := render.Render(w, r, gr.NewGardenResponse(r.Context(), garden)); err != nil {
		logger.WithError(err).Error("unable to render GardenResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// getGardenHealth responds with the Garden's health status bsed on querying InfluxDB for self-reported status
func (gr GardensResource) getGardenHealth(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to get Garden health")

	defer gr.influxdbClient.Close()

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	health := garden.Health(r.Context(), gr.influxdbClient)

	logger.Debugf("retrieved Garden health data: %+v", health)

	if err := render.Render(w, r, GardenHealthResponse{health}); err != nil {
		logger.WithError(err).Error("unable to render GardenHealthResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// gardenAction reads a GardenAction request and uses it to execute one of the actions
// that is available to run against a Plant. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (gr GardensResource) gardenAction(w http.ResponseWriter, r *http.Request) {
	logger := contextLogger(r.Context())
	logger.Info("received request to execute GardenAction")

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	action := &GardenActionRequest{}
	if err := render.Bind(r, action); err != nil {
		logger.WithError(err).Error("invalid request for GardenAction")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	logger.Debugf("garden action: %+v", action)

	if err := gr.worker.ExecuteGardenAction(garden, action.GardenAction); err != nil {
		logger.WithError(err).Error("unable to execute GardenAction")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusAccepted)
	render.DefaultResponder(w, r, nil)
}
