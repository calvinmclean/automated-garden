package action

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// ZoneAction collects all the possible actions for a Zone into a single struct so these can easily be
// received as one request
type ZoneAction struct {
	Water *WaterAction `json:"water" form:"water"`
}

// String...
func (action *ZoneAction) String() string {
	return fmt.Sprintf("%+v", *action.Water)
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *ZoneAction) Bind(*http.Request) error {
	if action == nil || action.Water == nil {
		return errors.New("missing required action fields")
	}

	return nil
}

// WaterAction is an action for watering a Zone for the specified amount of time
type WaterAction struct {
	Duration      *pkg.Duration `json:"duration" form:"duration"`
	IgnoreWeather bool          `json:"ignore_weather"`

	// Source is only used internally and is not set by incoming commands
	Source Source `json:"-"`
}

// WaterMessage is the message being sent over MQTT to the embedded garden controller
type WaterMessage struct {
	Duration int64  `json:"duration"`
	ZoneID   string `json:"zone_id"`
	Position uint   `json:"position"`
	EventID  string `json:"id"`
	Source   Source `json:"source"`

	// Start is a boolean showing if this record is the start or finish event. It is excluded
	// from JSON because it is just used when parsing incoming messages that log the start/finish
	Start bool `json:"-"`
}

// String...
func (m *WaterMessage) String() string {
	return fmt.Sprintf("%+v", *m)
}

// Source shows how an action was triggered
type Source string

const (
	SourceSchedule     Source = "schedule"
	SourceCommand      Source = "command"
	SourceWaterRoutine Source = "water_routine"
)
