package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GetGardenHealth returns a GardenHealth struct after querying InfluxDB for the Garden controller's last contact time
func (w *Worker) GetGardenHealth(ctx context.Context, g *pkg.Garden) *pkg.GardenHealth {
	if w.influxdbClient == nil {
		return nil
	}

	lastContact, err := w.influxdbClient.GetLastContact(ctx, g.TopicPrefix)
	if err != nil {
		return &pkg.GardenHealth{
			Status:  "N/A",
			Details: err.Error(),
		}
	}

	if lastContact.IsZero() {
		return &pkg.GardenHealth{
			Status:  pkg.HealthStatusDown,
			Details: "no last contact time available",
		}
	}

	// Garden is considered "UP" if it's last contact was less than 5 minutes ago
	between := time.Since(lastContact)
	up := between < 5*time.Minute

	status := pkg.HealthStatusUp
	if !up {
		status = pkg.HealthStatusDown
	}

	return &pkg.GardenHealth{
		Status:      status,
		LastContact: &lastContact,
		Details:     fmt.Sprintf("last contact from Garden was %v ago", between.Truncate(time.Millisecond)),
	}
}
