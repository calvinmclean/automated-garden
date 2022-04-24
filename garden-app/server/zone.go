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
)

const (
	zoneBasePath  = "/zones"
	zonePathParam = "zoneID"
	zoneCtxKey    = contextKey("zone")
)

// ZonesResource encapsulates the structs and dependencies necessary for the "/zones" API
// to function, including storage, scheduling, and caching
type ZonesResource struct {
	GardensResource
}

// NewZonesResource creates a new ZonesResource
func NewZonesResource(gr GardensResource) (ZonesResource, error) {
	zr := ZonesResource{
		GardensResource: gr,
	}

	// Initialize water Jobs for each Zone from the storage client
	allGardens, err := zr.storageClient.GetGardens(false)
	if err != nil {
		return zr, err
	}
	for _, g := range allGardens {
		allZones, err := zr.storageClient.GetZones(g.ID, false)
		if err != nil {
			return zr, err
		}
		for _, z := range allZones {
			if err = zr.scheduler.ScheduleWaterAction(g, z); err != nil {
				err = fmt.Errorf("unable to add water Job for Zone %v: %v", z.ID, err)
				return zr, err
			}
		}
	}

	return zr, err
}

// routes creates all of the routing that is prefixed by "/zone" for interacting with Zone resources
func (zr ZonesResource) routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", zr.createZone)
	r.Get("/", zr.getAllZones)
	r.Route(fmt.Sprintf("/{%s}", zonePathParam), func(r chi.Router) {
		r.Use(zr.zoneContextMiddleware)

		r.Get("/", zr.getZone)
		r.Patch("/", zr.updateZone)
		r.Delete("/", zr.endDateZone)

		// Add new middleware to restrict certain paths to non-end-dated Zones
		r.Route("/", func(r chi.Router) {
			r.Use(zr.restrictEndDatedMiddleware)

			r.Post("/action", zr.zoneAction)
			r.Get("/history", zr.WaterHistory)
		})
	})
	return r
}

// restrictEndDatedMiddleware will return a 400 response if the requested Zone is end-dated
func (zr ZonesResource) restrictEndDatedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zone := r.Context().Value(zoneCtxKey).(*pkg.Zone)

		if zone.EndDated() {
			render.Render(w, r, ErrInvalidRequest(fmt.Errorf("resource not available for end-dated Zone")))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// zoneContextMiddleware middleware is used to load a Zone object from the URL
// parameters passed through as the request. In case the Zone could not be found,
// we stop here and return a 404.
func (zr ZonesResource) zoneContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

		zoneID, err := xid.FromString(chi.URLParam(r, zonePathParam))
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		zone := garden.Zones[zoneID]
		if zone == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		// t := context.WithValue(r.Context(), gardenCtxKey, garden)
		ctx := context.WithValue(r.Context(), zoneCtxKey, zone)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// zoneAction reads a ZoneAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (zr ZonesResource) zoneAction(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	zone := r.Context().Value(zoneCtxKey).(*pkg.Zone)

	action := &ZoneActionRequest{}
	if err := render.Bind(r, action); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Received request to perform action on Zone %s\n", zone.ID)
	if err := action.Execute(garden, zone, zr.scheduler); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusAccepted)
	render.DefaultResponder(w, r, nil)
}

