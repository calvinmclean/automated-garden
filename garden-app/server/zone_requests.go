package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/rs/xid"
)

// ZoneRequest wraps a Zone into a request so we can handle Bind/Render in this package
type ZoneRequest struct {
	*pkg.Zone
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (z *ZoneRequest) Bind(r *http.Request) error {
	if z == nil || z.Zone == nil {
		return errors.New("missing required Zone fields")
	}

	if z.Position == nil {
		return errors.New("missing required zone_position field")
	}
	if z.WaterSchedule == nil {
		return errors.New("missing required water_schedule field")
	}
	if z.WaterSchedule.Interval == "" {
		return errors.New("missing required water_schedule.interval field")
	}
	if z.WaterSchedule.Duration == "" {
		return errors.New("missing required water_schedule.duration field")
	}
	// Check that Duration is valid Duration
	if _, err := time.ParseDuration(z.WaterSchedule.Duration); err != nil {
		return fmt.Errorf("invalid duration format for water_schedule.duration: %s", z.WaterSchedule.Duration)
	}
	if z.WaterSchedule.StartTime == nil {
		return errors.New("missing required water_schedule.start_time field")
	}
	if z.Name == "" {
		return errors.New("missing required name field")
	}

	return nil
}

// UpdateZoneRequest wraps a Zone into a request so we can handle Bind/Render in this package
// It has different validation than the ZoneRequest
type UpdateZoneRequest struct {
	*pkg.Zone
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (z *UpdateZoneRequest) Bind(r *http.Request) error {
	if z == nil || z.Zone == nil {
		return errors.New("missing required Zone fields")
	}

	if z.ID != xid.NilID() {
		return errors.New("updating ID is not allowed")
	}
	if z.EndDate != nil {
		return errors.New("to end-date a Zone, please use the DELETE endpoint")
	}

	if z.Zone.WaterSchedule != nil {
		// Check that Duration is valid Duration
		if z.WaterSchedule.Duration != "" {
			if _, err := time.ParseDuration(z.WaterSchedule.Duration); err != nil {
				return fmt.Errorf("invalid duration format for water_schedule.duration: %s", z.WaterSchedule.Duration)
			}
		}
		// Check that StartTime is in the future
		if z.WaterSchedule.StartTime != nil && time.Since(*z.WaterSchedule.StartTime) > 0 {
			return fmt.Errorf("unable to set water_schedule.start_time to time in the past")
		}
	}

	return nil
}

// ZoneActionRequest wraps a ZoneAction into a request so we can handle Bind/Render in this package
type ZoneActionRequest struct {
	*action.ZoneAction
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *ZoneActionRequest) Bind(r *http.Request) error {
	// ZoneAction is nil if no ZoneAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || action.ZoneAction == nil || (action.Water == nil) {
		return errors.New("missing required action fields")
	}
	return nil
}
