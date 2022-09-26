package action

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GardenAction collects all the possible actions for a Garden into a single struct so these can easily be
// received as one request
type GardenAction struct {
	Light *LightAction `json:"light"`
	Stop  *StopAction  `json:"stop"`
}

// LightAction is an action for turning on or off a light for the Garden. The State field is optional and it will just toggle
// the current state if left empty.
type LightAction struct {
	State       pkg.LightState `json:"state"`
	ForDuration string         `json:"for_duration"`
}

// StopAction is an action for stopping watering of a Plant. It doesn't stop watering a specific Plant, only what is
// currently watering and optionally clearing the queue of Plants to water.
type StopAction struct {
	All bool `json:"all"`
}
