package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/html"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	prommetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	metrics_middleware "github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
)

// API contains all HTTP API handling and logic
type API struct {
	*babyapi.API[*babyapi.NilResource]
	gardens             *GardensAPI
	zones               *ZonesAPI
	weatherClients      *WeatherClientsAPI
	notificationClients *NotificationClientsAPI
	waterSchedules      *WaterSchedulesAPI
}

// NewAPI intializes an API without any integrations or clients. Use api.Setup(...) before running
func NewAPI() *API {
	api := &API{
		API:                 babyapi.NewRootAPI("garden-app", "/"),
		gardens:             NewGardenAPI(),
		zones:               NewZonesAPI(),
		weatherClients:      NewWeatherClientsAPI(),
		notificationClients: NewNotificationClientsAPI(),
		waterSchedules:      NewWaterSchedulesAPI(),
	}
	api.gardens.AddNestedAPI(api.zones)

	api.API.
		AddMiddleware(std.HandlerProvider("", metrics_middleware.New(metrics_middleware.Config{
			Recorder: prommetrics.NewRecorder(prommetrics.Config{Prefix: "garden_app"}),
		}))).
		AddCustomRoute(http.MethodGet, "/metrics", promhttp.Handler()).
		AddCustomRoute(http.MethodGet, "/", http.RedirectHandler("/gardens", http.StatusFound)).
		AddNestedAPI(api.gardens).
		AddNestedAPI(api.weatherClients).
		AddNestedAPI(api.notificationClients).
		AddNestedAPI(api.waterSchedules)

	return api
}

// Setup will prepare to run by setting up clients and doing any final configurations for the API
func (api *API) Setup(cfg Config, validateData bool) error {
	html.SetFS(templates, "templates/*")
	html.SetFuncs(templateFuncs)

	logger := cfg.LogConfig.NewLogger().With("source", "server")
	slog.SetDefault(logger)

	// Initialize Storage Client
	logger.Info("initializing storage client", "driver", cfg.StorageConfig.Driver)
	storageClient, err := storage.NewClient(cfg.StorageConfig)
	if err != nil {
		return fmt.Errorf("unable to initialize storage client: %v", err)
	}

	if validateData {
		err = validateAllStoredResources(storageClient)
		if err != nil {
			return fmt.Errorf("error validating all existing stored data: %w", err)
		}
	}

	// Initialize MQTT Client
	logger.With(
		"client_id", cfg.MQTTConfig.ClientID,
		"broker", cfg.MQTTConfig.Broker,
		"port", cfg.MQTTConfig.Port,
	).Info("initializing MQTT client")
	mqttClient, err := mqtt.NewClient(cfg.MQTTConfig, mqtt.DefaultHandler(logger), mqtt.TopicHandler{
		Topic:   "+/data/water",
		Handler: paho.MessageHandler(NewMQTTHandler(storageClient, logger).Handle),
	})
	if err != nil {
		return fmt.Errorf("unable to initialize MQTT client: %v", err)
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

	err = api.setup(cfg, storageClient, influxdbClient, worker)
	if err != nil {
		return err
	}

	worker.StartAsync()

	go func() {
		<-api.Done()
		worker.Stop()
	}()

	return nil
}

func (api *API) setup(cfg Config, storageClient *storage.Client, influxdbClient influxdb.Client, worker *worker.Worker) error {
	if cfg.ReadOnly {
		api.API.AddMiddleware(readOnlyMiddleware)
	}

	err := api.gardens.setup(cfg, storageClient, influxdbClient, worker)
	if err != nil {
		return fmt.Errorf("error setting up Gardens API: %w", err)
	}

	err = api.waterSchedules.setup(storageClient, worker)
	if err != nil {
		return fmt.Errorf("error setting up WaterSchedules API: %w", err)
	}

	api.zones.setup(storageClient, influxdbClient, worker)
	api.weatherClients.setup(storageClient)
	api.notificationClients.setup(storageClient)

	return nil
}

// validateAllStoredResources will read all resources from storage and make sure they are valid for the types
func validateAllStoredResources(storageClient *storage.Client) error {
	gardens, err := storageClient.Gardens.GetAll(context.Background(), nil)
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

	zones, err := storageClient.Zones.GetAll(context.Background(), nil)
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

	waterSchedules, err := storageClient.WaterSchedules.GetAll(context.Background(), nil)
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

	weatherClients, err := storageClient.WeatherClientConfigs.GetAll(context.Background(), nil)
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
