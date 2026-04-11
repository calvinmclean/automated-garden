package server

import (
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

// NotificationClientResponse wraps a notification client for API responses
type NotificationClientResponse struct {
	*notifications.Client
	Links []Link `json:"links,omitempty"`
}

// Render adds links and sets HTMX triggers for HTML responses
func (resp *NotificationClientResponse) Render(w http.ResponseWriter, r *http.Request) error {
	if resp != nil {
		resp.Links = append(resp.Links,
			Link{
				Rel:  "self",
				HRef: fmt.Sprintf("%s/%s", notificationClientsBasePath, resp.ID),
			},
		)

		// Trigger refresh of notification client lists and dropdowns
		if render.GetAcceptedContentType(r) == render.ContentTypeHTML {
			switch r.Method {
			case http.MethodPut, http.MethodPost:
				w.Header().Add("HX-Trigger", "newNotificationClient")
			}
		}
	}
	return nil
}

// AllNotificationClientsResponse wraps a list of notification clients
type AllNotificationClientsResponse struct {
	babyapi.ResourceList[*NotificationClientResponse]
}

// Render delegates to ResourceList's Render
func (resp AllNotificationClientsResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return resp.ResourceList.Render(w, r)
}

// HTML renders the settings list for HTMX requests
func (resp AllNotificationClientsResponse) HTML(w http.ResponseWriter, r *http.Request) string {
	return settingsListTemplate.Render(r, map[string]any{
		"Items": resp.Items,
	})
}
