package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/rs/xid"
)

const (
	// LightTimeFormat is used to control format of time fields
	LightTimeFormat = "15:04:05-07:00"
)

const (
	// LightStateOff is the value used to turn off a light
	LightStateOff LightState = iota
	// LightStateOn is the value used to turn on a light
	LightStateOn
)

var (
	stateToString = []string{"OFF", "ON"}
	stringToState = map[string]LightState{
		`"OFF"`: LightStateOff,
		`"ON"`:  LightStateOn,
	}
)

// LightState is an enum representing the state of a Light (ON or OFF)
type LightState int

// Return the string representation of this LightState
func (l LightState) String() string {
	return stateToString[l]
}

// MarshalJSON will convert LightState into it's JSON string representation
func (l LightState) MarshalJSON() ([]byte, error) {
	if int(l) >= len(stateToString) {
		return []byte{}, fmt.Errorf("cannot convert %d to %T", int(l), l)
	}
	return json.Marshal(stateToString[l])
}

// UnmarshalJSON with convert LightState's JSON string representation, ignoring case, into a LightState
func (l *LightState) UnmarshalJSON(data []byte) error {
	upper := strings.ToUpper(string(data))
	var ok bool
	*l, ok = stringToState[upper]
	if !ok {
		return fmt.Errorf("cannot unmarshal %s into Go value of type %T", string(data), l)
	}
	return nil
}

// Garden is the representation of a single garden-controller device. It is the container for Plants
type Garden struct {
	Name          string            `json:"name" yaml:"name,omitempty"`
	ID            xid.ID            `json:"id" yaml:"id,omitempty"`
	Plants        map[xid.ID]*Plant `json:"plants" yaml:"plants,omitempty"`
	MaxPlants     *uint             `json:"max_plants" yaml:"max_plants"`
	CreatedAt     *time.Time        `json:"created_at" yaml:"created_at,omitempty"`
	EndDate       *time.Time        `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	LightSchedule *LightSchedule    `json:"light_schedule,omitempty" yaml:"light_schedule,omitempty"`
}

// GardenHealth holds information about the Garden controller's health status
type GardenHealth struct {
	Status      string     `json:"status,omitempty"`
	Details     string     `json:"details,omitempty"`
	LastContact *time.Time `json:"last_contact,omitempty"`
}

// LightSchedule allows the user to control when the Garden light is turned on and off
// "Time" should be in the format of LightTimeFormat constant ("15:04:05-07:00")
type LightSchedule struct {
	Duration    string     `json:"duration" yaml:"duration"`
	StartTime   string     `json:"start_time" yaml:"start_time"`
	AdhocOnTime *time.Time `json:"adhoc_on_time,omitempty" yaml:"adhoc_on_time,omitempty"`
}

// Health returns a GardenHealth struct after querying InfluxDB for the Garden controller's last contact time
func (g *Garden) Health(ctx context.Context, influxdbClient influxdb.Client) GardenHealth {
	lastContact, err := influxdbClient.GetLastContact(ctx, g.Name)
	if err != nil {
		return GardenHealth{
			Status:  "N/A",
			Details: err.Error(),
		}
	}

	if lastContact.IsZero() {
		return GardenHealth{
			Status:  "DOWN",
			Details: "no last contact time available",
		}
	}

	// Garden is considered "UP" if it's last contact was less than 5 minutes ago
	between := time.Since(lastContact)
	up := between < 5*time.Minute

	status := "UP"
	if !up {
		status = "DOWN"
	}

	return GardenHealth{
		Status:      status,
		LastContact: &lastContact,
		Details:     fmt.Sprintf("last contact from Garden was %v ago", between),
	}
}

// EndDated returns true if the Garden is end-dated
func (g *Garden) EndDated() bool {
	return g.EndDate != nil && g.EndDate.Before(time.Now())
}

// Patch allows for easily updating individual fields of a Garden by passing in a new Garden containing
// the desired values
func (g *Garden) Patch(newGarden *Garden) {
	if newGarden.Name != "" {
		g.Name = newGarden.Name
	}
	if newGarden.MaxPlants != nil {
		g.MaxPlants = newGarden.MaxPlants
	}
	if newGarden.CreatedAt != nil {
		g.CreatedAt = newGarden.CreatedAt
	}
	if g.EndDate != nil && newGarden.EndDate == nil {
		g.EndDate = newGarden.EndDate
	}
	if newGarden.LightSchedule != nil {
		// If existing garden doesn't have a LightSchedule, it needs to be initialized first
		if g.LightSchedule == nil {
			g.LightSchedule = &LightSchedule{}
		}
		if newGarden.LightSchedule.Duration != "" {
			g.LightSchedule.Duration = newGarden.LightSchedule.Duration
		}
		if newGarden.LightSchedule.StartTime != "" {
			g.LightSchedule.StartTime = newGarden.LightSchedule.StartTime
		}
		if newGarden.LightSchedule.AdhocOnTime == nil {
			g.LightSchedule.AdhocOnTime = newGarden.LightSchedule.AdhocOnTime
		}
		// If both Duration and StartTime are empty, remove the schedule
		if newGarden.LightSchedule.Duration == "" && newGarden.LightSchedule.StartTime == "" {
			g.LightSchedule = nil
		}
	}
}

// NumPlants returns the number of non-end-dated Plants that are part of this Garden
func (g *Garden) NumPlants() uint {
	result := uint(0)
	for _, p := range g.Plants {
		if !p.EndDated() {
			result++
		}
	}
	return result
}
