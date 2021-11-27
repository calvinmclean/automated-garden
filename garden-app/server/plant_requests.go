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
	if p.WateringStrategy == nil {
		return errors.New("missing required watering_strategy field")
	}
	if p.WateringStrategy.Interval == "" {
		return errors.New("missing required watering_strategy.interval field")
	}
	if p.WateringStrategy.WateringAmount == 0 {
		return errors.New("missing required watering_strategy.watering_amount field")
	}
	if p.WateringStrategy.StartTime == "" {
		return errors.New("missing required watering_strategy.start_time field")
	}
	// Check that water time is valid
	_, err := time.Parse(pkg.WaterTimeFormat, p.WateringStrategy.StartTime)
	if err != nil {
		return fmt.Errorf("invalid time format for watering_strategy.start_time: %s", p.WateringStrategy.StartTime)
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

	if p.Plant.WateringStrategy != nil && p.WateringStrategy.StartTime != "" {
		// Check that water time is valid
		_, err := time.Parse(pkg.WaterTimeFormat, p.WateringStrategy.StartTime)
		if err != nil {
			return fmt.Errorf("invalid time format for watering_strategy.start_time: %s", p.WateringStrategy.StartTime)
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
