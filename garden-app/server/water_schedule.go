package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/server/html"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	waterScheduleBasePath   = "/water_schedules"
	waterScheduleIDLogField = "water_schedule_id"
)

// WaterSchedulesAPI provides and API for interacting with WaterSchedules
type WaterSchedulesAPI struct {
	*babyapi.API[*pkg.WaterSchedule]

	storageClient *storage.Client
	worker        *worker.Worker
}

// NewWaterSchedulesAPI creates a new WaterSchedulesResource
func NewWaterSchedulesAPI(storageClient *storage.Client, worker *worker.Worker) (WaterSchedulesAPI, error) {
	api := WaterSchedulesAPI{
		storageClient: storageClient,
		worker:        worker,
	}

	// Initialize WaterActions for each WaterSchedule from the storage client
	allWaterSchedules, err := api.storageClient.WaterSchedules.GetAll(func(ws *pkg.WaterSchedule) bool {
		return !ws.EndDated()
	})
	if err != nil {
		return api, fmt.Errorf("unable to get WaterSchedules: %v", err)
	}
	for _, ws := range allWaterSchedules {
		if err = api.worker.ScheduleWaterAction(ws); err != nil {
			return api, fmt.Errorf("unable to add WaterAction for WaterSchedule %v: %v", ws.ID, err)
		}
	}

	api.API = babyapi.NewAPI[*pkg.WaterSchedule]("WaterSchedules", waterScheduleBasePath, func() *pkg.WaterSchedule { return &pkg.WaterSchedule{} })
	api.SetStorage(api.storageClient.WaterSchedules)
	api.SetResponseWrapper(func(ws *pkg.WaterSchedule) render.Renderer {
		return api.NewWaterScheduleResponse(ws)
	})
	api.SetGetAllResponseWrapper(func(waterSchedules []*pkg.WaterSchedule) render.Renderer {
		resp := AllWaterSchedulesResponse{ResourceList: babyapi.ResourceList[*WaterScheduleResponse]{}}

		for _, w := range waterSchedules {
			resp.ResourceList.Items = append(resp.ResourceList.Items, api.NewWaterScheduleResponse(w))
		}

		return resp
	})

	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)

	api.SetBeforeDelete(func(r *http.Request) *babyapi.ErrResponse {
		id := api.GetIDParam(r)

		// Unable to delete WaterSchedule with associated Zones
		zones, err := api.storageClient.GetZonesUsingWaterSchedule(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to get Zones using WaterSchedule: %w", err))
		}
		if numZones := len(zones); numZones > 0 {
			return babyapi.ErrInvalidRequest(fmt.Errorf("unable to end-date WaterSchedule with %d Zones", numZones))
		}

		return nil
	})

	api.SetAfterDelete(func(r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		id := api.GetIDParam(r)

		// Remove scheduled WaterActions
		logger.Info("removing scheduled WaterActions for WaterSchedule")
		err := api.worker.RemoveJobsByID(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to remove scheduled WaterActions: %w", err))
		}

		return nil
	})

	api.SetGetAllFilter(EndDatedFilter[*pkg.WaterSchedule])

	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		switch r.URL.Query().Get("type") {
		case "create_modal":
			return html.Renderer(html.WaterScheduleModal, &pkg.WaterSchedule{
				ID: babyapi.NewID(),
			})
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodGet, "/components", api.GetRequestedResourceAndDo(func(r *http.Request, ws *pkg.WaterSchedule) (render.Renderer, *babyapi.ErrResponse) {
		switch r.URL.Query().Get("type") {
		case "edit_modal":
			return html.Renderer(html.WaterScheduleModal, ws), nil
		case "detail_modal":
			return html.Renderer(html.WaterScheduleDetailModal, ws), nil
		default:
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.ApplyExtension(extensions.HTMX[*pkg.WaterSchedule]{})

	return api, err
}

func (api *WaterSchedulesAPI) onCreateOrUpdate(_ *http.Request, ws *pkg.WaterSchedule) *babyapi.ErrResponse {
	// Validate the new WaterSchedule.WeatherControl
	if ws.WeatherControl != nil {
		err := api.weatherClientsExist(ws)
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
		err := api.worker.ResetWaterSchedule(ws)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to update/reset WaterSchedule: %w", err))
		}
	}

	return nil
}

func (api *WaterSchedulesAPI) weatherClientsExist(ws *pkg.WaterSchedule) error {
	if ws.HasTemperatureControl() {
		err := api.weatherClientExists(ws.WeatherControl.Temperature.ClientID)
		if err != nil {
			return fmt.Errorf("error getting client for TemperatureControl: %w", err)
		}
	}

	if ws.HasRainControl() {
		err := api.weatherClientExists(ws.WeatherControl.Rain.ClientID)
		if err != nil {
			return fmt.Errorf("error getting client for RainControl: %w", err)
		}
	}

	return nil
}

func (api *WaterSchedulesAPI) weatherClientExists(id xid.ID) error {
	_, err := api.storageClient.WeatherClientConfigs.Get(id.String())
	if err != nil {
		return fmt.Errorf("error getting WeatherClient with ID %q: %w", id, err)
	}
	return nil
}
