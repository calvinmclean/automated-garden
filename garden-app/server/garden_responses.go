package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GardenResponse is used to represent a Garden in the response body with the additional Moisture data
// and hypermedia Links fields
type GardenResponse struct {
	*pkg.Garden
	NextLightAction *NextLightAction `json:"next_light_action,omitempty"`
	NumPlants       int              `json:"num_plants"`
	Plants          Link             `json:"plants"`
	Links           []Link           `json:"links,omitempty"`
}

// NextLightAction contains the time and state for the next scheduled LightAction
type NextLightAction struct {
	Time  *time.Time `json:"time"`
	State string     `json:"state"`
}

// NewGardenResponse creates a self-referencing GardenResponse
func (gr GardensResource) NewGardenResponse(garden *pkg.Garden, links ...Link) *GardenResponse {
	plantsPath := fmt.Sprintf("%s/%s%s", gardenBasePath, garden.ID, plantBasePath)
	response := &GardenResponse{
		Garden:    garden,
		NumPlants: garden.NumPlants(),
		Plants:    Link{"collection", plantsPath},
	}
	response.Links = append(links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", gardenBasePath, garden.ID),
		},
	)
	if !garden.EndDated() {
		response.Links = append(response.Links,
			Link{
				"health",
				fmt.Sprintf("%s/%s/health", gardenBasePath, garden.ID),
			},
			Link{
				"plants",
				plantsPath,
			},
			Link{
				"action",
				fmt.Sprintf("%s/%s/action", gardenBasePath, garden.ID),
			},
		)

		if garden.LightSchedule != nil {
			nextOnTime := gr.getNextLightTime(garden, pkg.StateOn)
			nextOffTime := gr.getNextLightTime(garden, pkg.StateOff)
			if nextOnTime != nil && nextOffTime != nil {
				// If the nextOnTime is before the nextOffTime, that means the next light action will be the ON action
				if nextOnTime.Before(*nextOffTime) {
					response.NextLightAction = &NextLightAction{
						Time:  nextOnTime,
						State: pkg.StateOn,
					}
				} else {
					response.NextLightAction = &NextLightAction{
						Time:  nextOffTime,
						State: pkg.StateOff,
					}
				}
			}
		}
	}
	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (g *GardenResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// AllGardensResponse is a simple struct being used to render and return a list of all Gardens
type AllGardensResponse struct {
	Gardens []*GardenResponse `json:"gardens"`
}

// NewAllGardensResponse will create an AllGardensResponse from a list of Gardens
func (gr GardensResource) NewAllGardensResponse(gardens []*pkg.Garden) *AllGardensResponse {
	gardenResponses := []*GardenResponse{}
	for _, g := range gardens {
		gardenResponses = append(gardenResponses, gr.NewGardenResponse(g))
	}
	return &AllGardensResponse{gardenResponses}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (pr *AllGardensResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// GardenHealthResponse allpws for returning GardenHealth in an HTTP response
type GardenHealthResponse struct {
	pkg.GardenHealth
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (gh GardenHealthResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
