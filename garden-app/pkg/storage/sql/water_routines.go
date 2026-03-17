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

// WaterRoutineStorage implements babyapi.Storage interface for WaterRoutines using SQL
type WaterRoutineStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*pkg.WaterRoutine] = &WaterRoutineStorage{}

// NewWaterRoutineStorage creates a new WaterRoutineStorage instance
func NewWaterRoutineStorage(sqlDB *sql.DB) *WaterRoutineStorage {
	return &WaterRoutineStorage{
		q: db.New(sqlDB),
	}
}

// Get retrieves a WaterRoutine from storage by ID
func (s *WaterRoutineStorage) Get(ctx context.Context, id string) (*pkg.WaterRoutine, error) {
	dbWaterRoutine, err := s.q.GetWaterRoutine(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, babyapi.ErrNotFound
		}
		return nil, fmt.Errorf("error getting water routine: %w", err)
	}

	return dbWaterRoutineToWaterRoutine(dbWaterRoutine)
}

// Search returns all WaterRoutines from storage
func (s *WaterRoutineStorage) Search(ctx context.Context, _ string, _ url.Values) ([]*pkg.WaterRoutine, error) {
	dbWaterRoutines, err := s.q.ListWaterRoutines(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing water routines: %w", err)
	}

	waterRoutines := make([]*pkg.WaterRoutine, len(dbWaterRoutines))
	for i, dbWaterRoutine := range dbWaterRoutines {
		waterRoutine, err := dbWaterRoutineToWaterRoutine(dbWaterRoutine)
		if err != nil {
			return nil, fmt.Errorf("invalid water routine: %w", err)
		}

		waterRoutines[i] = waterRoutine
	}

	return waterRoutines, nil
}

// Set saves a WaterRoutine to storage (creates or updates)
func (s *WaterRoutineStorage) Set(ctx context.Context, waterRoutine *pkg.WaterRoutine) error {
	steps, err := json.Marshal(waterRoutine.Steps)
	if err != nil {
		return fmt.Errorf("error marshaling steps: %w", err)
	}

	return s.q.UpsertWaterRoutine(ctx, db.UpsertWaterRoutineParams{
		ID:    waterRoutine.ID.String(),
		Name:  waterRoutine.Name,
		Steps: steps,
	})
}

// Delete removes a WaterRoutine from storage
func (s *WaterRoutineStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteWaterRoutine(ctx, id)
}

func dbWaterRoutineToWaterRoutine(dbWaterRoutine db.WaterRoutine) (*pkg.WaterRoutine, error) {
	waterRoutineID, err := parseID(dbWaterRoutine.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid water routine ID: %w", err)
	}

	waterRoutine := &pkg.WaterRoutine{
		ID:   waterRoutineID,
		Name: dbWaterRoutine.Name,
	}

	if len(dbWaterRoutine.Steps) > 0 {
		var steps []pkg.WaterRoutineStep
		err := json.Unmarshal(dbWaterRoutine.Steps, &steps)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling steps: %w", err)
		}
		waterRoutine.Steps = steps
	}

	return waterRoutine, nil
}
