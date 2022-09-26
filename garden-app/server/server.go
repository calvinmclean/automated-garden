package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/sirupsen/logrus"
)

// Config holds all the options and sub-configs for the server
type Config struct {
	WebConfig      `mapstructure:"web_server"`
	InfluxDBConfig influxdb.Config `mapstructure:"influxdb"`
	MQTTConfig     mqtt.Config     `mapstructure:"mqtt"`
	StorageConfig  storage.Config  `mapstructure:"storage"`
	WeatherConfig  weather.Config  `mapstructure:"weather"`
	LogLevel       logrus.Level
}

// WebConfig is used to allow reading the "web_server" section into the main Config struct
type WebConfig struct {
	Port int `mapstructure:"port"`
}

// Run sets up and runs the webserver. This is the main entrypoint to our webserver application
// and is called by the "server" command
func Run(config Config) {
	baseLogger := logrus.New()
	baseLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		ForceColors:   true,
		FullTimestamp: true,
	})
	baseLogger.SetLevel(config.LogLevel)
	logger := baseLogger.WithField("source", "server")

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(middleware.Timeout(3 * time.Second))

	// RESTy routes for Garden and Plant API
	gardenResource, err := NewGardenResource(config, logger)
	if err != nil {
		logger.WithError(err).Errorf("error initializing '%s' endpoint", gardenBasePath)
		os.Exit(1)
	}
	plantsResource, err := NewPlantsResource(gardenResource)
	if err != nil {
		logger.WithError(err).Errorf("error initializing '%s' endpoint", plantBasePath)
		os.Exit(1)
	}
	zonesResource, err := NewZonesResource(gardenResource, logger)
	if err != nil {
		logger.WithError(err).Errorf("error initializing '%s' endpoint", zoneBasePath)
		os.Exit(1)
	}
	r.Mount(gardenBasePath, gardenResource.routes(plantsResource, zonesResource))

	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r)
}
