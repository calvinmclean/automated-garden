package weather

import (
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/netatmo"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/xid"
)

var weatherClientSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Namespace: "garden_app",
	Name:      "weather_client_duration_seconds",
	Help:      "summary of weather client calls",
}, []string{"function", "cached"})

// Client is an interface defining the possible methods used to interact with the weather client APIs
type Client interface {
	GetTotalRain(since time.Duration) (float32, error)
	GetAverageHighTemperature(since time.Duration) (float32, error)
}

// Config is used to identify and configure a client type
type Config struct {
	ID      xid.ID                 `json:"id" yaml:"id"`
	Type    string                 `json:"type" yaml:"type"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// NewClient will use the config to create and return the correct type of weather client. If no type is provided, this will
// return a nil client rather than an error since Weather client is not required
func NewClient(c *Config) (Client, error) {
	switch c.Type {
	case "netatmo":
		return newMetricsWrapperClient(netatmo.NewClient(c.Options))
	case "fake":
		return newMetricsWrapperClient(fake.NewClient(c.Options))
	default:
		return nil, fmt.Errorf("invalid type '%s'", c.Type)
	}
}

// Patch allows modifying an existing Config with fields from a new one
func (c *Config) Patch(newConfig *Config) {
	if newConfig.Type != "" {
		c.Type = newConfig.Type
	}

	for k, v := range newConfig.Options {
		c.Options[k] = v
	}
}

// clientWrapper wraps any other implementation of the interface in order to add basic Prometheus summary metrics
// and caching
type clientWrapper struct {
	Client
	responseCache *cache.Cache
}

// newMetricsWrapperClient returns the input error as-is and the input client wrapped with a Prometheus metrics
// collector. It is intended to directly wrap functions to create other clients
func newMetricsWrapperClient(client Client, err error) (Client, error) {
	prometheus.MustRegister(weatherClientSummary)
	return &clientWrapper{client, cache.New(5*time.Minute, 1*time.Minute)}, err
}

// GetTotalRain ...
func (c *clientWrapper) GetTotalRain(since time.Duration) (float32, error) {
	now := time.Now()
	cached := false
	defer func() {
		weatherClientSummary.WithLabelValues("GetTotalRain", fmt.Sprintf("%t", cached)).Observe(time.Since(now).Seconds())
	}()

	cacheKey := fmt.Sprintf("total_rain_%d", since)
	cachedData, found := c.responseCache.Get(cacheKey)
	if found {
		cached = true
		return cachedData.(float32), nil
	}

	totalRain, err := c.Client.GetTotalRain(since)
	if err != nil {
		return 0, err
	}
	c.responseCache.Set(cacheKey, totalRain, cache.DefaultExpiration)

	return totalRain, nil
}

// GetAverageHighTemperature ...
func (c *clientWrapper) GetAverageHighTemperature(since time.Duration) (float32, error) {
	now := time.Now()
	cached := false
	defer func() {
		weatherClientSummary.WithLabelValues("GetAverageHighTemperature", fmt.Sprintf("%t", cached)).Observe(time.Since(now).Seconds())
	}()

	cacheKey := fmt.Sprintf("avg_temp_%d", since)
	cachedData, found := c.responseCache.Get(cacheKey)
	if found {
		cached = true
		return cachedData.(float32), nil
	}

	avgTemp, err := c.Client.GetAverageHighTemperature(since)
	if err != nil {
		return 0, err
	}
	c.responseCache.Set(cacheKey, avgTemp, cache.DefaultExpiration)

	return avgTemp, nil
}
