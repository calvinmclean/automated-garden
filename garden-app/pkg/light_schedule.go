package pkg

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// LightStateOff is the value used to turn off a light
	LightStateOff LightState = iota
	// LightStateOn is the value used to turn on a light
	LightStateOn
	// LightStateToggle is the empty value that results in toggling
	LightStateToggle
)

var (
	stateToString = []string{"OFF", "ON", ""}
	stringToState = map[string]LightState{
		`"OFF"`: LightStateOff,
		`OFF`:   LightStateOff,
		`"ON"`:  LightStateOn,
		`ON`:    LightStateOn,
		`""`:    LightStateToggle,
		``:      LightStateToggle,
	}
)

// LightState is an enum representing the state of a Light (ON or OFF)
type LightState int

// Return the string representation of this LightState
func (l LightState) String() string {
	return stateToString[l]
}

// MarshalJSON will convert LightState into it's JSON string representation
func (l LightState) MarshalJSON() ([]byte, error) {
	if int(l) >= len(stateToString) {
		return []byte{}, fmt.Errorf("cannot convert %d to %T", int(l), l)
	}
	return json.Marshal(stateToString[l])
}

// UnmarshalJSON with convert LightState's JSON string representation, ignoring case, into a LightState
func (l *LightState) UnmarshalJSON(data []byte) error {
	return l.unmarshal(data)
}

func (l *LightState) UnmarshalText(data []byte) error {
	return l.unmarshal(data)
}

func (l *LightState) unmarshal(data []byte) error {
	upper := strings.ToUpper(string(data))
	var ok bool
	*l, ok = stringToState[upper]
	if !ok {
		return fmt.Errorf("cannot unmarshal %s into Go value of type %T", string(data), l)
	}
	return nil
}

// LightSchedule allows the user to control when the Garden light is turned on and off
// "Time" should be in the format of LightTimeFormat constant ("15:04:05-07:00")
type LightSchedule struct {
	Duration    *Duration  `json:"duration" yaml:"duration"`
	StartTime   *StartTime `json:"start_time" yaml:"start_time"`
	AdhocOnTime *time.Time `json:"adhoc_on_time,omitempty" yaml:"adhoc_on_time,omitempty"`
}

// String...
func (ls *LightSchedule) String() string {
	return fmt.Sprintf("%+v", *ls)
}

// Patch allows modifying the struct in-place with values from a different instance
func (ls *LightSchedule) Patch(newLightSchedule *LightSchedule) {
	if newLightSchedule.Duration != nil {
		ls.Duration = newLightSchedule.Duration
	}
	if newLightSchedule.StartTime != nil {
		ls.StartTime = newLightSchedule.StartTime
	}
	if newLightSchedule.AdhocOnTime == nil {
		ls.AdhocOnTime = nil
	}
}
