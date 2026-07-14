package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
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
	gardenBasePath        = "/gardens"
	maxFirmwareUploadSize = 3 * 1024 * 1024 // 3 MB
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
	api.SetSearchResponseWrapper(func(gardens iter.Seq2[*pkg.Garden, error]) render.Renderer {
		resp := AllGardensResponse{ResourceList: babyapi.ResourceList[*GardenResponse]{}}

		for g, err := range gardens {
			if err != nil {
				// Handle error - for now just skip items with errors
				continue
			}
			resp.ResourceList.Items = append(resp.ResourceList.Items, api.NewGardenResponse(g))
		}

		return resp
	})

	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)

	api.AddCustomIDRoute(http.MethodPost, "/action", api.GetRequestedResourceAndDo(api.gardenAction))
	api.AddCustomIDRoute(http.MethodGet, "/water_history", api.GetRequestedResourceAndDo(api.gardenWaterHistory))

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

	api.SetBeforeDelete(func(_ http.ResponseWriter, r *http.Request) *babyapi.ErrResponse {
		logger, _ := babyapi.GetLoggerFromContext(r.Context())
		gardenID := api.GetIDParam(r)

		// Don't allow end-dating a Garden with active Zones
		numZones, err := api.numZones(r.Context(), gardenID)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("error getting number of Zones for garden: %w", err))
		}
		if numZones > 0 {
			zoneErr := errors.New("unable to end-date Garden with active Zones")
			logger.Error("unable to end-date Garden", "error", zoneErr)
			return babyapi.ErrInvalidRequest(zoneErr)
		}

		return nil
	})

	api.SetAfterDelete(func(_ http.ResponseWriter, r *http.Request) *babyapi.ErrResponse {
		logger, _ := babyapi.GetLoggerFromContext(r.Context())
		gardenID := api.GetIDParam(r)

		// Remove scheduled light and fan actions
		logger.Debug("removing scheduled actions for Garden")
		if err := api.worker.RemoveJobsByID(gardenID); err != nil {
			logger.Error("unable to remove scheduled actions", "error", err)
			return babyapi.InternalServerError(err)
		}
		return nil
	})

	api.ApplyExtension(extensions.HTMX[*pkg.Garden]{})

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

func (api *GardensAPI) gardenModalRenderer(ctx context.Context, g *pkg.Garden) render.Renderer {
	notificationClients := make([]*notifications.Client, 0)
	for nc, err := range api.storageClient.NotificationClientConfigs.Search(ctx, "", nil) {
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("error getting all notification clients to create garden modal: %w", err))
		}
		notificationClients = append(notificationClients, nc)
	}

	slices.SortFunc(notificationClients, func(nc1 *notifications.Client, nc2 *notifications.Client) int {
		return strings.Compare(nc1.Name, nc2.Name)
	})

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
	for g, err := range api.storageClient.Gardens.Search(context.Background(), "", nil) {
		if err != nil {
			return fmt.Errorf("error getting gardens for light schedule initialization: %w", err)
		}
		if g.EndDated() {
			continue
		}
		if g.LightSchedule != nil {
			err = api.worker.ScheduleLightActions(g)
			if err != nil {
				return fmt.Errorf("unable to schedule LightAction for Garden %v: %v", g.ID, err)
			}
		}
		if g.FanSchedule != nil {
			err = api.worker.ScheduleFanActions(g)
			if err != nil {
				return fmt.Errorf("unable to schedule FanAction for Garden %v: %v", g.ID, err)
			}
		}
	}

	return nil
}

