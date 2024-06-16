package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ajg/form"
)

const (
	// startTimeFormat is used to control format of time fields
	startTimeFormat = "15:04:05Z07:00"
)

// StartTime allows for special handling of Time without the date and also allows several
// formats for decoding so it is more easily compatible with HTML forms
type StartTime struct {
	time.Time
}

type startTimeSplit struct {
	Hour   int
	Minute int
	TZ     string
}

func (st *startTimeSplit) String() string {
	return fmt.Sprintf("%02d:%02d:00%s", st.Hour, st.Minute, st.TZ)
}

func StartTimeFromString(startTime string) (*StartTime, error) {
	result, err := time.Parse(startTimeFormat, startTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing start time: %w", err)
	}

	return &StartTime{result}, nil
}

func (st *StartTime) String() string {
	return st.Format(startTimeFormat)
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
		var splitTime startTimeSplit
		err := json.Unmarshal(data, &splitTime)
		if err != nil {
			return err
		}

		timeString = splitTime.String()
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

func (st *StartTime) UnmarshalText(data []byte) error {
	var timeString string

	var splitTime startTimeSplit
	err := form.NewDecoder(bytes.NewBuffer(data)).Decode(&splitTime)
	if err != nil {
		timeString = string(data)
	} else {
		timeString = splitTime.String()
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
