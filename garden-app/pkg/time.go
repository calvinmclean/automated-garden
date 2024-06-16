package pkg

import (
	"fmt"
	"strconv"
	"time"
)

const (
	// StartTimeFormat is used to control format of time fields
	StartTimeFormat = "15:04:05Z07:00"
)

func parseStartTime(startTime string) (time.Time, error) {
	result, err := time.Parse(StartTimeFormat, startTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing start time: %w", err)
	}

	return result.UTC(), nil
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
