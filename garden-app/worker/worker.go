package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"

	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	maxFirmwareSize      = 3 * 1024 * 1024 // 3 MB
	firmwareAssetName    = "firmware.bin"
	controllerReleaseTag = "controller-latest"
	githubRepo           = "calvinmclean/automated-garden"
)

var (
	scheduleJobsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "garden_app",
		Name:      "scheduled_jobs",
		Help:      "gauge of the currently-scheduled jobs",
	}, []string{"type", "id"})
	schedulerErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "garden_app",
		Name:      "scheduler_errors",
		Help:      "count of errors that occur in the background and do not have any visibility except logs",
	}, []string{"type", "id"})
)

func init() {
	sync.OnceFunc(func() {
		prometheus.MustRegister(
			scheduleJobsGauge,
			schedulerErrors,
		)
	})()
}

// Worker contains the necessary clients to schedule and execute actions
type Worker struct {
	storageClient  *storage.Client
	influxdbClient influxdb.Client
	mqttClient     mqtt.Client
	scheduler      *gocron.Scheduler
	logger         *slog.Logger
	httpClient     *http.Client

	// controllerSetupURLFunc builds the URL for the controller's WiFiManager paramsave endpoint.
	// It is overridable for tests.
	controllerSetupURLFunc func(topicPrefix string) string

	// firmwareUpdateReleaseURLFunc returns the GitHub API URL for the controller-latest release.
	// It is overridable for tests.
	firmwareUpdateReleaseURLFunc func() string

	// firmwareUpdateUploadURLFunc builds the URL for the controller's WiFiManager update endpoint.
	// It is overridable for tests.
	firmwareUpdateUploadURLFunc func(topicPrefix string) string

	// When Garden health messages are received, Timers are created to track their
	// uptime and notify if they go down
	downTimers map[string]clock.Timer
}

// WorkerOption configures a Worker during creation
type WorkerOption func(*Worker)

// WithHTTPClient sets the HTTP client used for controller setup requests
func WithHTTPClient(client *http.Client) WorkerOption {
	return func(w *Worker) {
		w.httpClient = client
	}
}

// WithControllerSetupURLFunc overrides the URL builder for controller setup requests
func WithControllerSetupURLFunc(fn func(topicPrefix string) string) WorkerOption {
	return func(w *Worker) {
		w.controllerSetupURLFunc = fn
	}
}

// WithFirmwareUpdateReleaseURLFunc overrides the GitHub release URL used for latest firmware updates
func WithFirmwareUpdateReleaseURLFunc(fn func() string) WorkerOption {
	return func(w *Worker) {
		w.firmwareUpdateReleaseURLFunc = fn
	}
}

// WithFirmwareUpdateUploadURLFunc overrides the URL builder for firmware upload requests
func WithFirmwareUpdateUploadURLFunc(fn func(topicPrefix string) string) WorkerOption {
	return func(w *Worker) {
		w.firmwareUpdateUploadURLFunc = fn
	}
}

// NewWorker creates a Worker with specified clients
func NewWorker(
	storageClient *storage.Client,
	influxdbClient influxdb.Client,
	mqttClient mqtt.Client,
	logger *slog.Logger,
	options ...WorkerOption,
) *Worker {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.CustomTime(clock.DefaultClock)
	w := &Worker{
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		mqttClient:     mqttClient,
		scheduler:      scheduler,
		logger:         logger.With("source", "worker"),
		downTimers:     map[string]clock.Timer{},
		httpClient:     http.DefaultClient,
		controllerSetupURLFunc: func(topicPrefix string) string {
			return fmt.Sprintf("http://%s.local/paramsave", topicPrefix)
		},
		firmwareUpdateReleaseURLFunc: func() string {
			return fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", githubRepo, controllerReleaseTag)
		},
		firmwareUpdateUploadURLFunc: func(topicPrefix string) string {
			return fmt.Sprintf("http://%s.local/u", topicPrefix)
		},
	}

	for _, option := range options {
		option(w)
	}

	return w
}

// StartAsync starts the Worker's background jobs
func (w *Worker) StartAsync() {
	w.scheduler.StartAsync()
	w.setupMQTT()
	w.syncLightStateAllGardens()
	w.syncFanStateAllGardens()
}

func (w *Worker) setupMQTT() {
	if _, isMock := w.mqttClient.(*mqtt.MockClient); isMock || w.mqttClient == nil {
		return
	}
	w.mqttClient.AddHandler(mqtt.TopicHandler{
		Topic:   "+/data/water",
		Handler: w.handleWaterCompleteStatusMessage,
	})
	w.mqttClient.AddHandler(mqtt.TopicHandler{
		Topic:   "+/data/logs",
		Handler: w.handleGardenStartupMessage,
	})
	w.mqttClient.AddHandler(mqtt.TopicHandler{
		Topic:   "+/data/health",
		Handler: w.healthMessageHandler,
	})
	w.mqttClient.AddHandler(mqtt.TopicHandler{
		Topic:   "+/data/info",
		Handler: w.handleControllerInfoMessage,
	})

	if err := w.mqttClient.Connect(); err != nil {
		w.logger.Error("failed to connect to MQTT broker", "error", err)
	}
}

func (w *Worker) syncLightStateAllGardens() {
	if w.storageClient == nil {
		return
	}

	ctx := context.Background()
	for g, err := range w.storageClient.Gardens.Search(ctx, "", nil) {
		if err != nil {
			w.logger.Error("error getting garden for light state sync", "error", err)
			continue
		}
		logger := w.contextLogger(g, nil, nil)

		err := w.setExpectedLightState(ctx, g)
		if err != nil {
			logger.Error("error setting expected LightState", "error", err)
		}
	}
}

func (w *Worker) syncFanStateAllGardens() {
	if w.storageClient == nil {
		return
	}

	ctx := context.Background()
	for g, err := range w.storageClient.Gardens.Search(ctx, "", nil) {
		if err != nil {
			w.logger.Error("error getting garden for fan state sync", "error", err)
			continue
		}
		logger := w.contextLogger(g, nil, nil)

		err := w.setExpectedFanState(ctx, g)
		if err != nil {
			logger.Error("error setting expected FanState", "error", err)
		}
	}
}

// Stop stops the Worker's background jobs
func (w *Worker) Stop() {
	w.scheduler.Stop()
	if w.mqttClient != nil {
		w.mqttClient.Disconnect(100)
	}
	if w.influxdbClient != nil {
		w.influxdbClient.Close()
	}

	prometheus.Unregister(scheduleJobsGauge)
	prometheus.Unregister(schedulerErrors)
}
