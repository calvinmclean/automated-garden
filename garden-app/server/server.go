package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// Config holds all the options and sub-configs for the server
type Config struct {
	WebConfig      `mapstructure:"web_server"`
	InfluxDBConfig influxdb.Config `mapstructure:"influxdb"`
	MQTTConfig     mqtt.Config     `mapstructure:"mqtt"`
	StorageConfig  storage.Config  `mapstructure:"storage"`
}

// WebConfig is used to allow reading the "web_server" section into the main Config struct
type WebConfig struct {
	Port int `mapstructure:"port"`
}

type contextKey string

// Run sets up and runs the webserver. This is the main entrypoint to our webserver application
// and is called by the "server" command
func Run(config Config) {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	// RESTy routes for Plant API actions
	// The PlantsResource will initialize the Scheduler and Storage Client
	plantsResource, err := NewPlantsResource(config)
	if err != nil {
		logger.Error("Error initializing '/plants' endpoint: ", err)
		os.Exit(1)
	}
	r.Mount("/plants", plantsResource.routes())

	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r)
}
