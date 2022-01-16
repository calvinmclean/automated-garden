package server

import (
	"errors"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// PlantRequest wraps a Plant into a request so we can handle Bind/Render in this package
type PlantRequest struct {
	*pkg.Plant
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (p *PlantRequest) Bind(r *http.Request) error {
	if p == nil || p.Plant == nil {
		return errors.New("missing required Plant fields")
	}

	if p.Name == "" {
		return errors.New("missing required name field")
	}
	if p.ZoneID == xid.NilID() {
		return errors.New("missing required zone_id field")
	}

	return nil
}

// UpdatePlantRequest wraps a Plant into a request so we can handle Bind/Render in this package
// It has different validation than the PlantRequest
type UpdatePlantRequest struct {
	*pkg.Plant
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (p *UpdatePlantRequest) Bind(r *http.Request) error {
	if p == nil || p.Plant == nil {
		return errors.New("missing required Plant fields")
	}

	if p.ID != xid.NilID() {
		return errors.New("updating ID is not allowed")
	}
	if p.EndDate != nil {
		return errors.New("to end-date a Plant, please use the DELETE endpoint")
	}

	return nil
}
