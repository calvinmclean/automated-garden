package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

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

	var position, skipCount any
	if zone.Position != nil {
		position = int64(*zone.Position)
	}
	if zone.SkipCount != nil {
		skipCount = int64(*zone.SkipCount)
	}

	var endDate sql.NullTime
	if zone.EndDate != nil {
		endDate = sql.NullTime{Time: *zone.EndDate, Valid: true}
	}

	var details pkg.ZoneDetails
	if zone.Details != nil {
		details = *zone.Details
	}

	return s.q.UpsertZone(ctx, db.UpsertZoneParams{
		ID:                 zone.ID.String(),
		Name:               zone.Name,
		GardenID:           zone.GardenID.String(),
		DetailsDescription: details.Description,
		DetailsNotes:       details.Notes,
		Position:           position,
		SkipCount:          skipCount,
		CreatedAt:          *zone.CreatedAt,
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
func (s *ZoneStorage) Search(ctx context.Context, gardenID string, _ url.Values) ([]*pkg.Zone, error) {
	dbZones, err := s.q.ListZones(ctx, gardenID)
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
		ID:       zoneID,
		Name:     dbZone.Name,
		GardenID: gardenID,
		Details: &pkg.ZoneDetails{
			Description: dbZone.DetailsDescription,
			Notes:       dbZone.DetailsNotes,
		},
		CreatedAt: &dbZone.CreatedAt,
	}

	if pos, ok := dbZone.Position.(int64); ok {
		position := uint(pos)
		zone.Position = &position
	}

	if skip, ok := dbZone.SkipCount.(int64); ok {
		skipCount := uint(skip)
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
