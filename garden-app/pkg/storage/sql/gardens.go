package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

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
func (s *GardenStorage) Search(ctx context.Context, _ string, q url.Values) ([]*pkg.Garden, error) {
	getEndDated := q.Get("end_dated") == "true"

	listGardens := s.q.ListAllGardens
	if !getEndDated {
		listGardens = func(ctx context.Context) ([]db.Garden, error) {
			return s.q.ListActiveGardens(ctx, sql.NullString{String: time.Now().Format(time.RFC3339), Valid: true})
		}
	}

	dbGardens, err := listGardens(ctx)
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
	var endDate sql.NullString
	if garden.EndDate != nil {
		endDate = sql.NullString{String: garden.EndDate.Format(time.RFC3339), Valid: true}
	}

	var notificationClientID sql.NullString
	if garden.NotificationClientID != nil {
		notificationClientID = sql.NullString{String: *garden.NotificationClientID, Valid: true}
	}

	var notificationSettings sql.NullString
	if garden.NotificationSettings != nil {
		notificationSettingsStr, err := json.Marshal(garden.NotificationSettings)
		if err != nil {
			return fmt.Errorf("error marshaling notification settings: %w", err)
		}
		notificationSettings = sql.NullString{
			String: string(notificationSettingsStr),
			Valid:  true,
		}
	}

	var controllerConfig sql.NullString
	if garden.ControllerConfig != nil {
		controllerConfigStr, err := json.Marshal(garden.ControllerConfig)
		if err != nil {
			return fmt.Errorf("error marshaling controller config: %w", err)
		}
		controllerConfig = sql.NullString{
			String: string(controllerConfigStr),
			Valid:  true,
		}
	}

	var lightSchedule sql.NullString
	if garden.LightSchedule != nil {
		lightScheduleStr, err := json.Marshal(garden.LightSchedule)
		if err != nil {
			return fmt.Errorf("error marshaling light schedule: %w", err)
		}
		lightSchedule = sql.NullString{
			String: string(lightScheduleStr),
			Valid:  true,
		}
	}

	var maxZones int64
	if garden.MaxZones != nil {
		var err error
		maxZones, err = safeUintToInt64(*garden.MaxZones)
		if err != nil {
			return fmt.Errorf("invalid MaxZones: %w", err)
		}
	}

	var tempHumidSensor bool
	if garden.TemperatureHumiditySensor != nil {
		tempHumidSensor = *garden.TemperatureHumiditySensor
	}

	createdAt := time.Now().Format(time.RFC3339)
	if garden.CreatedAt != nil {
		createdAt = garden.CreatedAt.Format(time.RFC3339)
	}

	return s.q.UpsertGarden(ctx, db.UpsertGardenParams{
		ID:                   garden.ID.String(),
		Name:                 garden.Name,
		TopicPrefix:          garden.TopicPrefix,
		MaxZones:             maxZones,
		TempHumidSensor:      tempHumidSensor,
		CreatedAt:            createdAt,
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

	createdAt, err := time.Parse(time.RFC3339, dbGarden.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at: %w", err)
	}

	garden := &pkg.Garden{
		ID:          gardenID,
		Name:        dbGarden.Name,
		TopicPrefix: dbGarden.TopicPrefix,
		CreatedAt:   &createdAt,
	}

	mz, err := safeInt64ToUint(dbGarden.MaxZones)
	if err != nil {
		return nil, fmt.Errorf("invalid MaxZones: %w", err)
	}
	garden.MaxZones = &mz

	garden.TemperatureHumiditySensor = &dbGarden.TempHumidSensor

	if dbGarden.EndDate.Valid {
		endDate, err := time.Parse(time.RFC3339, dbGarden.EndDate.String)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date: %w", err)
		}
		garden.EndDate = &endDate
	}

	if dbGarden.NotificationClientID.Valid {
		garden.NotificationClientID = &dbGarden.NotificationClientID.String
	}

	if dbGarden.NotificationSettings.Valid && len(dbGarden.NotificationSettings.String) > 0 {
		var notificationSettings pkg.NotificationSettings
		err := json.Unmarshal([]byte(dbGarden.NotificationSettings.String), &notificationSettings)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling notification settings: %w", err)
		}
		garden.NotificationSettings = &notificationSettings
	}

	if dbGarden.ControllerConfig.Valid && len(dbGarden.ControllerConfig.String) > 0 {
		var controllerConfig pkg.ControllerConfig
		err := json.Unmarshal([]byte(dbGarden.ControllerConfig.String), &controllerConfig)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling controller config: %w", err)
		}
		garden.ControllerConfig = &controllerConfig
	}

	if dbGarden.LightSchedule.Valid && len(dbGarden.LightSchedule.String) > 0 {
		var lightSchedule pkg.LightSchedule
		err := json.Unmarshal([]byte(dbGarden.LightSchedule.String), &lightSchedule)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling light schedule: %w", err)
		}
		garden.LightSchedule = &lightSchedule
	}

	return garden, nil
}
