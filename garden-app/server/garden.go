package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// PlantsResource encapsulates the structs and dependencies necessary for the "/plants" API
// to function, including storage, scheduling, and caching
type GardenResource struct {
	storageClient storage.Client
	config        Config
}

const (
	gardenBasePath  = "/gardens"
	gardenPathParam = "gardenName"
	gardenCtxKey    = "garden"
)

// NewGardenResource creates a new GardenResource
func NewGardenResource(config Config) (gr GardenResource, err error) {
	gr = GardenResource{
		config: config,
	}

	gr.storageClient, err = storage.NewStorageClient(config.StorageConfig)
	if err != nil {
		err = fmt.Errorf("unable to initialize storage client: %v", err)
		return
	}

	return
}

// routes creates all of the routing that is prefixed by "/plant" for interacting with Plant resources
func (gr GardenResource) routes(pr PlantsResource) chi.Router {
	r := chi.NewRouter()
	// r.Post("/", gr.createGarden)
	r.Get("/", gr.getAllGardens)
	r.Route(fmt.Sprintf("/{%s}", gardenPathParam), func(r chi.Router) {
		r.Use(gr.gardenContextMiddleware)

		// r.Get("/", gr.getGarden)
		// r.Patch("/", gr.updateGarden)
		// r.Delete("/", gr.endDateGarden)

		r.Mount(plantBasePath, pr.routes())
	})
	return r
}

// gardenContextMiddleware middleware is used to load a Garden object from the URL
// parameters passed through as the request. In case the Garden could not be found,
// we stop here and return a 404.
func (gr GardenResource) gardenContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gardenName := chi.URLParam(r, gardenPathParam)

		garden, err := gr.storageClient.GetGarden(gardenName)
		if err != nil {
			render.Render(w, r, ServerError(err))
			return
		}
		if garden == nil {
			render.Render(w, r, ErrNotFoundResponse)
			return
		}

		ctx := context.WithValue(r.Context(), gardenCtxKey, garden)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getAllGardens will return a list of all Gardens
func (pr GardenResource) getAllGardens(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	gardens, err := pr.storageClient.GetGardens(getEndDated)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.Render(w, r, pr.NewAllGardensResponse(gardens)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}
