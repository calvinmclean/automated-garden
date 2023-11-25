package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	weatherClientsBasePath  = "/weather_clients"
	weatherClientIDLogField = "weather_client_id"
)

// WeatherClientsAPI encapsulates the structs and dependencies necessary for the WeatherClients API
// to function, including storage and configuring
type WeatherClientsAPI struct {
	storageClient *storage.TypedClient[*weather.Config]
	api           *babyapi.API[*weather.Config]
}

// NewWeatherClientsAPI creates a new WeatherClientsResource
func NewWeatherClientsAPI(storageClient *storage.Client) (*WeatherClientsAPI, error) {
	wcr := &WeatherClientsAPI{
		storageClient: storageClient.WeatherClientConfigs,
	}

	wcr.api = babyapi.NewAPI[*weather.Config](weatherClientsBasePath, func() *weather.Config { return &weather.Config{} })
	wcr.api.SetStorage(wcr.storageClient)

	wcr.api.ResponseWrapper(func(wc *weather.Config) render.Renderer {
		return &WeatherClientResponse{Config: wc}
	})

	wcr.api.AddCustomRoute(chi.Route{
		Pattern: "/",
		Handlers: map[string]http.Handler{
			http.MethodPost: wcr.api.ReadRequestBodyAndDo(wcr.createWeatherClient),
		},
	})

	wcr.api.AddCustomIDRoute(chi.Route{
		Pattern: "/test",
		Handlers: map[string]http.Handler{
			http.MethodGet: http.HandlerFunc(wcr.testWeatherClient),
		},
	})

	wcr.api.SetBeforeAfterDelete(
		func(r *http.Request) *babyapi.ErrResponse {
			id := wcr.api.GetIDParam(r)

			waterSchedules, err := storageClient.GetWaterSchedulesUsingWeatherClient(id)
			if err != nil {
				return babyapi.InternalServerError(fmt.Errorf("unable to get WaterSchedules using WeatherClient %q: %w", id, err))
			}

			if len(waterSchedules) > 0 {
				return babyapi.ErrInvalidRequest(fmt.Errorf("unable to delete WeatherClient used by %d WaterSchedules", len(waterSchedules)))
			}

			return nil
		},
		nil,
	)

	return wcr, nil
}

func (api *WeatherClientsAPI) testWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to test WeatherClient")

	weatherClient, httpErr := api.api.GetRequestedResource(r)
	if httpErr != nil {
		logger.Error("error getting requested resource", "error", httpErr.Error())
		render.Render(w, r, httpErr)
		return
	}

	wc, err := weather.NewClient(weatherClient, func(weatherClientOptions map[string]interface{}) error {
		weatherClient.Options = weatherClientOptions
		return api.storageClient.Set(weatherClient)
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

func (api *WeatherClientsAPI) createWeatherClient(r *http.Request, weatherClientConfig *weather.Config) (*weather.Config, *babyapi.ErrResponse) {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to create new WeatherClient")

	logger.Debug("request to create WeatherClient", "request", weatherClientConfig)

	// Assign values to fields that may not be set in the request
	weatherClientConfig.ID = xid.New()
	logger.Debug("new WeatherClient ID", weatherClientIDLogField, weatherClientConfig.ID)

	// Save the WeatherClient
	logger.Debug("saving WeatherClient")
	if err := api.storageClient.Set(weatherClientConfig); err != nil {
		logger.Error("unable to save WeatherClient Config", "error", err)
		return nil, babyapi.InternalServerError(err)
	}

	render.Status(r, http.StatusCreated)
	return weatherClientConfig, nil
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
func (resp *WeatherClientResponse) Render(w http.ResponseWriter, r *http.Request) error {
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
