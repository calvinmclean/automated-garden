package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	prommetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	metrics_middleware "github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
)

//go:embed dist/*
var dist embed.FS

// Config holds all the options and sub-configs for the server
type Config struct {
	WebConfig      `mapstructure:"web_server"`
	InfluxDBConfig influxdb.Config `mapstructure:"influxdb"`
	MQTTConfig     mqtt.Config     `mapstructure:"mqtt"`
	StorageConfig  storage.Config  `mapstructure:"storage"`
	LogConfig      LogConfig       `mapstructure:"log"`
}

// WebConfig is used to allow reading the "web_server" section into the main Config struct
type WebConfig struct {
	Port       int  `mapstructure:"port"`
	EnableCors bool `mapstructure:"enable_cors"`
}

// Server contains all of the necessary resources for running a server
type Server struct {
	*http.Server
	quit            chan os.Signal
	logger          *logrus.Entry
	gardensResource *GardensResource
	worker          *worker.Worker
}

// NewServer creates and initializes all server resources based on config
func NewServer(cfg Config, validateData bool) (*Server, error) {
	baseLogger := logrus.New()
	baseLogger.SetFormatter(cfg.LogConfig.GetFormatter())
	baseLogger.SetLevel(cfg.LogConfig.GetLogLevel())
	logger := baseLogger.WithField("source", "server")

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(middleware.Timeout(3 * time.Second))

	if cfg.EnableCors {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"https://*", "http://*"},
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))
	}

	// Configure HTTP metrics
	r.Use(std.HandlerProvider("", metrics_middleware.New(metrics_middleware.Config{
		Recorder: prommetrics.NewRecorder(prommetrics.Config{Prefix: "garden_app"}),
	})))
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	// Initialize Storage Client
	logger.WithField("driver", cfg.StorageConfig.Driver).Info("initializing storage client")
	storageClient, err := storage.NewClient(cfg.StorageConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize storage client: %v", err)
	}

	if validateData {
		err = validateAllStoredResources(storageClient)
		if err != nil {
			return nil, fmt.Errorf("error validating all existing stored data: %w", err)
		}
	}

	// Initialize MQTT Client
	logger.WithFields(logrus.Fields{
		"client_id": cfg.MQTTConfig.ClientID,
		"broker":    cfg.MQTTConfig.Broker,
		"port":      cfg.MQTTConfig.Port,
	}).Info("initializing MQTT client")
	mqttClient, err := mqtt.NewClient(cfg.MQTTConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize MQTT client: %v", err)
	}

	// Initialize InfluxDB Client
	logger.WithFields(logrus.Fields{
		"address": cfg.InfluxDBConfig.Address,
		"org":     cfg.InfluxDBConfig.Org,
		"bucket":  cfg.InfluxDBConfig.Bucket,
	}).Info("initializing InfluxDB client")
	influxdbClient := influxdb.NewClient(cfg.InfluxDBConfig)

	// Initialize Scheduler
	logger.Info("initializing scheduler")
	worker := worker.NewWorker(storageClient, influxdbClient, mqttClient, baseLogger)

	// Create API routes/handlers
	gardenResource, err := NewGardenResource(cfg, storageClient, influxdbClient, worker)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", gardenBasePath, err)
	}
	plantsResource, err := NewPlantsResource(gardenResource)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", plantBasePath, err)
	}
	zonesResource, err := NewZonesResource(storageClient, influxdbClient, worker)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", zoneBasePath, err)
	}

	static, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, fmt.Errorf("error setting up static webapp subtree: %w", err)
	}
	r.Handle("/*", http.FileServer(http.FS(static)))

	r.Route(gardenBasePath, func(r chi.Router) {
		r.Post("/", gardenResource.createGarden)
		r.Get("/", gardenResource.getAllGardens)

		r.Route(fmt.Sprintf("/{%s}", gardenPathParam), func(r chi.Router) {
			r.Use(gardenResource.gardenContextMiddleware)

			r.Get("/", get[*GardenResponse](getGardenFromContext))
			r.Patch("/", gardenResource.updateGarden)
			r.Delete("/", gardenResource.endDateGarden)

			// Add new middleware to restrict certain paths to non-end-dated Gardens
			r.Route("/", func(r chi.Router) {
				r.Use(restrictEndDatedMiddleware("Garden", gardenCtxKey))
				r.Post("/action", gardenResource.gardenAction)

				r.Route(plantBasePath, func(r chi.Router) {
					r.Post("/", plantsResource.createPlant)
					r.Get("/", plantsResource.getAllPlants)

					r.Route(fmt.Sprintf("/{%s}", plantPathParam), func(r chi.Router) {
						r.Use(plantsResource.plantContextMiddleware)

						r.Get("/", get[*PlantResponse](getPlantFromContext))

						r.Patch("/", plantsResource.updatePlant)
						r.Delete("/", plantsResource.endDatePlant)
					})
				})

				r.Route(zoneBasePath, func(r chi.Router) {
					r.Post("/", zonesResource.createZone)
					r.Get("/", zonesResource.getAllZones)

					r.Route(fmt.Sprintf("/{%s}", zonePathParam), func(r chi.Router) {
						r.Use(zonesResource.zoneContextMiddleware)

						r.Get("/", get[*ZoneResponse](getZoneFromContext))
						r.Patch("/", zonesResource.updateZone)
						r.Delete("/", zonesResource.endDateZone)

						// Add new middleware to restrict certain paths to non-end-dated Zones
						r.Route("/", func(r chi.Router) {
							r.Use(restrictEndDatedMiddleware("Zone", zoneCtxKey))

							r.Post("/action", zonesResource.zoneAction)
							r.Get("/history", zonesResource.waterHistory)
						})
					})
				})
			})
		})
	})

	weatherClientsResource, err := NewWeatherClientsResource(storageClient)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", weatherClientsBasePath, err)
	}
	r.Route(weatherClientsBasePath, func(r chi.Router) {
		r.Post("/", weatherClientsResource.createWeatherClient)
		r.Get("/", weatherClientsResource.getAllWeatherClients)

		r.Route(fmt.Sprintf("/{%s}", weatherClientPathParam), func(r chi.Router) {
			r.Use(weatherClientsResource.weatherClientContextMiddleware)

			r.Get("/", weatherClientsResource.getWeatherClient)
			r.Patch("/", weatherClientsResource.updateWeatherClient)
			r.Delete("/", weatherClientsResource.deleteWeatherClient)

			r.Get("/test", weatherClientsResource.testWeatherClient)
		})
	})

	waterSchedulesResource, err := NewWaterSchedulesResource(storageClient, worker)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", waterScheduleBasePath, err)
	}
	r.Route(waterScheduleBasePath, func(r chi.Router) {
		r.Post("/", waterSchedulesResource.createWaterSchedule)
		r.Get("/", waterSchedulesResource.getAllWaterSchedules)

		r.Route(fmt.Sprintf("/{%s}", waterSchedulePathParam), func(r chi.Router) {
			r.Use(waterSchedulesResource.waterScheduleContextMiddleware)

			r.Get("/", waterSchedulesResource.getWaterSchedule)
			r.Patch("/", waterSchedulesResource.updateWaterSchedule)
			r.Delete("/", waterSchedulesResource.endDateWaterSchedule)
		})
	})

	return &Server{
		// nolint:gosec
		&http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: r},
		make(chan os.Signal, 1),
		logger,
		gardenResource,
		worker,
	}, nil
}

