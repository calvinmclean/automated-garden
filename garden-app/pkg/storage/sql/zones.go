package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/sql/db"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
)

// ZoneStorage implements babyapi.Storage interface for Zones using SQL
type ZoneStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*pkg.Zone] = &ZoneStorage{}

// NewZoneStorage creates a new ZoneStorage instance
func NewZoneStorage(sqlDB *sql.DB) *ZoneStorage {
	return &ZoneStorage{
		q: db.New(sqlDB),
	}
}

// Set saves a Zone to storage (creates or updates)
func (s *ZoneStorage) Set(ctx context.Context, zone *pkg.Zone) error {
	var waterScheduleIDs string
	if len(zone.WaterScheduleIDs) > 0 {
		strIDs := make([]string, len(zone.WaterScheduleIDs))
		for i, id := range zone.WaterScheduleIDs {
			strIDs[i] = id.String()
		}
		waterScheduleIDs = strings.Join(strIDs, ",")
	}

	var position, skipCount sql.NullInt64
	if zone.Position != nil {
		position = sql.NullInt64{Int64: int64(*zone.Position), Valid: true}
	}
	if zone.SkipCount != nil {
		skipCount = sql.NullInt64{Int64: int64(*zone.SkipCount), Valid: true}
	}

	var endDate sql.NullTime
	if zone.EndDate != nil {
		endDate = sql.NullTime{Time: *zone.EndDate, Valid: true}
	}

	var details pkg.ZoneDetails
	if zone.Details != nil {
		details = *zone.Details
	}

	createdAt := time.Now()
	if zone.CreatedAt != nil {
		createdAt = *zone.CreatedAt
	}

	return s.q.UpsertZone(ctx, db.UpsertZoneParams{
		ID:                 zone.ID.String(),
		Name:               zone.Name,
		GardenID:           zone.GardenID.String(),
		DetailsDescription: sql.NullString{String: details.Description, Valid: details.Description != ""},
		DetailsNotes:       sql.NullString{String: details.Notes, Valid: details.Notes != ""},
		Position:           position,
		SkipCount:          skipCount,
		CreatedAt:          createdAt,
		EndDate:            endDate,
		WaterScheduleIds:   sql.NullString{String: waterScheduleIDs, Valid: len(waterScheduleIDs) > 0},
	})
}

// Get retrieves a Zone from storage by ID
func (s *ZoneStorage) Get(ctx context.Context, id string) (*pkg.Zone, error) {
	dbZone, err := s.q.GetZone(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, babyapi.ErrNotFound
		}
		return nil, fmt.Errorf("error getting zone: %w", err)
	}

	return dbZoneToZone(dbZone)
}

// List returns all Zones from storage
func (s *ZoneStorage) Search(ctx context.Context, gardenID string, q url.Values) ([]*pkg.Zone, error) {
	getEndDated := q.Get("end_dated") == "true"

	listZones := s.q.ListActiveZones
	if getEndDated {
		listZones = s.q.ListAllZones
	}

	dbZones, err := listZones(ctx, gardenID)
	if err != nil {
		return nil, fmt.Errorf("error listing zones: %w", err)
	}

	zones := make([]*pkg.Zone, len(dbZones))
	for i, dbZone := range dbZones {
		zone, err := dbZoneToZone(dbZone)
		if err != nil {
			return nil, fmt.Errorf("invalid zone: %w", err)
		}

		zones[i] = zone
	}

	return zones, nil
}

// Delete removes a Zone from storage
func (s *ZoneStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteZone(ctx, id)
}

// FindByWaterScheduleID returns all Zones associated with a given water schedule ID
func (s *ZoneStorage) FindByWaterScheduleID(ctx context.Context, waterScheduleID string) ([]*pkg.Zone, error) {
	dbZones, err := s.q.FindZonesByWaterScheduleID(ctx, waterScheduleID)
	if err != nil {
		return nil, fmt.Errorf("error finding zones by water schedule ID: %w", err)
	}

	zones := make([]*pkg.Zone, len(dbZones))
	for i, dbZone := range dbZones {
		zones[i], err = dbZoneToZone(dbZone)
	}

	return zones, nil
}

func dbZoneToZone(dbZone db.Zone) (*pkg.Zone, error) {
	zoneID, err := parseID(dbZone.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid zone ID: %w", err)
	}
	gardenID, err := xid.FromString(dbZone.GardenID)
	if err != nil {
		return nil, fmt.Errorf("invalid garden ID: %w", err)
	}

	zone := &pkg.Zone{
		ID:        zoneID,
		Name:      dbZone.Name,
		GardenID:  gardenID,
		CreatedAt: &dbZone.CreatedAt,
	}

	if dbZone.DetailsDescription.Valid || dbZone.DetailsNotes.Valid {
		zone.Details = &pkg.ZoneDetails{}
		if dbZone.DetailsDescription.Valid {
			zone.Details.Description = dbZone.DetailsDescription.String
		}
		if dbZone.DetailsNotes.Valid {
			zone.Details.Notes = dbZone.DetailsNotes.String
		}
	}

	if dbZone.Position.Valid {
		position := uint(dbZone.Position.Int64)
		zone.Position = &position
	}

	if dbZone.SkipCount.Valid {
		skipCount := uint(dbZone.SkipCount.Int64)
		zone.SkipCount = &skipCount
	}

	if dbZone.EndDate.Valid {
		zone.EndDate = &dbZone.EndDate.Time
	}

	if dbZone.WaterScheduleIds.Valid && dbZone.WaterScheduleIds.String != "" {
		strIDs := strings.Split(dbZone.WaterScheduleIds.String, ",")
		zone.WaterScheduleIDs = make([]xid.ID, len(strIDs))
		for i, strID := range strIDs {
			id, err := xid.FromString(strID)
			if err != nil {
				return nil, fmt.Errorf("invalid water schedule ID: %w", err)
			}
			zone.WaterScheduleIDs[i] = id
		}
	}

	return zone, nil
}
