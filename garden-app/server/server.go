package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	metrics_middleware "github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
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

// Server contains all of the necessary resources for running a server
type Server struct {
	*http.Server
	quit            chan os.Signal
	logger          *logrus.Entry
	gardensResource GardensResource
}

// NewServer creates and initializes all server resources based on config
func NewServer(cfg Config) (*Server, error) {
	baseLogger := logrus.New()
	baseLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		ForceColors:   true,
		FullTimestamp: true,
	})
	baseLogger.SetLevel(cfg.LogLevel)
	logger := baseLogger.WithField("source", "server")

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(middleware.Timeout(3 * time.Second))

	// Configure HTTP metrics
	r.Use(std.HandlerProvider("", metrics_middleware.New(metrics_middleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{}),
	})))
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	// RESTy routes for Garden and Plant API
	gardenResource, err := NewGardenResource(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", gardenBasePath, err)
	}
	plantsResource, err := NewPlantsResource(gardenResource)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", plantBasePath, err)
	}
	zonesResource, err := NewZonesResource(gardenResource, logger)
	if err != nil {
		return nil, fmt.Errorf("error initializing '%s' endpoint: %w", zoneBasePath, err)
	}
	r.Mount(gardenBasePath, gardenResource.routes(plantsResource, zonesResource))

	return &Server{
		&http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: r},
		make(chan os.Signal, 1),
		logger,
		gardenResource,
	}, nil
}

// Start will run the server until it is stopped (blocking)
func (s *Server) Start() {
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

		s.Shutdown(context.Background())
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
