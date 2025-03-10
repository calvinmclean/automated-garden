package pkg

import (
	"errors"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
)

type WaterStatus string

const (
	WaterStatusSent      WaterStatus = "sent"
	WaterStatusStarted   WaterStatus = "start"
	WaterStatusCompleted WaterStatus = "complete"
)

var (
	// ErrElapsedExceedsDuration occurs when watering Started but is still not reported as Complete after
	// the duration has been exceeded. This likely means that the controller was interrupted during watering
	ErrElapsedExceedsDuration = errors.New("elapsed time exceeds expected watering duration")

	// ErrSentButNotStarted occurs if the latest watering is Sent, but not Started and the previous watering
	// is Completed or Started with ErrElapsedExceedsDuration
	ErrSentButNotStarted = errors.New("watering was sent but never started")
)

// WaterHistory holds information about a WaterEvent that occurred in the past
type WaterHistory struct {
	Duration    Duration    `json:"duration" mapstructure:"duration"`
	EventID     string      `json:"event_id" mapstructure:"event_id"`
	Status      WaterStatus `json:"status" mapstructure:"status"`
	Source      string      `json:"source" mapstructure:"source"`
	SentAt      time.Time   `json:"sent_at" mapstructure:"sent_at"`
	StartedAt   time.Time   `json:"started_at,omitzero" mapstructure:"started_at"`
	CompletedAt time.Time   `json:"completed_at,omitzero" mapstructure:"completed_at"`
}

// WaterHistoryProgress is used to show watering progress or errors in the UI
type WaterHistoryProgress struct {
	Duration Duration
	Elapsed  Duration
	Progress float32
	Queue    uint
	Error    error
}

func (p WaterHistoryProgress) Percent() string {
	return fmt.Sprintf("%.0f%%", p.Progress*100)
}

// OneSecondProgress is the amount that Progress will increase every second.
// This is used for incrementing a dynamic progress bar in the UI
func (p WaterHistoryProgress) OneSecondProgress() float32 {
	return (1 / float32(p.Duration.Duration.Seconds()))
}

// CalculateWaterProgress parses the WaterHistory to create WaterHistoryProgress based on the recent entries.
// If the most recent event is Started, this calculates how far along it is.
// If the most recent event is Completed, it will show for 1 hour and then is considered irrelevant.
// If most recent events are Sent, this counts the "Queue" until the most recent Started or Completed event.
//
// There are a few scenarios that will be presented as an error:
//   - ErrElapsedExceedsDuration: Status is Started, but the elapsed time since then exceeds the specified duration.
//     This means the controller was likely interrupted before completing
//   - ErrSentButNotStarted: Status is Sent and the previous event was Completed > 1s ago, meaning the controller
//     is likely offline or was interrupted before starting watering
func CalculateWaterProgress(history []WaterHistory) WaterHistoryProgress {
	if len(history) == 0 {
		return WaterHistoryProgress{}
	}

	var prev WaterHistory
	var queue uint

	for _, event := range history {
		switch event.Status {
		case WaterStatusStarted:
			elapsed := clock.Since(event.StartedAt)

			progress := WaterHistoryProgress{
				Duration: event.Duration,
				Elapsed:  Duration{Duration: elapsed},
				Progress: float32(elapsed) / float32(event.Duration.Duration),
				Queue:    queue,
			}

			if elapsed > event.Duration.Duration {
				progress.Error = ErrElapsedExceedsDuration
				progress.Progress = 0
			}

			return progress
		case WaterStatusCompleted:
			elapsed := clock.Since(event.CompletedAt)

			// WaterEvents completed over an hour ago are not relevant
			if elapsed > time.Hour {
				return WaterHistoryProgress{}
			}

			// If this is the first event, then nothing is in-progress
			if queue == 0 {
				return WaterHistoryProgress{}
			}

			// If an event was Sent after this one, and this was completed > 1s ago, we have a problem
			if prev.Status == WaterStatusSent && elapsed >= time.Second {
				return WaterHistoryProgress{
					Error: ErrSentButNotStarted,
					Queue: queue,
				}
			}

			return WaterHistoryProgress{
				Duration: event.Duration,
				Elapsed:  Duration{Duration: elapsed},
				Progress: 1.0,
				Queue:    queue,
			}
		case WaterStatusSent:
			prev = event
			queue++
		}
	}

	return WaterHistoryProgress{
		Duration: Duration{},
		Elapsed:  Duration{},
		Progress: 0,
		Queue:    queue,
	}
}
