package worker

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

// Worker contains the necessary clients to schedule and execute actions
type Worker struct {
	storageClient  storage.Client
	influxdbClient influxdb.Client
	mqttClient     mqtt.Client
	weatherClient  weather.Client
	scheduler      *gocron.Scheduler
	logger         *logrus.Logger
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
		logger:         logger,
	}
}

// StartAsync starts the Worker's background jobs
func (w *Worker) StartAsync() {
	w.scheduler.StartAsync()
}

// Stop stops the Worker's background jobs
func (w *Worker) Stop() {
	w.scheduler.Stop()
}
