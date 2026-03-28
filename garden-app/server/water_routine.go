package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"

	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/go-chi/render"
)

const (
	waterRoutineBasePath   = "/water_routines"
	waterRoutineIDLogField = "water_routine_id"
)

type WaterRoutineAPI struct {
	*babyapi.API[*pkg.WaterRoutine]

	storageClient *storage.Client
	worker        *worker.Worker
}

func NewWaterRoutineAPI() *WaterRoutineAPI {
	api := &WaterRoutineAPI{}

	api.API = babyapi.NewAPI("WaterRoutines", waterRoutineBasePath, func() *pkg.WaterRoutine { return &pkg.WaterRoutine{} })
	api.SetResponseWrapper(func(wr *pkg.WaterRoutine) render.Renderer {
		return &WaterRoutineResponse{wr}
	})
	api.SetSearchResponseWrapper(func(waterRoutines []*pkg.WaterRoutine) render.Renderer {
		resp := AllWaterRoutinesResponse{ResourceList: babyapi.ResourceList[*WaterRoutineResponse]{}}

		for _, wr := range waterRoutines {
			resp.ResourceList.Items = append(resp.ResourceList.Items, &WaterRoutineResponse{wr})
		}

		return resp
	})
	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)
	api.AddCustomIDRoute(http.MethodPost, "/run", api.GetRequestedResourceAndDo(api.runWatering))

	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		switch r.URL.Query().Get("type") {
		case "create_modal":
			return api.waterRoutineModalRenderer(r.Context(), &pkg.WaterRoutine{
				ID: babyapi.NewID(),
			})
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodGet, "/components", api.GetRequestedResourceAndDo(func(_ http.ResponseWriter, r *http.Request, wr *pkg.WaterRoutine) (render.Renderer, *babyapi.ErrResponse) {
		switch r.URL.Query().Get("type") {
		case "edit_modal":
			return api.waterRoutineModalRenderer(r.Context(), wr), nil
		default:
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.ApplyExtension(extensions.HTMX[*pkg.WaterRoutine]{})

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

func (api *WaterRoutineAPI) setup(storageClient *storage.Client, worker *worker.Worker) {
	api.storageClient = storageClient
	api.worker = worker
	api.SetStorage(api.storageClient.WaterRoutines)
}

func (api *WaterRoutineAPI) waterRoutineModalRenderer(ctx context.Context, wr *pkg.WaterRoutine) render.Renderer {
	gardens, err := api.storageClient.Gardens.Search(ctx, "", nil)
	if err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error getting all gardens to create water routine modal: %w", err))
	}

	type GardenZones struct {
		GardenName string
		Zones      []*pkg.Zone
	}

	groupedZones := make(map[string]*GardenZones)
	for _, garden := range gardens {
		gz := &GardenZones{
			GardenName: garden.Name,
			Zones:      []*pkg.Zone{},
		}

		zones, err := api.storageClient.Zones.Search(ctx, garden.GetID(), nil)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("error getting zones for garden %s: %w", garden.GetID(), err))
		}

		gz.Zones = zones

		if len(gz.Zones) > 0 {
			groupedZones[garden.GetID()] = gz
		}
	}

	return waterRoutineModalTemplate.Renderer(map[string]any{
		"WaterRoutine": wr,
		"GroupedZones": groupedZones,
	})
}

func (api *WaterRoutineAPI) onCreateOrUpdate(_ http.ResponseWriter, r *http.Request, wr *pkg.WaterRoutine) *babyapi.ErrResponse {
	// Make sure all Zones exist and validate duration
	for i, step := range wr.Steps {
		// Validate Zone exists
		_, err := api.storageClient.Zones.Get(r.Context(), step.ZoneID.String())
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				return babyapi.ErrInvalidRequest(fmt.Errorf("unable to get Zone: %w", err))
			}
			return babyapi.InternalServerError(err)
		}

		// Validate Duration is not 0
		if step.Duration == nil || step.Duration.Duration == 0 {
			return babyapi.ErrInvalidRequest(fmt.Errorf("step %d: duration must be greater than 0", i+1))
		}
	}

	return nil
}

func (api *WaterRoutineAPI) runWatering(_ http.ResponseWriter, r *http.Request, wr *pkg.WaterRoutine) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute WaterRoutine")

	for _, step := range wr.Steps {
		stepLogger := logger.With("zone_id", step.ZoneID.String(), "duration", step.Duration.String())

		zone, err := api.storageClient.Zones.Get(r.Context(), step.ZoneID.String())
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				stepLogger.Warn("zone not found")
				continue
			}
			return nil, babyapi.InternalServerError(err)
		}

		stepLogger = stepLogger.With("garden_id", zone.GardenID.String())

		if zone.EndDated() {
			stepLogger.Warn("unable to execute action on end-dated zone")
			continue
		}

		garden, err := api.storageClient.Gardens.Get(r.Context(), zone.GardenID.String())
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				stepLogger.Warn("garden not found")
				continue
			}
			return nil, babyapi.InternalServerError(err)
		}

		zoneAction := &action.ZoneAction{
			Water: &action.WaterAction{
				Duration:      step.Duration,
				IgnoreWeather: true,
				Source:        action.SourceWaterRoutine,
			},
		}
		stepLogger.Info("zone action", "action", zoneAction)

		if err := api.worker.ExecuteZoneAction(garden, zone, zoneAction); err != nil {
			stepLogger.Error("unable to execute ZoneAction", "error", err)
			return nil, babyapi.InternalServerError(err)
		}
	}

	render.Status(r, http.StatusAccepted)
	return &ZoneActionResponse{}, nil
}
