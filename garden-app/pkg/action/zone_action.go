package action

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// ZoneAction collects all the possible actions for a Zone into a single struct so these can easily be
// received as one request
type ZoneAction struct {
	Water *WaterAction `json:"water"`
}

// String...
func (action *ZoneAction) String() string {
	return fmt.Sprintf("%+v", *action.Water)
}

// WaterAction is an action for watering a Zone for the specified amount of time
type WaterAction struct {
	Duration       *pkg.Duration `json:"duration"`
	IgnoreMoisture bool          `json:"ignore_moisture"`
	IgnoreWeather  bool          `json:"ignore_weather"`
}

// WaterMessage is the message being sent over MQTT to the embedded garden controller
type WaterMessage struct {
	Duration int64  `json:"duration"`
	ZoneID   string `json:"id"`
	Position uint   `json:"position"`
}

// String...
func (m *WaterMessage) String() string {
	return fmt.Sprintf("%+v", *m)
}
