package action

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GardenAction collects all the possible actions for a Garden into a single struct so these can easily be
// received as one request
type GardenAction struct {
	Light *LightAction `json:"light"`
	Stop  *StopAction  `json:"stop"`
}

// String...
func (action *GardenAction) String() string {
	return fmt.Sprintf("{LightAction: %+v, StopAction: %+v}", action.Light, action.Stop)
}

// LightAction is an action for turning on or off a light for the Garden. The State field is optional and it will just toggle
// the current state if left empty.
type LightAction struct {
	State       pkg.LightState `json:"state"`
	ForDuration *pkg.Duration  `json:"for_duration"`
}

// StopAction is an action for stopping watering of a Zone. It doesn't stop watering a specific Zone, only what is
// currently watering and optionally clearing the queue of Zones to water.
type StopAction struct {
	All bool `json:"all"`
}
