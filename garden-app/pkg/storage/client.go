package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/kv"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// Config is used to identify and configure a storage client
type Config struct {
	Driver  string                 `mapstructure:"driver"`
	Options map[string]interface{} `mapstructure:"options"`
}

// Client is a "generic" interface used to interact with our storage backend (DB, file, etc)
// It was separated from BaseClient to allow a single implementation of more advanced methods that are
// able to only rely on existing interface methods (reduces duplication in implementation packages)
type Client interface {
	BaseClient

	GetZonesUsingWaterSchedule(xid.ID) ([]*pkg.ZoneAndGarden, error)
	GetWaterSchedulesUsingWeatherClient(xid.ID) ([]*pkg.WaterSchedule, error)
}

// BaseClient holds the required methods for interacting with base resources directly in the storage backend
type BaseClient interface {
	GetGarden(xid.ID) (*pkg.Garden, error)
	GetGardens(bool) ([]*pkg.Garden, error)
	SaveGarden(*pkg.Garden) error
	DeleteGarden(xid.ID) error

	GetZones(xid.ID, bool) ([]*pkg.Zone, error)
	SaveZone(xid.ID, *pkg.Zone) error
	DeleteZone(xid.ID, xid.ID) error

	SavePlant(xid.ID, *pkg.Plant) error
	DeletePlant(xid.ID, xid.ID) error

	GetWeatherClient(xid.ID) (weather.Client, error)
	GetWeatherClientConfig(xid.ID) (*weather.Config, error)
	GetWeatherClientConfigs() ([]*weather.Config, error)
	SaveWeatherClientConfig(*weather.Config) error
	DeleteWeatherClientConfig(xid.ID) error

	GetWaterSchedule(xid.ID) (*pkg.WaterSchedule, error)
	GetMultipleWaterSchedules([]xid.ID) ([]*pkg.WaterSchedule, error)
	GetWaterSchedules(bool) ([]*pkg.WaterSchedule, error)
	SaveWaterSchedule(*pkg.WaterSchedule) error
	DeleteWaterSchedule(xid.ID) error
}

// NewClient will use the config to create and return the correct type of storage client
func NewClient(config Config) (Client, error) {
	client, err := kv.NewClient(config.Driver, config.Options)
	if err != nil {
		return nil, fmt.Errorf("error creating new KV client: %w", err)
	}

	return &extendedClient{client}, nil
}

type extendedClient struct {
	BaseClient
}

// GetZonesUsingWaterSchedule will find all Zones that use this WaterSchedule and return the Zones along with the Gardens they belong to
func (c *extendedClient) GetZonesUsingWaterSchedule(id xid.ID) ([]*pkg.ZoneAndGarden, error) {
	gardens, err := c.GetGardens(false)
	if err != nil {
		return nil, fmt.Errorf("unable to get all Gardens: %w", err)
	}

	results := []*pkg.ZoneAndGarden{}
	for _, g := range gardens {
		zones, err := c.GetZones(g.ID, false)
		if err != nil {
			return nil, fmt.Errorf("unable to get all Zones for Garden %q: %w", g.ID, err)
		}

		for _, z := range zones {
			for _, wsID := range z.WaterScheduleIDs {
				if wsID == id {
					results = append(results, &pkg.ZoneAndGarden{Zone: z, Garden: g})
				}
			}
		}
	}

	return results, nil
}

// GetWaterSchedulesUsingWeatherClient will return all WaterSchedules that rely on this WeatherClient
func (c *extendedClient) GetWaterSchedulesUsingWeatherClient(id xid.ID) ([]*pkg.WaterSchedule, error) {
	waterSchedules, err := c.GetWaterSchedules(false)
	if err != nil {
		return nil, fmt.Errorf("unable to get all WaterSchedules: %w", err)
	}

	results := []*pkg.WaterSchedule{}
	for _, ws := range waterSchedules {
		if ws.HasWeatherControl() {
			if ws.HasRainControl() {
				if ws.WeatherControl.Rain.ClientID == id {
					results = append(results, ws)
				}
			}
			if ws.HasTemperatureControl() {
				if ws.WeatherControl.Temperature.ClientID == id {
					results = append(results, ws)
				}
			}
		}
	}

	return results, nil
}
