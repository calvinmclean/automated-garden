package server

import (
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

const (
	weatherClientsBasePath  = "/weather_clients"
	weatherClientPathParam  = "clientID"
	weatherClientIDLogField = "weather_client_id"
)

// WeatherClientsResource encapsulates the structs and dependencies necessary for the WeatherClients API
// to function, including storage and configuring
type WeatherClientsResource struct {
	storageClient storage.Client
}

// NewWeatherClientsResource creates a new WeatherClientsResource
func NewWeatherClientsResource(logger *logrus.Entry, storageClient storage.Client) (WeatherClientsResource, error) {
	wc := WeatherClientsResource{
		storageClient: storageClient,
	}

	return wc, nil
}

func (wc WeatherClientsResource) weatherClientContextMiddleware(next http.Handler) http.Handler {
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

		weatherClientConfig, err := wc.storageClient.GetWeatherClientConfig(weatherClientID)
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

		ctx = newContextWithWeatherClient(ctx, weatherClientConfig)
		ctx = newContextWithLogger(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (wc WeatherClientsResource) getAllWeatherClients(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to get all WeatherClients")

	weatherClientConfigs, err := wc.storageClient.GetWeatherClientConfigs()
	if err != nil {
		logger.WithError(err).Error("unable to get all WeatherClients")
		render.Render(w, r, ErrRender(err))
		return
	}
	logger.Debugf("found %d WeatherClients", len(weatherClientConfigs))

	if err := render.Render(w, r, wc.NewAllWeatherClientsResponse(r.Context(), weatherClientConfigs)); err != nil {
		logger.WithError(err).Error("unable to render AllWeatherClientsResponse")
		render.Render(w, r, ErrRender(err))
	}
}

func (wc WeatherClientsResource) getWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to get WeatherClients")

	weatherClient := getWeatherClientFromContext(r.Context())
	logger.Debugf("responding with WeatherClients: %+v", weatherClient)

	gardenResponse := wc.NewWeatherClientResponse(r.Context(), weatherClient)
	if err := render.Render(w, r, gardenResponse); err != nil {
		logger.WithError(err).Error("unable to render WeatherClientResponse")
		render.Render(w, r, ErrRender(err))
	}
}

func (wc WeatherClientsResource) createWeatherClient(w http.ResponseWriter, r *http.Request) {
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
	if err := wc.storageClient.SaveWeatherClientConfig(weatherClientConfig); err != nil {
		logger.WithError(err).Error("unable to save WeatherClient Config")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, wc.NewWeatherClientResponse(r.Context(), weatherClientConfig)); err != nil {
		logger.WithError(err).Error("unable to render WeatherClientResponse")
		render.Render(w, r, ErrRender(err))
	}
}

func (wc WeatherClientsResource) updateWeatherClient(w http.ResponseWriter, r *http.Request) {
	logger := getLoggerFromContext(r.Context())
	logger.Info("received request to update WeatherClient")

	request := &UpdateWeatherClientRequest{}
	if err := render.Bind(r, request); err != nil {
		logger.WithError(err).Error("invalid request to update WeatherClient")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	logger.Debugf("request to update WeatherClient: %+v", request.Config)

	weatherClient := getWeatherClientFromContext(r.Context())

	weatherClient.Patch(request.Config)

	// Save the WeatherClient
	logger.Debug("saving WeatherClient")
	if err := wc.storageClient.SaveWeatherClientConfig(weatherClient); err != nil {
		logger.WithError(err).Error("unable to save WeatherClient Config")
		render.Render(w, r, InternalServerError(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, wc.NewWeatherClientResponse(r.Context(), weatherClient)); err != nil {
		logger.WithError(err).Error("unable to render WeatherClientResponse")
		render.Render(w, r, ErrRender(err))
	}
}
