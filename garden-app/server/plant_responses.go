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
	gardenPath := fmt.Sprintf("%s/%s", gardenBasePath, plant.GardenID)
	links = append(links,
		Link{
			"self",
			fmt.Sprintf("%s%s/%s", gardenPath, plantBasePath, plant.ID),
		},
		Link{
			"garden",
			gardenPath,
		},
	)
	if !plant.EndDated() {
		links = append(links,
			Link{
				"actions",
				fmt.Sprintf("%s%s/%s/actions", gardenPath, plantBasePath, plant.ID),
			},
			Link{
				"history",
				fmt.Sprintf("%s%s/%s/history", gardenPath, plantBasePath, plant.ID),
			},
		)
	}
	return &PlantResponse{
		plant,
		moisture,
		pr.getNextWateringTime(plant),
		links,
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (p *PlantResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// PlantWateringHistoryResponse wraps a slice of WateringHistory structs plus some aggregate stats for an HTTP response
type PlantWateringHistoryResponse struct {
	History []pkg.WateringHistory `json:"history"`
	Count   int                   `json:"count"`
	Average string                `json:"average"`
	Total   string                `json:"total"`
}

// NewPlantWateringHistoryResponse creates a response by creating some basic statistics about a list of history events
func NewPlantWateringHistoryResponse(history []pkg.WateringHistory) PlantWateringHistoryResponse {
	total := 0
	for _, h := range history {
		total += h.WateringAmount
	}
	// The average needs to be parsed as a string because I do not want to convert to int and lose precision
	average, err := time.ParseDuration(fmt.Sprintf("%fms", float64(total)/float64(len(history))))
	if err != nil {
		average = -1
	}
	return PlantWateringHistoryResponse{
		History: history,
		Count:   len(history),
		Average: average.String(),
		Total:   time.Duration(total * int(time.Millisecond)).String(),
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (resp PlantWateringHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
