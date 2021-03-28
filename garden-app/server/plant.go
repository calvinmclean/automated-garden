package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

// AllPlantsResponse is a simple struct being used to render and return a list of all Plants
type AllPlantsResponse struct {
	Plants []*PlantResponse `json:"plants"`
}

// NewAllPlantsResponse will create an AllPlantsResponse from a list of Plants
func NewAllPlantsResponse(plants []*api.Plant) *AllPlantsResponse {
	plantResponses := []*PlantResponse{}
	for _, p := range plants {
		plantResponses = append(plantResponses, NewPlantResponse(p, 0))
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
	Moisture float64 `json:"moisture,omitempty"`
	Links    []Link  `json:"links,omitempty"`
}

// NewPlantResponse creates a self-referencing PlantResponse
func NewPlantResponse(plant *api.Plant, moisture float64, links ...Link) *PlantResponse {
	return &PlantResponse{
		plant,
		moisture,
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

// Link is used for HATEOAS-style RESP hypermedia
type Link struct {
	Rel  string `json:"rel"`
	HRef string `json:"href"`
}

var moistureCache = map[xid.ID]float64{}

// plantRouter creates all of the routing that is prefixed by "/plant" for interacting
// with Plant resources
func plantRouter(r chi.Router) {
	r.Post("/", createPlant)
	r.Get("/", getAllPlants)

	r.Route("/{plantID}", func(r chi.Router) {
		r.Use(plantContextMiddleware)

		r.Post("/action", plantAction)
		r.Get("/", getPlant)
		r.Put("/", updatePlant)
		r.Delete("/", endDatePlant)
	})
}

// plantContextMiddleware middleware is used to load a Plant object from the URL
// parameters passed through as the request. In case the Plant could not be found,
// we stop here and return a 404.
func plantContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Convert ID string to xid
		id, err := xid.FromString(chi.URLParam(r, "plantID"))
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		plant, err := storageClient.GetPlant(id)
		if err != nil {
			render.Render(w, r, ServerError(err))
			return
		}
		if plant == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		ctx := context.WithValue(r.Context(), "plant", plant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// plantAction reads an AggregateAction request and uses it to execute one of the actions
// that is available to run against a Plant. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func plantAction(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)

	data := &api.AggregateAction{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Received request to perform action on Plant %s\n", plant.ID)
	if err := data.Execute(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	// Save the Plant in case anything was changed (watering a plant might change the skip_count field)
	// TODO: consider giving the action the ability to use the storage client
	if err := storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, ServerError(err))
		return
	}

	render.Status(r, http.StatusAccepted)
	render.DefaultResponder(w, r, nil)
}

// getPlant simply returns the Plant requested by the provided ID
func getPlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)
	moisture, cached := moistureCache[plant.ID]
	plantResponse := NewPlantResponse(plant, moisture)
	if err := render.Render(w, r, plantResponse); err != nil {
		render.Render(w, r, ErrRender(err))
	}

	// If moisture was not already cached (and plant has moisture sensor), asynchronously get it and cache it
	// Otherwise, clear cache
	if !cached && plant.WateringStrategy.MinimumMoisture > 0 {
		go getAndCacheMoisture(plant)
	} else {
		delete(moistureCache, plant.ID)
	}
}

// updatePlant will change any specified fields of the Plant and save it
func updatePlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)

	// Read the request body into existing plant to overwrite fields
	if err := render.Bind(r, plant); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Update the watering schedule for the Plant
	if err := resetWateringSchedule(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	// Save the Plant
	if err := storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	if err := render.Render(w, r, NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it
func endDatePlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)

	// Set end date of Plant and save
	now := time.Now()
	plant.EndDate = &now
	if err := storageClient.SavePlant(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	// Remove scheduled watering Job
	if err := removeWateringSchedule(plant); err != nil {
		logger.Errorf("Unable to remove watering Job for Plant %s: %v", plant.ID.String(), err)
		render.Render(w, r, ServerError(err))
		return
	}

	if err := render.Render(w, r, NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllPlants will return a list of all Plants
func getAllPlants(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	plants := storageClient.GetPlants(getEndDated)
	if err := render.Render(w, r, NewAllPlantsResponse(plants)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// createPlant will create a new Plant resource
func createPlant(w http.ResponseWriter, r *http.Request) {
	plant := &api.Plant{}
	if err := render.Bind(r, plant); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Assign new unique ID and StartDate to plant
	plant.ID = xid.New()
	if plant.StartDate == nil {
		now := time.Now()
		plant.StartDate = &now
	}

	// Start watering schedule
	if err := addWateringSchedule(plant); err != nil {
		logger.Errorf("Unable to add watering Job for Plant %s: %v", plant.ID.String(), err)
	}

	// Save the Plant
	if err := storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, ServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, NewPlantResponse(plant, 0)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

func getAndCacheMoisture(plant *api.Plant) {
	moisture, err := plant.GetMoisture()
	if err != nil {
		logger.Errorf("unable to get moisture of Plant %v: %v", plant.ID, err)
	}
	moistureCache[plant.ID] = moisture
}
