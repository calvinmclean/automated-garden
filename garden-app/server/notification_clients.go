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
	notificationClientsBasePath  = "/notification_clients"
	notificationClientIDLogField = "notification_client_id"
)

// NotificationClientsAPI encapsulates the structs and dependencies necessary for the NotificationClients API
// to function, including storage and configuring
type NotificationClientsAPI struct {
	*babyapi.API[*notifications.Config]

	storageClient *storage.Client
}

// NewNotificationClientsAPI creates a new NotificationClientsResource
func NewNotificationClientsAPI(storageClient *storage.Client) (*NotificationClientsAPI, error) {
	api := &NotificationClientsAPI{
		storageClient: storageClient,
	}

	api.API = babyapi.NewAPI[*notifications.Config]("NotificationClients", notificationClientsBasePath, func() *notifications.Config { return &notifications.Config{} })
	api.SetStorage(api.storageClient.NotificationClientConfigs)

	api.SetOnCreateOrUpdate(func(_ *http.Request, nc *notifications.Config) *babyapi.ErrResponse {
		// make sure a valid NotificationClient can still be created
		_, err := notifications.NewClient(nc)
		if err != nil {
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid request to update NotificationClient: %w", err))
		}

		return nil
	})

	api.SetResponseWrapper(func(nc *notifications.Config) render.Renderer {
		return &NotificationClientResponse{Config: nc}
	})

	api.AddCustomIDRoute(http.MethodPost, "/test", babyapi.Handler(api.testNotificationClient))

	return api, nil
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

	nc, err := notifications.NewClient(notificationClient)
	if err != nil {
		logger.Error("unable to get NotificationClient", "error", err)
		return InternalServerError(err)
	}

	var req TestNotificationClientRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Error("unable to parse TestNotificationClientRequest", "error", err)
		return babyapi.ErrInvalidRequest(err)
	}

	err = nc.SendMessage(req.Title, req.Message)
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
	*notifications.Config

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
