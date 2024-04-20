package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/server/html"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	zoneBasePath = "/zones"
)

// ZonesAPI encapsulates the structs and dependencies necessary for the "/zones" API
// to function, including storage, scheduling, and caching
type ZonesAPI struct {
	*babyapi.API[*pkg.Zone]

	storageClient  *storage.Client
	influxdbClient influxdb.Client
	worker         *worker.Worker
}

// NewZonesAPI creates a new ZonesResource
func NewZonesAPI(storageClient *storage.Client, influxdbClient influxdb.Client, worker *worker.Worker) (ZonesAPI, error) {
	api := ZonesAPI{
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		worker:         worker,
	}

	api.API = babyapi.NewAPI[*pkg.Zone]("Zones", zoneBasePath, func() *pkg.Zone { return &pkg.Zone{} })
	api.SetStorage(api.storageClient.Zones)
	api.SetResponseWrapper(func(z *pkg.Zone) render.Renderer {
		return api.NewZoneResponse(z)
	})
	api.SetGetAllResponseWrapper(func(zones []*pkg.Zone) render.Renderer {
		resp := AllZonesResponse{ResourceList: babyapi.ResourceList[*ZoneResponse]{}, api: api.API}

		for _, z := range zones {
			resp.ResourceList.Items = append(resp.ResourceList.Items, api.NewZoneResponse(z))
		}

		return resp
	})

	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)

	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		switch r.URL.Query().Get("type") {
		case "create_modal":
			waterSchedules, err := storageClient.WaterSchedules.GetAll(nil)
			if err != nil {
				return babyapi.InternalServerError(fmt.Errorf("error getting all waterschedules to create zone modal: %w", err))
			}

			slices.SortFunc(waterSchedules, func(ws1 *pkg.WaterSchedule, ws2 *pkg.WaterSchedule) int {
				return strings.Compare(ws1.Name, ws2.Name)
			})

			g, err := storageClient.Gardens.Get(api.GetParentIDParam(r))
			if err != nil {
				return babyapi.InternalServerError(fmt.Errorf("error getting garden to create zone modal: %w", err))
			}

			// TODO: remove positions that are already in-use by Zones
			positions := []int{}
			for i := 0; i < int(*g.MaxZones); i++ {
				positions = append(positions, i)
			}

			return html.Renderer(html.CreateZoneModal, map[string]any{
				"Garden":         g,
				"WaterSchedules": waterSchedules,
				"Positions":      positions,
			})
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodPost, "/action", api.GetRequestedResourceAndDo(api.zoneAction))

	api.AddCustomIDRoute(http.MethodGet, "/history", api.GetRequestedResourceAndDo(api.waterHistory))

	api.SetGetAllFilter(func(r *http.Request) babyapi.FilterFunc[*pkg.Zone] {
		gardenID := api.GetParentIDParam(r)
		gardenIDFilter := filterZoneByGardenID(gardenID)

		endDateFilter := EndDatedFilter[*pkg.Zone](r)
		return func(z *pkg.Zone) bool {
			return gardenIDFilter(z) && endDateFilter(z)
		}
	})

	api.ApplyExtension(extensions.HTMX[*pkg.Zone]{})

	return api, nil
}

