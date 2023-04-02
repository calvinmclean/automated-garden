package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
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
	if z.WaterSchedule.Interval == nil {
		return errors.New("missing required water_schedule.interval field")
	}
	if z.WaterSchedule.Duration == nil {
		return errors.New("missing required water_schedule.duration field")
	}
	if z.WaterSchedule.StartTime == nil {
		return errors.New("missing required water_schedule.start_time field")
	}
	if z.Name == "" {
		return errors.New("missing required name field")
	}
	if z.WaterSchedule.WeatherControl != nil {
		err := ValidateWeatherControl(z.WaterSchedule.WeatherControl)
		if err != nil {
			return fmt.Errorf("error validating weather_control: %w", err)
		}
	}

	return nil
}

// ValidateWeatherControl validates input for the WeatherControl of a Zone
func ValidateWeatherControl(wc *weather.Control) error {
	if wc.Temperature != nil {
		err := ValidateScaleControl(wc.Temperature)
		if err != nil {
			return fmt.Errorf("error validating temperature_control: %w", err)
		}
	}
	if wc.Rain != nil {
		err := ValidateScaleControl(wc.Rain)
		if err != nil {
			return fmt.Errorf("error validating rain_control: %w", err)
		}
	}
	if wc.SoilMoisture != nil {
		if wc.SoilMoisture.MinimumMoisture == nil {
			return errors.New("missing required field: minimum_moisture")
		}
	}
	return nil
}

// ValidateScaleControl validates input for ScaleControl
func ValidateScaleControl(sc *weather.ScaleControl) error {
	errStringFormat := "missing required field: %s"
	if sc.BaselineValue == nil {
		return fmt.Errorf(errStringFormat, "baseline_value")
	}
	if sc.Factor == nil {
		return fmt.Errorf(errStringFormat, "factor")
	}
	if *sc.Factor > float32(1) || *sc.Factor < float32(0) {
		return errors.New("factor must be between 0 and 1")
	}
	if sc.Range == nil {
		return fmt.Errorf(errStringFormat, "range")
	}
	if *sc.Range < float32(0) {
		return errors.New("range must be a positive number")
	}
	if sc.ClientID.IsNil() {
		return fmt.Errorf(errStringFormat, "client_id")
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
