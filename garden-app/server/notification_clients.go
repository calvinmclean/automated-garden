package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/go-chi/render"
)

const (
	notificationClientsBasePath = "/notification_clients"
)

// NotificationClientsAPI encapsulates the structs and dependencies necessary for the NotificationClients API
// to function, including storage and configuring
type NotificationClientsAPI struct {
	*babyapi.API[*notifications.Client]

	storageClient *storage.Client
}

// NewNotificationClientsAPI creates a new NotificationClientsResource
func NewNotificationClientsAPI() *NotificationClientsAPI {
	api := &NotificationClientsAPI{}

	api.API = babyapi.NewAPI[*notifications.Client]("NotificationClients", notificationClientsBasePath, func() *notifications.Client { return &notifications.Client{} })

	api.SetOnCreateOrUpdate(func(_ http.ResponseWriter, _ *http.Request, nc *notifications.Client) *babyapi.ErrResponse {
		// make sure a valid NotificationClient can still be created
		err := nc.TestCreate()
		if err != nil {
			return babyapi.ErrInvalidRequest(fmt.Errorf("error initializing client: %w", err))
		}

		return nil
	})

	api.SetResponseWrapper(func(nc *notifications.Client) render.Renderer {
		return &NotificationClientResponse{Client: nc}
	})

	api.SetSearchResponseWrapper(func(ncs []*notifications.Client) render.Renderer {
		result := make([]*NotificationClientResponse, len(ncs))
		for i, nc := range ncs {
			result[i] = &NotificationClientResponse{Client: nc}
		}
		return AllNotificationClientsResponse{
			ResourceList: babyapi.ResourceList[*NotificationClientResponse]{Items: result},
		}
	})

	// Component routes for the settings modal
	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(api.handleComponents))
	api.AddCustomIDRoute(http.MethodGet, "/components", babyapi.Handler(api.handleIDComponents))

	api.AddCustomIDRoute(http.MethodPost, "/test", babyapi.Handler(api.testNotificationClient))

	api.ApplyExtension(extensions.HTMX[*notifications.Client]{})

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

// handleComponents handles non-ID component requests (settings_modal, settings_list, add_row)
func (api *NotificationClientsAPI) handleComponents(_ http.ResponseWriter, r *http.Request) render.Renderer {
	componentType := r.URL.Query().Get("type")

	switch componentType {
	case "settings_modal":
		return api.settingsModalRenderer(r.Context())
	case "settings_list":
		return api.settingsListRenderer(r.Context())
	case "add_row":
		return notificationClientAddRowTemplate.Renderer(&notifications.Client{ID: NewID()})
	case "remove_add_row":
		return removeAddRowTemplate.Renderer(nil)
	default:
		return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component type: %s", componentType))
	}
}

// handleIDComponents handles ID-based component requests (edit_row, row)
func (api *NotificationClientsAPI) handleIDComponents(_ http.ResponseWriter, r *http.Request) render.Renderer {
	componentType := r.URL.Query().Get("type")

	notificationClient, apiErr := api.GetRequestedResource(r)
	if apiErr != nil {
		return apiErr
	}

	switch componentType {
	case "edit_row":
		return notificationClientEditRowTemplate.Renderer(notificationClient)
	case "row":
		return notificationClientRowTemplate.Renderer(notificationClient)
	default:
		return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component type: %s", componentType))
	}
}

// settingsModalRenderer returns the full settings modal with notification clients list
func (api *NotificationClientsAPI) settingsModalRenderer(ctx context.Context) render.Renderer {
	return settingsModalTemplate.Renderer(nil)
}

// settingsListRenderer returns just the notification clients list for HTMX refresh
func (api *NotificationClientsAPI) settingsListRenderer(ctx context.Context) render.Renderer {
	notificationClients, err := api.storageClient.NotificationClientConfigs.Search(ctx, "", nil)
	if err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error fetching notification clients: %w", err))
	}

	// Sort by name
	slices.SortFunc(notificationClients, func(a, b *notifications.Client) int {
		return strings.Compare(a.Name, b.Name)
	})

	return settingsListTemplate.Renderer(map[string]any{
		"Items": notificationClients,
	})
}

func (api *NotificationClientsAPI) setup(storageClient *storage.Client) {
	api.storageClient = storageClient

	api.SetStorage(api.storageClient.NotificationClientConfigs)
}

type TestNotificationClientRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (api *NotificationClientsAPI) testNotificationClient(_ http.ResponseWriter, r *http.Request) render.Renderer {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to test NotificationClient")

	notificationClient, httpErr := api.GetRequestedResource(r)
	if httpErr != nil {
		logger.Error("error getting requested resource", "error", httpErr.Error())
		return httpErr
	}

	var req TestNotificationClientRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Error("unable to parse TestNotificationClientRequest", "error", err)
		return babyapi.ErrInvalidRequest(err)
	}

	err = notificationClient.SendMessage(req.Title, req.Message)
	if err != nil {
		return babyapi.ErrInvalidRequest(err)
	}

	return nil
}

// NotificationClientTestResponse is used to return WeatherData from testing that the client works
type NotificationClientTestResponse struct {
	WeatherData
}

// Render ...
func (resp *NotificationClientTestResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
