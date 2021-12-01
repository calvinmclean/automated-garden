package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// PlantRequest wraps a Plant into a request so we can handle Bind/Render in this package
type PlantRequest struct {
	*pkg.Plant
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (p *PlantRequest) Bind(r *http.Request) error {
	if p == nil || p.Plant == nil {
		return errors.New("missing required Plant fields")
	}

	if p.PlantPosition == nil {
		return errors.New("missing required plant_position field")
	}
	if p.WaterSchedule == nil {
		return errors.New("missing required water_schedule field")
	}
	if p.WaterSchedule.Interval == "" {
		return errors.New("missing required water_schedule.interval field")
	}
	if p.WaterSchedule.Duration == "" {
		return errors.New("missing required water_schedule.duration field")
	}
	// Check that Duration is valid Duration
	if _, err := time.ParseDuration(p.WaterSchedule.Duration); err != nil {
		return fmt.Errorf("invalid duration format for water_schedule.duration: %s", p.WaterSchedule.Duration)
	}
	if p.WaterSchedule.StartTime == nil {
		return errors.New("missing required water_schedule.start_time field")
	}
	if p.Name == "" {
		return errors.New("missing required name field")
	}
	if p.GardenID != xid.NilID() {
		return errors.New("manual specification of garden ID is not allowed")
	}

	return nil
}

// UpdatePlantRequest wraps a Plant into a request so we can handle Bind/Render in this package
// It has different validation than the PlantRequest
type UpdatePlantRequest struct {
	*pkg.Plant
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (p *UpdatePlantRequest) Bind(r *http.Request) error {
	if p == nil || p.Plant == nil {
		return errors.New("missing required Plant fields")
	}

	if p.ID != xid.NilID() {
		return errors.New("updating ID is not allowed")
	}
	if p.GardenID != xid.NilID() {
		return errors.New("updating garden ID is not allowed")
	}
	if p.EndDate != nil {
		return errors.New("to end-date a Plant, please use the DELETE endpoint")
	}

	if p.Plant.WaterSchedule != nil {
		// Check that Duration is valid Duration
		if p.WaterSchedule.Duration != "" {
			if _, err := time.ParseDuration(p.WaterSchedule.Duration); err != nil {
				return fmt.Errorf("invalid duration format for water_schedule.duration: %s", p.WaterSchedule.Duration)
			}
		}
		// Check that StartTime is in the future
		if p.WaterSchedule.StartTime != nil && time.Since(*p.WaterSchedule.StartTime) > 0 {
			return fmt.Errorf("unable to set water_schedule.start_time to time in the past")
		}
	}

	return nil
}

// PlantActionRequest wraps a PlantAction into a request so we can handle Bind/Render in this package
type PlantActionRequest struct {
	*pkg.PlantAction
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *PlantActionRequest) Bind(r *http.Request) error {
	// PlantAction is nil if no PlantAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || action.PlantAction == nil || (action.Water == nil) {
		return errors.New("missing required action fields")
	}
	return nil
}
