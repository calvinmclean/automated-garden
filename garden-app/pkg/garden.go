package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/babyapi"
)

const (
	// LightTimeFormat is used to control format of time fields
	LightTimeFormat = "15:04:05-07:00"
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
		`"ON"`:  LightStateOn,
		`""`:    LightStateToggle,
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
	upper := strings.ToUpper(string(data))
	var ok bool
	*l, ok = stringToState[upper]
	if !ok {
		return fmt.Errorf("cannot unmarshal %s into Go value of type %T", string(data), l)
	}
	return nil
}

// Garden is the representation of a single garden-controller device
type Garden struct {
	Name                      string         `json:"name" yaml:"name,omitempty"`
	TopicPrefix               string         `json:"topic_prefix,omitempty" yaml:"topic_prefix,omitempty"`
	ID                        babyapi.ID     `json:"id" yaml:"id,omitempty"`
	MaxZones                  *uint          `json:"max_zones" yaml:"max_zones"`
	CreatedAt                 *time.Time     `json:"created_at" yaml:"created_at,omitempty"`
	EndDate                   *time.Time     `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	LightSchedule             *LightSchedule `json:"light_schedule,omitempty" yaml:"light_schedule,omitempty"`
	TemperatureHumiditySensor *bool          `json:"temperature_humidity_sensor,omitempty" yaml:"temperature_humidity_sensor,omitempty"`
}

func (g *Garden) GetID() string {
	return g.ID.String()
}

// String...
func (g *Garden) String() string {
	return fmt.Sprintf("%+v", *g)
}

// GardenHealth holds information about the Garden controller's health status
type GardenHealth struct {
	Status      string     `json:"status,omitempty"`
	Details     string     `json:"details,omitempty"`
	LastContact *time.Time `json:"last_contact,omitempty"`
}

// LightSchedule allows the user to control when the Garden light is turned on and off
// "Time" should be in the format of LightTimeFormat constant ("15:04:05-07:00")
type LightSchedule struct {
	Duration    *Duration  `json:"duration" yaml:"duration"`
	StartTime   string     `json:"start_time" yaml:"start_time"`
	AdhocOnTime *time.Time `json:"adhoc_on_time,omitempty" yaml:"adhoc_on_time,omitempty"`
}

// String...
func (ls *LightSchedule) String() string {
	return fmt.Sprintf("%+v", *ls)
}

// Patch allows modifying the struct in-place with values from a different instance
func (ls *LightSchedule) Patch(new *LightSchedule) {
	if new.Duration != nil {
		ls.Duration = new.Duration
	}
	if new.StartTime != "" {
		ls.StartTime = new.StartTime
	}
	if new.AdhocOnTime == nil {
		ls.AdhocOnTime = nil
	}
}

// Health returns a GardenHealth struct after querying InfluxDB for the Garden controller's last contact time
func (g *Garden) Health(ctx context.Context, influxdbClient influxdb.Client) *GardenHealth {
	lastContact, err := influxdbClient.GetLastContact(ctx, g.TopicPrefix)
	if err != nil {
		return &GardenHealth{
			Status:  "N/A",
			Details: err.Error(),
		}
	}

	if lastContact.IsZero() {
		return &GardenHealth{
			Status:  "DOWN",
			Details: "no last contact time available",
		}
	}

	// Garden is considered "UP" if it's last contact was less than 5 minutes ago
	between := time.Since(lastContact)
	up := between < 5*time.Minute

	status := "UP"
	if !up {
		status = "DOWN"
	}

	return &GardenHealth{
		Status:      status,
		LastContact: &lastContact,
		Details:     fmt.Sprintf("last contact from Garden was %v ago", between.Truncate(time.Millisecond)),
	}
}

// EndDated returns true if the Garden is end-dated
func (g *Garden) EndDated() bool {
	return g.EndDate != nil && g.EndDate.Before(time.Now())
}

func (g *Garden) SetEndDate(now time.Time) {
	g.EndDate = &now
}

// Patch allows for easily updating individual fields of a Garden by passing in a new Garden containing
// the desired values
func (g *Garden) Patch(newGarden *Garden) *babyapi.ErrResponse {
	if newGarden.Name != "" {
		g.Name = newGarden.Name
	}
	if newGarden.TopicPrefix != "" {
		g.TopicPrefix = newGarden.TopicPrefix
	}
	if newGarden.MaxZones != nil {
		g.MaxZones = newGarden.MaxZones
	}
	if newGarden.CreatedAt != nil {
		g.CreatedAt = newGarden.CreatedAt
	}
	if g.EndDate != nil && newGarden.EndDate == nil {
		g.EndDate = newGarden.EndDate
	}
	if newGarden.LightSchedule != nil {
		// If existing garden doesn't have a LightSchedule, it needs to be initialized first
		if g.LightSchedule == nil {
			g.LightSchedule = &LightSchedule{}
		}
		g.LightSchedule.Patch(newGarden.LightSchedule)

		// If both Duration and StartTime are empty, remove the schedule
		if newGarden.LightSchedule.Duration == nil && newGarden.LightSchedule.StartTime == "" {
			g.LightSchedule = nil
		}
	}
	if newGarden.TemperatureHumiditySensor != nil {
		g.TemperatureHumiditySensor = newGarden.TemperatureHumiditySensor
	}

	return nil
}

// HasTemperatureHumiditySensor determines if the Garden has a sensor configured
func (g *Garden) HasTemperatureHumiditySensor() bool {
	return g.TemperatureHumiditySensor != nil && *g.TemperatureHumiditySensor
}

func (g *Garden) Bind(r *http.Request) error {
	if g == nil {
		return errors.New("missing required Garden fields")
	}

	err := g.ID.Bind(r)
	if err != nil {
		return err
	}

	switch r.Method {
	case http.MethodPost:
		now := time.Now()
		g.CreatedAt = &now
		fallthrough
	case http.MethodPut:
		if g.Name == "" {
			return errors.New("missing required name field")
		}
		if g.TopicPrefix == "" {
			return errors.New("missing required topic_prefix field")
		}
		illegalRegexp := regexp.MustCompile(`[\$\#\*\>\+\/]`)
		if illegalRegexp.MatchString(g.TopicPrefix) {
			return errors.New("one or more invalid characters in Garden topic_prefix")
		}
		if g.MaxZones == nil {
			return errors.New("missing required max_zones field")
		} else if *g.MaxZones == 0 {
			return errors.New("max_zones must not be 0")
		}
		if g.LightSchedule != nil {
			if g.LightSchedule.Duration == nil {
				return errors.New("missing required light_schedule.duration field")
			}

			// Check that Duration is valid Duration
			if g.LightSchedule.Duration.Duration >= 24*time.Hour {
				return fmt.Errorf("invalid light_schedule.duration >= 24 hours: %s", g.LightSchedule.Duration)
			}

			if g.LightSchedule.StartTime == "" {
				return errors.New("missing required light_schedule.start_time field")
			}
			// Check that LightSchedule.StartTime is valid
			_, err := time.Parse(LightTimeFormat, g.LightSchedule.StartTime)
			if err != nil {
				return fmt.Errorf("invalid time format for light_schedule.start_time: %s", g.LightSchedule.StartTime)
			}
		}
	case http.MethodPatch:
		illegalRegexp := regexp.MustCompile(`[\$\#\*\>\+\/]`)
		if illegalRegexp.MatchString(g.TopicPrefix) {
			return errors.New("one or more invalid characters in Garden topic_prefix")
		}
		if g.EndDate != nil {
			return errors.New("to end-date a Garden, please use the DELETE endpoint")
		}
		if g.MaxZones != nil && *g.MaxZones == 0 {
			return errors.New("max_zones must not be 0")
		}

		if g.LightSchedule != nil {
			// Check that Duration is valid Duration
			if g.LightSchedule.Duration != nil {
				if g.LightSchedule.Duration.Duration >= 24*time.Hour {
					return fmt.Errorf("invalid light_schedule.duration >= 24 hours: %s", g.LightSchedule.Duration)
				}
			}
			// Check that LightSchedule.StartTime is valid
			if g.LightSchedule.StartTime != "" {
				_, err := time.Parse(LightTimeFormat, g.LightSchedule.StartTime)
				if err != nil {
					return fmt.Errorf("invalid time format for light_schedule.start_time: %s", g.LightSchedule.StartTime)
				}
			}
		}
	}

	return nil
}

func (g *Garden) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
