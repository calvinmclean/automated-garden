package pkg

import (
	"time"

	"github.com/rs/xid"
)

const (
	// WaterTimeFormat is used to control format of time fields
	WaterTimeFormat = "15:04:05-07:00"
)

// Plant is the representation of the most important resource for this application, a Plant.
// This includes some general information like name and unique ID, a start and end date to show when
// the Plant was in the system, plus some information for watering like the duration to water for, how
// often to water, and the PlantPosition field will tell the microcontroller which plant to water.
// Some integers in this struct are pointers because it allows differentiating 0-value from empty.
type Plant struct {
	Name          string         `json:"name" yaml:"name,omitempty"`
	Details       *Details       `json:"details,omitempty" yaml:"details,omitempty"`
	ID            xid.ID         `json:"id" yaml:"id,omitempty"`
	GardenID      xid.ID         `json:"garden_id" yaml:"garden_id,omitempty"`
	PlantPosition *uint          `json:"plant_position" yaml:"plant_position"`
	CreatedAt     *time.Time     `json:"created_at" yaml:"created_at,omitempty"`
	EndDate       *time.Time     `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	WaterSchedule *WaterSchedule `json:"water_schedule,omitempty" yaml:"water_schedule,omitempty"`
}

// Details is a struct holding some additional details about a Plant that are primarily for user convenience
// and are generally not used by the application
type Details struct {
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	Notes         string `json:"notes,omitempty" yaml:"notes,omitempty"`
	TimeToHarvest string `json:"time_to_harvest,omitempty" yaml:"time_to_harvest,omitempty"`
	Count         int    `json:"count,omitempty" yaml:"count,omitempty"`
}

// WaterSchedule allows the user to have more control over how the Plant is watered using an Interval
// and optional MinimumMoisture which acts as the threshold the Plant's soil should be above.
// StartTime specifies when the watering interval should originate from. It can be used to increase/decrease delays in watering.
type WaterSchedule struct {
	Duration        string     `json:"duration" yaml:"duration"`
	Interval        string     `json:"interval" yaml:"interval"`
	MinimumMoisture int        `json:"minimum_moisture,omitempty" yaml:"minimum_moisture,omitempty"`
	StartTime       *time.Time `json:"start_time" yaml:"start_time"`
}

// WateringHistory holds information about a WateringEvent that occurred in the past
type WateringHistory struct {
	Duration   string    `json:"duration"`
	RecordTime time.Time `json:"record_time"`
}

// WateringAction creates the default/basic WateringAction for this Plant
func (p *Plant) WateringAction() *WaterAction {
	wateringDuration, _ := time.ParseDuration(p.WaterSchedule.Duration)
	return &WaterAction{Duration: wateringDuration.Milliseconds()}
}

// EndDated returns true if the Plant is end-dated
func (p *Plant) EndDated() bool {
	return p.EndDate != nil && p.EndDate.Before(time.Now())
}

// Patch allows for easily updating individual fields of a Plant by passing in a new Plant containing
// the desired values
func (p *Plant) Patch(newPlant *Plant) {
	if newPlant.Name != "" {
		p.Name = newPlant.Name
	}
	if newPlant.PlantPosition != nil {
		p.PlantPosition = newPlant.PlantPosition
	}
	if newPlant.CreatedAt != nil {
		p.CreatedAt = newPlant.CreatedAt
	}
	if p.EndDate != nil && newPlant.EndDate == nil {
		p.EndDate = newPlant.EndDate
	}

	if newPlant.WaterSchedule != nil {
		// Initiate WaterSchedule if it is nil
		if p.WaterSchedule == nil {
			p.WaterSchedule = &WaterSchedule{}
		}
		if newPlant.WaterSchedule.Duration != "" {
			p.WaterSchedule.Duration = newPlant.WaterSchedule.Duration
		}
		if newPlant.WaterSchedule.Interval != "" {
			p.WaterSchedule.Interval = newPlant.WaterSchedule.Interval
		}
		if newPlant.WaterSchedule.MinimumMoisture != 0 {
			p.WaterSchedule.MinimumMoisture = newPlant.WaterSchedule.MinimumMoisture
		}
		if newPlant.WaterSchedule.StartTime != nil {
			p.WaterSchedule.StartTime = newPlant.WaterSchedule.StartTime
		}
	}

	if newPlant.Details != nil {
		// Initiate Details if it is nil
		if p.Details == nil {
			p.Details = &Details{}
		}
		if newPlant.Details.Description != "" {
			p.Details.Description = newPlant.Details.Description
		}
		if newPlant.Details.Notes != "" {
			p.Details.Notes = newPlant.Details.Notes
		}
		if newPlant.Details.TimeToHarvest != "" {
			p.Details.TimeToHarvest = newPlant.Details.TimeToHarvest
		}
		if newPlant.Details.Count != 0 {
			p.Details.Count = newPlant.Details.Count
		}
	}
}
