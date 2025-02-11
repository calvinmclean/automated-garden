package worker

import (
	"log/slog"
	"sync"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"

	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
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

	// When Garden health messages are received, Timers are created to track their
	// uptime and notify if they go down
	downTimers map[string]clock.Timer
	// Wait for any downtime notifications before shutting down
	downtimeWG *sync.WaitGroup
}

// NewWorker creates a Worker with specified clients
func NewWorker(
	storageClient *storage.Client,
	influxdbClient influxdb.Client,
	mqttClient mqtt.Client,
	logger *slog.Logger,
) *Worker {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.CustomTime(clock.DefaultClock)
	return &Worker{
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		mqttClient:     mqttClient,
		scheduler:      scheduler,
		logger:         logger.With("source", "worker"),
		downTimers:     map[string]clock.Timer{},
		downtimeWG:     &sync.WaitGroup{},
	}
}

// StartAsync starts the Worker's background jobs
func (w *Worker) StartAsync() {
	w.scheduler.StartAsync()

	// Skip adding handler when mocked since it's not used
	_, isMock := w.mqttClient.(*mqtt.MockClient)
	if isMock || w.mqttClient == nil {
		return
	}

	w.mqttClient.AddHandler(mqtt.TopicHandler{
		Topic:   "+/data/water",
		Handler: w.handleWaterCompleteMessage,
	})
	w.mqttClient.AddHandler(mqtt.TopicHandler{
		Topic:   "+/data/logs",
		Handler: w.handleGardenStartupMessage,
	})
	w.mqttClient.AddHandler(mqtt.TopicHandler{
		Topic:   "+/data/health",
		Handler: w.healthMessageHandler,
	})

	if err := w.mqttClient.Connect(); err != nil {
		w.logger.Error("failed to connect to MQTT broker", "error", err)
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

	w.downtimeWG.Wait()

	prometheus.Unregister(scheduleJobsGauge)
	prometheus.Unregister(schedulerErrors)
}
