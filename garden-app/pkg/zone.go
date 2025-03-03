package pkg

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
)

const currentZoneVersion = uint(1)

// Zone represents a "waterable resource" that is owned by a Garden..
// This allows for more complex Garden setups where a large irrigation system will be watering entire groups of
// Zones rather than watering individually. This contains the important information for managing WaterSchedules
// and some additional details describing the Zone. The Position is an integer that tells the controller which
// part of hardware needs to be switched on to start watering
type Zone struct {
	Name             string       `json:"name" yaml:"name,omitempty"`
	Details          *ZoneDetails `json:"details,omitempty" yaml:"details,omitempty"`
	ID               babyapi.ID   `json:"id" yaml:"id,omitempty"`
	GardenID         xid.ID       `json:"garden_id" yaml:"garden_id,omitempty"`
	Position         *uint        `json:"position" yaml:"position"`
	CreatedAt        *time.Time   `json:"created_at" yaml:"created_at,omitempty"`
	EndDate          *time.Time   `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	WaterScheduleIDs []xid.ID     `json:"water_schedule_ids" yaml:"water_schedule_ids"`
	SkipCount        *uint        `json:"skip_count" yaml:"skip_count"`
	Version          uint         `json:"version,omitempty" yaml:"version"`
}

func (z *Zone) GetVersion() uint {
	return z.Version
}

func (z *Zone) SetVersion(v uint) {
	z.Version = v
}

func (z *Zone) GetID() string {
	return z.ID.String()
}

// String...
func (z *Zone) String() string {
	return fmt.Sprintf("%+v", *z)
}

// EndDated returns true if the Zone is end-dated
func (z *Zone) EndDated() bool {
	return z.EndDate != nil && z.EndDate.Before(clock.Now())
}

func (z *Zone) SetEndDate(now time.Time) {
	z.EndDate = &now
}

// Patch allows for easily updating individual fields of a Zone by passing in a new Zone containing
// the desired values
func (z *Zone) Patch(newZone *Zone) *babyapi.ErrResponse {
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
	if newZone.SkipCount != nil {
		z.SkipCount = newZone.SkipCount
	}

	if len(newZone.WaterScheduleIDs) != 0 {
		z.WaterScheduleIDs = newZone.WaterScheduleIDs
	}

	if newZone.Details != nil {
		// Initiate Details if it is nil
		if z.Details == nil {
			z.Details = &ZoneDetails{}
		}
		z.Details.Patch(newZone.Details)
	}

	return nil
}

// ZoneDetails is a struct holding some additional details about a Zone that are primarily for user convenience
// and are generally not used by the application
type ZoneDetails struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Notes       string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// String...
func (zd *ZoneDetails) String() string {
	return fmt.Sprintf("%+v", *zd)
}

// Patch allows modifying the struct in-place with values from a different instance
func (zd *ZoneDetails) Patch(newZoneDetails *ZoneDetails) {
	if newZoneDetails.Description != "" {
		zd.Description = newZoneDetails.Description
	}
	if newZoneDetails.Notes != "" {
		zd.Notes = newZoneDetails.Notes
	}
}

// WaterHistory holds information about a WaterEvent that occurred in the past
type WaterHistory struct {
	Duration   string    `json:"duration"`
	RecordTime time.Time `json:"record_time"`
	EventID    string    `json:"event_id"`
}

// ZoneAndGarden allows grouping the Zone and Garden it belongs too and is useful in some cases
// where both are needed in a return value
type ZoneAndGarden struct {
	*Zone
	*Garden
}

func (z *Zone) Bind(r *http.Request) error {
	if z == nil {
		return errors.New("missing required Zone fields")
	}

	err := z.ID.Bind(r)
	if err != nil {
		return err
	}

	// Remove any zero-valued WaterScheduleIDs. This can happen because HTML form input requires specifying
	// an index, so any skipped check boxes will result in zero-valued xids
	wsIDs := []xid.ID{}
	for _, wsID := range z.WaterScheduleIDs {
		if !wsID.IsZero() {
			wsIDs = append(wsIDs, wsID)
		}
	}
	z.WaterScheduleIDs = wsIDs

	now := clock.Now()
	switch r.Method {
	case http.MethodPost:
		z.CreatedAt = &now
		fallthrough
	case http.MethodPut:
		if z.Version == 0 {
			z.Version = currentZoneVersion
		}
		if z.CreatedAt == nil || z.CreatedAt.IsZero() {
			z.CreatedAt = &now
		}
		if z.Position == nil {
			return errors.New("missing required position field")
		}
		if z.Name == "" {
			return errors.New("missing required name field")
		}
	case http.MethodPatch:
		if z.EndDate != nil {
			return errors.New("to end-date a Zone, please use the DELETE endpoint")
		}
		if !z.GardenID.IsNil() {
			return errors.New("unable to change GardenID")
		}
	}

	return nil
}

func (z *Zone) Render(_ http.ResponseWriter, _ *http.Request) error {
	// Version is excluded from responses because it's not important external information
	z.Version = 0
	return nil
}
