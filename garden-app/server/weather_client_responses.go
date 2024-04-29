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

	Links []Link `json:"links,omitempty"`
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

	if render.GetAcceptedContentType(r) == render.ContentTypeHTML && r.Method == http.MethodPut {
		w.Header().Add("HX-Trigger", "newWeatherClient")
	}

	return nil
}

type AllWeatherClientsResponse struct {
	babyapi.ResourceList[*WeatherClientResponse]
}

func (aws AllWeatherClientsResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return aws.ResourceList.Render(w, r)
}

func (aws AllWeatherClientsResponse) HTML(r *http.Request) string {
	slices.SortFunc(aws.Items, func(w *WeatherClientResponse, x *WeatherClientResponse) int {
		return strings.Compare(w.Type, x.Type)
	})

	if r.URL.Query().Get("refresh") == "true" {
		return weatherClientsTemplate.Render(r, aws)
	}

	return weatherClientsPageTemplate.Render(r, aws)
}
