package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/db"
)

// AdditionalQueries implements AdditionalQueries interface for SQL storage
type AdditionalQueries struct {
	q *db.Queries
}

// NewAdditionalQueries creates a new AdditionalQueries instance
func NewAdditionalQueries(sqlDB *sql.DB) *AdditionalQueries {
	return &AdditionalQueries{
		q: db.New(sqlDB),
	}
}

// GetZonesUsingWaterSchedule will find all Zones that use this WaterSchedule and return the Zones along with the Gardens they belong to
func (a *AdditionalQueries) GetZonesUsingWaterSchedule(id string) ([]*pkg.ZoneAndGarden, error) {
	ctx := context.Background()

	dbZones, err := a.q.FindZonesByWaterScheduleID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error finding zones by water schedule ID: %w", err)
	}

	results := make([]*pkg.ZoneAndGarden, 0, len(dbZones))
	for _, dbZone := range dbZones {
		zone, err := dbZoneToZone(dbZone)
		if err != nil {
			return nil, fmt.Errorf("invalid zone: %w", err)
		}

		if zone.EndDated() {
			continue
		}

		dbGarden, err := a.q.GetGarden(ctx, dbZone.GardenID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("error getting garden: %w", err)
		}

		garden, err := dbGardenToGarden(dbGarden)
		if err != nil {
			return nil, fmt.Errorf("invalid garden: %w", err)
		}

		if garden.EndDated() {
			continue
		}

		results = append(results, &pkg.ZoneAndGarden{
			Zone:   zone,
			Garden: garden,
		})
	}

	return results, nil
}

// GetWaterSchedulesUsingWeatherClient will return all WaterSchedules that rely on this WeatherClient
func (a *AdditionalQueries) GetWaterSchedulesUsingWeatherClient(id string) ([]*pkg.WaterSchedule, error) {
	ctx := context.Background()

	dbWaterSchedules, err := a.q.FindWaterSchedulesByWeatherClientID(ctx, db.FindWaterSchedulesByWeatherClientIDParams{
		WeatherControl:   sql.NullString{String: id, Valid: true},
		WeatherControl_2: sql.NullString{String: id, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("error finding water schedules by weather client ID: %w", err)
	}

	waterSchedules := make([]*pkg.WaterSchedule, 0, len(dbWaterSchedules))
	for _, dbWaterSchedule := range dbWaterSchedules {
		waterSchedule, err := dbWaterScheduleToWaterSchedule(dbWaterSchedule)
		if err != nil {
			return nil, fmt.Errorf("invalid water schedule: %w", err)
		}

		waterSchedules = append(waterSchedules, waterSchedule)
	}

	return waterSchedules, nil
}
