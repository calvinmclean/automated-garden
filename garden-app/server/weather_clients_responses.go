package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
)

// WeatherClientResponse is a simple struct being used to render and return a WeatherClient
type WeatherClientResponse struct {
	*weather.Config
	Links []Link `json:"links,omitempty"`
}

// NewWeatherClientResponse creates a new WeatherClientResponse
func (wcr WeatherClientsResource) NewWeatherClientResponse(ctx context.Context, weatherClient *weather.Config, links ...Link) *WeatherClientResponse {
	response := &WeatherClientResponse{
		Config: weatherClient,
	}
	response.Links = append(links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", weatherClientsBasePath, weatherClient.ID),
		},
	)

	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (resp *WeatherClientResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// AllWeatherClientsResponse is a simple struct being used to render and return a list of all WeatherClients
type AllWeatherClientsResponse struct {
	WeatherClients []*WeatherClientResponse `json:"weather_clients"`
}

// NewAllWeatherClientsResponse will create an AllWeatherClientResponse from a list of Zones
func (wcr WeatherClientsResource) NewAllWeatherClientsResponse(ctx context.Context, weatherClients []*weather.Config) *AllWeatherClientsResponse {
	weatherClientResponses := []*WeatherClientResponse{}
	for _, c := range weatherClients {
		weatherClientResponses = append(weatherClientResponses, wcr.NewWeatherClientResponse(ctx, c))
	}
	return &AllWeatherClientsResponse{weatherClientResponses}
}

// Render ...
func (wr *AllWeatherClientsResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// WeatherClientTestResponse is used to return WeatherData from testing that the client works
type WeatherClientTestResponse struct {
	WeatherData
}

// Render ...
func (resp *WeatherClientTestResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
