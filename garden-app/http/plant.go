package http

import (
	"context"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/calvinmclean/automated-garden/garden-app/api/actions"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

// AllPlantsResponse is a simple struct being used to render and return a list of all Plants
type AllPlantsResponse struct {
	Plants []*api.Plant `json:"plants"`
	plants map[string]*api.Plant
}

// Render will take the map of Plants and convert it to a list for a more RESTy response
func (pr *AllPlantsResponse) Render(w http.ResponseWriter, r *http.Request) error {
	for _, p := range pr.plants {
		pr.Plants = append(pr.Plants, p)
	}
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
	})
}

// plantContextMiddleware middleware is used to load a Plant object from the URL
// parameters passed through as the request. In case the Plant could not be found,
// we stop here and return a 404.
func plantContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plantID := chi.URLParam(r, "plantID")

		plant, ok := plants[plantID]
		if !ok {
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
	data.Execute(plant)
}

// getPlant simply returns the Plant requested by the provided ID
func getPlant(w http.ResponseWriter, r *http.Request) {
	plant := r.Context().Value("plant").(*api.Plant)
	if err := render.Render(w, r, plant); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllPlants will return a list of all Plants
func getAllPlants(w http.ResponseWriter, r *http.Request) {
	if err := render.Render(w, r, &AllPlantsResponse{plants: plants}); err != nil {
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

	// Assign new unique ID to plant
	plant.ID = xid.New().String()

	// Add to my map
	plants[plant.ID] = plant

	render.Status(r, http.StatusCreated)
	render.Render(w, r, plant)
}