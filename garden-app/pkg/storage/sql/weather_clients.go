package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/sql/db"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
)

// WeatherClientStorage implements babyapi.Storage interface for WeatherClient Configs using SQL
type WeatherClientStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*weather.Config] = &WeatherClientStorage{}

// NewWeatherClientStorage creates a new WeatherClientStorage instance
func NewWeatherClientStorage(sqlDB *sql.DB) *WeatherClientStorage {
	return &WeatherClientStorage{
		q: db.New(sqlDB),
	}
}

// Get retrieves a WeatherClient Config from storage by ID
func (s *WeatherClientStorage) Get(ctx context.Context, id string) (*weather.Config, error) {
	dbWeatherClient, err := s.q.GetWeatherClient(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, babyapi.ErrNotFound
		}
		return nil, fmt.Errorf("error getting weather client: %w", err)
	}

	return dbWeatherClientToWeatherClient(dbWeatherClient)
}

// Search returns all WeatherClient Configs from storage
func (s *WeatherClientStorage) Search(ctx context.Context, _ string, _ url.Values) ([]*weather.Config, error) {
	dbWeatherClients, err := s.q.ListWeatherClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing weather clients: %w", err)
	}

	weatherClients := make([]*weather.Config, len(dbWeatherClients))
	for i, dbWeatherClient := range dbWeatherClients {
		weatherClient, err := dbWeatherClientToWeatherClient(dbWeatherClient)
		if err != nil {
			return nil, fmt.Errorf("invalid weather client: %w", err)
		}

		weatherClients[i] = weatherClient
	}

	return weatherClients, nil
}

// Set saves a WeatherClient Config to storage (creates or updates)
func (s *WeatherClientStorage) Set(ctx context.Context, weatherClient *weather.Config) error {
	options, err := json.Marshal(weatherClient.Options)
	if err != nil {
		return fmt.Errorf("error marshaling options: %w", err)
	}

	return s.q.UpsertWeatherClient(ctx, db.UpsertWeatherClientParams{
		ID:      weatherClient.ID.String(),
		Type:    weatherClient.Type,
		Options: options,
	})
}

// Delete removes a WeatherClient Config from storage
func (s *WeatherClientStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteWeatherClient(ctx, id)
}

func dbWeatherClientToWeatherClient(dbWeatherClient db.WeatherClient) (*weather.Config, error) {
	weatherClientID, err := parseID(dbWeatherClient.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid weather client ID: %w", err)
	}

	weatherClient := &weather.Config{
		ID:   weatherClientID,
		Type: dbWeatherClient.Type,
	}

	if len(dbWeatherClient.Options) > 0 {
		var options map[string]any
		err := json.Unmarshal(dbWeatherClient.Options, &options)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling options: %w", err)
		}
		weatherClient.Options = options
	}

	return weatherClient, nil
}
