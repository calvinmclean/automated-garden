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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
	"github.com/rs/xid"
)

const (
	gardenBasePath  = "/gardens"
	gardenPathParam = "gardenID"
	gardenCtxKey    = contextKey("garden")
)

// GardensResource encapsulates the structs and dependencies necessary for the "/gardens" API
// to function, including storage and configurating
type GardensResource struct {
	storageClient  storage.Client
	influxdbClient influxdb.Client
	mqttClient     mqtt.Client
	scheduler      *gocron.Scheduler
	config         Config
}

// NewGardenResource creates a new GardenResource
func NewGardenResource(config Config) (gr GardensResource, err error) {
	gr = GardensResource{
		scheduler: gocron.NewScheduler(time.Local),
		config:    config,
	}
	gr.scheduler.StartAsync()

	// Initialize MQTT Client
	gr.mqttClient, err = mqtt.NewMQTTClient(gr.config.MQTTConfig, nil)
	if err != nil {
		err = fmt.Errorf("unable to initialize MQTT client: %v", err)
		return
	}

	// Initialize Storage Client
	gr.storageClient, err = storage.NewStorageClient(config.StorageConfig)
	if err != nil {
		err = fmt.Errorf("unable to initialize storage client: %v", err)
		return
	}

	// Initialize InfluxDB Client
	gr.influxdbClient = influxdb.NewClient(gr.config.InfluxDBConfig)

	// Initialize lighting schedules for all Gardens
	allGardens, err := gr.storageClient.GetGardens(false)
	if err != nil {
		return gr, err
	}
	for _, g := range allGardens {
		if g.LightSchedule != nil {
			if err = gr.scheduleLightActions(g); err != nil {
				err = fmt.Errorf("unable to add lighting Job for Garden %v: %v", g.ID, err)
				return gr, err
			}
		}
	}

	return
}

// routes creates all of the routing that is prefixed by "/plant" for interacting with Plant resources
func (gr GardensResource) routes(pr PlantsResource) chi.Router {
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
		})
	})
	return r
}

