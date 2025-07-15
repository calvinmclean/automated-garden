package server

import (
	"context"
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

func NewWeatherClientsAPI() *WeatherClientsAPI {
	api := &WeatherClientsAPI{}

	api.API = babyapi.NewAPI("WeatherClients", weatherClientsBasePath, func() *weather.Config { return &weather.Config{} })

	api.SetOnCreateOrUpdate(func(_ http.ResponseWriter, _ *http.Request, wc *weather.Config) *babyapi.ErrResponse {
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
	api.SetGetAllResponseWrapper(func(wcs []*weather.Config) render.Renderer {
		resp := AllWeatherClientsResponse{ResourceList: babyapi.ResourceList[*WeatherClientResponse]{}}

		for _, wc := range wcs {
			resp.ResourceList.Items = append(resp.ResourceList.Items, &WeatherClientResponse{Config: wc})
		}

		return resp
	})

	api.AddCustomIDRoute(http.MethodGet, "/test", babyapi.Handler(api.testWeatherClient))

	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		switch r.URL.Query().Get("type") {
		case "create_modal":
			return weatherClientModalTemplate.Renderer(&weather.Config{
				ID: NewID(),
			})
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodGet, "/components", api.GetRequestedResourceAndDo(func(_ http.ResponseWriter, r *http.Request, wc *weather.Config) (render.Renderer, *babyapi.ErrResponse) {
		switch r.URL.Query().Get("type") {
		case "edit_modal":
			return weatherClientModalTemplate.Renderer(wc), nil
		default:
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.SetBeforeDelete(func(_ http.ResponseWriter, r *http.Request) *babyapi.ErrResponse {
		id := api.GetIDParam(r)

		waterSchedules, err := api.storageClient.GetWaterSchedulesUsingWeatherClient(id)
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("unable to get WaterSchedules using WeatherClient %q: %w", id, err))
		}

		if len(waterSchedules) > 0 {
			return babyapi.ErrInvalidRequest(fmt.Errorf("unable to delete WeatherClient used by %d WaterSchedules", len(waterSchedules)))
		}

		return nil
	})

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

func (api *WeatherClientsAPI) setup(storageClient *storage.Client) {
	api.storageClient = storageClient

	api.SetStorage(api.storageClient.WeatherClientConfigs)
}

func (api *WeatherClientsAPI) testWeatherClient(_ http.ResponseWriter, r *http.Request) render.Renderer {
	logger := babyapi.GetLoggerFromContext(r.Context())
	logger.Info("received request to test WeatherClient")

	weatherClient, httpErr := api.GetRequestedResource(r)
	if httpErr != nil {
		logger.Error("error getting requested resource", "error", httpErr.Error())
		return httpErr
	}

	weatherData, err := api.getWeatherData(r.Context(), weatherClient)
	if err != nil {
		logger.Error("unable to get weather data", "error", err)
		return InternalServerError(err)
	}

	return &WeatherClientTestResponse{WeatherData: weatherData}
}

func (api *WeatherClientsAPI) getWeatherData(ctx context.Context, weatherClient *weather.Config) (WeatherData, error) {
	wc, err := weather.NewClient(weatherClient, func(weatherClientOptions map[string]interface{}) error {
		weatherClient.Options = weatherClientOptions
		return api.storageClient.WeatherClientConfigs.Set(ctx, weatherClient)
	})
	if err != nil {
		return WeatherData{}, fmt.Errorf("error getting weather client: %w", err)
	}

	rd, err := wc.GetTotalRain(72 * time.Hour)
	if err != nil {
		return WeatherData{}, fmt.Errorf("unable to get total rain in the last 72 hours: %w", err)
	}

	td, err := wc.GetAverageHighTemperature(72 * time.Hour)
	if err != nil {
		return WeatherData{}, fmt.Errorf("unable to get average high temperature in the last 72 hours: %w", err)
	}

	return WeatherData{
		Rain: &RainData{
			MM: rd,
		},
		Temperature: &TemperatureData{
			Celsius: td,
		},
	}, nil
}
