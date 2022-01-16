package pkg

import (
	"time"

	"github.com/rs/xid"
)

// Plant is the representation of the most important resource for this application, a Plant.
// This includes some general information like name and unique ID, a start and end date to show when
// the Plant was in the system, plus some information for watering like the duration to water for, how
// often to water, and the PlantPosition field will tell the microcontroller which plant to water.
// Some integers in this struct are pointers because it allows differentiating 0-value from empty.
type Plant struct {
	Name      string        `json:"name" yaml:"name,omitempty"`
	Details   *PlantDetails `json:"details,omitempty" yaml:"details,omitempty"`
	ID        xid.ID        `json:"id" yaml:"id,omitempty"`
	ZoneID    xid.ID        `json:"zone_id" yaml:"zone_id"`
	CreatedAt *time.Time    `json:"created_at" yaml:"created_at,omitempty"`
	EndDate   *time.Time    `json:"end_date,omitempty" yaml:"end_date,omitempty"`
}

// PlantDetails is a struct holding some additional details about a Plant that are primarily for user convenience
// and are generally not used by the application
type PlantDetails struct {
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	Notes         string `json:"notes,omitempty" yaml:"notes,omitempty"`
	TimeToHarvest string `json:"time_to_harvest,omitempty" yaml:"time_to_harvest,omitempty"`
	Count         int    `json:"count,omitempty" yaml:"count,omitempty"`
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
	if newPlant.ZoneID != xid.NilID() {
		p.ZoneID = newPlant.ZoneID
	}
	if newPlant.CreatedAt != nil {
		p.CreatedAt = newPlant.CreatedAt
	}
	if p.EndDate != nil && newPlant.EndDate == nil {
		p.EndDate = newPlant.EndDate
	}

	if newPlant.Details != nil {
		// Initiate Details if it is nil
		if p.Details == nil {
			p.Details = &PlantDetails{}
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
