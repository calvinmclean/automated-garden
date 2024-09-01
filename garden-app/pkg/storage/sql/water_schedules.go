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
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
)

// WaterScheduleStorage implements babyapi.Storage interface for WaterSchedules using SQL
type WaterScheduleStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*pkg.WaterSchedule] = &WaterScheduleStorage{}

// NewWaterScheduleStorage creates a new WaterScheduleStorage instance
func NewWaterScheduleStorage(sqlDB *sql.DB) *WaterScheduleStorage {
	return &WaterScheduleStorage{
		q: db.New(sqlDB),
	}
}

// Get retrieves a WaterSchedule from storage by ID
func (s *WaterScheduleStorage) Get(ctx context.Context, id string) (*pkg.WaterSchedule, error) {
	dbWaterSchedule, err := s.q.GetWaterSchedule(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, babyapi.ErrNotFound
		}
		return nil, fmt.Errorf("error getting water schedule: %w", err)
	}

	return dbWaterScheduleToWaterSchedule(dbWaterSchedule)
}

// Search returns all WaterSchedules from storage
func (s *WaterScheduleStorage) Search(ctx context.Context, _ string, _ url.Values) ([]*pkg.WaterSchedule, error) {
	dbWaterSchedules, err := s.q.ListWaterSchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing water schedules: %w", err)
	}

	waterSchedules := make([]*pkg.WaterSchedule, len(dbWaterSchedules))
	for i, dbWaterSchedule := range dbWaterSchedules {
		waterSchedule, err := dbWaterScheduleToWaterSchedule(dbWaterSchedule)
		if err != nil {
			return nil, fmt.Errorf("invalid water schedule: %w", err)
		}

		waterSchedules[i] = waterSchedule
	}

	return waterSchedules, nil
}

// Set saves a WaterSchedule to storage (creates or updates)
func (s *WaterScheduleStorage) Set(ctx context.Context, waterSchedule *pkg.WaterSchedule) error {
	var name, description sql.NullString
	if waterSchedule.Name != "" {
		name = sql.NullString{String: waterSchedule.Name, Valid: true}
	}
	if waterSchedule.Description != "" {
		description = sql.NullString{String: waterSchedule.Description, Valid: true}
	}

	var duration, interval interface{}
	if waterSchedule.Duration != nil {
		duration = int64(waterSchedule.Duration.Duration)
	}
	if waterSchedule.Interval != nil {
		interval = int64(waterSchedule.Interval.Duration)
	}

	var startTime string
	if waterSchedule.StartTime != nil {
		startTime = waterSchedule.StartTime.String()
	}

	var endDate sql.NullTime
	if waterSchedule.EndDate != nil {
		endDate = sql.NullTime{Time: *waterSchedule.EndDate, Valid: true}
	}

	var activePeriodStartMonth, activePeriodEndMonth sql.NullString
	if waterSchedule.ActivePeriod != nil {
		activePeriodStartMonth = sql.NullString{String: waterSchedule.ActivePeriod.StartMonth, Valid: true}
		activePeriodEndMonth = sql.NullString{String: waterSchedule.ActivePeriod.EndMonth, Valid: true}
	}

	var weatherControl json.RawMessage
	if waterSchedule.WeatherControl != nil {
		var err error
		weatherControl, err = json.Marshal(waterSchedule.WeatherControl)
		if err != nil {
			return fmt.Errorf("error marshaling weather control: %w", err)
		}
	}

	var notificationClientID sql.NullString
	if waterSchedule.NotificationClientID != nil {
		notificationClientID = sql.NullString{String: *waterSchedule.NotificationClientID, Valid: true}
	}

	return s.q.UpsertWaterSchedule(ctx, db.UpsertWaterScheduleParams{
		ID:                     waterSchedule.ID.String(),
		Name:                   name,
		Description:            description,
		Duration:               duration,
		Interval:               interval,
		StartDate:              *waterSchedule.StartDate,
		StartTime:              startTime,
		EndDate:                endDate,
		ActivePeriodStartMonth: activePeriodStartMonth,
		ActivePeriodEndMonth:   activePeriodEndMonth,
		WeatherControl:         weatherControl,
		NotificationClientID:   notificationClientID,
	})
}

// Delete removes a WaterSchedule from storage
func (s *WaterScheduleStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteWaterSchedule(ctx, id)
}

func dbWaterScheduleToWaterSchedule(dbWaterSchedule db.WaterSchedule) (*pkg.WaterSchedule, error) {
	waterScheduleID, err := parseID(dbWaterSchedule.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid water schedule ID: %w", err)
	}

	waterSchedule := &pkg.WaterSchedule{
		ID:        waterScheduleID,
		StartDate: &dbWaterSchedule.StartDate,
	}

	if dbWaterSchedule.Name.Valid {
		waterSchedule.Name = dbWaterSchedule.Name.String
	}

	if dbWaterSchedule.Description.Valid {
		waterSchedule.Description = dbWaterSchedule.Description.String
	}

	if dur, ok := dbWaterSchedule.Duration.(int64); ok {
		duration := pkg.Duration{Duration: time.Duration(dur)}
		waterSchedule.Duration = &duration
	}

	if intvl, ok := dbWaterSchedule.Interval.(int64); ok {
		interval := pkg.Duration{Duration: time.Duration(intvl)}
		waterSchedule.Interval = &interval
	}

	if dbWaterSchedule.StartTime != "" {
		startTime, err := pkg.StartTimeFromString(dbWaterSchedule.StartTime)
		if err != nil {
			return nil, fmt.Errorf("error parsing start time: %w", err)
		}
		waterSchedule.StartTime = startTime
	}

	if dbWaterSchedule.EndDate.Valid {
		waterSchedule.EndDate = &dbWaterSchedule.EndDate.Time
	}

	if dbWaterSchedule.ActivePeriodStartMonth.Valid && dbWaterSchedule.ActivePeriodEndMonth.Valid {
		waterSchedule.ActivePeriod = &pkg.ActivePeriod{
			StartMonth: dbWaterSchedule.ActivePeriodStartMonth.String,
			EndMonth:   dbWaterSchedule.ActivePeriodEndMonth.String,
		}
	}

	if len(dbWaterSchedule.WeatherControl) > 0 {
		var weatherControl weather.Control
		err := json.Unmarshal(dbWaterSchedule.WeatherControl, &weatherControl)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling weather control: %w", err)
		}
		waterSchedule.WeatherControl = &weatherControl
	}

	if dbWaterSchedule.NotificationClientID.Valid {
		waterSchedule.NotificationClientID = &dbWaterSchedule.NotificationClientID.String
	}

	return waterSchedule, nil
}
