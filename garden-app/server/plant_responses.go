package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// AllPlantsResponse is a simple struct being used to render and return a list of all Plants
type AllPlantsResponse struct {
	Plants []*PlantResponse `json:"plants"`
}

// NewAllPlantsResponse will create an AllPlantsResponse from a list of Plants
func (pr PlantsResource) NewAllPlantsResponse(plants []*pkg.Plant) *AllPlantsResponse {
	plantResponses := []*PlantResponse{}
	for _, p := range plants {
		plantResponses = append(plantResponses, pr.NewPlantResponse(p, 0))
	}
	return &AllPlantsResponse{plantResponses}
}

// Render will take the map of Plants and convert it to a list for a more RESTy response
func (pr *AllPlantsResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// PlantResponse is used to represent a Plant in the response body with the additional Moisture data
// and hypermedia Links fields
type PlantResponse struct {
	*pkg.Plant
	Moisture         float64    `json:"moisture,omitempty"`
	NextWateringTime *time.Time `json:"next_watering_time,omitempty"`
	Links            []Link     `json:"links,omitempty"`
}

// NewPlantResponse creates a self-referencing PlantResponse
func (pr PlantsResource) NewPlantResponse(plant *pkg.Plant, moisture float64, links ...Link) *PlantResponse {
	return &PlantResponse{
		plant,
		moisture,
		pr.getNextWateringTime(plant),
		append(links, Link{
			"self",
			fmt.Sprintf("/plants/%s", plant.ID),
		}),
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (p *PlantResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
