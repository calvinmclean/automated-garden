package server

import (
	"context"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/calvinmclean/automated-garden/garden-app/api/actions"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

// AllPlantsResponse is a simple struct being used to render and return a list of all Plants
type AllPlantsResponse struct {
	Plants []*api.Plant `json:"plants"`
}

// Render will take the map of Plants and convert it to a list for a more RESTy response
func (pr *AllPlantsResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// plantRouter creates all of the routing that is prefixed by "/plant" for interacting
// with Plant resources
func plantRouter(r chi.Router) {
	r.Post("/", createPlant)
	r.Get("/", getAllPlants)

	r.Route("/{plantID}", func(r chi.Router) {
		r.Use(plantContextMiddleware)

		r.Post("/", plantAction)
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

	data := &actions.AggregateAction{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Recieved request to perform action on Plant %s\n", plant.ID)
	if err := data.Execute(plant); err != nil {
		render.Render(w, r, ServerError(err))
		return
	}

	// Save the Plant in case anything was changed
	// TODO: consider giving the action the ability to use the storage client
	if err := storageClient.SavePlant(plant); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, ServerError(err))
		return
	}
}

// getPlant simply returns the Plant requested by the provided ID
func getPlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)
	if err := render.Render(w, r, plant); err != nil {
		render.Render(w, r, ErrRender(err))
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

	if err := render.Render(w, r, plant); err != nil {
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

	if err := render.Render(w, r, plant); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllPlants will return a list of all Plants
func getAllPlants(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	plants := storageClient.GetPlants(getEndDated)
	if err := render.Render(w, r, &AllPlantsResponse{plants}); err != nil {
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
	if err := render.Render(w, r, plant); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}
