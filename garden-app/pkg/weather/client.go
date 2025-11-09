package weather

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/netatmo"
	"github.com/calvinmclean/babyapi"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	responseCache = cache.New(5*time.Minute, 1*time.Minute)

	weatherClientSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "garden_app",
		Name:      "weather_client_duration_seconds",
		Help:      "summary of weather client calls",
	}, []string{"function", "cached"})
)

func init() {
	prometheus.MustRegister(weatherClientSummary)
}

// Client is an interface defining the possible methods used to interact with the weather client APIs
type Client interface {
	GetTotalRain(since time.Duration) (float32, error)
	GetAverageHighTemperature(since time.Duration) (float32, error)
}

// Config is used to identify and configure a client type
type Config struct {
	ID      babyapi.ID     `json:"id" yaml:"id"`
	Type    string         `json:"type" yaml:"type"`
	Options map[string]any `json:"options" yaml:"options"`
}

func (wc *Config) GetID() string {
	return wc.ID.String()
}

func (wc *Config) ParentID() string {
	return ""
}

func (wc *Config) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (wc *Config) Bind(r *http.Request) error {
	if wc == nil {
		return errors.New("missing required WeatherClient fields")
	}

	err := wc.ID.Bind(r)
	if err != nil {
		return err
	}

	switch r.Method {
	case http.MethodPut, http.MethodPost:
		if wc.Type == "" {
			return errors.New("missing required type field")
		}
		if wc.Options == nil {
			return errors.New("missing required options field")
		}
	}

	return nil
}

// NewClient will use the config to create and return the correct type of weather client. If no type is provided, this will
// return a nil client rather than an error since Weather client is not required
func NewClient(c *Config, storageCallback func(map[string]any) error) (client Client, err error) {
	switch c.Type {
	case "netatmo":
		client, err = netatmo.NewClient(c.Options, storageCallback)
	case "fake":
		client, err = fake.NewClient(c.Options)
	default:
		err = fmt.Errorf("invalid type '%s'", c.Type)
	}
	if err != nil {
		return nil, err
	}

	return newMetricsWrapperClient(client, c), nil
}

// Patch allows modifying an existing Config with fields from a new one
func (wc *Config) Patch(newConfig *Config) *babyapi.ErrResponse {
	if newConfig.Type != "" {
		wc.Type = newConfig.Type
	}

	if wc.Options == nil && newConfig.Options != nil {
		wc.Options = map[string]any{}
	}
	maps.Copy(wc.Options, newConfig.Options)

	return nil
}

// EndDated allows this to satisfy an interface even though the resources does not have end-dates
func (*Config) EndDated() bool {
	return false
}

func (*Config) SetEndDate(_ time.Time) {}

// clientWrapper wraps any other implementation of the interface in order to add basic Prometheus summary metrics
// and caching
type clientWrapper struct {
	Client
	*Config
}

// newMetricsWrapperClient returns the input client wrapped with a Prometheus metrics collector. It is intended to
// directly wrap functions to create other clients
func newMetricsWrapperClient(client Client, config *Config) Client {
	return &clientWrapper{client, config}
}

// GetTotalRain ...
func (c *clientWrapper) GetTotalRain(since time.Duration) (float32, error) {
	now := clock.Now()
	cached := false
	defer func() {
		weatherClientSummary.WithLabelValues("GetTotalRain", fmt.Sprintf("%t", cached)).Observe(time.Since(now).Seconds())
	}()

	cacheKey := fmt.Sprintf("total_rain_%d_%s", since, c.Config.ID)
	cachedData, found := responseCache.Get(cacheKey)
	if found {
		cached = true
		return cachedData.(float32), nil
	}

	totalRain, err := c.Client.GetTotalRain(since)
	if err != nil {
		return 0, err
	}
	responseCache.Set(cacheKey, totalRain, cache.DefaultExpiration)

	return totalRain, nil
}

// GetAverageHighTemperature ...
func (c *clientWrapper) GetAverageHighTemperature(since time.Duration) (float32, error) {
	now := clock.Now()
	cached := false
	defer func() {
		weatherClientSummary.WithLabelValues("GetAverageHighTemperature", fmt.Sprintf("%t", cached)).Observe(time.Since(now).Seconds())
	}()

	cacheKey := fmt.Sprintf("avg_temp_%d_%s", since, c.Config.ID)
	cachedData, found := responseCache.Get(cacheKey)
	if found {
		cached = true
		return cachedData.(float32), nil
	}

	avgTemp, err := c.Client.GetAverageHighTemperature(since)
	if err != nil {
		return 0, err
	}
	responseCache.Set(cacheKey, avgTemp, cache.DefaultExpiration)

	return avgTemp, nil
}

func ResetCache() {
	responseCache = cache.New(5*time.Minute, 1*time.Minute)
}
