package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
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

	api.AddCustomIDRoute(http.MethodPost, "/test", babyapi.Handler(api.testNotificationClient))

	api.EnableMCP(babyapi.MCPPermRead)

	return api
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

type NotificationClientResponse struct {
	*notifications.Client

	Links []Link `json:"links,omitempty"`
}

// Render ...
func (resp *NotificationClientResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	if resp != nil {
		resp.Links = append(resp.Links,
			Link{
				"self",
				fmt.Sprintf("%s/%s", notificationClientsBasePath, resp.ID),
			},
		)
	}
	return nil
}
