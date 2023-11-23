package server

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
)

// GardenRequest wraps a Garden into a request so we can handle Bind/Render in this package
type GardenRequest struct {
	*pkg.Garden
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (g *GardenRequest) Bind(_ *http.Request) error {
	if g == nil || g.Garden == nil {
		return errors.New("missing required Garden fields")
	}
	if g.Name == "" {
		return errors.New("missing required name field")
	}
	if g.TopicPrefix == "" {
		return errors.New("missing required topic_prefix field")
	}
	illegalRegexp := regexp.MustCompile(`[\$\#\*\>\+\/]`)
	if illegalRegexp.MatchString(g.TopicPrefix) {
		return errors.New("one or more invalid characters in Garden topic_prefix")
	}
	if g.MaxZones == nil {
		return errors.New("missing required max_zones field")
	} else if *g.MaxZones == 0 {
		return errors.New("max_zones must not be 0")
	}
	if len(g.Zones) > 0 {
		return errors.New("cannot add or modify Zones with this request")
	}
	if g.LightSchedule != nil {
		if g.LightSchedule.Duration == nil {
			return errors.New("missing required light_schedule.duration field")
		}

		// Check that Duration is valid Duration
		if g.LightSchedule.Duration.Duration >= 24*time.Hour {
			return fmt.Errorf("invalid light_schedule.duration >= 24 hours: %s", g.LightSchedule.Duration)
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
func (g *UpdateGardenRequest) Bind(_ *http.Request) error {
	if g == nil || g.Garden == nil {
		return errors.New("missing required Garden fields")
	}
	illegalRegexp := regexp.MustCompile(`[\$\#\*\>\+\/]`)
	if illegalRegexp.MatchString(g.TopicPrefix) {
		return errors.New("one or more invalid characters in Garden topic_prefix")
	}
	if g.EndDate != nil {
		return errors.New("to end-date a Garden, please use the DELETE endpoint")
	}
	if g.MaxZones != nil && *g.MaxZones == 0 {
		return errors.New("max_zones must not be 0")
	}

	if g.LightSchedule != nil {
		// Check that Duration is valid Duration
		if g.LightSchedule.Duration != nil {
			if g.LightSchedule.Duration.Duration >= 24*time.Hour {
				return fmt.Errorf("invalid light_schedule.duration >= 24 hours: %s", g.LightSchedule.Duration)
			}
		}
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

// GardenActionRequest wraps a GardenAction into a request so we can handle Bind/Render in this package
type GardenActionRequest struct {
	*action.GardenAction

	// These are flattened fields so the API can be compatible with HTMX's hx-post. The Bind method will convert
	// these flattened values to fit the base GardenAction type
	LightState *pkg.LightState `json:"light_state"`
	StopAll    *bool           `json:"stop_all"`
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid and convert HTMX flattened values
func (gar *GardenActionRequest) Bind(_ *http.Request) error {
	if gar == nil {
		return errors.New("missing required action fields")
	}
	if gar.GardenAction == nil {
		gar.GardenAction = &action.GardenAction{}
	}
	if gar.LightState != nil {
		gar.Light = &action.LightAction{State: *gar.LightState}
	}
	if gar.StopAll != nil {
		gar.Stop = &action.StopAction{All: *gar.StopAll}
	}

	if gar.Light == nil && gar.Stop == nil {
		return errors.New("missing required action fields")
	}
	if gar.Light != nil && gar.Light.ForDuration != nil {
		if gar.Light.ForDuration.Duration < 0 {
			return errors.New("delay duration must be greater than 0")
		}
	}
	return nil
}
