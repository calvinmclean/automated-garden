package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/xid"

	// sqlite driver import
	_ "modernc.org/sqlite"
)

//go:generate sqlc generate

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds configuration for the storage backend
type Config struct {
	ConnectionString string `mapstructure:"connection_string" yaml:"connection_string"`
}

type Client struct {
	Gardens                   babyapi.Storage[*pkg.Garden]
	Zones                     babyapi.Storage[*pkg.Zone]
	WaterSchedules            babyapi.Storage[*pkg.WaterSchedule]
	WeatherClientConfigs      babyapi.Storage[*weather.Config]
	NotificationClientConfigs babyapi.Storage[*notifications.Client]
	WaterRoutines             babyapi.Storage[*pkg.WaterRoutine]

	*AdditionalQueries
}

// NewClient creates a new storage.Client using SQL backend.
// It initializes the database connection using the provided config.
func NewClient(config Config) (*Client, error) {
	connectionString := config.ConnectionString
	// Use shared cache for in-memory databases to allow multiple connections to share the same database
	// Generate a unique database name for each client so tests don't interfere with each other
	if connectionString == ":memory:" {
		// Generate a random identifier for the database name
		randomBytes := make([]byte, 8)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("error generating random database name: %w", err)
		}
		randomName := hex.EncodeToString(randomBytes)
		connectionString = fmt.Sprintf("file:mem%s?mode=memory&cache=shared", randomName)
	}

	db, err := sql.Open("sqlite", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite database: %w", err)
	}

	err = runMigrations(db)
	if err != nil {
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	return &Client{
		Gardens:                   NewGardenStorage(db),
		Zones:                     NewZoneStorage(db),
		WaterSchedules:            NewWaterScheduleStorage(db),
		WeatherClientConfigs:      NewWeatherClientStorage(db),
		NotificationClientConfigs: NewNotificationClientStorage(db),
		WaterRoutines:             NewWaterRoutineStorage(db),
		AdditionalQueries:         NewAdditionalQueries(db),
	}, nil
}

// GetWeatherClient retrieves a WeatherClient by ID and initializes it
func (c *Client) GetWeatherClient(id xid.ID) (weather.Client, error) {
	clientConfig, err := c.WeatherClientConfigs.Get(context.Background(), id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting weather client config: %w", err)
	}

	if clientConfig == nil {
		return nil, fmt.Errorf("weather client config not found")
	}

	return weather.NewClient(clientConfig, func(weatherClientOptions map[string]any) error {
		clientConfig.Options = weatherClientOptions
		return c.WeatherClientConfigs.Set(context.Background(), clientConfig)
	})
}

// runMigrations executes all pending database migrations
func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("error creating migration driver: %w", err)
	}

	migrations, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("error creating migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", migrations, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("error creating migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error running migrations: %w", err)
	}

	return nil
}
