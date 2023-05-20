package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

const (
	zoneBasePath   = "/zones"
	zonePathParam  = "zoneID"
	zoneIDLogField = "zone_id"
)

// ZonesResource encapsulates the structs and dependencies necessary for the "/zones" API
// to function, including storage, scheduling, and caching
type ZonesResource struct {
	GardensResource
}

// NewZonesResource creates a new ZonesResource
func NewZonesResource(gr GardensResource, logger *logrus.Entry) (ZonesResource, error) {
	zr := ZonesResource{
		GardensResource: gr,
	}

	return zr, nil
}

// zoneContextMiddleware middleware is used to load a Zone object from the URL
// parameters passed through as the request. In case the Zone could not be found,
// we stop here and return a 404.
func (zr ZonesResource) zoneContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		zoneIDString := chi.URLParam(r, zonePathParam)
		logger := getLoggerFromContext(ctx).WithField(zoneIDLogField, zoneIDString)
		zoneID, err := xid.FromString(zoneIDString)
		if err != nil {
			logger.WithError(err).Error("unable to parse Zone ID")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		garden := getGardenFromContext(ctx)
		zone := garden.Zones[zoneID]
		if zone == nil {
			logger.Info("zone not found")
			render.Render(w, r, ErrNotFoundResponse)
			return
		}
		logger.Debugf("found Zone: %+v", zone)

		ctx = newContextWithZone(ctx, zone)
		ctx = newContextWithLogger(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// zoneAction reads a ZoneAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (zr ZonesResource) zoneAction(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to execute ZoneAction")

	garden := getGardenFromContext(r.Context())
	zone := getZoneFromContext(r.Context())

	action := &ZoneActionRequest{}
	if err := render.Bind(r, action); err != nil {
		logger.WithError(err).Error("invalid request for ZoneAction")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	logger.Debugf("zone action: %+v", action)

	if err := zr.worker.ExecuteZoneAction(garden, zone, action.ZoneAction); err != nil {
		logger.WithError(err).Error("unable to execute ZoneAction")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusAccepted)
	render.DefaultResponder(w, r, nil)
}

// getZone simply returns the Zone requested by the provided ID
func (zr ZonesResource) getZone(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to get Zone")

	garden := getGardenFromContext(r.Context())
	zone := getZoneFromContext(r.Context())
	logger.Debugf("responding with Zone: %+v", zone)

	if err := render.Render(w, r, zr.NewZoneResponse(r.Context(), garden, zone)); err != nil {
		logger.WithError(err).Error("unable to render ZoneResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// updateZone will change any specified fields of the Zone and save it
func (zr ZonesResource) updateZone(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to update Zone")

	zone := getZoneFromContext(r.Context())
	request := &UpdateZoneRequest{}
	garden := getGardenFromContext(r.Context())

	// Read the request body into existing zone to overwrite fields
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid update Zone request")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if request.Zone.WaterScheduleID != xid.NilID() {
		// Validate water schedule exists
		_, err := zr.storageClient.GetWaterSchedule(request.Zone.WaterScheduleID)
		if err != nil {
			err := fmt.Errorf("error getting WaterSchedule with ID %q", request.Zone.WaterScheduleID)
			logger.WithError(err).Error("invalid request to update Zone")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
	}

	zone.Patch(request.Zone)
	logger.Debugf("zone after patching: %+v", zone)

	// Save the Zone
	if err := zr.storageClient.SaveZone(garden.ID, zone); err != nil {
		logger.WithError(err).Error("unable to save updated Zone")
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, zr.NewZoneResponse(r.Context(), garden, zone)); err != nil {
		logger.WithError(err).Error("unable to render ZoneResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// endDateZone will mark the Zone's end date as now and save it
func (zr ZonesResource) endDateZone(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to end-date Zone")

	garden := getGardenFromContext(r.Context())
	zone := getZoneFromContext(r.Context())
	now := time.Now()

	// Unable to delete Zone with associated Plants
	if numPlants := len(garden.PlantsByZone(zone.ID, false)); numPlants > 0 {
		err := fmt.Errorf("unable to end-date Zone with %d Plants", numPlants)
		logger.WithError(err).Error("unable to end-date Zone")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Permanently delete the Zone if it is already end-dated
	if zone.EndDated() {
		logger.Info("permanently deleting Zone")

		if err := zr.storageClient.DeleteZone(garden.ID, zone.ID); err != nil {
			logger.WithError(err).Error("unable to delete Zone")
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of Zone and save
	zone.EndDate = &now
	logger.Debug("saving end-dated Zone")
	if err := zr.storageClient.SaveZone(garden.ID, zone); err != nil {
		logger.WithError(err).Error("unable to save end-dated Zone")
		render.Render(w, r, InternalServerError(err))
		return
	}
	logger.Debug("saved end-dated Zone")

	// Remove scheduled WaterActions
	logger.Info("removing scheduled WaterActions for Zone")
	if err := zr.worker.RemoveJobsByID(zone.ID); err != nil {
		logger.WithError(err).Error("unable to remove scheduled WaterActions")
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, zr.NewZoneResponse(r.Context(), garden, zone)); err != nil {
		logger.WithError(err).Error("unable to render ZoneResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// getAllZones will return a list of all Zones
func (zr ZonesResource) getAllZones(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"

	logger := getLoggerFromContext(r.Context()).WithField("include_end_dated", getEndDated)
	logger.Info("received request to get all Zones")

	garden := getGardenFromContext(r.Context())
	zones := []*pkg.Zone{}
	for _, z := range garden.Zones {
		if getEndDated || (z.EndDate == nil || z.EndDate.After(time.Now())) {
			zones = append(zones, z)
		}
	}
	logger.Debugf("found %d Zones", len(zones))

	if err := render.Render(w, r, zr.NewAllZonesResponse(r.Context(), zones, garden)); err != nil {
		logger.WithError(err).Error("unable to render AllZonesResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// createZone will create a new Zone resource
func (zr ZonesResource) createZone(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to create new Zone")

	request := &ZoneRequest{}
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid request to create Zone")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	zone := request.Zone
	logger.Debugf("request to create Zone: %+v", zone)

	garden := getGardenFromContext(r.Context())

	// Validate that adding a Zone does not exceed Garden.MaxZones
	if garden.NumZones()+1 > *garden.MaxZones {
		err := fmt.Errorf("adding a Zone would exceed Garden's max_zones=%d", *garden.MaxZones)
		logger.WithError(err).Error("invalid request to create Zone")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	// Validate that ZonePosition works for a Garden with MaxZones (remember ZonePosition is zero-indexed)
	if *zone.Position >= *garden.MaxZones {
		err := fmt.Errorf("position invalid for Garden with max_zones=%d", *garden.MaxZones)
		logger.WithError(err).Error("invalid request to create Zone")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Validate water schedule exists
	_, err := zr.storageClient.GetWaterSchedule(zone.WaterScheduleID)
	if err != nil {
		err := fmt.Errorf("error getting WaterSchedule with ID %q", zone.WaterScheduleID)
		logger.WithError(err).Error("invalid request to create Zone")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Assign values to fields that may not be set in the request
	zone.ID = xid.New()
	if zone.CreatedAt == nil {
		now := time.Now()
		zone.CreatedAt = &now
	}
	logger.Debugf("new zone ID: %v", zone.ID)

	// Save the Zone
	logger.Debug("saving Zone")
	if err := zr.storageClient.SaveZone(garden.ID, zone); err != nil {
		logger.WithError(err).Error("unable to save Zone")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, zr.NewZoneResponse(r.Context(), garden, zone)); err != nil {
		logger.WithError(err).Error("unable to render ZoneResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// WaterHistory responds with the Zone's recent water events read from InfluxDB
func (zr ZonesResource) waterHistory(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to get Zone water history")

	garden := getGardenFromContext(r.Context())
	zone := getZoneFromContext(r.Context())

	// Read query parameters and set default values
	timeRangeString := r.URL.Query().Get("range")
	if len(timeRangeString) == 0 {
		timeRangeString = "72h"
	}
	logger.Debugf("using time range: %s", timeRangeString)

	limitString := r.URL.Query().Get("limit")
	if len(limitString) == 0 {
		limitString = "0"
	}
	logger.Debugf("using limit: %s", limitString)

	// Parse query parameter strings into correct types
	timeRange, err := time.ParseDuration(timeRangeString)
	if err != nil {
		logger.WithError(err).Error("unable to parse time range")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	limit, err := strconv.ParseUint(limitString, 0, 64)
	if err != nil {
		logger.WithError(err).Error("unable to parse limit")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Debug("getting water history from InfluxDB")
	history, err := zr.getWaterHistory(r.Context(), zone, garden, timeRange, limit)
	if err != nil {
		logger.WithError(err).Error("unable to get water history from InfluxDB")
		render.Render(w, r, InternalServerError(err))
		return
	}
	logger.Debugf("water history: %+v", history)

	if err := render.Render(w, r, NewZoneWaterHistoryResponse(history)); err != nil {
		logger.WithError(err).Error("unable to render Zone water history response")
		render.Render(w, r, ErrRender(err))
	}
}

func (zr ZonesResource) getMoisture(ctx context.Context, g *pkg.Garden, z *pkg.Zone) (float64, error) {
	defer zr.influxdbClient.Close()

	moisture, err := zr.influxdbClient.GetMoisture(ctx, *z.Position, g.TopicPrefix)
	if err != nil {
		return 0, err
	}
	return moisture, err
}

// getWaterHistory gets previous WaterEvents for this Zone from InfluxDB
func (zr ZonesResource) getWaterHistory(ctx context.Context, zone *pkg.Zone, garden *pkg.Garden, timeRange time.Duration, limit uint64) (result []pkg.WaterHistory, err error) {
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
