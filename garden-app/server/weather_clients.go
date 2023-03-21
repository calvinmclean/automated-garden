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

		weatherClient, err := wc.storageClient.GetWeatherClient(weatherClientID)
		if err != nil {
			logger.WithError(err).Error("unable to get WeatherClient")
			render.Render(w, r, InternalServerError(err))
			return
		}
		if weatherClient == nil {
			logger.Info("WeatherClient not found")
			render.Render(w, r, ErrNotFoundResponse)
			return
		}
		logger.Debugf("found WeatherClient: %+v", weatherClient)

		ctx = newContextWithWeatherClient(ctx, weatherClient)
		ctx = newContextWithLogger(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (wc WeatherClientsResource) getAllWeatherClients(w http.ResponseWriter, r *http.Request) {
	getEndDated := r.URL.Query().Get("end_dated") == "true"

	logger := getLoggerFromContext(r.Context()).WithField("include_end_dated", getEndDated)
	logger.Info("received request to get all WeatherClients")

	weatherClients, err := wc.storageClient.GetWeatherClients(getEndDated)
	if err != nil {
		logger.WithError(err).Error("unable to get all WeatherClients")
		render.Render(w, r, ErrRender(err))
		return
	}
	logger.Debugf("found %d WeatherClients", len(weatherClients))

	if err := render.Render(w, r, wc.NewAllWeatherClientsResponse(r.Context(), weatherClients)); err != nil {
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
