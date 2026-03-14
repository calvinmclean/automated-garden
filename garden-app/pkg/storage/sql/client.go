package sql

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	_ "modernc.org/sqlite"
)

//go:generate sqlc generate

//go:embed schema.sql
var schema string

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
	dataSourceName := config.DataSourceName
	// Use shared cache for in-memory databases to allow multiple connections to share the same database
	// Generate a unique database name for each client so tests don't interfere with each other
	if dataSourceName == ":memory:" {
		// Generate a random identifier for the database name
		randomBytes := make([]byte, 8)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("error generating random database name: %w", err)
		}
		randomName := hex.EncodeToString(randomBytes)
		dataSourceName = fmt.Sprintf("file:mem%s?mode=memory&cache=shared", randomName)
	}

	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("error initializing database schema: %w", err)
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
