package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	waterScheduleBasePath   = "/water_schedules"
	waterSchedulePathParam  = "waterScheduleID"
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

	wsr.api.AddCustomRoute(chi.Route{
		Pattern: "/",
		Handlers: map[string]http.Handler{
			http.MethodPost: wsr.api.ReadRequestBodyAndDo(wsr.createWaterSchedule),
		},
	})

	wsr.api.SetPATCH(func(old, new *pkg.WaterSchedule) *babyapi.ErrResponse {
		wasEndDated := old.EndDated()

		old.Patch(new)

		// Validate the new WaterSchedule.WeatherControl
		if old.WeatherControl != nil {
			err := wsr.weatherClientsExist(old)
			if err != nil {
				return babyapi.ErrInvalidRequest(fmt.Errorf("unable to get WeatherClients for WaterSchedule: %w", err))
			}

			err = pkg.ValidateWeatherControl(old.WeatherControl)
			if err != nil {
				return babyapi.ErrInvalidRequest(fmt.Errorf("invalid WaterSchedule.WeatherControl after patching: %w", err))
			}
		}

		// Update the water schedule for the WaterSchedule if it was changed or EndDate is removed
		if (wasEndDated && old.EndDate == nil) || old.Interval != nil || old.Duration != nil || old.StartTime != nil {
			// logger.Info("updating/resetting WaterSchedule for WaterSchedule")
			err := wsr.worker.ResetWaterSchedule(old)
			if err != nil {
				return babyapi.InternalServerError(fmt.Errorf("unable to update/reset WaterSchedule: %w", err))
			}
		}

		return nil
	})

	wsr.api.SetBeforeDelete(func(r *http.Request, id string) error {
		ws, httpErr := wsr.api.GetRequestedResource(r)
		if err != nil {
			return httpErr
		}

		// Unable to delete WaterSchedule with associated Zones
		zones, err := wsr.storageClient.GetZonesUsingWaterSchedule(ws.ID)
		if err != nil {
			return fmt.Errorf("unable to get Zones using WaterSchedule: %w", err)
		}
		if numZones := len(zones); numZones > 0 {
			// TODO: 400 error
			return fmt.Errorf("unable to end-date WaterSchedule with %d Zones", numZones)
		}

		logger := babyapi.GetLoggerFromContext(r.Context())

		// TODO: after delete?
		// Remove scheduled WaterActions
		logger.Info("removing scheduled WaterActions for WaterSchedule")
		if err := wsr.worker.RemoveJobsByID(ws.ID); err != nil {
			return fmt.Errorf("unable to remove scheduled WaterActions: %w", err)
		}

		return nil
	})

	wsr.api.SetGetAllFilter(EndDatedFilter[*pkg.WaterSchedule])

	return wsr, err
}

// createWaterSchedule will create a new WaterSchedule resource
func (wsr *WaterSchedulesResource) createWaterSchedule(r *http.Request, ws *pkg.WaterSchedule) (*pkg.WaterSchedule, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to create new WaterSchedule")

	logger.Debug("request to create WaterSchedule", "water_schedule", ws)

	err := wsr.weatherClientsExist(ws)
	if err != nil {
		if errors.Is(err, babyapi.ErrNotFound) {
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("unable to get WeatherClients for WaterSchedule %w", err))
		}
		logger.Error("unable to get WeatherClients for WaterSchedule", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	// Assign values to fields that may not be set in the request
	ws.ID = xid.New()
	logger.Debug("new WaterSchedule ID", "id", ws.ID)

	// Save the WaterSchedule
	logger.Debug("saving WaterSchedule")
	if err := wsr.storageClient.WaterSchedules.Set(ws); err != nil {
		logger.Error("unable to save WaterSchedule", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	// Start WaterSchedule
	if err := wsr.worker.ScheduleWaterAction(ws); err != nil {
		logger.Error("unable to schedule WaterAction", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusCreated)
	return ws, nil
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
