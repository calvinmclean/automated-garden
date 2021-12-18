package server

import (
	"context"
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
func (pr PlantsResource) NewAllPlantsResponse(ctx context.Context, plants []*pkg.Plant, garden *pkg.Garden) *AllPlantsResponse {
	plantResponses := []*PlantResponse{}
	for _, p := range plants {
		plantResponses = append(plantResponses, pr.NewPlantResponse(ctx, garden, p))
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
func (pr PlantsResource) NewPlantResponse(ctx context.Context, garden *pkg.Garden, plant *pkg.Plant, links ...Link) *PlantResponse {
	gardenPath := fmt.Sprintf("%s/%s", gardenBasePath, garden.ID)
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
				"action",
				fmt.Sprintf("%s%s/%s/action", gardenPath, plantBasePath, plant.ID),
			},
			Link{
				"history",
				fmt.Sprintf("%s%s/%s/history", gardenPath, plantBasePath, plant.ID),
			},
		)
	}
	moisture := 0.0
	var err error
	if plant.WaterSchedule.MinimumMoisture > 0 && garden != nil {
		moisture, err = pr.getMoisture(ctx, garden, plant)
		if err != nil {
			// Log moisture error but do not return an error since this isn't critical information
			logger.Errorf("unable to get moisture of Plant %v: %v", plant.ID, err)
		}
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
	total := time.Duration(0)
	for _, h := range history {
		amountDuration, _ := time.ParseDuration(h.Duration)
		total += amountDuration
	}
	count := len(history)
	average := time.Duration(0)
	if count != 0 {
		average = time.Duration(int(total) / len(history))
	}
	return PlantWateringHistoryResponse{
		History: history,
		Count:   count,
		Average: average.String(),
		Total:   time.Duration(total).String(),
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (resp PlantWateringHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
