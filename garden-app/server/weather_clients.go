package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

const (
	weatherClientsBasePath  = "/weather_clients"
	weatherClientPathParam  = "clientID"
	weatherClientIDLogField = "weather_client_id"
)

// WeatherClientsResource encapsulates the structs and dependencies necessary for the WeatherClients API
// to function, including storage and configuring
type WeatherClientsResource struct {
	storageClient *storage.Client
}

// NewWeatherClientsResource creates a new WeatherClientsResource
func NewWeatherClientsResource(storageClient *storage.Client) (*WeatherClientsResource, error) {
	wc := &WeatherClientsResource{
		storageClient: storageClient,
	}

	return wc, nil
}

func (wcr *WeatherClientsResource) weatherClientContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		weatherClientIDString := chi.URLParam(r, weatherClientPathParam)
		logger := getLoggerFromContext(ctx).WithField(weatherClientIDLogField, weatherClientIDString)
		weatherClientID, err := xid.FromString(weatherClientIDString)
		if err != nil {
			logger.WithError(err).Error("unable to parse WeatherClient ID")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		weatherClientConfig, err := wcr.storageClient.GetWeatherClientConfig(weatherClientID)
		if err != nil {
			logger.WithError(err).Error("unable to get WeatherClient")
			render.Render(w, r, InternalServerError(err))
			return
		}
		if weatherClientConfig == nil {
			logger.Info("WeatherClient not found")
			render.Render(w, r, ErrNotFoundResponse)
			return
		}
		logger.Debugf("found WeatherClient: %+v", weatherClientConfig)

		ctx = newContextWithWeatherClient(ctx, wcr.NewWeatherClientResponse(weatherClientConfig))
		ctx = newContextWithLogger(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (wcr *WeatherClientsResource) getAllWeatherClients(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to get all WeatherClients")

	weatherClientConfigs, err := wcr.storageClient.GetWeatherClientConfigs()
	if err != nil {
		logger.WithError(err).Error("unable to get all WeatherClients")
		render.Render(w, r, InternalServerError(err))
		return
	}
	logger.Debugf("found %d WeatherClients", len(weatherClientConfigs))

	if err := render.Render(w, r, wcr.NewAllWeatherClientsResponse(weatherClientConfigs)); err != nil {
		logger.WithError(err).Error("unable to render AllWeatherClientsResponse")
		render.Render(w, r, ErrRender(err))
	}
}

func (wcr *WeatherClientsResource) createWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to create new WeatherClient")

	request := &WeatherClientRequest{}
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid request to create WeatherClient")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	weatherClientConfig := request.Config
	logger.Debugf("request to create WeatherClient: %+v", weatherClientConfig)

	// Assign values to fields that may not be set in the request
	weatherClientConfig.ID = xid.New()
	logger.Debugf("new WeatherClient ID: %v", weatherClientConfig.ID)

	// Save the WeatherClient
	logger.Debug("saving WeatherClient")
	if err := wcr.storageClient.SaveWeatherClientConfig(weatherClientConfig); err != nil {
		logger.WithError(err).Error("unable to save WeatherClient Config")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, wcr.NewWeatherClientResponse(weatherClientConfig)); err != nil {
		logger.WithError(err).Error("unable to render WeatherClientResponse")
		render.Render(w, r, ErrRender(err))
	}
}

func (wcr *WeatherClientsResource) updateWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to update WeatherClient")

	request := &UpdateWeatherClientRequest{}
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid request to update WeatherClient")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	logger.Debugf("request to update WeatherClient: %+v", request.Config)

	weatherClientResp := getWeatherClientFromContext(r.Context())
	weatherClient := weatherClientResp.Config

	weatherClient.Patch(request.Config)

	// make sure a valid WeatherClient can still be created
	_, err := weather.NewClient(weatherClient, func(map[string]interface{}) error { return nil })
	if err != nil {
		logger.WithError(err).Error("invalid request to update WeatherClient")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Save the WeatherClient
	logger.Debug("saving WeatherClient")
	if err := wcr.storageClient.SaveWeatherClientConfig(weatherClient); err != nil {
		logger.WithError(err).Error("unable to save WeatherClient Config")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusOK)
	if err := render.Render(w, r, weatherClientResp); err != nil {
		logger.WithError(err).Error("unable to render WeatherClientResponse")
		render.Render(w, r, ErrRender(err))
	}
}

// deleteWeatherClient will delete the WeatherClient config from storage
func (wcr *WeatherClientsResource) deleteWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to delete a WeatherClient")

	weatherClientResp := getWeatherClientFromContext(r.Context())
	weatherClient := weatherClientResp.Config

	// Unable to delete a WeatherClient that is being used by Zones
	err := wcr.checkIfClientIsBeingUsed(weatherClient)
	if err != nil {
		logger.WithError(err).Error("unable to delete WeatherClient used by Zone")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := wcr.storageClient.DeleteWeatherClientConfig(weatherClient.ID); err != nil {
		logger.WithError(err).Error("unable to delete WeatherClient")
		render.Render(w, r, InternalServerError(err))
		return
	}

	if err := render.Render(w, r, weatherClientResp); err != nil {
		logger.WithError(err).Error("unable to render WeatherClientResponse")
		render.Render(w, r, ErrRender(err))
	}
}

func (wcr *WeatherClientsResource) checkIfClientIsBeingUsed(weatherClient *weather.Config) error {
	waterSchedules, err := wcr.storageClient.GetWaterSchedulesUsingWeatherClient(weatherClient.ID)
	if err != nil {
		return fmt.Errorf("unable to get WaterSchedules using WeatherClient %q: %w", weatherClient.ID, err)
	}

	if len(waterSchedules) > 0 {
		return fmt.Errorf("unable to delete WeatherClient used by %d WaterSchedules", len(waterSchedules))
	}

	return nil
}

func (wcr *WeatherClientsResource) testWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to test WeatherClient")

	weatherClient := getWeatherClientFromContext(r.Context())

	wc, err := wcr.storageClient.GetWeatherClient(weatherClient.ID)
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
}
