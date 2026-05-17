package pkg

import (
	"fmt"
	"time"
)

// FanSchedule allows the user to control when a Garden's fan is turned on and off
type FanSchedule struct {
	ActiveTime *Duration `json:"active_time" yaml:"active_time"`
	// TODO: rename to Interval
	OffTime       *Duration  `json:"off_time" yaml:"off_time"`
	Power         *uint      `json:"power" yaml:"power"`
	OnlyWithLight bool       `json:"only_with_light" yaml:"only_with_light"`
	StartTime     *StartTime `json:"start_time,omitempty" yaml:"start_time,omitempty"`
}

// String returns a string representation of the FanSchedule
func (fs *FanSchedule) String() string {
	return fmt.Sprintf("%+v", *fs)
}

// Patch allows modifying the struct in-place with values from a different instance
func (fs *FanSchedule) Patch(newFanSchedule *FanSchedule) {
	if newFanSchedule.ActiveTime != nil {
		fs.ActiveTime = newFanSchedule.ActiveTime
	}
	if newFanSchedule.OffTime != nil {
		fs.OffTime = newFanSchedule.OffTime
	}
	if newFanSchedule.Power != nil {
		fs.Power = newFanSchedule.Power
	}
	fs.OnlyWithLight = newFanSchedule.OnlyWithLight
	if newFanSchedule.StartTime != nil {
		fs.StartTime = newFanSchedule.StartTime
	}
}

// Interval returns the total interval between fan cycle starts
func (fs FanSchedule) Interval() time.Duration {
	if fs.ActiveTime == nil || fs.OffTime == nil {
		return 0
	}
	return fs.ActiveTime.Duration + fs.OffTime.Duration
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
