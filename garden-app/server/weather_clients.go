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
	weatherClientPathParam  = "clientID"
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

	wcr.api.AddCustomRoute(chi.Route{
		Pattern: "/",
		Handlers: map[string]http.Handler{
			http.MethodPost: wcr.api.ReadRequestBodyAndDo(func(r *http.Request, weatherClientConfig *weather.Config) render.Renderer {
				logger := getLoggerFromContext(r.Context())
				logger.Info("received request to create new WeatherClient")

				logger.Debugf("request to create WeatherClient: %+v", weatherClientConfig)

				// Assign values to fields that may not be set in the request
				weatherClientConfig.ID = xid.New()
				logger.Debugf("new WeatherClient ID: %v", weatherClientConfig.ID)

				// Save the WeatherClient
				logger.Debug("saving WeatherClient")
				if err := wcr.storageClient.Set(weatherClientConfig); err != nil {
					logger.WithError(err).Error("unable to save WeatherClient Config")
					return InternalServerError(err)
				}

				render.Status(r, http.StatusCreated)
				return weatherClientConfig
			}),
		},
	})

	wcr.api.AddCustomIDRoute(chi.Route{
		Pattern: "/test",
		Handlers: map[string]http.Handler{
			http.MethodGet: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				logger := getLoggerFromContext(r.Context())
				logger.Info("received request to test WeatherClient")

				weatherClient, httpErr := wcr.api.GetRequestedResource(r)
				if httpErr != nil {
					logger.Error("error getting requested resource", "error", httpErr.Error())
					render.Render(w, r, httpErr)
					return
				}

				wc, err := weather.NewClient(weatherClient, func(weatherClientOptions map[string]interface{}) error {
					weatherClient.Options = weatherClientOptions
					return wcr.storageClient.Set(weatherClient)
				})
				if err != nil {
					logger.WithError(err).Error("unable to get WeatherClient")
					render.Render(w, r, InternalServerError(err))
					return
				}

				rd, err := wc.GetTotalRain(72 * time.Hour)
				if err != nil {
					logger.WithError(err).Error("unable to get total rain in the last 72 hours")
					render.Render(w, r, InternalServerError(err))
					return
				}

				td, err := wc.GetAverageHighTemperature(72 * time.Hour)
				if err != nil {
					logger.WithError(err).Error("unable to get average high temperature in the last 72 hours")
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
					logger.WithError(err).Error("unable to render WeatherClientResponse")
					render.Render(w, r, ErrRender(err))
				}
			}),
		},
	})

	wcr.api.SetPATCH(func(old, new *weather.Config) error {
		old.Patch(new)

		// make sure a valid WeatherClient can still be created
		_, err := weather.NewClient(old, func(map[string]interface{}) error { return nil })
		if err != nil {
			return fmt.Errorf("invalid request to update WeatherClient: %w", err)
		}

		return nil
	})

	wcr.api.SetBeforeDelete(func(r *http.Request, id string) error {
		waterSchedules, err := storageClient.GetWaterSchedulesUsingWeatherClient(id)
		if err != nil {
			return fmt.Errorf("unable to get WaterSchedules using WeatherClient %q: %w", id, err)
		}

		if len(waterSchedules) > 0 {
			return fmt.Errorf("unable to delete WeatherClient used by %d WaterSchedules", len(waterSchedules))
		}

		if err != nil {
			return fmt.Errorf("unable to delete WeatherClient used by WaterSchedule: %w", err)
		}
		return nil
	})

	return wcr, nil
}

func (wcr *WeatherClientsAPI) Router() chi.Router {
	return wcr.api.Router()
}

// WeatherClientTestResponse is used to return WeatherData from testing that the client works
type WeatherClientTestResponse struct {
	WeatherData
}

// Render ...
func (resp *WeatherClientTestResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
