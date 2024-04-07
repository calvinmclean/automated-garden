package action

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GardenAction collects all the possible actions for a Garden into a single struct so these can easily be
// received as one request
type GardenAction struct {
	Light *LightAction `json:"light" form:"light"`
	Stop  *StopAction  `json:"stop" form:"stop"`
}

// String...
func (action *GardenAction) String() string {
	return fmt.Sprintf("{LightAction: %+v, StopAction: %+v}", action.Light, action.Stop)
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *GardenAction) Bind(_ *http.Request) error {
	if action == nil || (action.Light == nil && action.Stop == nil) {
		return errors.New("missing required action fields")
	}

	if action.Light != nil && action.Light.ForDuration != nil {
		if action.Light.ForDuration.Duration < 0 {
			return errors.New("delay duration must be greater than 0")
		}
	}
	return nil
}

// LightAction is an action for turning on or off a light for the Garden. The State field is optional and it will just toggle
// the current state if left empty.
type LightAction struct {
	State       pkg.LightState `json:"state" form:"state"`
	ForDuration *pkg.Duration  `json:"for_duration" form:"for_duration"`
}

// StopAction is an action for stopping watering of a Zone. It doesn't stop watering a specific Zone, only what is
// currently watering and optionally clearing the queue of Zones to water.
type StopAction struct {
	All bool `json:"all" form:"all"`
}
