package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/concurrent"
	"github.com/calvinmclean/babyapi"

	"github.com/go-chi/render"
)

const (
	// influxDBTimeout is the timeout for InfluxDB queries.
	influxDBTimeout = 3 * time.Second
)

// ActiveWatering contains information about a currently-watering zone
type ActiveWatering struct {
	ZoneName string
	Progress pkg.WaterHistoryProgress
}

// GardenResponse is used to represent a Garden in the response body with additional data
// and hypermedia Links fields
type GardenResponse struct {
	*pkg.Garden
	NextLightAction         *NextLightAction         `json:"next_light_action,omitempty"`
	Health                  *pkg.GardenHealth        `json:"health,omitempty"`
	TemperatureHumidityData *TemperatureHumidityData `json:"temperature_humidity_data,omitempty"`
	NumZones                uint                     `json:"num_zones"`
	Links                   []Link                   `json:"links,omitempty"`
	ActiveWatering          *ActiveWatering          `json:"-"` // HTML only
	WateringQueue           uint                     `json:"-"` // HTML only

	api *GardensAPI
}

// NextLightAction contains the time and state for the next scheduled LightAction
type NextLightAction struct {
	Time  *time.Time     `json:"time"`
	State pkg.LightState `json:"state"`
}

// TemperatureHumidityData has the temperature and humidity of the Garden
type TemperatureHumidityData struct {
	TemperatureCelsius float64 `json:"temperature_celsius"`
	HumidityPercentage float64 `json:"humidity_percentage"`
}