// zoneAction reads a ZoneAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (api *ZonesAPI) zoneAction(r *http.Request, zone *pkg.Zone) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute ZoneAction")

	if zone.EndDated() {
		return nil, babyapi.ErrInvalidRequest(errors.New("unable to execute action on end-dated zone"))
	}
	garden, httpErr := api.getGardenFromRequest(r)
	if httpErr != nil {
		logger.Error("unable to get garden for zone", "error", httpErr)
		return nil, httpErr
	}

	zoneAction := &action.ZoneAction{}
	if err := render.Bind(r, zoneAction); err != nil {
		logger.Error("invalid request for ZoneAction", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Info("zone action", "action", zoneAction)

	if err := api.worker.ExecuteZoneAction(garden, zone, zoneAction); err != nil {
		logger.Error("unable to execute ZoneAction", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusAccepted)
	return &ZoneActionResponse{}, nil
}

func (api *ZonesAPI) waterSchedulesExist(ids []xid.ID) error {
	for _, id := range ids {
		_, err := api.storageClient.WaterSchedules.Get(id.String())
		if err != nil {
			return fmt.Errorf("error getting WaterSchedule with ID %q: %w", id, err)
		}
	}

	return nil
}

func (api *ZonesAPI) getGardenFromRequest(r *http.Request) (*pkg.Garden, *babyapi.ErrResponse) {
	garden, err := babyapi.GetResourceFromContext[*pkg.Garden](r.Context(), api.ParentContextKey())
	if err != nil {
		if errors.Is(err, babyapi.ErrNotFound) {
			return nil, babyapi.ErrNotFoundResponse
		}
		err = fmt.Errorf("error getting Garden %q for Zone: %w", api.GetParentIDParam(r), err)
		return nil, babyapi.InternalServerError(err)
	}

	return garden, nil
}

func (api *ZonesAPI) onCreateOrUpdate(r *http.Request, zone *pkg.Zone) *babyapi.ErrResponse {
	logger := babyapi.GetLoggerFromContext(r.Context())

	gardenID := api.GetParentIDParam(r)
	if !zone.GardenID.IsNil() && gardenID != zone.GardenID.String() {
		return babyapi.ErrInvalidRequest(fmt.Errorf("garden_id for zone must match URL path"))
	}

	garden, httpErr := api.getGardenFromRequest(r)
	if httpErr != nil {
		logger.Error("unable to get garden for zone", "error", httpErr)
		return httpErr
	}

	zonesForGarden, err := api.storageClient.Gardens.GetAll(func(g *pkg.Garden) bool {
		return g.ID.String() == gardenID
	})
	if err != nil {
		err = fmt.Errorf("error getting all zones for Garden %q: %w", gardenID, err)
		logger.Error("unable to get all zones", "error", err)
		return babyapi.InternalServerError(err)
	}

	zone.GardenID, err = xid.FromString(gardenID)
	if err != nil {
		return babyapi.ErrInvalidRequest(fmt.Errorf("invalid GardenID: %w", err))
	}

	// Validate that adding a Zone does not exceed Garden.MaxZones
	if uint(len(zonesForGarden)+1) > *garden.MaxZones {
		err := fmt.Errorf("adding a Zone would exceed Garden's max_zones=%d", *garden.MaxZones)
		logger.Error("invalid request to create Zone", "error", err)
		return babyapi.ErrInvalidRequest(err)
	}
	// Validate that ZonePosition works for a Garden with MaxZones (remember ZonePosition is zero-indexed)
	if *zone.Position >= *garden.MaxZones {
		err := fmt.Errorf("position invalid for Garden with max_zones=%d", *garden.MaxZones)
		logger.Error("invalid request to create Zone", "error", err)
		return babyapi.ErrInvalidRequest(err)
	}
	// Validate water schedules exists
	err = api.waterSchedulesExist(zone.WaterScheduleIDs)
	if err != nil {
		if errors.Is(err, babyapi.ErrNotFound) {
			logger.Error("invalid request to create Zone", "error", err)
			return babyapi.ErrInvalidRequest(err)
		}
		logger.Error("unable to get WaterSchedules for new Zone", "water_schedule_ids", zone.WaterScheduleIDs, "error", err)
		return babyapi.InternalServerError(err)
	}

	return nil
}

func rangeQueryParam(r *http.Request) (time.Duration, error) {
	timeRangeString := r.URL.Query().Get("range")
	if len(timeRangeString) == 0 {
		timeRangeString = "72h"
	}

	timeRange, err := time.ParseDuration(timeRangeString)
	if err != nil {
		return 0, err
	}
	return timeRange, nil
}

func limitQueryParam(r *http.Request) (uint64, error) {
	limitString := r.URL.Query().Get("limit")
	if len(limitString) == 0 {
		limitString = "0"
	}

	limit, err := strconv.ParseUint(limitString, 0, 64)
	if err != nil {
		return 0, err
	}

	return limit, nil
}

// WaterHistory responds with the Zone's recent water events read from InfluxDB
func (api *ZonesAPI) waterHistory(r *http.Request, zone *pkg.Zone) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to get Zone water history")

	history, apiErr := api.getWaterHistoryFromRequest(r, zone, logger)
	if apiErr != nil {
		return nil, apiErr
	}

	return NewZoneWaterHistoryResponse(history), nil
}

func (api *ZonesAPI) getWaterHistoryFromRequest(r *http.Request, zone *pkg.Zone, logger *slog.Logger) ([]pkg.WaterHistory, *babyapi.ErrResponse) {
	garden, httpErr := api.getGardenFromRequest(r)
	if httpErr != nil {
		logger.Error("unable to get garden for zone", "error", httpErr)
		return nil, httpErr
	}

	timeRange, err := rangeQueryParam(r)
	if err != nil {
		logger.Error("unable to parse time range", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("using time range", "time_range", timeRange)

	limit, err := limitQueryParam(r)
	if err != nil {
		logger.Error("unable to parse limit", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("using limit", "limit", limit)

	logger.Debug("getting water history from InfluxDB")
	history, err := api.getWaterHistory(r.Context(), zone, garden, timeRange, limit)
	if err != nil {
		logger.Error("unable to get water history from InfluxDB", "error", err)
		return nil, babyapi.InternalServerError(err)
	}
	logger.Debug("water history", "history", history)

	return history, nil
}

func (api *ZonesAPI) getMoisture(ctx context.Context, g *pkg.Garden, z *pkg.Zone) (float64, error) {
	defer api.influxdbClient.Close()

	moisture, err := api.influxdbClient.GetMoisture(ctx, *z.Position, g.TopicPrefix)
	if err != nil {
		return 0, err
	}
	return moisture, err
}

// getWaterHistory gets previous WaterEvents for this Zone from InfluxDB
func (api *ZonesAPI) getWaterHistory(ctx context.Context, zone *pkg.Zone, garden *pkg.Garden, timeRange time.Duration, limit uint64) (result []pkg.WaterHistory, err error) {
	defer api.influxdbClient.Close()

	history, err := api.influxdbClient.GetWaterHistory(ctx, *zone.Position, garden.TopicPrefix, timeRange, limit)
	if err != nil {
		return
	}

	for _, h := range history {
		result = append(result, pkg.WaterHistory{
			Duration:   (time.Duration(h["Duration"].(int)) * time.Millisecond).String(),
			RecordTime: h["RecordTime"].(time.Time),
		})
	}
	return
}

func excludeWeatherData(r *http.Request) bool {
	result := r.URL.Query().Get("exclude_weather_data") == "true"
	return result
}
