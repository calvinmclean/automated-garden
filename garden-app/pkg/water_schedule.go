package pkg

import (
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// WaterSchedule allows the user to have more control over how the Plant is watered using an Interval
// and optional MinimumMoisture which acts as the threshold the Plant's soil should be above.
// StartTime specifies when the watering interval should originate from. It can be used to increase/decrease delays in watering.
type WaterSchedule struct {
	ID             xid.ID           `json:"id" yaml:"id"`
	Duration       *Duration        `json:"duration" yaml:"duration"`
	Interval       *Duration        `json:"interval" yaml:"interval"`
	StartTime      *time.Time       `json:"start_time" yaml:"start_time"`
	EndDate        *time.Time       `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	WeatherControl *weather.Control `json:"weather_control,omitempty"`
}

// String...
func (ws *WaterSchedule) String() string {
	return fmt.Sprintf("%+v", *ws)
}

// EndDated returns true if the WaterSchedule is end-dated
func (ws *WaterSchedule) EndDated() bool {
	return ws.EndDate != nil && ws.EndDate.Before(time.Now())
}

// HasWeatherControl is used to determine if weather conditions should be checked before watering the Zone
// This checks that WeatherControl is defined and has at least one type of control configured
func (ws *WaterSchedule) HasWeatherControl() bool {
	return ws != nil &&
		(ws.HasRainControl() || ws.HasSoilMoistureControl() || ws.HasTemperatureControl())
}

// Patch allows modifying the struct in-place with values from a different instance
func (ws *WaterSchedule) Patch(new *WaterSchedule) {
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
		ws.WeatherControl.Temperature != nil &&
		*ws.WeatherControl.Temperature.Factor != 0
}
