package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/yaml"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// Config is used to identify and configure a storage client
type Config struct {
	Type    string            `mapstructure:"type"`
	Options map[string]string `mapstructure:"options"`
}

// Client is a "generic" interface used to interact with our storage backend (DB, file, etc)
type Client interface {
	GetGarden(xid.ID) (*pkg.Garden, error)
	GetGardens(bool) ([]*pkg.Garden, error)
	SaveGarden(*pkg.Garden) error
	DeleteGarden(xid.ID) error

	GetZone(xid.ID, xid.ID) (*pkg.Zone, error)
	GetZones(xid.ID, bool) ([]*pkg.Zone, error)
	SaveZone(xid.ID, *pkg.Zone) error
	DeleteZone(xid.ID, xid.ID) error

	GetPlant(xid.ID, xid.ID) (*pkg.Plant, error)
	GetPlants(xid.ID, bool) ([]*pkg.Plant, error)
	SavePlant(xid.ID, *pkg.Plant) error
	DeletePlant(xid.ID, xid.ID) error

	GetWeatherClient(xid.ID) (weather.Client, error)
	GetWeatherClientConfig(xid.ID) (*weather.Config, error)
	GetWeatherClientConfigs() ([]*weather.Config, error)
	SaveWeatherClientConfig(*weather.Config) error
	DeleteWeatherClientConfig(xid.ID) error

	GetWaterSchedule(xid.ID) (*pkg.WaterSchedule, error)
	GetWaterSchedules(bool) ([]*pkg.WaterSchedule, error)
	SaveWaterSchedule(*pkg.WaterSchedule) error
	DeleteWaterSchedule(xid.ID) error

	GetZonesUsingWaterSchedule(xid.ID) ([]*pkg.ZoneAndGarden, error)
	GetWaterSchedulesUsingWeatherClient(xid.ID) ([]*pkg.WaterSchedule, error)
}

// NewClient will use the config to create and return the correct type of storage client
func NewClient(config Config) (Client, error) {
	switch config.Type {
	case "YAML", "yaml", "ConfigMap", "configmap":
		return yaml.NewClient(config.Type, config.Options)
	default:
		return nil, fmt.Errorf("invalid type '%s'", config.Type)
	}
}
