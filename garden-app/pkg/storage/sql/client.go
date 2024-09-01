package sql

import (
	"database/sql"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	_ "modernc.org/sqlite"
)

//go:generate sqlc generate

// Config holds configuration for the SQL backend
type Config struct {
	DataSourceName string `mapstructure:"data_source_name" yaml:"data_source_name"`
}

type Client struct {
	Gardens                   babyapi.Storage[*pkg.Garden]
	Zones                     babyapi.Storage[*pkg.Zone]
	WaterSchedules            babyapi.Storage[*pkg.WaterSchedule]
	WeatherClientConfigs      babyapi.Storage[*weather.Config]
	NotificationClientConfigs babyapi.Storage[*notifications.Client]
	WaterRoutines             babyapi.Storage[*pkg.WaterRoutine]
}

// NewClient creates a new storage.Client using SQL backend.
// It initializes the database connection using the provided config.
func NewClient(config Config) (*Client, error) {
	db, err := sql.Open("sqlite", config.DataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite database: %w", err)
	}

	return &Client{
		Gardens:                   NewGardenStorage(db),
		Zones:                     NewZoneStorage(db),
		WaterSchedules:            NewWaterScheduleStorage(db),
		WeatherClientConfigs:      NewWeatherClientStorage(db),
		NotificationClientConfigs: NewNotificationClientStorage(db),
		WaterRoutines:             NewWaterRoutineStorage(db),
	}, nil
}
