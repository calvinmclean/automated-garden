package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
)

// GardenResponse is used to represent a Garden in the response body with the additional Moisture data
// and hypermedia Links fields
type GardenResponse struct {
	*pkg.Garden
	NextLightAction         *NextLightAction         `json:"next_light_action,omitempty"`
	Health                  *pkg.GardenHealth        `json:"health,omitempty"`
	TemperatureHumidityData *TemperatureHumidityData `json:"temperature_humidity_data,omitempty"`
	NumZones                uint                     `json:"num_zones"`
	Links                   []Link                   `json:"links,omitempty"`

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

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (g *GardenResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	zonesPath := fmt.Sprintf("%s/%s%s", gardenBasePath, g.Garden.ID, zoneBasePath)

	var err error
	g.NumZones, err = g.api.numZones(g.ID.String())
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

	g.Health = g.Garden.Health(ctx, g.api.influxdbClient)

	if g.Garden.LightSchedule != nil {
		nextOnTime := g.api.worker.GetNextLightTime(g.Garden, pkg.LightStateOn)
		nextOffTime := g.api.worker.GetNextLightTime(g.Garden, pkg.LightStateOff)
		if nextOnTime != nil && nextOffTime != nil {
			// If the nextOnTime is before the nextOffTime, that means the next light action will be the ON action
			if nextOnTime.Before(*nextOffTime) {
				g.NextLightAction = &NextLightAction{
					Time:  nextOnTime,
					State: pkg.LightStateOn,
				}
			} else {
				g.NextLightAction = &NextLightAction{
					Time:  nextOffTime,
					State: pkg.LightStateOff,
				}
			}
		} else if nextOnTime != nil {
			g.NextLightAction = &NextLightAction{
				Time:  nextOnTime,
				State: pkg.LightStateOn,
			}
		} else if nextOffTime != nil {
			g.NextLightAction = &NextLightAction{
				Time:  nextOffTime,
				State: pkg.LightStateOff,
			}
		}
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

	return nil
}

// AllGardensResponse is a simple struct being used to render and return a list of all Gardens
type AllGardensResponse struct {
	babyapi.ResourceList[*GardenResponse]
}

func (agr AllGardensResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return agr.ResourceList.Render(w, r)
}

func (agr AllGardensResponse) HTML(r *http.Request) string {
	slices.SortFunc(agr.Items, func(g *GardenResponse, h *GardenResponse) int {
		return strings.Compare(g.Name, h.Name)
	})

	return gardensTemplate.Render(r, agr)
}

func (api *GardensAPI) getAllZones(gardenID string, getEndDated bool) ([]*pkg.Zone, error) {
	zones, err := api.storageClient.Zones.GetAll(func(z *pkg.Zone) bool {
		gardenIDFilter := filterZoneByGardenID(gardenID)
		endDateFilter := storage.FilterEndDated[*pkg.Zone](getEndDated)

		return gardenIDFilter(z) && endDateFilter(z)
	})
	if err != nil {
		return nil, fmt.Errorf("error getting Zones for Garden: %w", err)
	}

	return zones, nil
}

// NumZones returns the number of non-end-dated Zones that are part of this Garden
func (api *GardensAPI) numZones(gardenID string) (uint, error) {
	zones, err := api.getAllZones(gardenID, false)
	if err != nil {
		return 0, err
	}

	return uint(len(zones)), nil
}

type GardenActionResponse struct{}

func (*GardenActionResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