// getZone simply returns the Zone requested by the provided ID
func (zr ZonesResource) getZone(w http.ResponseWriter, r *http.Request) {
	zone := r.Context().Value(zoneCtxKey).(*pkg.Zone)
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	zoneResponse := zr.NewZoneResponse(r.Context(), garden, zone)
	if err := render.Render(w, r, zoneResponse); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// updateZone will change any specified fields of the Zone and save it
func (zr ZonesResource) updateZone(w http.ResponseWriter, r *http.Request) {
	zone := r.Context().Value(zoneCtxKey).(*pkg.Zone)
	request := &UpdateZoneRequest{}
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Read the request body into existing zone to overwrite fields
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	zone.Patch(request.Zone)

	// Save the Zone
	if err := zr.storageClient.SaveZone(garden.ID, zone); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Update the water schedule for the Zone if it was changed or EndDate is removed
	if request.Zone.WaterSchedule != nil || request.Zone.EndDate == nil {
		if err := zr.scheduler.ResetWaterSchedule(garden, zone); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
	}

	if err := render.Render(w, r, zr.NewZoneResponse(r.Context(), garden, zone)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDateZone will mark the Zone's end date as now and save it
func (zr ZonesResource) endDateZone(w http.ResponseWriter, r *http.Request) {
	zone := r.Context().Value(zoneCtxKey).(*pkg.Zone)
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	now := time.Now()

	// Unable to delete Zone with associated Plants
	if numPlants := len(garden.PlantsByZone(zone.ID, false)); numPlants > 0 {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("unable to delete Zone with %d Plants", numPlants)))
		return
	}

	// Permanently delete the Zone if it is already end-dated
	if zone.EndDated() {
		if err := zr.storageClient.DeleteZone(garden.ID, zone.ID); err != nil {
			render.Render(w, r, InternalServerError(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(""))
		return
	}

	// Set end date of Zone and save
	zone.EndDate = &now
	if err := zr.storageClient.SaveZone(garden.ID, zone); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	// Remove scheduled water Job
	if err := zr.scheduler.RemoveJobsByID(zone.ID); err != nil {
		logger.Errorf("Unable to remove water Job for Zone %s: %v", zone.ID.String(), err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, zr.NewZoneResponse(r.Context(), garden, zone)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllZones will return a list of all Zones
func (zr ZonesResource) getAllZones(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	zones := []*pkg.Zone{}
	for _, z := range garden.Zones {
		if getEndDated || (z.EndDate == nil || z.EndDate.After(time.Now())) {
			zones = append(zones, z)
		}
	}
	if err := render.Render(w, r, zr.NewAllZonesResponse(r.Context(), zones, garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// createZone will create a new Zone resource
func (zr ZonesResource) createZone(w http.ResponseWriter, r *http.Request) {
	request := &ZoneRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	zone := request.Zone

	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Validate that adding a Zone does not exceed Garden.MaxZones
	if garden.NumZones()+1 > *garden.MaxZones {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("adding a Zone would exceed Garden's max_zones=%d", *garden.MaxZones)))
		return
	}
	// Validate that ZonePosition works for a Garden with MaxZones (remember ZonePosition is zero-indexed)
	if *zone.Position >= *garden.MaxZones {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("position invalid for Garden with max_zones=%d", *garden.MaxZones)))
		return
	}

	// Assign values to fields that may not be set in the request
	zone.ID = xid.New()
	if zone.CreatedAt == nil {
		now := time.Now()
		zone.CreatedAt = &now
	}

	// Start water schedule
	if err := zr.scheduler.ScheduleWaterAction(garden, zone); err != nil {
		logger.Errorf("Unable to add water Job for Zone %v: %v", zone.ID, err)
	}

	// Save the Zone
	if err := zr.storageClient.SaveZone(garden.ID, zone); err != nil {
		logger.Error("Error saving zone: ", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, zr.NewZoneResponse(r.Context(), garden, zone)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// WaterHistory responds with the Zone's recent water events read from InfluxDB
func (zr ZonesResource) WaterHistory(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	zone := r.Context().Value(zoneCtxKey).(*pkg.Zone)

	// Read query parameters and set default values
	timeRangeString := r.URL.Query().Get("range")
	if len(timeRangeString) == 0 {
		timeRangeString = "72h"
	}
	limitString := r.URL.Query().Get("limit")
	if len(limitString) == 0 {
		limitString = "0"
	}

	// Parse query parameter strings into correct types
	timeRange, err := time.ParseDuration(timeRangeString)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	limit, err := strconv.ParseUint(limitString, 0, 64)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	history, err := zr.getWaterHistory(r.Context(), zone, garden, timeRange, limit)
	if err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}
	if err := render.Render(w, r, NewZoneWaterHistoryResponse(history)); err != nil {
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