func (api *GardensAPI) onCreateOrUpdate(_ http.ResponseWriter, r *http.Request, garden *pkg.Garden) *babyapi.ErrResponse {
	logger, _ := babyapi.GetLoggerFromContext(r.Context())

	numZones, err := api.numZones(r.Context(), garden.ID.String())
	if err != nil {
		return babyapi.InternalServerError(err)
	}
	if *garden.MaxZones < numZones {
		return babyapi.ErrInvalidRequest(fmt.Errorf("unable to set max_zones less than current num_zones=%d", numZones))
	}

	// If LightSchedule is empty, remove the scheduled Job
	if garden.LightSchedule == nil {
		logger.Debug("removing LightSchedule")
		if err := api.worker.RemoveJobsByTag(garden.ID.String(), "light"); err != nil {
			logger.Error("unable to remove LightSchedule for Garden", "error", err)
			return babyapi.InternalServerError(err)
		}
	}

	// If FanSchedule is empty, remove the scheduled Job
	if garden.FanSchedule == nil {
		logger.Info("removing FanSchedule")
		if err := api.worker.RemoveJobsByTag(garden.ID.String(), "fan"); err != nil {
			logger.Error("unable to remove FanSchedule for Garden", "error", err)
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
		logger.Debug("updating/resetting LightSchedule for Garden")
		if err := api.worker.ResetLightSchedule(garden); err != nil {
			logger.Error("unable to update/reset LightSchedule", "light_schedule", garden.LightSchedule, "error", err)
		}
	}

	if garden.FanSchedule != nil {
		logger.Info("updating/resetting FanSchedule for Garden")
		if err := api.worker.ResetFanSchedule(garden); err != nil {
			logger.Error("unable to update/reset FanSchedule", "fan_schedule", garden.FanSchedule, "error", err)
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
	logger, _ := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to execute GardenAction")

	if garden.EndDated() {
		return nil, babyapi.ErrInvalidRequest(errors.New("unable to execute action on end-dated garden"))
	}

	var gardenAction *action.GardenAction
	var err error
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		gardenAction, err = parseFirmwareUpdateAction(r)
	} else {
		gardenAction = &action.GardenAction{}
		err = render.Bind(r, gardenAction)
	}
	if err != nil {
		logger.Error("invalid request for GardenAction", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("garden action", "action", gardenAction)

	if err := api.worker.ExecuteGardenAction(r.Context(), garden, gardenAction); err != nil {
		logger.Error("unable to execute GardenAction", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusAccepted)
	return &GardenActionResponse{}, nil
}

func parseFirmwareUpdateAction(r *http.Request) (*action.GardenAction, error) {
	err := r.ParseMultipartForm(int64(maxFirmwareUploadSize + 1024))
	if err != nil {
		return nil, fmt.Errorf("unable to parse multipart form: %w", err)
	}

	latest := r.FormValue("firmware_update.latest") == "true"

	fwAction := &action.FirmwareUpdateAction{
		Latest: latest,
	}

	if !latest {
		file, header, err := r.FormFile("firmware_update.file")
		if err != nil {
			return nil, errors.New("firmware_update file is required when latest is not true")
		}
		defer func() { _ = file.Close() }()

		if filepath.Ext(header.Filename) != ".bin" {
			return nil, errors.New("firmware file must have .bin extension")
		}

		fwAction.FileData, err = io.ReadAll(io.LimitReader(file, int64(maxFirmwareUploadSize)+1))
		if err != nil {
			return nil, fmt.Errorf("unable to read firmware file: %w", err)
		}

		if len(fwAction.FileData) > maxFirmwareUploadSize {
			return nil, errors.New("firmware file exceeds maximum size of 3 MB")
		}
	}

	return &action.GardenAction{
		FirmwareUpdate: fwAction,
	}, nil
}

// gardenWaterHistory responds with recent water events for all Zones in a Garden
func (api *GardensAPI) gardenWaterHistory(_ http.ResponseWriter, r *http.Request, garden *pkg.Garden) (render.Renderer, *babyapi.ErrResponse) {
	logger, _ := babyapi.GetLoggerFromContext(r.Context())
	logger.Debug("received request to get Garden water history")

	timeRange, err := rangeQueryParam(r)
	if err != nil {
		logger.Error("unable to parse time range", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("using time range", "time_range", timeRange)

	limit, err := limitQueryParam(r, 20)
	if err != nil {
		logger.Error("unable to parse limit", "error", err)
		return nil, babyapi.ErrInvalidRequest(err)
	}
	logger.Debug("using limit", "limit", limit)

	logger.Debug("getting garden water history from InfluxDB")
	history, err := api.influxdbClient.GetGardenWaterHistory(r.Context(), garden.TopicPrefix, timeRange, limit, true)
	if err != nil {
		logger.Error("unable to get garden water history from InfluxDB", "error", err)
		return nil, babyapi.InternalServerError(err)
	}
	logger.Debug("garden water history", "history", history)

	zones, err := api.getAllZones(r.Context(), garden.ID.String(), false)
	if err != nil {
		logger.Error("unable to get zones for garden water history", "error", err)
		return nil, babyapi.InternalServerError(err)
	}
	zoneNames := make(map[string]string, len(zones))
	for _, zone := range zones {
		zoneNames[zone.GetID()] = zone.Name
	}

	return NewGardenWaterHistoryResponse(history, zoneNames, garden), nil
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
