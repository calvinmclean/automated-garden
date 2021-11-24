package server

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
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

// GardenActionRequest wraps a GardenAction into a request so we can handle Bind/Render in this package
type GardenActionRequest struct {
	*pkg.GardenAction
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *GardenActionRequest) Bind(r *http.Request) error {
	// PlantAction is nil if no PlantAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || action.GardenAction == nil || (action.Light == nil && action.Stop == nil) {
		return errors.New("missing required action fields")
	}
	// Validate that action.Light.State is "", "ON", or "OFF" (case insensitive)
	state := strings.ToUpper(action.Light.State)
	if state != "" && state != pkg.StateOn && state != pkg.StateOff {
		return fmt.Errorf("invalid \"state\" provided: \"%s\"", action.Light.State)
	}
	return nil
}

// GardenRequest wraps a Garden into a request so we can handle Bind/Render in this package
type GardenRequest struct {
	*pkg.Garden
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (g *GardenRequest) Bind(r *http.Request) error {
	if g == nil || g.Garden == nil {
		return errors.New("missing required Garden fields")
	}
	if g.Name == "" {
		return errors.New("missing required name field")
	}
	illegalRegexp := regexp.MustCompile(`[\$\#\*\>\+\/]`)
	if illegalRegexp.MatchString(g.Name) {
		return errors.New("one or more invalid characters in Garden name")
	}
	if len(g.Plants) > 0 {
		return errors.New("cannot add or modify Plants with this request")
	}

	if g.LightSchedule != nil {
		if g.LightSchedule.Duration == "" {
			return errors.New("missing required light_schedule.duration field")
		}
		if g.LightSchedule.StartTime == "" {
			return errors.New("missing required light_schedule.start_time field")
		}
		// Check that LightSchedule.StartTime is valid
		_, err := time.Parse(pkg.LightTimeFormat, g.LightSchedule.StartTime)
		if err != nil {
			return fmt.Errorf("invalid time format for light_schedule.start_time: %s", g.LightSchedule.StartTime)
		}
	}

	return nil
}

// UpdateGardenRequest wraps a GardenRequest to change how validation occurs
type UpdateGardenRequest struct {
	*pkg.Garden
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (g *UpdateGardenRequest) Bind(r *http.Request) error {
	if g == nil || g.Garden == nil {
		return errors.New("missing required Garden fields")
	}
	illegalRegexp := regexp.MustCompile(`[\$\#\*\>\+\/]`)
	if illegalRegexp.MatchString(g.Name) {
		return errors.New("one or more invalid characters in Garden name")
	}
	if len(g.Plants) > 0 {
		return errors.New("cannot add or modify Plants with this request")
	}
	if g.EndDate != nil {
		return errors.New("to end-date a Garden, please use the DELETE endpoint")
	}

	if g.LightSchedule != nil {
		// Check that LightSchedule.StartTime is valid
		if g.LightSchedule.StartTime != "" {
			_, err := time.Parse(pkg.LightTimeFormat, g.LightSchedule.StartTime)
			if err != nil {
				return fmt.Errorf("invalid time format for light_schedule.start_time: %s", g.LightSchedule.StartTime)
			}
		}
	}
	return nil
}
