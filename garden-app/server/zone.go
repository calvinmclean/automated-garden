package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	zoneBasePath   = "/zones"
	zonePathParam  = "zoneID"
	zoneIDLogField = "zone_id"
)

// ZonesResource encapsulates the structs and dependencies necessary for the "/zones" API
// to function, including storage, scheduling, and caching
type ZonesResource struct {
	storageClient  *storage.Client
	influxdbClient influxdb.Client
	worker         *worker.Worker
	api            *babyapi.API[*pkg.Zone]
}

// NewZonesResource creates a new ZonesResource
func NewZonesResource(storageClient *storage.Client, influxdbClient influxdb.Client, worker *worker.Worker) (ZonesResource, error) {
	zr := ZonesResource{
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		worker:         worker,
	}

	zr.api = babyapi.NewAPI[*pkg.Zone](zoneBasePath, func() *pkg.Zone { return &pkg.Zone{} })
	zr.api.SetStorage(zr.storageClient.Zones)
	zr.api.ResponseWrapper(func(z *pkg.Zone) render.Renderer {
		return zr.NewZoneResponse(z)
	})

	zr.api.AddCustomRoute(chi.Route{
		Pattern: "/",
		Handlers: map[string]http.Handler{
			http.MethodPost: zr.api.ReadRequestBodyAndDo(zr.createZone),
		},
	})

	zr.api.AddCustomIDRoute(chi.Route{
		Pattern: "/action",
		Handlers: map[string]http.Handler{
			http.MethodPost: zr.api.GetRequestedResourceAndDo(zr.zoneAction),
		},
	})

	zr.api.AddCustomIDRoute(chi.Route{
		Pattern: "/history",
		Handlers: map[string]http.Handler{
			http.MethodGet: zr.api.GetRequestedResourceAndDo(zr.waterHistory),
		},
	})

	zr.api.SetPATCH(func(old, new *pkg.Zone) *babyapi.ErrResponse {
		if len(new.WaterScheduleIDs) != 0 {
			err := zr.waterSchedulesExist(new.WaterScheduleIDs)
			if err != nil {
				if errors.Is(err, babyapi.ErrNotFound) {
					err = fmt.Errorf("unable to update Zone with non-existent WaterSchedule %q: %w", new.WaterScheduleIDs, err)
					return babyapi.ErrInvalidRequest(err)
				}
				return babyapi.InternalServerError(fmt.Errorf("unable to get WaterSchedules %q for updating Zone: %w", new.WaterScheduleIDs, err))
			}
		}

		old.Patch(new)

		return nil
	})

	zr.api.SetGetAllFilter(func(r *http.Request) babyapi.FilterFunc[*pkg.Zone] {
		// TODO: improve how these url params are accessed
		// TODO: put this in middleware since it's used in mutlple parts?
		gardenID := chi.URLParam(r, "/gardensID")
		gardenIDFilter := filterZoneByGardenID(gardenID)

		endDateFilter := EndDatedFilter[*pkg.Zone](r)
		return func(z *pkg.Zone) bool {
			return gardenIDFilter(z) && endDateFilter(z)
		}
	})

	return zr, nil
}