// restrictEndDatedMiddleware will return a 400 response if the requested Garden is end-dated
func (gr GardensResource) restrictEndDatedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

		if garden.EndDated() {
			render.Render(w, r, ErrInvalidRequest(fmt.Errorf("resource not available for end-dated Garden")))
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
		gardenID, err := xid.FromString(chi.URLParam(r, gardenPathParam))
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		garden, err := gr.storageClient.GetGarden(gardenID)
		if err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
		if garden == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		ctx := context.WithValue(r.Context(), gardenCtxKey, garden)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (gr GardensResource) createGarden(w http.ResponseWriter, r *http.Request) {
	request := &GardenRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	garden := request.Garden

	// Assign new unique ID and CreatedAt to garden
	garden.ID = xid.New()
	if garden.CreatedAt == nil {
		now := time.Now()
		garden.CreatedAt = &now
	}

	// Start lighting schedule (if applicable)
	if garden.LightSchedule != nil {
		if err := gr.scheduleLightActions(garden); err != nil {
			logger.Errorf("Unable to add lighting Job for Garden %v: %v", garden.ID, err)
		}
	}

	// Save the Garden
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, gr.NewGardenResponse(garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllGardens will return a list of all Gardens
func (gr GardensResource) getAllGardens(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	gardens, err := gr.storageClient.GetGardens(getEndDated)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.Render(w, r, gr.NewAllGardensResponse(gardens)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getGarden will return a garden by ID/name
func (gr GardensResource) getGarden(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	gardenResponse := gr.NewGardenResponse(garden)
	if err := render.Render(w, r, gardenResponse); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it. If the Garden is already
// end-dated, it will permanently delete it
func (gr GardensResource) endDateGarden(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	now := time.Now()

	// Don't allow end-dating a Garden with active Plants
	if garden.NumPlants() > 0 {
		render.Render(w, r, ErrInvalidRequest(errors.New("unable to end-date Garden with active Plants")))
		return
	}

	// Permanently delete the Garden if it is already end-dated
	if garden.EndDated() {
		if err := gr.storageClient.DeleteGarden(garden.ID); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of Garden and save
	garden.EndDate = &now
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Remove scheduled lighting actions
	if err := gr.removeJobsByID(garden.ID); err != nil {
		logger.Errorf("Unable to remove watering Job for Garden %s: %v", garden.ID.String(), err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, gr.NewGardenResponse(garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// updateGarden updates any fields in the existing Garden from the request
func (gr GardensResource) updateGarden(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	request := &UpdateGardenRequest{}

	// Read the request body into existing garden to overwrite fields
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Validate that new MaxPlants (if defined) is not less than NumPlants
	if request.Garden.MaxPlants != nil && *request.Garden.MaxPlants < garden.NumPlants() {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("unable to set max_plants less than current num_plants=%d", garden.NumPlants())))
		return
	}

	garden.Patch(request.Garden)

	// If LightSchedule is empty, remove the scheduled Job
	if garden.LightSchedule == nil {
		if err := gr.removeJobsByID(garden.ID); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	// Save the Garden
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Update the lighting schedule for the Garden (if it exists)
	if garden.LightSchedule != nil {
		if err := gr.resetLightingSchedule(garden); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	if err := render.Render(w, r, gr.NewGardenResponse(garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getGardenHealth responds with the Garden's health status bsed on querying InfluxDB for self-reported status
func (gr GardensResource) getGardenHealth(w http.ResponseWriter, r *http.Request) {
	defer gr.influxdbClient.Close()

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	health := garden.Health(r.Context(), gr.influxdbClient)
	if err := render.Render(w, r, GardenHealthResponse{health}); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// gardenAction reads a GardenAction request and uses it to execute one of the actions
// that is available to run against a Plant. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (gr GardensResource) gardenAction(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	action := &GardenActionRequest{}
	if err := render.Bind(r, action); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Received request to perform action on Garden %s", garden.ID)
	if err := action.Execute(garden, gr.mqttClient); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// If this is a LightAction with specified duration, additional steps are necessary
	if action.Light != nil && action.Light.ForDuration != "" {
		logger.Infof("LightAction requests delay for %s", action.Light.ForDuration)
		// Read delay Duration string into a time.Duration
		delayDuration, err := time.ParseDuration(action.Light.ForDuration)
		if err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}

		nextOnTime := gr.getNextLightTime(garden, pkg.StateOn)
		nextOffTime := gr.getNextLightTime(garden, pkg.StateOff)

		var adhocTime time.Time

		// If nextOffTime is before nextOnTime, then the light was probably ON and we need to schedule now + delay to turn back on. No need to delete any schedules
		if nextOffTime.Before(*nextOnTime) {
			now := time.Now()
			adhocTime = now.Add(delayDuration)
		} else {
			// If nextOffTime is after nextOnTime, then light was not ON yet and we need to delete nextOnTime and schedule nextOnTime + delay. Then we need to reschedule the regular ON time
			// Delete existing ON schedule
			if err := gr.removeLightScheduleWithState(garden, pkg.StateOn); err != nil {
				render.Render(w, r, InternalServerError(err))
				return
			}

			// Add new ON schedule with action.Light.ForDuration that executes once
			adhocTime = nextOnTime.Add(delayDuration)

			// Add new regular ON schedule starting 24 hours from today's Date + g.LightSchedule.StartTime
			err = gr.rescheduleLightOnAction(garden)
			if err != nil {
				render.Render(w, r, InternalServerError(err))
				return
			}
		}

		// Add new lightSchedule with AdhocTime and Save Garden
		garden.LightSchedule.AdhocOnTime = &adhocTime
		err = gr.scheduleAdhocLightAction(garden)
		if err != nil {
			logger.Errorf("Error adding adhoc schedule: %v", err)
			render.Render(w, r, InternalServerError(err))
			return
		}
		err = gr.storageClient.SaveGarden(garden)
		if err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	render.Status(r, http.StatusAccepted)
	render.DefaultResponder(w, r, nil)
}
