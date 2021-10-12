package pkg

import (
	"context"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/rs/xid"
)

// Garden is the representation of a single garden-controller device. It is the container for Plants
type Garden struct {
	Name      string            `json:"name" yaml:"name,omitempty"`
	ID        xid.ID            `json:"id" yaml:"id,omitempty"`
	Plants    map[xid.ID]*Plant `json:"plants" yaml:"plants,omitempty"`
	CreatedAt *time.Time        `json:"created_at" yaml:"created_at,omitempty"`
	EndDate   *time.Time        `json:"end_date,omitempty" yaml:"end_date,omitempty"`
}

// GardenHealth holds information about the Garden controller's health status
type GardenHealth struct {
	Status      string     `json:"status,omitempty"`
	Details     string     `json:"details,omitempty"`
	LastContact *time.Time `json:"last_contact,omitempty"`
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

	// Garden is considered "UP" if it's last contact was after 5 minutes ago
	up := lastContact.After(time.Now().Add(-5 * time.Minute))

	status := "UP"
	if !up {
		status = "DOWN"
	}

	return GardenHealth{
		Status:      status,
		LastContact: &lastContact,
	}
}
