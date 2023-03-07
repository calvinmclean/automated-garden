package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
)

type WeatherClientResponse struct {
	*weather.Config
	Links []Link `json:"links,omitempty"`
}

func (wc WeatherClientsResource) NewWeatherClientResponse(ctx context.Context, weatherClient *weather.Config, links ...Link) *WeatherClientResponse {
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
