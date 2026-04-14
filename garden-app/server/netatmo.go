package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

// startOAuth initiates the OAuth flow for Netatmo
func (api *WeatherClientsAPI) startOAuth(_ http.ResponseWriter, r *http.Request, wc *weather.Config) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())

	// Validate that client_id and client_secret exist
	clientID, ok := wc.Options["client_id"].(string)
	if !ok || clientID == "" {
		return nil, babyapi.ErrInvalidRequest(errors.New("client_id is required"))
	}

	clientSecret, ok := wc.Options["client_secret"].(string)
	if !ok || clientSecret == "" {
		return nil, babyapi.ErrInvalidRequest(errors.New("client_secret is required"))
	}

	// Build redirect URI
	baseURL := api.getBaseURL(r)
	redirectURI := fmt.Sprintf("%s/weather_clients/%s/netatmo/oauth/callback", baseURL, wc.GetID())

	// Generate and store state with redirect URI
	state := api.oauthStateCache.Store(wc.GetID(), redirectURI, 5*time.Minute)

	// Build Netatmo OAuth URL
	authURL := fmt.Sprintf(
		"https://api.netatmo.com/oauth2/authorize?client_id=%s&redirect_uri=%s&scope=read_station&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
	)

	logger.Info("starting OAuth flow for Netatmo", "weather_client_id", wc.GetID())

	// Use HX-Trigger to tell HTMX to open the OAuth popup via JavaScript
	return &OAuthStartResponse{AuthURL: authURL}, nil
}

// OAuthStartResponse triggers a client-side event to open the OAuth popup
type OAuthStartResponse struct {
	AuthURL string
}

