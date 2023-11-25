package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	gardenBasePath = "/gardens"
)

// GardensResource encapsulates the structs and dependencies necessary for the "/gardens" API
// to function, including storage and configurating
type GardensResource struct {
	storageClient  *storage.Client
	influxdbClient influxdb.Client
	worker         *worker.Worker
	config         Config
	api            *babyapi.API[*pkg.Garden]
}

// NewGardenResource creates a new GardenResource
func NewGardenResource(config Config, storageClient *storage.Client, influxdbClient influxdb.Client, worker *worker.Worker) (*GardensResource, error) {
	gr := &GardensResource{
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		worker:         worker,
		config:         config,
	}

	// Initialize light schedules for all Gardens
	allGardens, err := gr.storageClient.Gardens.GetAll(storage.FilterEndDated[*pkg.Garden](false))
	if err != nil {
		return gr, err
	}
	for _, g := range allGardens {
		if g.LightSchedule != nil {
			if err = gr.worker.ScheduleLightActions(g); err != nil {
				return gr, fmt.Errorf("unable to schedule LightAction for Garden %v: %v", g.ID, err)
			}
		}
	}

	gr.api = babyapi.NewAPI[*pkg.Garden](gardenBasePath, func() *pkg.Garden { return &pkg.Garden{} })
	gr.api.SetStorage(gr.storageClient.Gardens)
	gr.api.ResponseWrapper(func(g *pkg.Garden) render.Renderer {
		return gr.NewGardenResponse(g)
	})

	gr.api.AddCustomRoute(chi.Route{
		Pattern: "/",
		Handlers: map[string]http.Handler{
			http.MethodPost: gr.api.ReadRequestBodyAndDo(gr.createGarden),
		},
	})

	gr.api.AddCustomIDRoute(chi.Route{
		Pattern: "/action",
		Handlers: map[string]http.Handler{
			http.MethodPost: gr.api.GetRequestedResourceAndDo(gr.gardenAction),
		},
	})

	// TODO: Add logger to patch or change how it is used
	gr.api.SetPATCH(func(old, new *pkg.Garden) *babyapi.ErrResponse {
		// Validate that new MaxZones (if defined) is not less than NumZones
		if new.MaxZones != nil {
			numZones, err := gr.numZones(old.ID.String())
			if err != nil {
				return babyapi.InternalServerError(err)
			}
			if *new.MaxZones < numZones {
				return babyapi.ErrInvalidRequest(fmt.Errorf("unable to set max_zones less than current num_zones=%d", numZones))
			}
		}

		old.Patch(new)

		// TODO: AfterPatch?
		// If LightSchedule is empty, remove the scheduled Job
		if old.LightSchedule == nil {
			// logger.Info("removing LightSchedule")
			if err := gr.worker.RemoveJobsByID(old.ID.String()); err != nil {
				// logger.WithError(err).Error("unable to remove LightSchedule for Garden")
				return babyapi.InternalServerError(err)
			}
		}

		// Update the light schedule for the Garden (if it exists)
		if old.LightSchedule != nil {
			// logger.Info("updating/resetting LightSchedule for Garden")
			if err := gr.worker.ResetLightSchedule(old); err != nil {
				// logger.WithError(err).Errorf("unable to update/reset LightSchedule: %+v", old.LightSchedule)
				return babyapi.InternalServerError(err)
			}
		}

		return nil
	})

	gr.api.SetGetAllFilter(EndDatedFilter[*pkg.Garden])

	gr.api.SetBeforeDelete(func(r *http.Request, gardenID string) error {
		// Don't allow end-dating a Garden with active Zones
		numZones, err := gr.numZones(gardenID)
		if err != nil {
			return fmt.Errorf("error getting number of Zones for garden: %w", err)
		}
		if numZones > 0 {
			err := errors.New("unable to end-date Garden with active Zones")
			// logger.WithError(err).Error("unable to end-date Garden")
			// render.Render(w, r, ErrInvalidRequest(err))
			// TODO: return 400 error?
			return err
		}

		// TODO: After Delete?
		// Remove scheduled light actions
		// logger.Info("removing scheduled LightActions for Garden")
		if err := gr.worker.RemoveJobsByID(gardenID); err != nil {
			// logger.WithError(err).Error("unable to remove scheduled LightActions")
			// render.Render(w, r, InternalServerError(err))
			return err
		}

		return nil
	})

	return gr, nil
}

func (gr *GardensResource) createGarden(r *http.Request, garden *pkg.Garden) (*pkg.Garden, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to create new Garden")

	logger.Debug("request to create Garden", "garden", garden)

	// Assign new unique ID and CreatedAt to garden
	garden.ID = xid.New()
	if garden.CreatedAt == nil {
		now := time.Now()
		garden.CreatedAt = &now
	}
	logger.Debug("new garden ID", "id", garden.ID)

	// Start light schedule (if applicable)
	if garden.LightSchedule != nil {
		if err := gr.worker.ScheduleLightActions(garden); err != nil {
			logger.Error("unable to schedule LightAction", "error", err)
			return nil, babyapi.InternalServerError(err)
		}
	}

	// Save the Garden
	logger.Debug("saving Garden")
	err := gr.storageClient.Gardens.Set(garden)
	if err != nil {
		logger.Error("unable to save Garden", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusCreated)
	return garden, nil
}

// gardenAction reads a GardenAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (gr *GardensResource) gardenAction(r *http.Request, garden *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute GardenAction")

	action := &GardenActionRequest{}
	if err := render.Bind(r, action); err != nil {
		logger.Error("invalid request for GardenAction", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("garden action", "action", action)

	if err := gr.worker.ExecuteGardenAction(garden, action.GardenAction); err != nil {
		logger.Error("unable to execute GardenAction", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusAccepted)
	return &GardenActionResponse{}, nil
}
