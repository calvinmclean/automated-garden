// Package pkg provides domain models and utilities for the garden application
package pkg

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

const cronPrefix = "cron:"

// Duration is a wrapper around the time.Duration that allows it to be used as interger or string representation. It also
// supports inputting a cron string as an interval instead if using the prefix "cron:"
type Duration struct {
	time.Duration
	Cron string
}

// SchedulerFunc is a wrapper around gocron's fluent style to easily choose the cron or duration-based scheduling
func (d *Duration) SchedulerFunc(s *gocron.Scheduler) *gocron.Scheduler {
	if d.Cron != "" {
		return s.Cron(d.Cron)
	}
	return s.Every(d.Duration)
}

// MarshalJSON will convert Duration into the string representation
func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON with allow reading a Duration as a string or integer into time.Duration
func (d *Duration) UnmarshalJSON(data []byte) error {
	var value any
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	switch v := value.(type) {
	case string:
		d.Duration, d.Cron, err = parseString(v)
		if err != nil {
			return fmt.Errorf("invalid json input for Duration: %w", err)
		}
	case float64:
		d.Duration = time.Duration(v)
	default:
		return fmt.Errorf("unexpected type %T, must be string or number", v)
	}

	return nil
}

// UnmarshalText is used for HTML form decoding. It doesn't know if it's an integer or string representation
// so it will try both. It parses integer input as milliseconds
func (d *Duration) UnmarshalText(data []byte) error {
	v, err := strconv.Atoi(string(data))
	if err == nil {
		d.Duration = time.Duration(v) * time.Millisecond
		return nil
	}

	d.Duration, d.Cron, err = parseString(string(data))
	if err == nil {
		return nil
	}
	if d.Duration == 0 && d.Cron == "" {
		d = nil
		return nil
	}

	return fmt.Errorf("invalid text input for Duration: %w", err)
}

// UnmarshalYAML with allow reading a Duration as a string or integer into time.Duration
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case "!!str":
		var err error
		d.Duration, d.Cron, err = parseString(value.Value)
		if err != nil {
			return fmt.Errorf("invalid yaml input for Duration: %w", err)
		}
	case "!!int":
		v, err := strconv.Atoi(value.Value)
		if err != nil {
			return err
		}
		d.Duration = time.Duration(v)
	default:
		return fmt.Errorf("unexpected type %s, must be string or number", value.Tag)
	}

	return nil
}

// MarshalYAML will convert Duration into the string representation
func (d *Duration) MarshalYAML() (any, error) {
	if d.Cron != "" {
		return cronPrefix + d.Cron, nil
	}
	return d.String(), nil
}

func parseString(input string) (time.Duration, string, error) {
	if input == "" || input == `""` {
		return 0, "", nil
	}

	input = strings.Trim(input, `"`)

	if !strings.HasPrefix(input, cronPrefix) {
		d, err := parseDurationWithDays(input)
		if err != nil {
			return 0, "", fmt.Errorf("invalid format for time.Duration: %w", err)
		}
		return d, "", nil
	}

	cronStr := strings.TrimPrefix(input, cronPrefix)
	_, err := cron.ParseStandard(cronStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid cron expression: %w", err)
	}

	return 0, cronStr, nil
}

// parseDurationWithDays parses a duration string that may contain days (d) suffix.
// It converts days to hours and delegates the rest to time.ParseDuration.
// Examples: "5d" -> 120h, "5d1h" -> 121h, "2d30m" -> 48h30m, "1h30m" -> 1h30m
func parseDurationWithDays(input string) (time.Duration, error) {
	daysStr, remainder, found := strings.Cut(input, "d")
	if !found {
		return time.ParseDuration(input)
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		return 0, fmt.Errorf("invalid days value: %w", err)
	}

	totalHours := days * 24

	if remainder == "" {
		return time.Duration(totalHours) * time.Hour, nil
	}

	remainderDuration, err := time.ParseDuration(remainder)
	if err != nil {
		return 0, fmt.Errorf("invalid remainder duration: %w", err)
	}

	return time.Duration(totalHours)*time.Hour + remainderDuration, nil
}

// String returns a string representation of the Duration using FormatDurationShort.
// For cron expressions, it returns the cron prefix format.
func (d Duration) String() string {
	if d.Cron != "" {
		return cronPrefix + d.Cron
	}
	return FormatDurationShort(d.Duration)
}

// FormatDurationShort formats a duration in a short, readable format.
// It uses days (d) for durations of 24 hours or more, with optional hours, minutes, and seconds.
// Examples: "4d", "3d2h", "2h30m", "45m", "30s"
func FormatDurationShort(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	// Round to nearest second to avoid sub-second precision
	d = d.Round(time.Second)

	var result strings.Builder

	days := int(d.Hours()) / 24
	if days > 0 {
		fmt.Fprintf(&result, "%dd", days)
	}

	hours := int(d.Hours()) % 24
	if hours > 0 {
		fmt.Fprintf(&result, "%dh", hours)
	}

	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		fmt.Fprintf(&result, "%dm", minutes)
	}

	seconds := int(d.Seconds()) % 60
	if seconds > 0 {
		fmt.Fprintf(&result, "%ds", seconds)
	}

	return result.String()
}
