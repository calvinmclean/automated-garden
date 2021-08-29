package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	gardenBasePath  = "/gardens"
	gardenPathParam = "gardenID"
	gardenCtxKey    = contextKey("garden")
)

// GardensResource encapsulates the structs and dependencies necessary for the "/gardens" API
// to function, including storage and configurating
type GardensResource struct {
	storageClient storage.Client
	config        Config
}

// NewGardenResource creates a new GardenResource
func NewGardenResource(config Config) (gr GardensResource, err error) {
	gr = GardensResource{
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
func (gr GardensResource) routes(pr PlantsResource) chi.Router {
	r := chi.NewRouter()
	r.Post("/", gr.createGarden)
	r.Get("/", gr.getAllGardens)
	r.Route(fmt.Sprintf("/{%s}", gardenPathParam), func(r chi.Router) {
		r.Use(gr.gardenContextMiddleware)

		r.Get("/", gr.getGarden)
		r.Patch("/", gr.updateGarden)
		r.Delete("/", gr.endDateGarden)

		r.Mount(plantBasePath, pr.routes())
	})
	return r
}

// gardenContextMiddleware middleware is used to load a Garden object from the URL
// parameters passed through as the request. In case the Garden could not be found,
// we stop here and return a 404.
func (gr GardensResource) gardenContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gardenID, err := xid.FromString(chi.URLParam(r, gardenPathParam))
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		garden, err := gr.storageClient.GetGarden(gardenID)
		if err != nil {
			render.Render(w, r, InternalServerError(err))
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

func (gr GardensResource) createGarden(w http.ResponseWriter, r *http.Request) {
	request := &GardenRequest{}
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	garden := request.Garden

	// Assign new unique ID and CreatedAt to garden
	garden.ID = xid.New()
	if garden.CreatedAt == nil {
		now := time.Now()
		garden.CreatedAt = &now
	}

	// Save the Garden
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		logger.Error("Error saving plant: ", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, gr.NewGardenResponse(garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getAllGardens will return a list of all Gardens
func (gr GardensResource) getAllGardens(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	gardens, err := gr.storageClient.GetGardens(getEndDated)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.Render(w, r, gr.NewAllGardensResponse(gardens)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// getGarden will return a garden by ID/name
func (gr GardensResource) getGarden(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)
	gardenResponse := gr.NewGardenResponse(garden)
	if err := render.Render(w, r, gardenResponse); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// endDatePlant will mark the Plant's end date as now and save it
func (gr GardensResource) endDateGarden(w http.ResponseWriter, r *http.Request) {
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Set end date of Garden and save
	now := time.Now()
	garden.EndDate = &now
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, gr.NewGardenResponse(garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// updateGarden updates any fields in the existing Garden from the request
func (gr GardensResource) updateGarden(w http.ResponseWriter, r *http.Request) {
	request := &GardenRequest{}
	garden := r.Context().Value(gardenCtxKey).(*pkg.Garden)

	// Read the request body into existing garden to overwrite fields
	if err := render.Bind(r, request); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Manually update garden fields that are allowed to be changed
	garden.Name = request.Name

	// Save the Garden
	if err := gr.storageClient.SaveGarden(garden); err != nil {
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, gr.NewGardenResponse(garden)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}
