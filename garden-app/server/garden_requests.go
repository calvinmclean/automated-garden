package server

import (
	"errors"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
)

// GardenActionRequest wraps a GardenAction into a request so we can handle Bind/Render in this package
type GardenActionRequest struct {
	*action.GardenAction

	// These are flattened fields so the API can be compatible with HTMX's hx-post. The Bind method will convert
	// these flattened values to fit the base GardenAction type
	LightState *pkg.LightState `json:"light_state"`
	StopAll    *bool           `json:"stop_all"`
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid and convert HTMX flattened values
func (gar *GardenActionRequest) Bind(_ *http.Request) error {
	if gar == nil {
		return errors.New("missing required action fields")
	}
	if gar.GardenAction == nil {
		gar.GardenAction = &action.GardenAction{}
	}
	if gar.LightState != nil {
		gar.Light = &action.LightAction{State: *gar.LightState}
	}
	if gar.StopAll != nil {
		gar.Stop = &action.StopAction{All: *gar.StopAll}
	}

	if gar.Light == nil && gar.Stop == nil {
		return errors.New("missing required action fields")
	}
	if gar.Light != nil && gar.Light.ForDuration != nil {
		if gar.Light.ForDuration.Duration < 0 {
			return errors.New("delay duration must be greater than 0")
		}
	}
	return nil
}
