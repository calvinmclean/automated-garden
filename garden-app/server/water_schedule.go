package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	waterScheduleBasePath   = "/water_schedules"
	waterScheduleIDLogField = "water_schedule_id"
)

// WaterSchedulesResource provides and API for interacting with WaterSchedules
type WaterSchedulesResource struct {
	storageClient *storage.Client
	api           *babyapi.API[*pkg.WaterSchedule]
	worker        *worker.Worker
}

// NewWaterSchedulesResource creates a new WaterSchedulesResource
func NewWaterSchedulesResource(storageClient *storage.Client, worker *worker.Worker) (WaterSchedulesResource, error) {
	wsr := WaterSchedulesResource{
		storageClient: storageClient,
		worker:        worker,
	}

	// Initialize WaterActions for each WaterSchedule from the storage client
	allWaterSchedules, err := wsr.storageClient.WaterSchedules.GetAll(func(ws *pkg.WaterSchedule) bool {
		return !ws.EndDated()
	})
	if err != nil {
		return wsr, fmt.Errorf("unable to get WaterSchedules: %v", err)
	}
	for _, ws := range allWaterSchedules {
		if err = wsr.worker.ScheduleWaterAction(ws); err != nil {
			return wsr, fmt.Errorf("unable to add WaterAction for WaterSchedule %v: %v", ws.ID, err)
		}
	}

	wsr.api = babyapi.NewAPI[*pkg.WaterSchedule](waterScheduleBasePath, func() *pkg.WaterSchedule { return &pkg.WaterSchedule{} })
	wsr.api.SetStorage(wsr.storageClient.WaterSchedules)
	wsr.api.ResponseWrapper(func(ws *pkg.WaterSchedule) render.Renderer {
		return wsr.NewWaterScheduleResponse(ws)
	})

	wsr.api.SetOnCreateOrUpdate(wsr.onCreateOrUpdate)

	wsr.api.SetBeforeDelete(func(r *http.Request) *babyapi.ErrResponse {
		id := wsr.api.GetIDParam(r)

		// Unable to delete WaterSchedule with associated Zones
		zones, err := wsr.storageClient.GetZonesUsingWaterSchedule(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to get Zones using WaterSchedule: %w", err))
		}
		if numZones := len(zones); numZones > 0 {
			return babyapi.ErrInvalidRequest(fmt.Errorf("unable to end-date WaterSchedule with %d Zones", numZones))
		}

		return nil
	})

	wsr.api.SetAfterDelete(func(r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		id := wsr.api.GetIDParam(r)

		// Remove scheduled WaterActions
		logger.Info("removing scheduled WaterActions for WaterSchedule")
		err := wsr.worker.RemoveJobsByID(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to remove scheduled WaterActions: %w", err))
		}

		return nil
	})

	wsr.api.SetGetAllFilter(EndDatedFilter[*pkg.WaterSchedule])

	return wsr, err
}

func (wsr *WaterSchedulesResource) onCreateOrUpdate(r *http.Request, ws *pkg.WaterSchedule) *babyapi.ErrResponse {
	// Validate the new WaterSchedule.WeatherControl
	if ws.WeatherControl != nil {
		err := wsr.weatherClientsExist(ws)
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				return babyapi.ErrInvalidRequest(fmt.Errorf("unable to get WeatherClients for WaterSchedule: %w", err))
			}
			return babyapi.InternalServerError(err)
		}

		err = pkg.ValidateWeatherControl(ws.WeatherControl)
		if err != nil {
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid WaterSchedule.WeatherControl after patching: %w", err))
		}
	}

	if !ws.EndDated() {
		// logger.Info("updating/resetting WaterSchedule for WaterSchedule")
		err := wsr.worker.ResetWaterSchedule(ws)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to update/reset WaterSchedule: %w", err))
		}
	}

	return nil
}

func (wsr *WaterSchedulesResource) weatherClientsExist(ws *pkg.WaterSchedule) error {
	if ws.HasTemperatureControl() {
		err := wsr.weatherClientExists(ws.WeatherControl.Temperature.ClientID)
		if err != nil {
			return fmt.Errorf("error getting client for TemperatureControl: %w", err)
		}
	}

	if ws.HasRainControl() {
		err := wsr.weatherClientExists(ws.WeatherControl.Rain.ClientID)
		if err != nil {
			return fmt.Errorf("error getting client for RainControl: %w", err)
		}
	}

	return nil
}

func (wsr *WaterSchedulesResource) weatherClientExists(id xid.ID) error {
	_, err := wsr.storageClient.WaterSchedules.Get(id.String())
	if err != nil {
		return fmt.Errorf("error getting WeatherClient with ID %q: %w", id, err)
	}
	return nil
}
