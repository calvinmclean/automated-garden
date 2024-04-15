package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/server/templates"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

const (
	gardenBasePath = "/gardens"
)

// GardensAPI encapsulates the structs and dependencies necessary for the "/gardens" API
// to function, including storage and configurating
type GardensAPI struct {
	*babyapi.API[*pkg.Garden]

	storageClient  *storage.Client
	influxdbClient influxdb.Client
	worker         *worker.Worker
	config         Config
}

// NewGardensAPI creates a new GardenResource
func NewGardensAPI(config Config, storageClient *storage.Client, influxdbClient influxdb.Client, worker *worker.Worker) (*GardensAPI, error) {
	gr := &GardensAPI{
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

	gr.API = babyapi.NewAPI[*pkg.Garden]("Gardens", gardenBasePath, func() *pkg.Garden { return &pkg.Garden{} })
	gr.SetStorage(gr.storageClient.Gardens)
	gr.SetResponseWrapper(func(g *pkg.Garden) render.Renderer {
		return gr.NewGardenResponse(g)
	})
	gr.SetGetAllResponseWrapper(func(gardens []*pkg.Garden) render.Renderer {
		resp := AllGardensResponse{ResourceList: babyapi.ResourceList[*GardenResponse]{}}

		for _, g := range gardens {
			resp.ResourceList.Items = append(resp.ResourceList.Items, gr.NewGardenResponse(g))
		}

		return resp
	})

	gr.SetOnCreateOrUpdate(gr.onCreateOrUpdate)

	gr.AddCustomIDRoute(http.MethodPost, "/action", gr.GetRequestedResourceAndDo(gr.gardenAction))
	gr.AddCustomIDRoute(http.MethodGet, "/modal", gr.GetRequestedResourceAndDo(func(_ *http.Request, g *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
		return templates.Renderer(templates.EditGardenModal, g), nil
	}))

	gr.SetGetAllFilter(EndDatedFilter[*pkg.Garden])

	gr.SetBeforeDelete(func(r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		gardenID := gr.GetIDParam(r)

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

	gr.SetAfterDelete(func(r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		gardenID := gr.GetIDParam(r)

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

func (gr *GardensAPI) onCreateOrUpdate(r *http.Request, garden *pkg.Garden) *babyapi.ErrResponse {
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
func (gr *GardensAPI) gardenAction(r *http.Request, garden *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute GardenAction")

	if garden.EndDated() {
		return nil, babyapi.ErrInvalidRequest(errors.New("unable to execute action on end-dated garden"))
	}

	gardenAction := &action.GardenAction{}
	if err := render.Bind(r, gardenAction); err != nil {
		logger.Error("invalid request for GardenAction", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("garden action", "action", gardenAction)

	if err := gr.worker.ExecuteGardenAction(garden, gardenAction); err != nil {
		logger.Error("unable to execute GardenAction", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusAccepted)
	return &GardenActionResponse{}, nil
}