// Render implements render.Renderer
func (o *OAuthStartResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// HTML implements the HTMLer interface for babyapi HTML rendering
func (o *OAuthStartResponse) HTML(w http.ResponseWriter, _ *http.Request) string {
	// Set HX-Trigger header to trigger the OAuth popup via JavaScript
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"openOAuthPopup": "%s"}`, o.AuthURL))
	return ""
}

// handleOAuthCallback handles the OAuth callback from Netatmo
func (api *WeatherClientsAPI) handleOAuthCallback(_ http.ResponseWriter, r *http.Request) render.Renderer {
	logger := babyapi.GetLoggerFromContext(r.Context())

	// Parse query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	// Handle OAuth error from Netatmo
	if errorParam != "" {
		logger.Error("OAuth error from Netatmo", "error", errorParam)
		return oauthCallbackTemplate.Renderer(&OAuthCallbackData{
			Success: false,
			Error:   fmt.Sprintf("Netatmo authorization failed: %s", errorParam),
		})
	}

	// Validate state
	wcID, redirectURI, valid := api.oauthStateCache.Validate(state)
	if !valid {
		logger.Error("invalid or expired state parameter")
		return oauthCallbackTemplate.Renderer(&OAuthCallbackData{
			Success: false,
			Error:   "Invalid or expired session. Please try again.",
		})
	}

	// Get the weather client config
	wc, err := api.Storage.Get(r.Context(), wcID)
	if err != nil {
		logger.Error("error getting weather client", "error", err)
		return oauthCallbackTemplate.Renderer(&OAuthCallbackData{
			Success: false,
			Error:   "Weather client not found",
		})
	}

	// Exchange code for tokens
	tokens, err := api.exchangeCodeForTokens(r.Context(), code, redirectURI, wc)
	if err != nil {
		logger.Error("error exchanging code for tokens", "error", err)
		return oauthCallbackTemplate.Renderer(&OAuthCallbackData{
			Success: false,
			Error:   fmt.Sprintf("Failed to get access token: %v", err),
		})
	}

	// Store tokens in the config
	if wc.Options == nil {
		wc.Options = map[string]any{}
	}
	wc.Options["authentication"] = tokens

	// Save the updated config
	err = api.Storage.Set(r.Context(), wc)
	if err != nil {
		logger.Error("error saving weather client", "error", err)
		return oauthCallbackTemplate.Renderer(&OAuthCallbackData{
			Success: false,
			Error:   "Failed to save authentication",
		})
	}

	logger.Info("successfully authenticated with Netatmo", "weather_client_id", wcID)

	return oauthCallbackTemplate.Renderer(&OAuthCallbackData{
		Success: true,
	})
}

// exchangeCodeForTokens exchanges the authorization code for access/refresh tokens
func (api *WeatherClientsAPI) exchangeCodeForTokens(ctx context.Context, code string, redirectURI string, wc *weather.Config) (map[string]any, error) {
	clientID := wc.Options["client_id"].(string)
	clientSecret := wc.Options["client_secret"].(string)

	formData := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.netatmo.com/oauth2/token",
		strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// nolint:gosec // URL is hardcoded OAuth2 token endpoint, not user input
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		AccessToken  string `json:"access_token"`  // nolint:gosec // JSON struct field for token response
		RefreshToken string `json:"refresh_token"` // nolint:gosec // JSON struct field for token response
		ExpiresIn    int    `json:"expires_in"`
	}

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing token response: %w", err)
	}

	return map[string]any{
		"access_token":    tokenResponse.AccessToken,
		"refresh_token":   tokenResponse.RefreshToken,
		"expires_in":      tokenResponse.ExpiresIn,
		"expiration_date": time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second).Format(time.RFC3339Nano),
	}, nil
}

// OAuthCallbackData is passed to the OAuth callback template
type OAuthCallbackData struct {
	Success bool
	Error   string
}

// getNetatmoStations fetches the user's weather stations from Netatmo API
func (api *WeatherClientsAPI) getNetatmoStations(_ http.ResponseWriter, r *http.Request, wc *weather.Config) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())

	// Check if authentication exists
	auth, ok := wc.Options["authentication"].(map[string]any)
	if !ok || auth == nil {
		return nil, babyapi.ErrInvalidRequest(errors.New("netatmo authentication required"))
	}

	accessToken, _ := auth["access_token"].(string)
	if accessToken == "" {
		return nil, babyapi.ErrInvalidRequest(errors.New("access token not found"))
	}

	// Fetch stations from Netatmo API
	stations, err := api.fetchNetatmoStations(r.Context(), accessToken)
	if err != nil {
		logger.Error("error fetching Netatmo stations", "error", err)
		return nil, babyapi.InternalServerError(fmt.Errorf("failed to fetch stations: %w", err))
	}

	return netatmoStationsTemplate.Renderer(map[string]any{
		"Stations": stations,
	}), nil
}

// getNetatmoModules fetches modules for a selected station
func (api *WeatherClientsAPI) getNetatmoModules(_ http.ResponseWriter, r *http.Request, wc *weather.Config) (render.Renderer, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())

	stationID := r.URL.Query().Get("Options.station_id")
	if stationID == "" {
		stationID = wc.Options["station_id"].(string)
	}
	if stationID == "" {
		return nil, babyapi.ErrInvalidRequest(errors.New("station_id is required"))
	}

	// Check if authentication exists
	auth, ok := wc.Options["authentication"].(map[string]any)
	if !ok || auth == nil {
		return nil, babyapi.ErrInvalidRequest(errors.New("netatmo authentication required"))
	}

	accessToken, _ := auth["access_token"].(string)
	if accessToken == "" {
		return nil, babyapi.ErrInvalidRequest(errors.New("access token not found"))
	}

	// Fetch modules for the station
	rainModules, outdoorModules, err := api.fetchNetatmoModules(r.Context(), accessToken, stationID)
	if err != nil {
		logger.Error("error fetching Netatmo modules", "error", err, "station_id", stationID)
		return nil, babyapi.InternalServerError(fmt.Errorf("failed to fetch modules: %w", err))
	}

	return netatmoModulesTemplate.Renderer(map[string]any{
		"RainModules":    rainModules,
		"OutdoorModules": outdoorModules,
	}), nil
}

// fetchNetatmoStations calls the Netatmo API to get user's stations
func (api *WeatherClientsAPI) fetchNetatmoStations(ctx context.Context, accessToken string) ([]map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.netatmo.com/api/getstationsdata", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Accept", "application/json")

	// nolint:gosec // URL is hardcoded Netatmo API endpoint, not user input
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Body struct {
			Devices []struct {
				ID          string `json:"_id"`
				StationName string `json:"station_name"`
			} `json:"devices"`
		} `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	stations := make([]map[string]string, 0, len(result.Body.Devices))
	for _, device := range result.Body.Devices {
		stations = append(stations, map[string]string{
			"ID":   device.ID,
			"Name": device.StationName,
		})
	}

	return stations, nil
}

// fetchNetatmoModules calls the Netatmo API to get modules for a station
func (api *WeatherClientsAPI) fetchNetatmoModules(ctx context.Context, accessToken, stationID string) ([]map[string]string, []map[string]string, error) {
	u, err := url.Parse("https://api.netatmo.com/api/getstationsdata")
	if err != nil {
		return nil, nil, err
	}
	q := u.Query()
	q.Set("device_id", stationID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Body struct {
			Devices []struct {
				Modules []struct {
					ID         string `json:"_id"`
					ModuleName string `json:"module_name"`
					Type       string `json:"type"`
				} `json:"modules"`
			} `json:"devices"`
		} `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, err
	}

	var rainModules, outdoorModules []map[string]string
	if len(result.Body.Devices) > 0 {
		for _, module := range result.Body.Devices[0].Modules {
			mod := map[string]string{
				"ID":   module.ID,
				"Name": module.ModuleName,
			}
			// Netatmo module types: NAModule1 = outdoor, NAModule2 = anemometer, NAModule3 = rain
			switch module.Type {
			case "NAModule3": // Rain module
				rainModules = append(rainModules, mod)
			case "NAModule1": // Outdoor module
				outdoorModules = append(outdoorModules, mod)
			}
		}
	}

	return rainModules, outdoorModules, nil
}
