package pkg

import (
	"fmt"
	"time"
)

// FanSchedule allows the user to control when a Garden's fan is turned on and off
type FanSchedule struct {
	Duration      *Duration `json:"duration" yaml:"duration"`
	Interval      *Duration `json:"interval" yaml:"interval"`
	Power         *uint     `json:"power" yaml:"power"`
	OnlyWithLight bool      `json:"only_with_light" yaml:"only_with_light"`
}

// String returns a string representation of the FanSchedule
func (fs *FanSchedule) String() string {
	return fmt.Sprintf("%+v", *fs)
}

// Patch allows modifying the struct in-place with values from a different instance
func (fs *FanSchedule) Patch(newFanSchedule *FanSchedule) {
	if newFanSchedule.Duration != nil {
		fs.Duration = newFanSchedule.Duration
	}
	if newFanSchedule.Interval != nil {
		fs.Interval = newFanSchedule.Interval
	}
	if newFanSchedule.Power != nil {
		fs.Power = newFanSchedule.Power
	}
	fs.OnlyWithLight = newFanSchedule.OnlyWithLight
}

// CycleDuration returns the total cycle duration (active time + interval)
func (fs FanSchedule) CycleDuration() time.Duration {
	if fs.Duration == nil || fs.Interval == nil {
		return 0
	}
	return fs.Duration.Duration + fs.Interval.Duration
}

// PowerToPWM converts the 0-100 power percentage to a 0-255 PWM value
func (fs FanSchedule) PowerToPWM() uint8 {
	if fs.Power == nil {
		return 0
	}
	// Power is validated to be 0-100, so this conversion is safe
	//nolint:gosec
	return uint8(*fs.Power * 255 / 100)
}

// IsActiveAtTime returns true if the fan should be running at the given time
func (fs FanSchedule) IsActiveAtTime(now time.Time) bool {
	_, willBeActive := fs.NextChange(now)
	// NextChange returns the state after the transition, so current state is the opposite
	return !willBeActive
}

// NextChange determines the next time the fan will change state and what the new state will be.
// It returns the next transition time and true if the fan will be ON after the transition.
func (fs FanSchedule) NextChange(now time.Time) (time.Time, bool) {
	cycleDuration := fs.CycleDuration()
	if cycleDuration == 0 || fs.Duration == nil {
		return time.Time{}, false
	}

	anchor := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// If anchor is in the future, the first cycle hasn't started yet
	if anchor.After(now) {
		return anchor, true
	}

	elapsed := now.Sub(anchor)
	cyclePos := elapsed % cycleDuration

	if cyclePos < fs.Duration.Duration {
		// Currently ON, next change is when duration ends
		return now.Add(fs.Duration.Duration - cyclePos), false
	}

	// Currently OFF, next change is start of next cycle
	return now.Add(cycleDuration - cyclePos), true
}