// Start will run the server until it is stopped (blocking)
func (s *Server) Start() {
	s.worker.StartAsync()
	go func() {
		shutdownErr := s.ListenAndServe()
		if shutdownErr != http.ErrServerClosed {
			s.logger.WithError(shutdownErr).Errorf("server shutdown error")
		}
	}()

	// Shutdown gracefully on Ctrl+C
	wg := &sync.WaitGroup{}
	wg.Add(1)
	signal.Notify(s.quit, os.Interrupt, syscall.SIGTERM)
	var shutdownStart time.Time
	go func() {
		<-s.quit
		shutdownStart = time.Now()
		s.logger.Info("gracefully shutting down server")

		err := s.Shutdown(context.Background())
		if err != nil {
			s.logger.WithError(err).Error("unable to shutdown server")
		}
		s.gardensResource.worker.Stop()

		wg.Done()
	}()
	wg.Wait()
	s.logger.WithField("time_elapsed", time.Since(shutdownStart)).Info("server shutdown gracefully")
}

// Stop shuts down the server
func (s *Server) Stop() {
	s.quit <- os.Interrupt
}

// validateAllStoredResources will read all resources from storage and make sure they are valid for the types
func validateAllStoredResources(storageClient *storage.Client) error {
	gardens, err := storageClient.GetGardens(true)
	if err != nil {
		return fmt.Errorf("unable to get all Gardens: %w", err)
	}

	for _, g := range gardens {
		// Remove Plants and Zones because g.Bind doesn't allow them
		plants := g.Plants
		g.Plants = nil
		zones := g.Zones
		g.Zones = nil

		if g.ID.IsNil() {
			return errors.New("invalid Garden: missing required field 'id'")
		}
		err = (&GardenRequest{g}).Bind(nil)
		if err != nil {
			return fmt.Errorf("invalid Garden %q: %w", g.ID, err)
		}

		for _, z := range zones {
			if z.ID.IsNil() {
				return errors.New("invalid Zone: missing required field 'id'")
			}
			err = (&ZoneRequest{z}).Bind(nil)
			if err != nil {
				return fmt.Errorf("invalid Zone %q: %w", z.ID, err)
			}
		}

		for _, p := range plants {
			if p.ID.IsNil() {
				return errors.New("invalid Plant: missing required field 'id'")
			}
			err = (&PlantRequest{p}).Bind(nil)
			if err != nil {
				return fmt.Errorf("invalid Plant %q: %w", p.ID, err)
			}
		}
	}

	waterSchedules, err := storageClient.GetWaterSchedules(true)
	if err != nil {
		return fmt.Errorf("unable to get all WaterSchedules: %w", err)
	}

	for _, ws := range waterSchedules {
		if ws.ID.IsNil() {
			return errors.New("invalid WaterSchedule: missing required field 'id'")
		}
		err = (&WaterScheduleRequest{ws}).Bind(nil)
		if err != nil {
			return fmt.Errorf("invalid WaterSchedule %q: %w", ws.ID, err)
		}
	}

	weatherClients, err := storageClient.GetWeatherClientConfigs()
	if err != nil {
		return fmt.Errorf("unable to get all WeatherClients: %w", err)
	}

	for _, wc := range weatherClients {
		if wc.ID.IsNil() {
			return errors.New("invalid WeatherClient: missing required field 'id'")
		}
		err = (&WeatherClientRequest{wc}).Bind(nil)
		if err != nil {
			return fmt.Errorf("invalid WeatherClient %q: %w", wc.ID, err)
		}
	}

	return nil
}
