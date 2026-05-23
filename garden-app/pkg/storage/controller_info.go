package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/db"
)

// ControllerInfoStorage implements storage for ControllerInfo
type ControllerInfoStorage struct {
	q *db.Queries
}

// NewControllerInfoStorage creates a new ControllerInfoStorage instance
func NewControllerInfoStorage(sqlDB *sql.DB) *ControllerInfoStorage {
	return &ControllerInfoStorage{
		q: db.New(sqlDB),
	}
}

// Get retrieves ControllerInfo for a garden
func (s *ControllerInfoStorage) Get(ctx context.Context, gardenID string) (*pkg.ControllerInfo, error) {
	dbInfo, err := s.q.GetControllerInfo(ctx, gardenID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting controller info: %w", err)
	}
	return dbControllerInfoToControllerInfo(dbInfo), nil
}

// Upsert creates or updates ControllerInfo for a garden
func (s *ControllerInfoStorage) Upsert(ctx context.Context, info *pkg.ControllerInfo) error {
	var macAddress sql.NullString
	if info.MACAddress != "" {
		macAddress = sql.NullString{String: info.MACAddress, Valid: true}
	}

	var ipAddress sql.NullString
	if info.IPAddress != "" {
		ipAddress = sql.NullString{String: info.IPAddress, Valid: true}
	}

	var firmwareVersion sql.NullString
	if info.FirmwareVersion != "" {
		firmwareVersion = sql.NullString{String: info.FirmwareVersion, Valid: true}
	}

	updatedAt := time.Now().Format(time.RFC3339)
	if info.UpdatedAt != nil {
		updatedAt = info.UpdatedAt.Format(time.RFC3339)
	}

	return s.q.UpsertControllerInfo(ctx, db.UpsertControllerInfoParams{
		GardenID:        info.GardenID,
		MacAddress:      macAddress,
		IpAddress:       ipAddress,
		FirmwareVersion: firmwareVersion,
		UpdatedAt:       updatedAt,
	})
}

// Delete removes ControllerInfo for a garden
func (s *ControllerInfoStorage) Delete(ctx context.Context, gardenID string) error {
	return s.q.DeleteControllerInfo(ctx, gardenID)
}

func dbControllerInfoToControllerInfo(dbInfo db.GardenControllerInfo) *pkg.ControllerInfo {
	info := &pkg.ControllerInfo{
		GardenID: dbInfo.GardenID,
	}

	if dbInfo.MacAddress.Valid {
		info.MACAddress = dbInfo.MacAddress.String
	}
	if dbInfo.IpAddress.Valid {
		info.IPAddress = dbInfo.IpAddress.String
	}
	if dbInfo.FirmwareVersion.Valid {
		info.FirmwareVersion = dbInfo.FirmwareVersion.String
	}

	if dbInfo.UpdatedAt != "" {
		updatedAt, err := time.Parse(time.RFC3339, dbInfo.UpdatedAt)
		if err == nil {
			info.UpdatedAt = &updatedAt
		}
	}

	return info
}
