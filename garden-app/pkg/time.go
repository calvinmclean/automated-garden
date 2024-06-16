package pkg

import (
	"fmt"
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

	return result, nil
}
