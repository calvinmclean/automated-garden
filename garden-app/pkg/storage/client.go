package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/sql"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"

	"github.com/calvinmclean/babyapi"
	"github.com/mitchellh/mapstructure"
)

// Config is used to identify and configure a storage client
type Config struct {
	Driver  string         `mapstructure:"driver" yaml:"driver"`
	Options map[string]any `mapstructure:"options" yaml:"options"`
}

// AdditionalQueries are queries that are implemented outside of the base babyapi implementations
type AdditionalQueries interface {
	GetZonesUsingWaterSchedule(id string) ([]*pkg.ZoneAndGarden, error)
	GetWaterSchedulesUsingWeatherClient(id string) ([]*pkg.WaterSchedule, error)
}

type Client struct {
	Gardens                   babyapi.Storage[*pkg.Garden]
	Zones                     babyapi.Storage[*pkg.Zone]
	WaterSchedules            babyapi.Storage[*pkg.WaterSchedule]
	WeatherClientConfigs      babyapi.Storage[*weather.Config]
	NotificationClientConfigs babyapi.Storage[*notifications.Client]
	WaterRoutines             babyapi.Storage[*pkg.WaterRoutine]

	AdditionalQueries
}

func NewClient(config Config) (*Client, error) {
	if config.Driver != "sqlite" {
		return nil, fmt.Errorf("invalid driver: %q (only sqlite is supported)", config.Driver)
	}

	var sqlConfig sql.Config
	err := mapstructure.Decode(config.Options, &sqlConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding SQL config: %w", err)
	}

	sqlClient, err := sql.NewClient(sqlConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating SQL client: %w", err)
	}

	return &Client{
		Gardens:                   sqlClient.Gardens,
		Zones:                     sqlClient.Zones,
		WaterSchedules:            sqlClient.WaterSchedules,
		WeatherClientConfigs:      sqlClient.WeatherClientConfigs,
		NotificationClientConfigs: sqlClient.NotificationClientConfigs,
		WaterRoutines:             sqlClient.WaterRoutines,
		AdditionalQueries:         sqlClient.AdditionalQueries,
	}, nil
}