// zoneAction reads a ZoneAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (zr *ZonesResource) zoneAction(r *http.Request, zone *pkg.Zone) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute ZoneAction")

	if zone.EndDated() {
		return nil, babyapi.ErrInvalidRequest(errors.New("unable to execute action on end-dated zone"))
	}

	// TODO: improve how these url params are accessed
	// TODO: put this in middleware since it's used in mutlple parts?
	gardenID := chi.URLParam(r, "/gardensID")
	garden, err := zr.storageClient.Gardens.Get(gardenID)
	if err != nil {
		err = fmt.Errorf("error getting Garden %q for Zone: %w", gardenID, err)
		logger.Error("unable to get garden for zone", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	action := &ZoneActionRequest{}
	if err := render.Bind(r, action); err != nil {
		logger.Error("invalid request for ZoneAction", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("zone action", "action", action)

	if err := zr.worker.ExecuteZoneAction(garden, zone, action.ZoneAction); err != nil {
		logger.Error("unable to execute ZoneAction", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusAccepted)
	return &ZoneActionResponse{}, nil
}

func (zr *ZonesResource) waterSchedulesExist(ids []xid.ID) error {
	for _, id := range ids {
		_, err := zr.storageClient.WaterSchedules.Get(id.String())
		if err != nil {
			return fmt.Errorf("error getting WaterSchedule with ID %q: %w", id, err)
		}
	}

	return nil
}

// createZone will create a new Zone resource
func (zr *ZonesResource) createZone(r *http.Request, zone *pkg.Zone) (*pkg.Zone, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to create new Zone")

	logger.Debug("request to create Zone", "zone", zone)

	// TODO: improve how these url params are accessed
	// TODO: put this in middleware since it's used in mutlple parts?
	gardenID := chi.URLParam(r, "/gardensID")
	garden, err := zr.storageClient.Gardens.Get(gardenID)
	if err != nil {
		err = fmt.Errorf("error getting Garden %q for Zone: %w", gardenID, err)
		logger.Error("unable to get garden for zone", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	zonesForGarden, err := zr.storageClient.Gardens.GetAll(func(g *pkg.Garden) bool {
		return g.ID.String() == gardenID
	})
	if err != nil {
		err = fmt.Errorf("error getting all zones for Garden %q: %w", gardenID, err)
		logger.Error("unable to get all zones", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	zone.GardenID, err = xid.FromString(gardenID)
	if err != nil {
		return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid GardenID: %w", err))
	}

	// Validate that adding a Zone does not exceed Garden.MaxZones
	if uint(len(zonesForGarden)+1) > *garden.MaxZones {
		err := fmt.Errorf("adding a Zone would exceed Garden's max_zones=%d", *garden.MaxZones)
		logger.Error("invalid request to create Zone", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	// Validate that ZonePosition works for a Garden with MaxZones (remember ZonePosition is zero-indexed)
	if *zone.Position >= *garden.MaxZones {
		err := fmt.Errorf("position invalid for Garden with max_zones=%d", *garden.MaxZones)
		logger.Error("invalid request to create Zone", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	// Validate water schedules exists
	err = zr.waterSchedulesExist(zone.WaterScheduleIDs)
	if err != nil {
		if errors.Is(err, babyapi.ErrNotFound) {
			err = fmt.Errorf("unable to create Zone with non-existent WaterSchedule %q: %w", zone.WaterScheduleIDs, err)
			logger.Error("invalid request to create Zone", "error", err)
			return nil, babyapi.ErrInvalidRequest(err)
		}
		logger.Error("unable to get WaterSchedules for new Zone", "water_schedule_ids", zone.WaterScheduleIDs, "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	// Assign values to fields that may not be set in the request
	zone.ID = xid.New()
	if zone.CreatedAt == nil {
		now := time.Now()
		zone.CreatedAt = &now
	}
	logger.Debug("new zone ID", "id", zone.ID)

	// Save the Zone
	logger.Debug("saving Zone")
	if err := zr.storageClient.Zones.Set(zone); err != nil {
		logger.Error("unable to save Zone", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusCreated)
	return zone, nil
}

// WaterHistory responds with the Zone's recent water events read from InfluxDB
func (zr *ZonesResource) waterHistory(r *http.Request, zone *pkg.Zone) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to get Zone water history")

	// TODO: improve how these url params are accessed
	// TODO: put this in middleware since it's used in mutlple parts?
	gardenID := chi.URLParam(r, "/gardensID")
	garden, err := zr.storageClient.Gardens.Get(gardenID)
	if err != nil {
		err = fmt.Errorf("error getting Garden %q for Zone: %w", gardenID, err)
		logger.Error("unable to get garden for zone", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	// Read query parameters and set default values
	timeRangeString := r.URL.Query().Get("range")
	if len(timeRangeString) == 0 {
		timeRangeString = "72h"
	}
	logger.Debug("using time range", "time_range", timeRangeString)

	limitString := r.URL.Query().Get("limit")
	if len(limitString) == 0 {
		limitString = "0"
	}
	logger.Debug("using limit", "limit", limitString)

	// Parse query parameter strings into correct types
	timeRange, err := time.ParseDuration(timeRangeString)
	if err != nil {
		logger.Error("unable to parse time range", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	limit, err := strconv.ParseUint(limitString, 0, 64)
	if err != nil {
		logger.Error("unable to parse limit", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}

	logger.Debug("getting water history from InfluxDB")
	history, err := zr.getWaterHistory(r.Context(), zone, garden, timeRange, limit)
	if err != nil {
		logger.Error("unable to get water history from InfluxDB", "error", err)
		return nil, babyapi.InternalServerError(err)
	}
	logger.Debug("water history", "history", history)

	return NewZoneWaterHistoryResponse(history), nil
}

func (zr *ZonesResource) getMoisture(ctx context.Context, g *pkg.Garden, z *pkg.Zone) (float64, error) {
	defer zr.influxdbClient.Close()

	moisture, err := zr.influxdbClient.GetMoisture(ctx, *z.Position, g.TopicPrefix)
	if err != nil {
		return 0, err
	}
	return moisture, err
}

// getWaterHistory gets previous WaterEvents for this Zone from InfluxDB
func (zr *ZonesResource) getWaterHistory(ctx context.Context, zone *pkg.Zone, garden *pkg.Garden, timeRange time.Duration, limit uint64) (result []pkg.WaterHistory, err error) {
	defer zr.influxdbClient.Close()

	history, err := zr.influxdbClient.GetWaterHistory(ctx, *zone.Position, garden.TopicPrefix, timeRange, limit)
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

// ZoneActionRequest wraps a ZoneAction into a request so we can handle Bind/Render in this package
type ZoneActionRequest struct {
	*action.ZoneAction
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *ZoneActionRequest) Bind(_ *http.Request) error {
	// ZoneAction is nil if no ZoneAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || action.ZoneAction == nil || (action.Water == nil) {
		return errors.New("missing required action fields")
	}
	return nil
}
