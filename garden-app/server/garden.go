package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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

	gr.api = babyapi.NewAPI[*pkg.Garden]("Gardens", gardenBasePath, func() *pkg.Garden { return &pkg.Garden{} })
	gr.api.SetStorage(gr.storageClient.Gardens)
	gr.api.ResponseWrapper(func(g *pkg.Garden) render.Renderer {
		return gr.NewGardenResponse(g)
	})

	gr.api.SetOnCreateOrUpdate(gr.onCreateOrUpdate)

	gr.api.AddCustomIDRoute(chi.Route{
		Pattern: "/action",
		Handlers: map[string]http.Handler{
			http.MethodPost: gr.api.GetRequestedResourceAndDo(gr.gardenAction),
		},
	})

	gr.api.SetGetAllFilter(EndDatedFilter[*pkg.Garden])

	gr.api.SetBeforeDelete(func(r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		gardenID := gr.api.GetIDParam(r)

		// Don't allow end-dating a Garden with active Zones
		numZones, err := gr.numZones(gardenID)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("error getting number of Zones for garden: %w", err))
		}
		if numZones > 0 {
			err := errors.New("unable to end-date Garden with active Zones")
			logger.Error("unable to end-date Garden", "error", err)
			return babyapi.ErrInvalidRequest(err)
		}

		return nil
	})

	gr.api.SetAfterDelete(func(r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		gardenID := gr.api.GetIDParam(r)

		// Remove scheduled light actions
		logger.Info("removing scheduled LightActions for Garden")
		if err := gr.worker.RemoveJobsByID(gardenID); err != nil {
			logger.Error("unable to remove scheduled LightActions", "error", err)
			return babyapi.InternalServerError(err)
		}
		return nil
	})

	return gr, nil
}

func (gr *GardensResource) onCreateOrUpdate(r *http.Request, garden *pkg.Garden) *babyapi.ErrResponse {
	logger := babyapi.GetLoggerFromContext(r.Context())

	numZones, err := gr.numZones(garden.ID.String())
	if err != nil {
		return babyapi.InternalServerError(err)
	}
	if *garden.MaxZones < numZones {
		return babyapi.ErrInvalidRequest(fmt.Errorf("unable to set max_zones less than current num_zones=%d", numZones))
	}

	// If LightSchedule is empty, remove the scheduled Job
	if garden.LightSchedule == nil {
		logger.Info("removing LightSchedule")
		if err := gr.worker.RemoveJobsByID(garden.ID.String()); err != nil {
			logger.Error("unable to remove LightSchedule for Garden", "error", err)
			return babyapi.InternalServerError(err)
		}
	}

	// Update the light schedule for the Garden (if it exists)
	if garden.LightSchedule != nil {
		logger.Info("updating/resetting LightSchedule for Garden")
		if err := gr.worker.ResetLightSchedule(garden); err != nil {
			logger.Error("unable to update/reset LightSchedule", "light_schedule", garden.LightSchedule, "error", err)
			return babyapi.InternalServerError(err)
		}
	}

	return nil
}

// gardenAction reads a GardenAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (gr *GardensResource) gardenAction(r *http.Request, garden *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute GardenAction")

	if garden.EndDated() {
		return nil, babyapi.ErrInvalidRequest(errors.New("unable to execute action on end-dated garden"))
	}

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
