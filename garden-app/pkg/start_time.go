package pkg

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const (
	// startTimeFormat is used to control format of time fields
	startTimeFormat = "15:04:05Z07:00"
)

// StartTime allows for special handling of Time without the date and also allows several
// formats for decoding so it is more easily compatible with HTML forms
type StartTime struct {
	Time time.Time `form:"-"`

	Hour   int
	Minute int
	TZ     string
}

func StartTimeFromString(startTime string) (*StartTime, error) {
	result, err := time.Parse(startTimeFormat, startTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing start time: %w", err)
	}

	return &StartTime{Time: result}, nil
}

func NewStartTime(t time.Time) *StartTime {
	return &StartTime{Time: t}
}

func (st *StartTime) String() string {
	return st.Time.Format(startTimeFormat)
}

// Validate is used after parsing from HTML form so the time can be parsed
func (st *StartTime) Validate() error {
	if !st.Time.IsZero() {
		return nil
	}

	result, err := time.Parse(startTimeFormat, fmt.Sprintf("%02d:%02d:00%s", st.Hour, st.Minute, st.TZ))
	if err != nil {
		return err
	}
	st.Time = result

	return nil
}

func (st *StartTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, st.String())), nil
}

func (st *StartTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	var timeString string
	switch v := value.(type) {
	case string:
		timeString = v
	case map[string]any:
		var splitTime struct {
			Hour   int
			Minute int
			TZ     string
		}
		err := json.Unmarshal(data, &splitTime)
		if err != nil {
			return err
		}

		timeString = fmt.Sprintf("%02d:%02d:00%s", splitTime.Hour, splitTime.Minute, splitTime.TZ)
	default:
		return fmt.Errorf("unexpected type %T, must be string or object", v)
	}

	startTime, err := StartTimeFromString(timeString)
	if err != nil {
		return err
	}
	st.Time = startTime.Time

	return nil
}

// TimeLocationFromOffset uses an offset minutes from JS `new Date().getTimezoneOffset()` and parses it into
// Go's time.Location. JS offsets are positive if they are behind UTC
func TimeLocationFromOffset(offsetMinutes string) (*time.Location, error) {
	offset, err := strconv.Atoi(offsetMinutes)
	if err != nil {
		return nil, err
	}

	offsetSeconds := offset * -60
	return time.FixedZone("UserLocation", offsetSeconds), nil
}
