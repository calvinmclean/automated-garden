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
// often to water, and the PlantPosition field will tell the microcontroller which plant to water
type Plant struct {
	Name             string           `json:"name" yaml:"name,omitempty"`
	Details          *Details         `json:"details,omitempty" yaml:"details,omitempty"`
	ID               xid.ID           `json:"id" yaml:"id,omitempty"`
	GardenID         xid.ID           `json:"garden_id" yaml:"garden_id,omitempty"`
	PlantPosition    int              `json:"plant_position" yaml:"plant_position"`
	CreatedAt        *time.Time       `json:"created_at" yaml:"created_at,omitempty"`
	EndDate          *time.Time       `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	SkipCount        int              `json:"skip_count,omitempty" yaml:"skip_count,omitempty"`
	WateringStrategy WateringStrategy `json:"watering_strategy,omitempty" yaml:"watering_strategy,omitempty"`
}

// Details is a struct holding some additional details about a Plant that are primarily for user convenience
// and are generally not used by the application
type Details struct {
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	Notes         string `json:"notes,omitempty" yaml:"notes,omitempty"`
	TimeToHarvest string `json:"time_to_harvest,omitempty" yaml:"time_to_harvest,omitempty"`
	Count         int    `json:"count,omitempty" yaml:"count,omitempty"`
}

// WateringStrategy allows the user to have more control over how the Plant is watered using an Interval
// and optional MinimumMoisture which acts as the threshold the Plant's soil should be above
// "Time" should be in the format of WaterTimeFormat constant ("15:04:05-07:00")
type WateringStrategy struct {
	WateringAmount  int    `json:"watering_amount" yaml:"watering_amount"`
	Interval        string `json:"interval" yaml:"interval"`
	MinimumMoisture int    `json:"minimum_moisture,omitempty" yaml:"minimum_moisture,omitempty"`
	StartTime       string `json:"start_time" yaml:"start_time"`
}

// WateringHistory holds information about a WateringEvent that occurred in the past
type WateringHistory struct {
	WateringAmount int       `json:"watering_amount"`
	RecordTime     time.Time `json:"record_time"`
}

// WateringAction creates the default/basic WateringAction for this Plant
func (p *Plant) WateringAction() *WaterAction {
	return &WaterAction{Duration: p.WateringStrategy.WateringAmount}
}
