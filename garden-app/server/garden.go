package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/go-chi/render"
)

const (
	gardenBasePath = "/gardens"
)

// GardensAPI encapsulates the structs and dependencies necessary for the "/gardens" API
// to function, including storage and configurating
type GardensAPI struct {
	*babyapi.API[*pkg.Garden]

	storageClient  *storage.Client
	influxdbClient influxdb.Client
	worker         *worker.Worker
	config         Config
}

func NewGardenAPI() *GardensAPI {
	api := &GardensAPI{}

	api.API = babyapi.NewAPI("Gardens", gardenBasePath, func() *pkg.Garden { return &pkg.Garden{} })
	api.SetResponseWrapper(func(g *pkg.Garden) render.Renderer {
		return api.NewGardenResponse(g)
	})
	api.SetGetAllResponseWrapper(func(gardens []*pkg.Garden) render.Renderer {
		resp := AllGardensResponse{ResourceList: babyapi.ResourceList[*GardenResponse]{}}

		for _, g := range gardens {
			resp.ResourceList.Items = append(resp.ResourceList.Items, api.NewGardenResponse(g))
		}

		return resp
	})

	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)

	api.AddCustomIDRoute(http.MethodPost, "/action", api.GetRequestedResourceAndDo(api.gardenAction))

	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		switch r.URL.Query().Get("type") {
		case "create_modal":
			return api.gardenModalRenderer(r.Context(), &pkg.Garden{
				ID: babyapi.NewID(),
			})
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodGet, "/components", api.GetRequestedResourceAndDo(func(r *http.Request, g *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
		switch r.URL.Query().Get("type") {
		case "edit_modal":
			return api.gardenModalRenderer(r.Context(), g), nil
		default:
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.SetBeforeDelete(func(_ http.ResponseWriter, r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		gardenID := api.GetIDParam(r)

		// Don't allow end-dating a Garden with active Zones
		numZones, err := api.numZones(r.Context(), gardenID)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("error getting number of Zones for garden: %w", err))
		}
		if numZones > 0 {
			err := errors.New("unable to end-date Garden with active Zones")
			logger.Error("unable to end-date Garden", "error", err)
			return babyapi.ErrInvalidRequest(err)
		}

		return nil
	})

	api.SetAfterDelete(func(_ http.ResponseWriter, r *http.Request) *babyapi.ErrResponse {
		logger := babyapi.GetLoggerFromContext(r.Context())
		gardenID := api.GetIDParam(r)

		// Remove scheduled light actions
		logger.Info("removing scheduled LightActions for Garden")
		if err := api.worker.RemoveJobsByID(gardenID); err != nil {
			logger.Error("unable to remove scheduled LightActions", "error", err)
			return babyapi.InternalServerError(err)
		}
		return nil
	})

	api.ApplyExtension(extensions.HTMX[*pkg.Garden]{})

	return api
}

func (api *GardensAPI) gardenModalRenderer(ctx context.Context, g *pkg.Garden) render.Renderer {
	notificationClients, err := api.storageClient.NotificationClientConfigs.GetAll(ctx, nil)
	if err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error getting all notification clients to create garden modal: %w", err))
	}

	slices.SortFunc(notificationClients, func(nc1 *notifications.Client, nc2 *notifications.Client) int {
		return strings.Compare(nc1.Name, nc2.Name)
	})

	fmt.Println("CLIENTS:", len(notificationClients), notificationClients)
	if len(notificationClients) == 1 {
		fmt.Println("X:", *notificationClients[0])
	}

	return gardenModalTemplate.Renderer(struct {
		*pkg.Garden
		NotificationClients []*notifications.Client
	}{g, notificationClients})
}

func (api *GardensAPI) setup(config Config, storageClient *storage.Client, influxdbClient influxdb.Client, worker *worker.Worker) error {
	api.storageClient = storageClient
	api.influxdbClient = influxdbClient
	api.worker = worker
	api.config = config

	api.SetStorage(api.storageClient.Gardens)

	// Initialize light schedules for all Gardens
	allGardens, err := api.storageClient.Gardens.GetAll(context.Background(), nil)
	if err != nil {
		return err
	}
	for _, g := range allGardens {
		if g.EndDated() || g.LightSchedule == nil {
			continue
		}
		err = api.worker.ScheduleLightActions(g)
		if err != nil {
			return fmt.Errorf("unable to schedule LightAction for Garden %v: %v", g.ID, err)
		}
	}

	return nil
}

func (api *GardensAPI) onCreateOrUpdate(_ http.ResponseWriter, r *http.Request, garden *pkg.Garden) *babyapi.ErrResponse {
	logger := babyapi.GetLoggerFromContext(r.Context())

	numZones, err := api.numZones(r.Context(), garden.ID.String())
	if err != nil {
		return babyapi.InternalServerError(err)
	}
	if *garden.MaxZones < numZones {
		return babyapi.ErrInvalidRequest(fmt.Errorf("unable to set max_zones less than current num_zones=%d", numZones))
	}

	// If LightSchedule is empty, remove the scheduled Job
	if garden.LightSchedule == nil {
		logger.Info("removing LightSchedule")
		if err := api.worker.RemoveJobsByID(garden.ID.String()); err != nil {
			logger.Error("unable to remove LightSchedule for Garden", "error", err)
			return babyapi.InternalServerError(err)
		}
	}

	if garden.LightSchedule != nil {
		// Validate NotificationClient exists
		if garden.LightSchedule.NotificationClientID != nil {
			apiErr := checkNotificationClientExists(r.Context(), api.storageClient, *garden.LightSchedule.NotificationClientID)
			if apiErr != nil {
				return apiErr
			}
		}

		// Update the light schedule for the Garden (if it exists)
		logger.Info("updating/resetting LightSchedule for Garden")
		if err := api.worker.ResetLightSchedule(garden); err != nil {
			logger.Error("unable to update/reset LightSchedule", "light_schedule", garden.LightSchedule, "error", err)
			return babyapi.InternalServerError(err)
		}
	}

	return nil
}

// gardenAction reads a GardenAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (api *GardensAPI) gardenAction(r *http.Request, garden *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute GardenAction")

	if garden.EndDated() {
		return nil, babyapi.ErrInvalidRequest(errors.New("unable to execute action on end-dated garden"))
	}

	gardenAction := &action.GardenAction{}
	if err := render.Bind(r, gardenAction); err != nil {
		logger.Error("invalid request for GardenAction", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("garden action", "action", gardenAction)

	if err := api.worker.ExecuteGardenAction(garden, gardenAction); err != nil {
		logger.Error("unable to execute GardenAction", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusAccepted)
	return &GardenActionResponse{}, nil
}

func checkNotificationClientExists(ctx context.Context, storageClient *storage.Client, id string) *babyapi.ErrResponse {
	_, err := storageClient.NotificationClientConfigs.Get(ctx, id)
	if err != nil {
		err = fmt.Errorf("error getting NotificationClient with ID %q: %w", id, err)

		if errors.Is(err, babyapi.ErrNotFound) {
			return babyapi.ErrInvalidRequest(err)
		}
		return babyapi.InternalServerError(err)
	}

	return nil
}
