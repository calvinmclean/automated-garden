package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

type WeatherClientTestResponse struct {
	WeatherData
}

func (resp *WeatherClientTestResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

type WeatherClientResponse struct {
	*weather.Config
	WeatherData *WeatherData `json:"weather_data,omitempty"`

	Links []Link `json:"links,omitempty"`

	api *WeatherClientsAPI
}

// Render ...
func (resp *WeatherClientResponse) Render(w http.ResponseWriter, r *http.Request) error {
	if resp != nil {
		resp.Links = append(resp.Links,
			Link{
				"self",
				fmt.Sprintf("%s/%s", weatherClientsBasePath, resp.ID),
			},
		)
	}

	// Check if we should fetch weather data
	// For HTML: skip by default for lazy loading, fetch when include_weather_data=true
	// For JSON: always fetch weather data to maintain API compatibility
	isHTML := render.GetAcceptedContentType(r) == render.ContentTypeHTML
	includeWeatherData := r.URL.Query().Get("include_weather_data") == "true"
	shouldFetchWeather := !isHTML || includeWeatherData

	if resp.api != nil && resp.Config != nil && shouldFetchWeather {
		units := getUnitsFromRequest(r)
		duration := getDurationFromRequest(r)
		weatherData, err := resp.api.getWeatherData(r.Context(), resp.Config, units, duration)
		if err == nil {
			resp.WeatherData = &weatherData
		}
	}

	if isHTML && r.Method == http.MethodPut {
		w.Header().Add("HX-Trigger", "newWeatherClient")
	}

	return nil
}

// HTML renders the weather client card for HTMX lazy loading
func (resp *WeatherClientResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	units := getUnitsFromRequest(r)
	duration := getDurationFromRequest(r)
	data := map[string]any{
		"Config":      resp.Config,
		"WeatherData": resp.WeatherData,
		"Units":       units,
		"Duration":    duration,
	}
	return weatherClientCardTemplate.Render(r, data)
}

type AllWeatherClientsResponse struct {
	babyapi.ResourceList[*WeatherClientResponse]
}

func (aws AllWeatherClientsResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return aws.ResourceList.Render(w, r)
}

func (aws AllWeatherClientsResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	slices.SortFunc(aws.Items, func(w *WeatherClientResponse, x *WeatherClientResponse) int {
		return strings.Compare(w.Name, x.Name)
	})

	units := getUnitsFromRequest(r)
	duration := getDurationFromRequest(r)
	data := map[string]any{
		"Items":    aws.Items,
		"Units":    units,
		"Duration": duration,
	}

	if r.URL.Query().Get("refresh") == "true" {
		return weatherClientsTemplate.Render(r, data)
	}

	return weatherClientsPageTemplate.Render(r, data)
}
