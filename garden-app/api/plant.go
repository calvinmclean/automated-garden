package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/rs/xid"
)

// Plant is the representation of the most important resource for this application, a Plant.
// This includes some general information like name and unique ID, a start and end date to show when
// the Plant was in the system, plus some information for watering like the duration to water for, how
// often to water, and the PlantPosition field will tell the microcontroller which plant to water
type Plant struct {
	Name           string     `json:"name" yaml:"name,omitempty"`
	ID             xid.ID     `json:"id" yaml:"id,omitempty"`
	WateringAmount int        `json:"watering_amount" yaml:"watering_amount,omitempty"`
	PlantPosition  int        `json:"plant_position" yaml:"plant_position"`
	Interval       string     `json:"interval" yaml:"interval,omitempty"`
	StartDate      *time.Time `json:"start_date" yaml:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty" yaml:"end_date,omitempty"`
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (p *Plant) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (p *Plant) Bind(r *http.Request) error {
	if p == nil {
		return errors.New("missing required action fields")
	}

	return nil
}
