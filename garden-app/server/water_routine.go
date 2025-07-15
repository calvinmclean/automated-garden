package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"

	"github.com/calvinmclean/babyapi"
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
	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)
	api.AddCustomIDRoute(http.MethodPost, "/run", api.GetRequestedResourceAndDo(api.runWatering))

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

func (api *WaterRoutineAPI) setup(storageClient *storage.Client, worker *worker.Worker) {
	api.storageClient = storageClient
	api.worker = worker
	api.SetStorage(api.storageClient.WaterRoutines)
}

func (api *WaterRoutineAPI) onCreateOrUpdate(_ http.ResponseWriter, r *http.Request, wr *pkg.WaterRoutine) *babyapi.ErrResponse {
	// Make sure all Zones exist
	for _, step := range wr.Steps {
		_, err := api.storageClient.Zones.Get(r.Context(), step.ZoneID.String())
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				return babyapi.ErrInvalidRequest(fmt.Errorf("unable to get Zone: %w", err))
			}
			return babyapi.InternalServerError(err)
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
