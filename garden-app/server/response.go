package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/go-chi/render"
)

var (
	// ErrNotFoundResponse is a basic error response for missing resource
	ErrNotFoundResponse = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}
)

// ErrResponse is a struct used to organize HTTP error responses
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

// Render will verify and render the ErrResponse
func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

// ErrInvalidRequest creates a 400 ErrResponse for a bad request
func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

// ErrRender creates a 422 response for errors encountered while rendering a response
func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

// ServerError creates a generic 500 error for a server-side error
func ServerError(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 500,
		StatusText:     "Server Error.",
		ErrorText:      err.Error(),
	}
}

// AllPlantsResponse is a simple struct being used to render and return a list of all Plants
type AllPlantsResponse struct {
	Plants []*PlantResponse `json:"plants"`
}

// NewAllPlantsResponse will create an AllPlantsResponse from a list of Plants
func (pr PlantsResource) NewAllPlantsResponse(plants []*api.Plant) *AllPlantsResponse {
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
	*api.Plant
	Moisture         float64    `json:"moisture,omitempty"`
	NextWateringTime *time.Time `json:"next_watering_time,omitempty"`
	Links            []Link     `json:"links,omitempty"`
}

// NewPlantResponse creates a self-referencing PlantResponse
func (pr PlantsResource) NewPlantResponse(plant *api.Plant, moisture float64, links ...Link) *PlantResponse {
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

// Link is used for HATEOAS-style REST hypermedia
type Link struct {
	Rel  string `json:"rel"`
	HRef string `json:"href"`
}
