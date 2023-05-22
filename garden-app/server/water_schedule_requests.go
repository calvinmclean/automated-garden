package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// WaterScheduleRequest wraps a WaterSchedule into a request so we can handle Bind/Render in this package
type WaterScheduleRequest struct {
	*pkg.WaterSchedule
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (ws *WaterScheduleRequest) Bind(r *http.Request) error {
	if ws == nil || ws.WaterSchedule == nil {
		return errors.New("missing required WaterSchedule fields")
	}

	if ws.Interval == nil {
		return errors.New("missing required interval field")
	}
	if ws.Duration == nil {
		return errors.New("missing required duration field")
	}
	if ws.StartTime == nil {
		return errors.New("missing required start_time field")
	}
	if ws.WeatherControl != nil {
		err := ValidateWeatherControl(ws.WeatherControl)
		if err != nil {
			return fmt.Errorf("error validating weather_control: %w", err)
		}
	}
	if ws.ActivePeriod != nil {
		err := ws.ActivePeriod.Validate()
		if err != nil {
			return fmt.Errorf("error validating active_period: %w", err)
		}
	}

	return nil
}

// ValidateWeatherControl validates input for the WeatherControl of a WaterSchedule
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

// UpdateWaterScheduleRequest wraps a WaterSchedule into a request so we can handle Bind/Render in this package
// It has different validation than the WaterScheduleRequest
type UpdateWaterScheduleRequest struct {
	*pkg.WaterSchedule
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (ws *UpdateWaterScheduleRequest) Bind(r *http.Request) error {
	if ws == nil || ws.WaterSchedule == nil {
		return errors.New("missing required WaterSchedule fields")
	}

	if ws.ID != xid.NilID() {
		return errors.New("updating ID is not allowed")
	}
	if ws.EndDate != nil {
		return errors.New("to end-date a WaterSchedule, please use the DELETE endpoint")
	}

	// Check that StartTime is in the future
	if ws.StartTime != nil && time.Since(*ws.StartTime) > 0 {
		return fmt.Errorf("unable to set start_time to time in the past")
	}

	if ws.ActivePeriod != nil {
		err := ws.ActivePeriod.Validate()
		if err != nil {
			return fmt.Errorf("error validating active_period: %w", err)
		}
	}

	return nil
}
