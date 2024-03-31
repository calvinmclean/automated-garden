package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

const (
	weatherClientsBasePath  = "/weather_clients"
	weatherClientIDLogField = "weather_client_id"
)

// WeatherClientsAPI encapsulates the structs and dependencies necessary for the WeatherClients API
// to function, including storage and configuring
type WeatherClientsAPI struct {
	*babyapi.API[*weather.Config]

	storageClient *storage.Client
}

// NewWeatherClientsAPI creates a new WeatherClientsResource
func NewWeatherClientsAPI(storageClient *storage.Client) (*WeatherClientsAPI, error) {
	api := &WeatherClientsAPI{
		storageClient: storageClient,
	}

	api.API = babyapi.NewAPI[*weather.Config]("WeatherClients", weatherClientsBasePath, func() *weather.Config { return &weather.Config{} })
	api.SetStorage(api.storageClient.WeatherClientConfigs)

	api.SetOnCreateOrUpdate(func(_ *http.Request, wc *weather.Config) *babyapi.ErrResponse {
		// make sure a valid WeatherClient can still be created
		_, err := weather.NewClient(wc, func(map[string]interface{}) error { return nil })
		if err != nil {
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid request to update WeatherClient: %w", err))
		}

		return nil
	})

	api.SetResponseWrapper(func(wc *weather.Config) render.Renderer {
		return &WeatherClientResponse{Config: wc}
	})

	api.AddCustomIDRoute(http.MethodGet, "/test", http.HandlerFunc(api.testWeatherClient))

	api.SetBeforeDelete(func(r *http.Request) *babyapi.ErrResponse {
		id := api.GetIDParam(r)

		waterSchedules, err := storageClient.GetWaterSchedulesUsingWeatherClient(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to get WaterSchedules using WeatherClient %q: %w", id, err))
		}

		if len(waterSchedules) > 0 {
			return babyapi.ErrInvalidRequest(fmt.Errorf("unable to delete WeatherClient used by %d WaterSchedules", len(waterSchedules)))
		}

		return nil
	})

	return api, nil
}

func (api *WeatherClientsAPI) testWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to test WeatherClient")

	weatherClient, httpErr := api.GetRequestedResource(r)
	if httpErr != nil {
		logger.Error("error getting requested resource", "error", httpErr.Error())
		render.Render(w, r, httpErr)
		return
	}

	wc, err := weather.NewClient(weatherClient, func(weatherClientOptions map[string]interface{}) error {
		weatherClient.Options = weatherClientOptions
		return api.storageClient.WeatherClientConfigs.Set(weatherClient)
	})
	if err != nil {
		logger.Error("unable to get WeatherClient", "error", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	rd, err := wc.GetTotalRain(72 * time.Hour)
	if err != nil {
		logger.Error("unable to get total rain in the last 72 hours", "error", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	td, err := wc.GetAverageHighTemperature(72 * time.Hour)
	if err != nil {
		logger.Error("unable to get average high temperature in the last 72 hours", "error", err)
		render.Render(w, r, InternalServerError(err))
		return
	}

	resp := &WeatherClientTestResponse{WeatherData: WeatherData{
		Rain: &RainData{
			MM: rd,
		},
		Temperature: &TemperatureData{
			Celsius: td,
		},
	}}

	if err := render.Render(w, r, resp); err != nil {
		logger.Error("unable to render WeatherClientResponse", "error", err)
		render.Render(w, r, ErrRender(err))
	}
}

// WeatherClientTestResponse is used to return WeatherData from testing that the client works
type WeatherClientTestResponse struct {
	WeatherData
}

// Render ...
func (resp *WeatherClientTestResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

type WeatherClientResponse struct {
	*weather.Config

	Links []Link `json:"links,omitempty"`
}

// Render ...
func (resp *WeatherClientResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	if resp != nil {
		resp.Links = append(resp.Links,
			Link{
				"self",
				fmt.Sprintf("%s/%s", "/weather_clients", resp.ID),
			},
		)
	}
	return nil
}