// NewGardenResponse creates a self-referencing GardenResponse
func (api *GardensAPI) NewGardenResponse(garden *pkg.Garden, links ...Link) *GardenResponse {
	return &GardenResponse{
		Garden: garden,
		Links:  links,
		api:    api,
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (g *GardenResponse) Render(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	zonesPath := fmt.Sprintf("%s/%s%s", gardenBasePath, g.Garden.ID, zoneBasePath)

	var err error
	g.NumZones, err = g.api.numZones(r.Context(), g.ID.String())
	if err != nil {
		return fmt.Errorf("error getting number of Zones for garden: %w", err)
	}
	g.Links = append(g.Links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", gardenBasePath, g.Garden.ID),
		},
	)

	if g.Garden.EndDated() {
		return nil
	}

	g.Links = append(g.Links,
		Link{
			"zones",
			zonesPath,
		},
		Link{
			"action",
			fmt.Sprintf("%s/%s/action", gardenBasePath, g.Garden.ID),
		},
	)

	logger := babyapi.GetLoggerFromContext(ctx)

	// By default, skip InfluxDB data fetching for fast page loads (lazy loading)
	// Set swap_data=true to fetch all InfluxDB data in a single request
	swapData := r.URL.Query().Get("swap_data") == "true"

	// Determine if we need to fetch InfluxDB data
	// For HTML: skip by default (lazy loading), fetch when swap_data=true
	// For JSON: always fetch health and sensor data, but not active watering
	isHTML := render.GetAcceptedContentType(r) == render.ContentTypeHTML
	needsInfluxData := !isHTML || (isHTML && swapData)

	// Fetch all InfluxDB-dependent data concurrently at the top level
	// This avoids nested concurrent execution and allows for a single timeout
	if needsInfluxData {
		// Only fetch active watering for HTML responses (where zones are displayed)
		// JSON API responses don't include active watering status
		fetchActiveWatering := isHTML && swapData
		g.fetchInfluxDBData(ctx, logger, fetchActiveWatering)
	}

	if render.GetAcceptedContentType(r) == render.ContentTypeHTML && r.Method == http.MethodPut {
		w.Header().Add("HX-Trigger", "newGarden")
	}

	if g.Garden.LightSchedule != nil {
		nextLightTime, nextLightState := g.Garden.LightSchedule.NextChange(clock.Now())
		g.NextLightAction = &NextLightAction{
			Time:  &nextLightTime,
			State: nextLightState,
		}

		loc := getLocationFromRequest(r)
		if loc == nil {
			loc = g.LightSchedule.StartTime.Time.Location()
		}

		offsetTime := g.NextLightAction.Time.In(loc)
		g.NextLightAction.Time = &offsetTime
	}

	return nil
}

// fetchInfluxDBData fetches all InfluxDB-dependent data for a garden concurrently.
// This includes health, temperature/humidity, and active watering status for all zones.
// All operations are run in a single concurrent batch.
func (g *GardenResponse) fetchInfluxDBData(ctx context.Context, logger *slog.Logger, includeActiveWatering bool) {
	// First, check cache for health data
	if g.api.healthCache != nil {
		cacheKey := g.Garden.GetID()
		if cachedHealth, found := g.api.healthCache.Get(cacheKey); found {
			g.Health = cachedHealth
		}
	}

	// Get zones first (this is from SQLite, not InfluxDB, so it's fast)
	var zones []*pkg.Zone
	if includeActiveWatering {
		var err error
		zones, err = g.api.getAllZones(ctx, g.ID.String(), false)
		if err != nil {
			logger.Warn("error getting zones for active watering check", "error", err)
			zones = nil
		}
	}

	// Build a single list of all InfluxDB tasks
	tasks := []concurrent.TaskFunc{
		{
			Name: "health",
			Fn: func(taskCtx context.Context) error {
				health := g.api.worker.GetGardenHealth(taskCtx, g.Garden)
				if health != nil {
					g.Health = health
					// Cache the health data
					if g.api.healthCache != nil {
						g.api.healthCache.Set(g.Garden.GetID(), health)
					}
				}
				return nil
			},
		},
	}

	// Add temperature/humidity task only if sensor is enabled
	if g.Garden.HasTemperatureHumiditySensor() {
		tasks = append(tasks, concurrent.TaskFunc{
			Name: "temperature-humidity",
			Fn: func(taskCtx context.Context) error {
				t, h, err := g.api.influxdbClient.GetTemperatureAndHumidity(taskCtx, g.Garden.TopicPrefix)
				if err != nil {
					return err
				}
				g.TemperatureHumidityData = &TemperatureHumidityData{
					TemperatureCelsius: t,
					HumidityPercentage: h,
				}
				return nil
			},
		})
	}

	// Add zone water history tasks to the same batch
	// Use a slice to collect results
	type zoneResult struct {
		zone     *pkg.Zone
		progress pkg.WaterHistoryProgress
	}
	results := make([]zoneResult, len(zones))

	for i, zone := range zones {
		i, zone := i, zone // capture loop variables
		tasks = append(tasks, concurrent.TaskFunc{
			Name: "zone-water-history-" + zone.GetID(),
			Fn: func(taskCtx context.Context) error {
				history, err := g.api.influxdbClient.GetWaterHistory(taskCtx, zone.GetID(), g.TopicPrefix, 72*time.Hour, 5)
				if err != nil {
					return err
				}
				slices.Reverse(history)
				results[i] = zoneResult{zone: zone, progress: pkg.CalculateWaterProgress(history)}
				return nil
			},
		})
	}

	// Execute ALL tasks concurrently with timeout (single level of concurrency)
	errors := concurrent.RunFuncs(ctx, influxDBTimeout, tasks)
	for taskName, err := range errors {
		logger.Warn("failed to fetch garden data", "task", taskName, "error", err)
	}

	// Process zone results after all tasks complete
	if includeActiveWatering {
		var activeZone *pkg.Zone
		var activeProgress pkg.WaterHistoryProgress
		var totalQueue uint

		for _, result := range results {
			if result.zone == nil {
				continue // skipped or errored
			}
			progress := result.progress

			// Check if this zone is currently watering (progress between 0 and 1)
			if progress.Progress > 0 && progress.Progress < 1.0 && activeZone == nil {
				activeZone = result.zone
				activeProgress = progress
			}

			// Accumulate queue count from all zones
			totalQueue += progress.Queue
		}

		// Always set total queue count
		g.WateringQueue = totalQueue

		// Set ActiveWatering if we found an actively watering zone
		if activeZone != nil {
			g.ActiveWatering = &ActiveWatering{
				ZoneName: activeZone.Name,
				Progress: activeProgress,
			}
		}
	}
}

// AllGardensResponse is a simple struct being used to render and return a list of all Gardens
type AllGardensResponse struct {
	babyapi.ResourceList[*GardenResponse]
}

func (agr AllGardensResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// Process all GardenResponse renders concurrently for better performance
	// This is especially important when each garden fetches data from InfluxDB
	ctx := r.Context()
	logger := babyapi.GetLoggerFromContext(ctx)

	tasks := make([]concurrent.TaskFunc, 0, len(agr.Items))
	for i := range agr.Items {
		item := agr.Items[i] // capture loop variable
		tasks = append(tasks, concurrent.TaskFunc{
			Name: "garden-" + item.GetID(),
			Fn: func(_ context.Context) error {
				// Use the original request context for rendering
				return item.Render(w, r)
			},
		})
	}

	// Execute all renders concurrently with a reasonable timeout
	// Using a longer timeout since this encompasses all garden data fetching
	errors := concurrent.RunFuncs(ctx, 10*time.Second, tasks)
	for taskName, err := range errors {
		logger.Warn("failed to render garden", "garden", taskName, "error", err)
	}

	return nil
}

func (g *GardenResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	// For swap_data, return just the data section (health, zones, watering) for lazy loading
	if r.URL.Query().Get("swap_data") == "true" {
		return gardenDataSectionTemplate.Render(r, g)
	}
	return gardenCardTemplate.Render(r, g)
}

func (agr AllGardensResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	slices.SortFunc(agr.Items, func(g *GardenResponse, h *GardenResponse) int {
		return strings.Compare(g.Name, h.Name)
	})

	if r.URL.Query().Get("refresh") == "true" {
		return gardensTemplate.Render(r, agr)
	}

	return gardensPageTemplate.Render(r, agr)
}

func (api *GardensAPI) getAllZones(ctx context.Context, gardenID string, getEndDated bool) ([]*pkg.Zone, error) {
	zones, err := api.storageClient.Zones.Search(ctx, gardenID, babyapi.EndDatedQueryParam(getEndDated))
	if err != nil {
		return nil, fmt.Errorf("error getting Zones for Garden: %w", err)
	}

	return zones, nil
}

// NumZones returns the number of non-end-dated Zones that are part of this Garden
func (api *GardensAPI) numZones(ctx context.Context, gardenID string) (uint, error) {
	zones, err := api.getAllZones(ctx, gardenID, false)
	if err != nil {
		return 0, err
	}

	return uint(len(zones)), nil
}

type GardenActionResponse struct{}

func (*GardenActionResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
