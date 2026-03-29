package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"

	"github.com/go-chi/render"
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

		api: api,
	}
}

// getActiveWatering calculates the active watering status for this garden
func (g *GardenResponse) getActiveWatering(ctx context.Context) {
	zones, err := g.api.getAllZones(ctx, g.ID.String(), false)
	if err != nil {
		return
	}

	var activeZone *pkg.Zone
	var activeProgress pkg.WaterHistoryProgress
	var totalQueue uint

	for _, zone := range zones {
		history, err := g.api.influxdbClient.GetWaterHistory(ctx, zone.GetID(), g.TopicPrefix, 72*time.Hour, 5)
		if err != nil {
			continue
		}

		// Reverse history to match chronological order expected by CalculateWaterProgress
		slices.Reverse(history)

		progress := pkg.CalculateWaterProgress(history)

		// Check if this zone is currently watering (progress between 0 and 1)
		if progress.Progress > 0 && progress.Progress < 1.0 && activeZone == nil {
			activeZone = zone
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

	g.Health = g.api.worker.GetGardenHealth(ctx, g.Garden)

	if g.Garden.LightSchedule != nil {
		nextLightTime, nextLightState := g.Garden.LightSchedule.NextChange(clock.Now())
		g.NextLightAction = &NextLightAction{
			Time:  &nextLightTime,
			State: nextLightState,
		}

		var loc *time.Location
		tzHeader := r.Header.Get("X-TZ-Offset")
		if tzHeader != "" {
			loc, err = pkg.TimeLocationFromOffset(tzHeader)
			if err != nil {
				return fmt.Errorf("error parsing timezone from header: %w", err)
			}
		}
		if loc == nil {
			loc = g.LightSchedule.StartTime.Time.Location()
		}

		offsetTime := g.NextLightAction.Time.In(loc)
		g.NextLightAction.Time = &offsetTime
	}

	if g.Garden.HasTemperatureHumiditySensor() {
		t, h, err := g.api.influxdbClient.GetTemperatureAndHumidity(ctx, g.Garden.TopicPrefix)
		if err != nil {
			logger := babyapi.GetLoggerFromContext(r.Context())
			logger.Error("error getting temperature and humidity data", "error", err)
			return nil
		}
		g.TemperatureHumidityData = &TemperatureHumidityData{
			TemperatureCelsius: t,
			HumidityPercentage: h,
		}
	}

	if render.GetAcceptedContentType(r) == render.ContentTypeHTML {
		if r.Method == http.MethodPut {
			w.Header().Add("HX-Trigger", "newGarden")
		}

		// Get active watering status for HTML responses
		g.getActiveWatering(ctx)
	}

	return nil
}

// AllGardensResponse is a simple struct being used to render and return a list of all Gardens
type AllGardensResponse struct {
	babyapi.ResourceList[*GardenResponse]
}

func (agr AllGardensResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return agr.ResourceList.Render(w, r)
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
