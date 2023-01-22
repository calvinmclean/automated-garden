package worker

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/metrics"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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

// Worker contains the necessary clients to schedule and execute actions
type Worker struct {
	storageClient  storage.Client
	influxdbClient influxdb.Client
	mqttClient     mqtt.Client
	weatherClient  weather.Client
	scheduler      *gocron.Scheduler
	logger         *logrus.Entry
	metrics        *metrics.Metrics
}

// NewWorker creates a Worker with specified clients
func NewWorker(
	storageClient storage.Client,
	influxdbClient influxdb.Client,
	mqttClient mqtt.Client,
	weatherClient weather.Client,
	logger *logrus.Logger,
) *Worker {
	return &Worker{
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		mqttClient:     mqttClient,
		weatherClient:  weatherClient,
		scheduler:      gocron.NewScheduler(time.Local),
		logger:         logger.WithField("source", "worker"),
		metrics:        metrics.New(scheduleJobsGauge, schedulerErrors),
	}
}

// StartAsync starts the Worker's background jobs
func (w *Worker) StartAsync() {
	w.scheduler.StartAsync()
	w.metrics.Register()
}

// Stop stops the Worker's background jobs
func (w *Worker) Stop() {
	w.scheduler.Stop()
	w.metrics.Unregister()
	if w.mqttClient != nil {
		w.mqttClient.Disconnect(100)
	}
	if w.influxdbClient != nil {
		w.influxdbClient.Close()
	}
}
