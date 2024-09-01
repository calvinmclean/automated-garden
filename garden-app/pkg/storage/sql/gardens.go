package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/sql/db"
	"github.com/calvinmclean/babyapi"
)

// GardenStorage implements babyapi.Storage interface for Gardens using SQL
type GardenStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*pkg.Garden] = &GardenStorage{}

// NewGardenStorage creates a new GardenStorage instance
func NewGardenStorage(sqlDB *sql.DB) *GardenStorage {
	return &GardenStorage{
		q: db.New(sqlDB),
	}
}

// Get retrieves a Garden from storage by ID
func (s *GardenStorage) Get(ctx context.Context, id string) (*pkg.Garden, error) {
	dbGarden, err := s.q.GetGarden(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, babyapi.ErrNotFound
		}
		return nil, fmt.Errorf("error getting garden: %w", err)
	}

	return dbGardenToGarden(dbGarden)
}

// Search returns all Gardens from storage
func (s *GardenStorage) Search(ctx context.Context, _ string, _ url.Values) ([]*pkg.Garden, error) {
	dbGardens, err := s.q.ListGardens(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing gardens: %w", err)
	}

	gardens := make([]*pkg.Garden, len(dbGardens))
	for i, dbGarden := range dbGardens {
		garden, err := dbGardenToGarden(dbGarden)
		if err != nil {
			return nil, fmt.Errorf("invalid garden: %w", err)
		}

		gardens[i] = garden
	}

	return gardens, nil
}

// Set saves a Garden to storage (creates or updates)
func (s *GardenStorage) Set(ctx context.Context, garden *pkg.Garden) error {
	var endDate sql.NullTime
	if garden.EndDate != nil {
		endDate = sql.NullTime{Time: *garden.EndDate, Valid: true}
	}

	var notificationClientID sql.NullString
	if garden.NotificationClientID != nil {
		notificationClientID = sql.NullString{String: *garden.NotificationClientID, Valid: true}
	}

	var notificationSettings json.RawMessage
	if garden.NotificationSettings != nil {
		var err error
		notificationSettings, err = json.Marshal(garden.NotificationSettings)
		if err != nil {
			return fmt.Errorf("error marshaling notification settings: %w", err)
		}
	}

	var controllerConfig json.RawMessage
	if garden.ControllerConfig != nil {
		var err error
		controllerConfig, err = json.Marshal(garden.ControllerConfig)
		if err != nil {
			return fmt.Errorf("error marshaling controller config: %w", err)
		}
	}

	var lightSchedule json.RawMessage
	if garden.LightSchedule != nil {
		var err error
		lightSchedule, err = json.Marshal(garden.LightSchedule)
		if err != nil {
			return fmt.Errorf("error marshaling light schedule: %w", err)
		}
	}

	var maxZones interface{}
	if garden.MaxZones != nil {
		maxZones = int64(*garden.MaxZones)
	}

	var tempHumidSensor bool
	if garden.TemperatureHumiditySensor != nil {
		tempHumidSensor = *garden.TemperatureHumiditySensor
	}

	return s.q.UpsertGarden(ctx, db.UpsertGardenParams{
		ID:                   garden.ID.String(),
		Name:                 garden.Name,
		TopicPrefix:          garden.TopicPrefix,
		MaxZones:             maxZones,
		TempHumidSensor:      tempHumidSensor,
		CreatedAt:            *garden.CreatedAt,
		EndDate:              endDate,
		NotificationClientID: notificationClientID,
		NotificationSettings: notificationSettings,
		ControllerConfig:     controllerConfig,
		LightSchedule:        lightSchedule,
	})
}

// Delete removes a Garden from storage
func (s *GardenStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteGarden(ctx, id)
}

func dbGardenToGarden(dbGarden db.Garden) (*pkg.Garden, error) {
	gardenID, err := parseID(dbGarden.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid garden ID: %w", err)
	}

	garden := &pkg.Garden{
		ID:          gardenID,
		Name:        dbGarden.Name,
		TopicPrefix: dbGarden.TopicPrefix,
		CreatedAt:   &dbGarden.CreatedAt,
	}

	if maxZones, ok := dbGarden.MaxZones.(int64); ok {
		mz := uint(maxZones)
		garden.MaxZones = &mz
	}

	garden.TemperatureHumiditySensor = &dbGarden.TempHumidSensor

	if dbGarden.EndDate.Valid {
		garden.EndDate = &dbGarden.EndDate.Time
	}

	if dbGarden.NotificationClientID.Valid {
		garden.NotificationClientID = &dbGarden.NotificationClientID.String
	}

	if len(dbGarden.NotificationSettings) > 0 {
		var notificationSettings pkg.NotificationSettings
		err := json.Unmarshal(dbGarden.NotificationSettings, &notificationSettings)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling notification settings: %w", err)
		}
		garden.NotificationSettings = &notificationSettings
	}

	if len(dbGarden.ControllerConfig) > 0 {
		var controllerConfig pkg.ControllerConfig
		err := json.Unmarshal(dbGarden.ControllerConfig, &controllerConfig)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling controller config: %w", err)
		}
		garden.ControllerConfig = &controllerConfig
	}

	if len(dbGarden.LightSchedule) > 0 {
		var lightSchedule pkg.LightSchedule
		err := json.Unmarshal(dbGarden.LightSchedule, &lightSchedule)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling light schedule: %w", err)
		}
		garden.LightSchedule = &lightSchedule
	}

	return garden, nil
}
