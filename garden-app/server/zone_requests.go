package server

import (
	"errors"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/rs/xid"
)

// ZoneRequest wraps a Zone into a request so we can handle Bind/Render in this package
type ZoneRequest struct {
	*pkg.Zone
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (z *ZoneRequest) Bind(_ *http.Request) error {
	if z == nil || z.Zone == nil {
		return errors.New("missing required Zone fields")
	}

	if z.Position == nil {
		return errors.New("missing required position field")
	}
	if len(z.WaterScheduleIDs) == 0 {
		return errors.New("missing required water_schedule_ids field")
	}
	if z.Name == "" {
		return errors.New("missing required name field")
	}

	return nil
}

// UpdateZoneRequest wraps a Zone into a request so we can handle Bind/Render in this package
// It has different validation than the ZoneRequest
type UpdateZoneRequest struct {
	*pkg.Zone
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (z *UpdateZoneRequest) Bind(_ *http.Request) error {
	if z == nil || z.Zone == nil {
		return errors.New("missing required Zone fields")
	}

	if z.ID != xid.NilID() {
		return errors.New("updating ID is not allowed")
	}
	if z.EndDate != nil {
		return errors.New("to end-date a Zone, please use the DELETE endpoint")
	}

	return nil
}

// ZoneActionRequest wraps a ZoneAction into a request so we can handle Bind/Render in this package
type ZoneActionRequest struct {
	*action.ZoneAction
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *ZoneActionRequest) Bind(_ *http.Request) error {
	// ZoneAction is nil if no ZoneAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || action.ZoneAction == nil || (action.Water == nil) {
		return errors.New("missing required action fields")
	}
	return nil
}
