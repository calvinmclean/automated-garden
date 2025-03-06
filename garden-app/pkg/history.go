package pkg

import "time"

type WaterStatus string

const (
	WaterStatusSent      WaterStatus = "sent"
	WaterStatusStarted   WaterStatus = "start"
	WaterStatusCompleted WaterStatus = "complete"
)

// WaterHistory holds information about a WaterEvent that occurred in the past
type WaterHistory struct {
	Duration    Duration    `json:"duration" mapstructure:"duration"`
	EventID     string      `json:"event_id" mapstructure:"event_id"`
	Status      WaterStatus `json:"status" mapstructure:"status"`
	SentAt      time.Time   `json:"sent_at" mapstructure:"sent_at"`
	StartedAt   time.Time   `json:"started_at,omitzero" mapstructure:"started_at"`
	CompletedAt time.Time   `json:"completed_at,omitzero" mapstructure:"completed_at"`
}
