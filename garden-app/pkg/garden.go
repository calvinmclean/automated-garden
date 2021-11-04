package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/rs/xid"
)

const (
	// LightTimeFormat is used to control format of time fields
	LightTimeFormat = "15:04:05-07:00"
)

// Garden is the representation of a single garden-controller device. It is the container for Plants
type Garden struct {
	Name          string            `json:"name" yaml:"name,omitempty"`
	ID            xid.ID            `json:"id" yaml:"id,omitempty"`
	Plants        map[xid.ID]*Plant `json:"plants" yaml:"plants,omitempty"`
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
	Duration  string `json:"duration" yaml:"duration"`
	StartTime string `json:"start_time" yaml:"start_time"`
}

// Health returns a GardenHealth struct after querying InfluxDB for the Garden controller's last contact time
func (g *Garden) Health(influxdbClient influxdb.Client) GardenHealth {
	ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
	defer cancel()

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
