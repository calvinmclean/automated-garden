package pkg

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
)

// WaterSchedule allows the user to have more control over how the Zone is watered using an Interval
// and optional MinimumMoisture which acts as the threshold the Zone's soil should be above.
// StartTime specifies when the watering interval should originate from. It can be used to increase/decrease delays in watering.
type WaterSchedule struct {
	ID             babyapi.ID       `json:"id" yaml:"id"`
	Duration       *Duration        `json:"duration" yaml:"duration"`
	Interval       *Duration        `json:"interval" yaml:"interval"`
	StartTime      *StartTime       `json:"start_time" yaml:"start_time"`
	EndDate        *time.Time       `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	WeatherControl *weather.Control `json:"weather_control,omitempty" yaml:"weather_control,omitempty"`
	Name           string           `json:"name,omitempty" yaml:"name,omitempty"`
	Description    string           `json:"description,omitempty" yaml:"description,omitempty"`
	ActivePeriod   *ActivePeriod    `json:"active_period,omitempty" yaml:"active_period,omitempty"`
}

func (ws *WaterSchedule) GetID() string {
	return ws.ID.String()
}

// String...
func (ws *WaterSchedule) String() string {
	return fmt.Sprintf("%+v", *ws)
}

// EndDated returns true if the WaterSchedule is end-dated
func (ws *WaterSchedule) EndDated() bool {
	return ws.EndDate != nil && ws.EndDate.Before(time.Now())
}

func (ws *WaterSchedule) SetEndDate(now time.Time) {
	ws.EndDate = &now
}

// HasWeatherControl is used to determine if weather conditions should be checked before watering the Zone
// This checks that WeatherControl is defined and has at least one type of control configured
func (ws *WaterSchedule) HasWeatherControl() bool {
	return ws != nil &&
		(ws.HasRainControl() || ws.HasSoilMoistureControl() || ws.HasTemperatureControl())
}

// Patch allows modifying the struct in-place with values from a different instance
func (ws *WaterSchedule) Patch(new *WaterSchedule) *babyapi.ErrResponse {
	if new.Duration != nil {
		ws.Duration = new.Duration
	}
	if new.Interval != nil {
		ws.Interval = new.Interval
	}
	if new.StartTime != nil {
		ws.StartTime = new.StartTime
	}
	if ws.EndDate != nil && new.EndDate == nil {
		ws.EndDate = new.EndDate
	}
	if new.WeatherControl != nil {
		if ws.WeatherControl == nil {
			ws.WeatherControl = &weather.Control{}
		}
		ws.WeatherControl.Patch(new.WeatherControl)
	}
	if new.Name != "" {
		ws.Name = new.Name
	}
	if new.Description != "" {
		ws.Description = new.Description
	}
	if new.ActivePeriod != nil {
		if ws.ActivePeriod == nil {
			ws.ActivePeriod = &ActivePeriod{}
		}
		ws.ActivePeriod.Patch(new.ActivePeriod)
	}

	return nil
}

// HasRainControl is used to determine if rain conditions should be checked before watering the Zone
func (ws *WaterSchedule) HasRainControl() bool {
	return ws.WeatherControl != nil &&
		ws.WeatherControl.Rain != nil
}

// HasSoilMoistureControl is used to determine if soil moisture conditions should be checked before watering the Zone
func (ws *WaterSchedule) HasSoilMoistureControl() bool {
	return ws.WeatherControl != nil &&
		ws.WeatherControl.SoilMoisture != nil &&
		ws.WeatherControl.SoilMoisture.MinimumMoisture != nil
}

// HasTemperatureControl is used to determine if configuration is available for environmental scaling
func (ws *WaterSchedule) HasTemperatureControl() bool {
	return ws.WeatherControl != nil &&
		ws.WeatherControl.Temperature != nil
}

// IsActive determines if the WaterSchedule is currently in it's ActivePeriod. Always true if no ActivePeriod is configured
func (ws *WaterSchedule) IsActive(now time.Time) bool {
	if ws.ActivePeriod == nil {
		return true
	}

	// Run validate to make sure start/end values are set. No chance of error since validation has already happened
	_ = ws.ActivePeriod.Validate()

	// Set current year to this year for easy comparison
	ws.ActivePeriod.start = ws.ActivePeriod.start.AddDate(now.Year(), 0, 0)
	ws.ActivePeriod.end = ws.ActivePeriod.end.AddDate(now.Year(), 0, 0)

	// Handle wraparound dates like December -> February (Winter)
	// If the period starts before now, we need to bump the end time by a year, otherwise
	// the start period needs to be last year
	if ws.ActivePeriod.start.After(ws.ActivePeriod.end) {
		if ws.ActivePeriod.start.Before(now) {
			ws.ActivePeriod.end = ws.ActivePeriod.end.AddDate(1, 0, 0)
		} else {
			ws.ActivePeriod.start = ws.ActivePeriod.start.AddDate(-1, 0, 0)
		}
	}

	return now.Month() == ws.ActivePeriod.start.Month() || // currently start month
		now.Month() == ws.ActivePeriod.end.Month() || // currently end month
		(now.After(ws.ActivePeriod.start) && now.Before(ws.ActivePeriod.end)) // somewhere in-between
}

// ActivePeriod contains the start and end months for when a WaterSchedule should be considered active. Both of these constraints are inclusive
type ActivePeriod struct {
	StartMonth string `json:"start_month" yaml:"start_month"`
	EndMonth   string `json:"end_month" yaml:"end_month"`

	start time.Time
	end   time.Time
}

// Validate parses the Month strings to make sure they are valid
func (ap *ActivePeriod) Validate() error {
	if ap == nil {
		return nil
	}

	var err error
	ap.start, err = time.Parse("January", ap.StartMonth)
	if err != nil {
		return fmt.Errorf("invalid StartMonth: %w", err)
	}
	ap.end, err = time.Parse("January", ap.EndMonth)
	if err != nil {
		return fmt.Errorf("invalid EndMonth: %w", err)
	}

	if ap.start.Month() == ap.end.Month() {
		return fmt.Errorf("StartMonth and EndMonth must be different")
	}

	return nil
}

// Patch allows for easily updating/editing an ActivePeriod
func (ap *ActivePeriod) Patch(new *ActivePeriod) {
	if new.StartMonth != "" {
		ap.StartMonth = new.StartMonth
	}
	if new.EndMonth != "" {
		ap.EndMonth = new.EndMonth
	}
}

// NextWaterDetails has information about the next time this WaterSchedule will be used
type NextWaterDetails struct {
	Time            *time.Time `json:"time,omitempty"`
	Duration        string     `json:"duration,omitempty"`
	WaterScheduleID *xid.ID    `json:"water_schedule_id,omitempty"`
	Message         string     `json:"message,omitempty"`
}

// WeatherData is used to represent the data used for WeatherControl to a user
type WeatherData struct {
	Rain                *RainData        `json:"rain,omitempty"`
	Temperature         *TemperatureData `json:"average_temperature,omitempty"`
	SoilMoisturePercent *float64         `json:"soil_moisture_percent,omitempty"`
}

// RainData shows the total rain in the last watering interval and the scaling factor it would result in
type RainData struct {
	MM          float32 `json:"mm"`
	ScaleFactor float32 `json:"scale_factor"`
}

// TemperatureData shows the average high temperatures in the last watering interval and the scaling factor it would result in
type TemperatureData struct {
	Celsius     float32 `json:"celsius"`
	ScaleFactor float32 `json:"scale_factor"`
}

func (ws *WaterSchedule) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (ws *WaterSchedule) Bind(r *http.Request) error {
	if ws == nil {
		return errors.New("missing required WaterSchedule fields")
	}
	err := ws.ID.Bind(r)
	if err != nil {
		return err
	}

	switch r.Method {
	case http.MethodPut, http.MethodPost:
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
			// Allow removing active period by setting empty for each. This is useful for HTML form
			if ws.ActivePeriod.StartMonth == "" && ws.ActivePeriod.EndMonth == "" {
				ws.ActivePeriod = nil
			}
		}
	case http.MethodPatch:
		if ws.EndDate != nil {
			return errors.New("to end-date a WaterSchedule, please use the DELETE endpoint")
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
			return errors.New("error validating moisture_control: missing required field: minimum_moisture")
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
