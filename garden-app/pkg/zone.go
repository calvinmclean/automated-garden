package pkg

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// Zone represents a "waterable resource" that is owned by a Garden and can be associated with multiple Plants.
// This allows for more complex Garden setups where a large irrigation system will be watering entire groups of
// Plants rather than watering individually. This contains the important information for managing WaterSchedules
// and some additional details describing the Zone. The Position is an integer that tells the controller which
// part of hardware needs to be switched on to start watering
type Zone struct {
	Name          string         `json:"name" yaml:"name,omitempty"`
	Details       *ZoneDetails   `json:"details,omitempty" yaml:"details,omitempty"`
	ID            xid.ID         `json:"id" yaml:"id,omitempty"`
	Position      *uint          `json:"position" yaml:"position"`
	CreatedAt     *time.Time     `json:"created_at" yaml:"created_at,omitempty"`
	EndDate       *time.Time     `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	WaterSchedule *WaterSchedule `json:"water_schedule,omitempty" yaml:"water_schedule,omitempty"`
}

// EndDated returns true if the Zone is end-dated
func (z *Zone) EndDated() bool {
	return z.EndDate != nil && z.EndDate.Before(time.Now())
}

// Patch allows for easily updating individual fields of a Zone by passing in a new Zone containing
// the desired values
func (z *Zone) Patch(newZone *Zone) {
	if newZone.Name != "" {
		z.Name = newZone.Name
	}
	if newZone.Position != nil {
		z.Position = newZone.Position
	}
	if newZone.CreatedAt != nil {
		z.CreatedAt = newZone.CreatedAt
	}
	if z.EndDate != nil && newZone.EndDate == nil {
		z.EndDate = newZone.EndDate
	}

	if newZone.WaterSchedule != nil {
		// Initiate WaterSchedule if it is nil
		if z.WaterSchedule == nil {
			z.WaterSchedule = &WaterSchedule{}
		}
		z.WaterSchedule.Patch(newZone.WaterSchedule)
	}

	if newZone.Details != nil {
		// Initiate Details if it is nil
		if z.Details == nil {
			z.Details = &ZoneDetails{}
		}
		z.Details.Patch(newZone.Details)
	}
}

// HasWeatherControl is used to determine if weather conditions should be checked before watering the Zone
// This checks that WeatherControl is defined and has at least one type of control configured
func (z *Zone) HasWeatherControl() bool {
	return z.WaterSchedule != nil &&
		(z.WaterSchedule.HasRainControl() || z.WaterSchedule.HasSoilMoistureControl() || z.WaterSchedule.HasTemperatureControl())
}

// ZoneDetails is a struct holding some additional details about a Zone that are primarily for user convenience
// and are generally not used by the application
type ZoneDetails struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Notes       string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// Patch allows modifying the struct in-place with values from a different instance
func (zd *ZoneDetails) Patch(new *ZoneDetails) {
	if new.Description != "" {
		zd.Description = new.Description
	}
	if new.Notes != "" {
		zd.Notes = new.Notes
	}
}

// WaterSchedule allows the user to have more control over how the Plant is watered using an Interval
// and optional MinimumMoisture which acts as the threshold the Plant's soil should be above.
// StartTime specifies when the watering interval should originate from. It can be used to increase/decrease delays in watering.
type WaterSchedule struct {
	Duration       *Duration        `json:"duration" yaml:"duration"`
	Interval       string           `json:"interval" yaml:"interval"`
	StartTime      *time.Time       `json:"start_time" yaml:"start_time"`
	WeatherControl *weather.Control `json:"weather_control,omitempty"`
}

// Patch allows modifying the struct in-place with values from a different instance
func (ws *WaterSchedule) Patch(new *WaterSchedule) {
	if new.Duration != nil {
		ws.Duration = new.Duration
	}
	if new.Interval != "" {
		ws.Interval = new.Interval
	}
	if new.StartTime != nil {
		ws.StartTime = new.StartTime
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
		ws.WeatherControl.SoilMoisture.MinimumMoisture > 0
}

// HasTemperatureControl is used to determine if configuration is available for environmental scaling
func (ws *WaterSchedule) HasTemperatureControl() bool {
	return ws.WeatherControl != nil &&
		ws.WeatherControl.Temperature != nil &&
		*ws.WeatherControl.Temperature.Factor != 0
}

// WaterHistory holds information about a WaterEvent that occurred in the past
type WaterHistory struct {
	Duration   string    `json:"duration"`
	RecordTime time.Time `json:"record_time"`
}
