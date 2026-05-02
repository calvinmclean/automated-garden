package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/units"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

type unitsContextKey struct{}

// unitsMiddleware reads the user's unit preference from context or storage.
// It checks the query parameter first (override), then falls back to the stored setting.
func unitsMiddleware(storageClient *storage.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userUnits := r.URL.Query().Get("units")
			us := units.UnitSystem(userUnits)
			if !us.IsMetric() && !us.IsImperial() {
				// Read from storage
				storedValue, err := storageClient.GetUserSetting(r.Context(), "units")
				storedUS := units.UnitSystem(storedValue)
				if err == nil && (storedUS.IsImperial() || storedUS.IsMetric()) {
					userUnits = storedValue
				} else {
					userUnits = string(units.Metric)
				}
			}
			ctx := context.WithValue(r.Context(), unitsContextKey{}, userUnits)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// getUnitsFromRequest retrieves the units from request context
func getUnitsFromRequest(r *http.Request) string {
	if userUnits, ok := r.Context().Value(unitsContextKey{}).(string); ok && userUnits != "" {
		return userUnits
	}
	return string(units.Metric)
}

// SettingsAPI encapsulates the structs and dependencies necessary for the Settings API
type SettingsAPI struct {
	storageClient *storage.Client
}

// NewSettingsAPI creates a new SettingsAPI
func NewSettingsAPI() *SettingsAPI {
	return &SettingsAPI{}
}

// Setup wires the storage client
func (api *SettingsAPI) Setup(storageClient *storage.Client) {
	api.storageClient = storageClient
}

// handleSettingsComponents handles settings modal component requests
func (api *SettingsAPI) handleSettingsComponents(_ http.ResponseWriter, r *http.Request) render.Renderer {
	componentType := r.URL.Query().Get("type")

	switch componentType {
	case "settings_modal":
		return api.settingsModalRenderer(r.Context())
	case "settings_list":
		return api.settingsListRenderer(r.Context())
	case "units_selector":
		return api.unitsSelectorRenderer(r.Context())
	default:
		return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component type: %s", componentType))
	}
}

// settingsModalRenderer returns the full settings modal
func (api *SettingsAPI) settingsModalRenderer(ctx context.Context) render.Renderer {
	userUnits, err := api.storageClient.GetUserSetting(ctx, "units")
	if err != nil {
		userUnits = string(units.Metric)
	}
	return settingsModalTemplate.Renderer(map[string]any{
		"Units":    userUnits,
		"IsMetric": units.UnitSystem(userUnits).IsMetric(),
	})
}

// settingsListRenderer returns just the notification clients list for HTMX refresh
func (api *SettingsAPI) settingsListRenderer(ctx context.Context) render.Renderer {
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

// unitsSelectorRenderer returns the units selector fragment
func (api *SettingsAPI) unitsSelectorRenderer(ctx context.Context) render.Renderer {
	userUnits, err := api.storageClient.GetUserSetting(ctx, "units")
	if err != nil {
		userUnits = string(units.Metric)
	}
	return unitsSelectorTemplate.Renderer(map[string]any{
		"Units":    userUnits,
		"IsMetric": units.UnitSystem(userUnits).IsMetric(),
	})
}

// UserSettingResponse is the JSON response for a user setting
type UserSettingResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Render implements render.Renderer
func (r *UserSettingResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// handleGetUserSetting returns a user setting value
func (api *SettingsAPI) handleGetUserSetting(_ http.ResponseWriter, r *http.Request) render.Renderer {
	key := r.PathValue("key")
	if key == "" {
		return babyapi.ErrInvalidRequest(fmt.Errorf("setting key is required"))
	}

	value, err := api.storageClient.GetUserSetting(r.Context(), key)
	if err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error getting setting: %w", err))
	}

	// Return HTML fragment if requested
	if render.GetAcceptedContentType(r) == render.ContentTypeHTML {
		return unitsSelectorTemplate.Renderer(map[string]any{
			"Units":    value,
			"IsMetric": units.UnitSystem(value).IsMetric(),
		})
	}

	// Return JSON
	return &UserSettingResponse{Key: key, Value: value}
}

// handleUpdateUserSetting updates a user setting
func (api *SettingsAPI) handleUpdateUserSetting(w http.ResponseWriter, r *http.Request) render.Renderer {
	key := r.PathValue("key")
	if key == "" {
		return babyapi.ErrInvalidRequest(fmt.Errorf("setting key is required"))
	}

	// Limit request body to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 1024) // 1KB limit for simple form data

	// Parse form data
	if err := r.ParseForm(); err != nil {
		return babyapi.ErrInvalidRequest(fmt.Errorf("error parsing form: %w", err))
	}

	value := r.FormValue("value")
	if value == "" {
		return babyapi.ErrInvalidRequest(fmt.Errorf("value is required"))
	}

	// Validate the setting value
	us := units.UnitSystem(value)
	if key == "units" && !us.IsMetric() && !us.IsImperial() {
		return babyapi.ErrInvalidRequest(fmt.Errorf("invalid units value: %s", value))
	}

	if err := api.storageClient.SetUserSetting(r.Context(), key, value); err != nil {
		return babyapi.InternalServerError(fmt.Errorf("error saving setting: %w", err))
	}

	// Set trigger for HTMX to refresh pages that display units
	w.Header().Add("HX-Trigger", "unitsChanged")

	// Return updated selector
	return unitsSelectorTemplate.Renderer(map[string]any{
		"Units":    value,
		"IsMetric": units.UnitSystem(value).IsMetric(),
	})
}
