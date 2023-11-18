package server

import (
	_ "embed"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GardenResponse is used to represent a Garden in the response body with the additional Moisture data
// and hypermedia Links fields
type GardenResponse struct {
	*pkg.Garden
	NextLightAction         *NextLightAction         `json:"next_light_action,omitempty"`
	Health                  *pkg.GardenHealth        `json:"health,omitempty"`
	TemperatureHumidityData *TemperatureHumidityData `json:"temperature_humidity_data,omitempty"`
	NumPlants               uint                     `json:"num_plants"`
	NumZones                uint                     `json:"num_zones"`
	Plants                  Link                     `json:"plants"`
	Zones                   Link                     `json:"zones"`
	Links                   []Link                   `json:"links,omitempty"`

	gr *GardensResource
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
func (gr *GardensResource) NewGardenResponse(garden *pkg.Garden, links ...Link) *GardenResponse {
	return &GardenResponse{
		Garden: garden,
		Links:  links,

		gr: gr,
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (g *GardenResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	plantsPath := fmt.Sprintf("%s/%s%s", gardenBasePath, g.Garden.ID, plantBasePath)
	zonesPath := fmt.Sprintf("%s/%s%s", gardenBasePath, g.Garden.ID, zoneBasePath)

	g.NumPlants = g.Garden.NumPlants()
	g.NumZones = g.Garden.NumZones()
	g.Plants = Link{"collection", plantsPath}
	g.Zones = Link{"collection", zonesPath}
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
			"plants",
			plantsPath,
		},
		Link{
			"zones",
			zonesPath,
		},
		Link{
			"action",
			fmt.Sprintf("%s/%s/action", gardenBasePath, g.Garden.ID),
		},
	)

	g.Health = g.Garden.Health(ctx, g.gr.influxdbClient)

	if g.Garden.LightSchedule != nil {
		nextOnTime := g.gr.worker.GetNextLightTime(g.Garden, pkg.LightStateOn)
		nextOffTime := g.gr.worker.GetNextLightTime(g.Garden, pkg.LightStateOff)
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
		t, h, err := g.gr.influxdbClient.GetTemperatureAndHumidity(ctx, g.Garden.TopicPrefix)
		if err != nil {
			logger := getLoggerFromContext(ctx).WithField(gardenIDLogField, g.Garden.ID.String())
			logger.WithError(err).Error("error getting temperature and humidity data: %w", err)
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
	Gardens []*GardenResponse `json:"gardens"`
}

//go:embed templates/gardens.html
var gardensHTML []byte

func (gr *AllGardensResponse) HTML() string {
	return string(gardensHTML)
}

// NewAllGardensResponse will create an AllGardensResponse from a list of Gardens
func (gr *GardensResource) NewAllGardensResponse(gardens []*pkg.Garden) *AllGardensResponse {
	gardenResponses := []*GardenResponse{}
	for _, g := range gardens {
		gardenResponses = append(gardenResponses, gr.NewGardenResponse(g))
	}
	return &AllGardensResponse{gardenResponses}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (agr *AllGardensResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	for _, g := range agr.Gardens {
		err := g.Render(nil, r)
		if err != nil {
			return fmt.Errorf("error rendering garden: %w", err)
		}
	}
	return nil
}
