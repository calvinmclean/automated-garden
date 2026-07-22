package pkg

import (
	"fmt"
	"time"
)

// ControllerLog represents a single log entry published by a garden controller.
// It is read from InfluxDB where the measurement is "logs" and the topic is
// "<topic_prefix>/data/logs".
type ControllerLog struct {
	Time    time.Time         `json:"time" mapstructure:"_time"`
	Level   string            `json:"level" mapstructure:"level"`
	Source  string            `json:"source" mapstructure:"source"`
	Message string            `json:"message" mapstructure:"message"`
	Details map[string]string `json:"details,omitempty" mapstructure:"-"`
}

// LevelClass returns a CSS class name for the log level so the UI can color-code it.
func (c ControllerLog) LevelClass() string {
	switch c.Level {
	case "error":
		return "danger"
	case "warn":
		return "warning"
	case "debug":
		return "primary"
	default:
		return "success"
	}
}

// DetailsString returns a newline-separated "key=value" rendering of Details.
func (c ControllerLog) DetailsString() string {
	if len(c.Details) == 0 {
		return ""
	}

	var s string
	for k, v := range c.Details {
		if s != "" {
			s += "\n"
		}
		s += fmt.Sprintf("%s=%s", k, v)
	}
	return s
}
