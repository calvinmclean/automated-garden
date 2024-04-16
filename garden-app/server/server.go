package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	prommetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	metrics_middleware "github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
)

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
	Port int `mapstructure:"port"`
}

// Server contains all of the necessary resources for running a server
type Server struct {
	rootAPI *babyapi.API[*babyapi.NilResource]
	cfg     Config
	logger  *slog.Logger
	worker  *worker.Worker
}

// NewServer creates and initializes all server resources based on config
func NewServer(cfg Config, validateData bool) (*Server, error) {
	logger := cfg.LogConfig.NewLogger().With("source", "server")

	rootAPI := babyapi.NewRootAPI("root", "/")

	// Configure HTTP metrics
	rootAPI.AddMiddleware(std.HandlerProvider("", metrics_middleware.New(metrics_middleware.Config{
		Recorder: prommetrics.NewRecorder(prommetrics.Config{Prefix: "garden_app"}),
	})))
	rootAPI.AddCustomRoute(http.MethodGet, "/metrics", promhttp.Handler())
	rootAPI.AddCustomRoute(http.MethodGet, "/", http.RedirectHandler("/gardens", http.StatusFound))

	// Initialize Storage Client
	logger.Info("initializing storage client", "driver", cfg.StorageConfig.Driver)
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
	logger.With(
		"client_id", cfg.MQTTConfig.ClientID,
		"broker", cfg.MQTTConfig.Broker,
		"port", cfg.MQTTConfig.Port,
	).Info("initializing MQTT client")
	mqttClient, err := mqtt.NewClient(cfg.MQTTConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize MQTT client: %v", err)
	}

	// Initialize InfluxDB Client
	logger.With(
		"address", cfg.InfluxDBConfig.Address,
		"org", cfg.InfluxDBConfig.Org,
		"bucket", cfg.InfluxDBConfig.Bucket,
	).Info("initializing InfluxDB client")
	influxdbClient := influxdb.NewClient(cfg.InfluxDBConfig)

	// Initialize Scheduler
	logger.Info("initializing scheduler")
	worker := worker.NewWorker(storageClient, influxdbClient, mqttClient, cfg.LogConfig.NewLogger())

	// Create API routes/handlers
	gardenAPI, err := NewGardensAPI(cfg, storageClient, influxdbClient, worker)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", gardenBasePath, err)
	}
	zonesResource, err := NewZonesAPI(storageClient, influxdbClient, worker)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", zoneBasePath, err)
	}

	rootAPI.AddNestedAPI(gardenAPI)
	gardenAPI.AddNestedAPI(zonesResource)

	weatherClientsAPI, err := NewWeatherClientsAPI(storageClient)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", weatherClientsBasePath, err)
	}
	rootAPI.AddNestedAPI(weatherClientsAPI)

	waterSchedulesAPI, err := NewWaterSchedulesAPI(storageClient, worker)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", waterScheduleBasePath, err)
	}
	rootAPI.AddNestedAPI(waterSchedulesAPI)

	return &Server{
		rootAPI,
		cfg,
		logger,
		worker,
	}, nil
}

// Start will run the server until it is stopped (blocking)
func (s *Server) Start() {
	// TODO: replace this by integrating with babyapi's RunCLI
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	s.worker.StartAsync()

	shutdownErr := s.rootAPI.WithContext(ctx).Serve(fmt.Sprintf(":%d", s.cfg.Port))
	if shutdownErr != nil && shutdownErr != http.ErrServerClosed {
		s.logger.Error("server shutdown error", "error", shutdownErr)
	}

	s.worker.Stop()
	s.logger.Info("server shutdown gracefully")
}

// Stop shuts down the server
func (s *Server) Stop() {
	s.rootAPI.Stop()
}

// validateAllStoredResources will read all resources from storage and make sure they are valid for the types
func validateAllStoredResources(storageClient *storage.Client) error {
	gardens, err := storageClient.Gardens.GetAll(storage.FilterEndDated[*pkg.Garden](true))
	if err != nil {
		return fmt.Errorf("unable to get all Gardens: %w", err)
	}

	for _, g := range gardens {
		if g.ID.IsNil() {
			return errors.New("invalid Garden: missing required field 'id'")
		}
		err = g.Bind(&http.Request{Method: http.MethodPut})
		if err != nil {
			return fmt.Errorf("invalid Garden %q: %w", g.ID, err)
		}
	}

	zones, err := storageClient.Zones.GetAll(nil)
	if err != nil {
		return fmt.Errorf("unable to get all Zones: %w", err)
	}

	for _, z := range zones {
		if z.ID.IsNil() {
			return errors.New("invalid Zone: missing required field 'id'")
		}
		err = z.Bind(&http.Request{Method: http.MethodPut})
		if err != nil {
			return fmt.Errorf("invalid Zone %q: %w", z.ID, err)
		}
	}

	waterSchedules, err := storageClient.WaterSchedules.GetAll(nil)
	if err != nil {
		return fmt.Errorf("unable to get all WaterSchedules: %w", err)
	}

	for _, ws := range waterSchedules {
		if ws.ID.IsNil() {
			return errors.New("invalid WaterSchedule: missing required field 'id'")
		}
		err = ws.Bind(&http.Request{Method: http.MethodPut})
		if err != nil {
			return fmt.Errorf("invalid WaterSchedule %q: %w", ws.ID, err)
		}
	}

	weatherClients, err := storageClient.WeatherClientConfigs.GetAll(nil)
	if err != nil {
		return fmt.Errorf("unable to get all WeatherClients: %w", err)
	}

	for _, wc := range weatherClients {
		if wc.ID.IsNil() {
			return errors.New("invalid WeatherClient: missing required field 'id'")
		}
		err = wc.Bind(&http.Request{Method: http.MethodPut})
		if err != nil {
			return fmt.Errorf("invalid WeatherClient %q: %w", wc.ID, err)
		}
	}

	return nil
}
