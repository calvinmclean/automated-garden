package server

import (
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GardenResponse is used to represent a Garden in the response body with the additional Moisture data
// and hypermedia Links fields
type GardenResponse struct {
	*pkg.Garden
	Plants Link   `json:"plants"`
	Links  []Link `json:"links,omitempty"`
}

// NewGardenResponse creates a self-referencing GardenResponse
func (gr GardensResource) NewGardenResponse(garden *pkg.Garden, links ...Link) *GardenResponse {
	plantsPath := fmt.Sprintf("%s/%s%s", gardenBasePath, garden.ID, plantBasePath)
	links = append(links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", gardenBasePath, garden.ID),
		},
	)
	if !garden.EndDated() {
		links = append(links,
			Link{
				"health",
				fmt.Sprintf("%s/%s/health", gardenBasePath, garden.ID),
			},
			Link{
				"plants",
				plantsPath,
			},
		)
	}
	return &GardenResponse{
		garden,
		Link{"collection", plantsPath},
		links,
	}
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
