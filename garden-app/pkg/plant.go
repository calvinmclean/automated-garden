package pkg

import (
	"time"

	"github.com/rs/xid"
)

// Plant is the representation of the most important resource for this application, a Plant.
// This includes some general information like name and unique ID, a start and end date to show when
// the Plant was in the system.
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

// Patch allows modifying the struct in-place with values from a different instance
func (pd *PlantDetails) Patch(new *PlantDetails) {
	if new.Description != "" {
		pd.Description = new.Description
	}
	if new.Notes != "" {
		pd.Notes = new.Notes
	}
	if new.TimeToHarvest != "" {
		pd.TimeToHarvest = new.TimeToHarvest
	}
	if new.Count != 0 {
		pd.Count = new.Count
	}
}

func (p *Plant) GetID() string {
	return p.ID.String()
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
		p.Details.Patch(newPlant.Details)
	}
}
