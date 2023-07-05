package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
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
	storageClient storage.Client
	worker        *worker.Worker
}

// NewWaterSchedulesResource creates a new WaterSchedulesResource
func NewWaterSchedulesResource(storageClient storage.Client, worker *worker.Worker) (WaterSchedulesResource, error) {
	wsr := WaterSchedulesResource{
		storageClient: storageClient,
		worker:        worker,
	}

	// Initialize WaterActions for each WaterSchedule from the storage client
	allWaterSchedules, err := wsr.storageClient.GetWaterSchedules(false)
	if err != nil {
		return wsr, fmt.Errorf("unable to get WaterSchedules: %v", err)
	}
	for _, ws := range allWaterSchedules {
		if err = wsr.worker.ScheduleWaterAction(ws); err != nil {
			return wsr, fmt.Errorf("unable to add WaterAction for WaterSchedule %v: %v", ws.ID, err)
		}
	}
	return wsr, err
}

// waterScheduleContextMiddleware middleware is used to load a WaterSchedule object from the URL
// parameters passed through as the request. In case the WaterSchedule could not be found,
// we stop here and return a 404.
func (wsr WaterSchedulesResource) waterScheduleContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		wsIDString := chi.URLParam(r, waterSchedulePathParam)
		logger := getLoggerFromContext(ctx).WithField(waterScheduleIDLogField, wsIDString)
		wsID, err := xid.FromString(wsIDString)
		if err != nil {
			logger.WithError(err).Error("unable to parse WaterSchedule ID")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		ws, err := wsr.storageClient.GetWaterSchedule(wsID)
		if err != nil {
			logger.WithError(err).Error("unable to get WaterSchedule")
			render.Render(w, r, InternalServerError(err))
			return
		}
		if ws == nil {
			logger.Info("WaterSchedule not found")
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		logger.Debugf("found WaterSchedule: %+v", ws)

		ctx = newContextWithWaterSchedule(ctx, ws)
		ctx = newContextWithLogger(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getWaterSchedule simply returns the WaterSchedule requested by the provided ID
func (wsr WaterSchedulesResource) getWaterSchedule(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to get WaterSchedule")

	ws := getWaterScheduleFromContext(r.Context())
	logger.Debugf("responding with WaterSchedule: %+v", ws)

	if err := render.Render(w, r, wsr.NewWaterScheduleResponse(r.Context(), ws, excludeWeatherData(r))); err != nil {
		logger.WithError(err).Error("unable to render WaterScheduleResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// updateWaterSchedule will change any specified fields of the WaterSchedule and save it
func (wsr WaterSchedulesResource) updateWaterSchedule(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to update WaterSchedule")

	ws := getWaterScheduleFromContext(r.Context())
	request := &UpdateWaterScheduleRequest{}

	// Read the request body into existing WaterSchedule to overwrite fields
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid update WaterSchedule request")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	wasEndDated := ws.EndDated()

	ws.Patch(request.WaterSchedule)
	logger.Debugf("WaterSchedule after patching: %+v", ws)

	// Validate the new WaterSchedule.WeatherControl
	if ws.WeatherControl != nil {
		err := ValidateWeatherControl(ws.WeatherControl)
		if err != nil {
			logger.WithError(err).Error("invalid WaterSchedule.WeatherControl after patching")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
	}

	// Save the WaterSchedule
	if err := wsr.storageClient.SaveWaterSchedule(ws); err != nil {
		logger.WithError(err).Error("unable to save updated WaterSchedule")
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Update the water schedule for the WaterSchedule if it was changed or EndDate is removed
	if (wasEndDated && request.EndDate == nil) || request.Interval != nil || request.Duration != nil || request.StartTime != nil {
		logger.Info("updating/resetting WaterSchedule for WaterSchedule")
		if err := wsr.worker.ResetWaterSchedule(ws); err != nil {
			logger.WithError(err).Errorf("unable to update/reset WaterSchedule: %+v", ws)
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	if err := render.Render(w, r, wsr.NewWaterScheduleResponse(r.Context(), ws, excludeWeatherData(r))); err != nil {
		logger.WithError(err).Error("unable to render WaterScheduleResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// endDateWaterSchedule will mark the WaterSchedule's end date as now and save it
func (wsr WaterSchedulesResource) endDateWaterSchedule(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to end-date WaterSchedule")

	ws := getWaterScheduleFromContext(r.Context())
	now := time.Now()

	// Unable to delete WaterSchedule with associated Zones
	zones, err := wsr.storageClient.GetZonesUsingWaterSchedule(ws.ID)
	if err != nil {
		logger.WithError(err).Error("unable to get Zones using WaterSchedule")
		render.Render(w, r, InternalServerError(err))
		return
	}
	if numZones := len(zones); numZones > 0 {
		err := fmt.Errorf("unable to end-date WaterSchedule with %d Zones", numZones)
		logger.WithError(err).Error("unable to end-date WaterSchedule")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Permanently delete the WaterSchedule if it is already end-dated
	if ws.EndDated() {
		logger.Info("permanently deleting WaterSchedule")

		if err := wsr.storageClient.DeleteWaterSchedule(ws.ID); err != nil {
			logger.WithError(err).Error("unable to delete WaterSchedule")
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of WaterSchedule and save
	ws.EndDate = &now
	logger.Debug("saving end-dated WaterSchedule")
	if err := wsr.storageClient.SaveWaterSchedule(ws); err != nil {
		logger.WithError(err).Error("unable to save end-dated WaterSchedule")
		render.Render(w, r, InternalServerError(err))
		return
	}
	logger.Debug("saved end-dated WaterSchedule")

	// Remove scheduled WaterActions
	logger.Info("removing scheduled WaterActions for WaterSchedule")
	if err := wsr.worker.RemoveJobsByID(ws.ID); err != nil {
		logger.WithError(err).Error("unable to remove scheduled WaterActions")
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, wsr.NewWaterScheduleResponse(r.Context(), ws, excludeWeatherData(r))); err != nil {
		logger.WithError(err).Error("unable to render WaterScheduleResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// getAllWaterSchedules will return a list of all WaterSchedules
func (wsr WaterSchedulesResource) getAllWaterSchedules(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"

	logger := getLoggerFromContext(r.Context()).WithField("include_end_dated", getEndDated)
	logger.Info("received request to get all WaterSchedules")

	waterSchedules, err := wsr.storageClient.GetWaterSchedules(getEndDated)
	if err != nil {
		logger.WithError(err).Error("unable to get all Gardens")
		render.Render(w, r, ErrRender(err))
		return
	}
	logger.Debugf("found %d WaterSchedules", len(waterSchedules))

	if err := render.Render(w, r, wsr.NewAllWaterSchedulesResponse(r.Context(), waterSchedules, excludeWeatherData(r))); err != nil {
		logger.WithError(err).Error("unable to render AllWaterSchedulesResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// createWaterSchedule will create a new WaterSchedule resource
func (wsr WaterSchedulesResource) createWaterSchedule(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to create new WaterSchedule")

	request := &WaterScheduleRequest{}
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid request to create WaterSchedule")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	ws := request.WaterSchedule
	logger.Debugf("request to create WaterSchedule: %+v", ws)

	// Assign values to fields that may not be set in the request
	ws.ID = xid.New()
	logger.Debugf("new WaterSchedule ID: %v", ws.ID)

	// Save the WaterSchedule
	logger.Debug("saving WaterSchedule")
	if err := wsr.storageClient.SaveWaterSchedule(ws); err != nil {
		logger.WithError(err).Error("unable to save WaterSchedule")
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Start WaterSchedule
	if err := wsr.worker.ScheduleWaterAction(ws); err != nil {
		logger.WithError(err).Error("unable to schedule WaterAction")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, wsr.NewWaterScheduleResponse(r.Context(), ws, excludeWeatherData(r))); err != nil {
		logger.WithError(err).Error("unable to render WaterScheduleResponse")
		render.Render(w, r, ErrRender(err))
	}
}
