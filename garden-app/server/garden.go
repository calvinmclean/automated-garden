package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/cache"
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
	healthCache    *cache.Cache[*pkg.GardenHealth]
	zonesAPI       *ZonesAPI
}

func NewGardenAPI() *GardensAPI {
	api := &GardensAPI{}

	// Initialize health cache with 2-minute TTL
	api.healthCache = cache.New[*pkg.GardenHealth](2 * time.Minute)

	api.API = babyapi.NewAPI("Gardens", gardenBasePath, func() *pkg.Garden { return &pkg.Garden{} })
	api.SetResponseWrapper(func(g *pkg.Garden) render.Renderer {
		return api.NewGardenResponse(g)
	})
	api.SetSearchResponseWrapper(func(gardens []*pkg.Garden) render.Renderer {
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
				ID: NewID(),
			})
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodGet, "/components", api.GetRequestedResourceAndDo(func(_ http.ResponseWriter, r *http.Request, g *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
		switch r.URL.Query().Get("type") {
		case "edit_modal":
			return api.gardenModalRenderer(r.Context(), g), nil
		default:
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	// New unified gardens view endpoint
	api.AddCustomRoute(http.MethodGet, "/new", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		return api.getGardensNewResponse(r)
	}))

	// Lazy-load zones for a garden in the new unified view
	api.AddCustomIDRoute(http.MethodGet, "/new/zones", api.GetRequestedResourceAndDo(func(_ http.ResponseWriter, r *http.Request, g *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
		return api.getGardenNewZonesResponse(r, g)
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

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

func (api *GardensAPI) gardenModalRenderer(ctx context.Context, g *pkg.Garden) render.Renderer {
	notificationClients, err := api.storageClient.NotificationClientConfigs.Search(ctx, "", nil)
	if err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error getting all notification clients to create garden modal: %w", err))
	}

	slices.SortFunc(notificationClients, func(nc1 *notifications.Client, nc2 *notifications.Client) int {
		return strings.Compare(nc1.Name, nc2.Name)
	})

	return gardenModalTemplate.Renderer(struct {
		*pkg.Garden
		NotificationClients []*notifications.Client
	}{g, notificationClients})
}

func (api *GardensAPI) setup(config Config, storageClient *storage.Client, influxdbClient influxdb.Client, worker *worker.Worker, zonesAPI *ZonesAPI) error {
	api.storageClient = storageClient
	api.influxdbClient = influxdbClient
	api.worker = worker
	api.config = config
	api.zonesAPI = zonesAPI

	api.SetStorage(api.storageClient.Gardens)

	// Initialize light schedules for all Gardens
	allGardens, err := api.storageClient.Gardens.Search(context.Background(), "", nil)
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

	// Validate NotificationClient exists
	if garden.NotificationClientID != nil {
		apiErr := checkNotificationClientExists(r.Context(), api.storageClient, *garden.NotificationClientID)
		if apiErr != nil {
			return apiErr
		}
	}

	if garden.LightSchedule != nil {
		// Update the light schedule for the Garden (if it exists)
		logger.Info("updating/resetting LightSchedule for Garden")
		if err := api.worker.ResetLightSchedule(garden); err != nil {
			logger.Error("unable to update/reset LightSchedule", "light_schedule", garden.LightSchedule, "error", err)
			return babyapi.InternalServerError(err)
		}
	}

	if r.Method == http.MethodPost && r.URL.Query().Get("create_zones") == "true" {
		err = api.createZonesForGarden(r.Context(), garden)
		if err != nil {
			logger.Error("create zones for new Garden", "error", err)
			return babyapi.InternalServerError(err)
		}
	}

	return nil
}

func (api *GardensAPI) createZonesForGarden(ctx context.Context, g *pkg.Garden) error {
	for i := range *g.MaxZones {
		position := i
		now := clock.Now()
		z := &pkg.Zone{
			ID:        babyapi.NewID(),
			GardenID:  g.ID.ID,
			Name:      fmt.Sprintf("Zone %d", i+1),
			Position:  &position,
			CreatedAt: &now,
		}

		err := api.storageClient.Zones.Set(ctx, z)
		if err != nil {
			return fmt.Errorf("error storing zone %d: %w", i, err)
		}
	}

	return nil
}

// gardenAction reads a GardenAction request and uses it to execute one of the actions
// that is available to run against a Zone. This one endpoint is used for all the different
// kinds of actions so the action information is carried in the request body
func (api *GardensAPI) gardenAction(_ http.ResponseWriter, r *http.Request, garden *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
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

// getGardensNewResponse returns the unified gardens view with zones
func (api *GardensAPI) getGardensNewResponse(r *http.Request) render.Renderer {
	ctx := r.Context()
	logger := babyapi.GetLoggerFromContext(ctx)

	gardens, err := api.storageClient.Gardens.Search(ctx, "", nil)
	if err != nil {
		logger.Error("error searching gardens", "error", err)
		return babyapi.InternalServerError(fmt.Errorf("error getting gardens: %w", err))
	}

	response := GardensNewResponse{
		Items: make([]*GardenSectionResponse, 0, len(gardens)),
	}

	for _, garden := range gardens {
		if garden.EndDated() {
			continue
		}
		gardenResponse := api.NewGardenResponse(garden)
		// Render the garden response to populate NextLightAction and other computed fields
		err := gardenResponse.Render(nil, r)
		if err != nil {
			logger.Error("error rendering garden response", "error", err, "garden", garden.ID)
			continue
		}
		response.Items = append(response.Items, &GardenSectionResponse{
			GardenResponse: gardenResponse,
			Zones:          nil, // Zones are lazy-loaded
		})
	}

	return response
}

// getGardenNewZonesResponse returns just the zones grid for lazy loading
func (api *GardensAPI) getGardenNewZonesResponse(r *http.Request, garden *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
	ctx := r.Context()
	logger := babyapi.GetLoggerFromContext(ctx)

	zones, err := api.getAllZones(ctx, garden.ID.String(), false)
	if err != nil {
		logger.Error("error getting zones for garden", "error", err)
		return nil, babyapi.InternalServerError(fmt.Errorf("error getting zones: %w", err))
	}

	// Create zone responses using the zones API and render them fully
	zoneResponses := make([]*ZoneResponse, 0, len(zones))
	for _, zone := range zones {
		zoneResp := api.zonesAPI.NewZoneResponse(zone)
		// Render the zone to populate NextWater and other computed fields
		err := zoneResp.Render(nil, r)
		if err != nil {
			logger.Error("error rendering zone response", "error", err, "zone", zone.ID)
			continue
		}
		zoneResponses = append(zoneResponses, zoneResp)
	}

	// Build response data with garden context for template
	data := map[string]any{
		"Items":  zoneResponses,
		"Garden": garden,
	}

	return gardensNewZonesTemplate.Renderer(data), nil
}
