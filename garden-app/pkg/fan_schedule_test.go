package pkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFanSchedulePatch(t *testing.T) {
	power50 := uint(50)
	power75 := uint(75)
	duration := &Duration{Duration: 30 * time.Minute}
	interval := &Duration{Duration: 2 * time.Hour}

	tests := []struct {
		name        string
		newSchedule *FanSchedule
	}{
		{
			"PatchDuration",
			&FanSchedule{Duration: duration},
		},
		{
			"PatchInterval",
			&FanSchedule{Interval: interval},
		},
		{
			"PatchPower",
			&FanSchedule{Power: &power50},
		},
		{
			"PatchOnlyWithLight",
			&FanSchedule{OnlyWithLight: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FanSchedule{}
			fs.Patch(tt.newSchedule)
			assert.EqualValues(t, tt.newSchedule, fs)
		})
	}

	t.Run("PatchMultipleFields", func(t *testing.T) {
		fs := &FanSchedule{}
		fs.Patch(&FanSchedule{Duration: duration, Interval: interval, Power: &power50, OnlyWithLight: true})
		assert.Equal(t, duration, fs.Duration)
		assert.Equal(t, interval, fs.Interval)
		assert.Equal(t, &power50, fs.Power)
		assert.True(t, fs.OnlyWithLight)
	})

	t.Run("PatchDoesNotOverwriteUnspecifiedFields", func(t *testing.T) {
		fs := &FanSchedule{Duration: duration, Power: &power50, OnlyWithLight: true}
		fs.Patch(&FanSchedule{Power: &power75, OnlyWithLight: true})
		assert.Equal(t, duration, fs.Duration)
		assert.Equal(t, &power75, fs.Power)
		assert.True(t, fs.OnlyWithLight)
	})
}

func TestFanScheduleCycleDuration(t *testing.T) {
	tests := []struct {
		name     string
		schedule *FanSchedule
		expected time.Duration
	}{
		{
			"ValidCycleDuration",
			&FanSchedule{
				Duration: &Duration{Duration: 30 * time.Minute},
				Interval: &Duration{Duration: 2 * time.Hour},
			},
			2*time.Hour + 30*time.Minute,
		},
		{
			"NilDuration",
			&FanSchedule{Interval: &Duration{Duration: 2 * time.Hour}},
			0,
		},
		{
			"NilInterval",
			&FanSchedule{Duration: &Duration{Duration: 30 * time.Minute}},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.schedule.CycleDuration())
		})
	}
}

func TestFanSchedulePowerToPWM(t *testing.T) {
	tests := []struct {
		name     string
		power    uint
		expected uint8
	}{
		{"Zero", 0, 0},
		{"50Percent", 50, 127},
		{"100Percent", 100, 255},
		{"25Percent", 25, 63},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FanSchedule{Power: &tt.power}
			assert.Equal(t, tt.expected, fs.PowerToPWM())
		})
	}

	t.Run("NilPower", func(t *testing.T) {
		fs := &FanSchedule{}
		assert.Equal(t, uint8(0), fs.PowerToPWM())
	})
}

func TestFanScheduleNextChange(t *testing.T) {
	loc := time.UTC
	duration := &Duration{Duration: 30 * time.Minute}
	interval := &Duration{Duration: 2 * time.Hour}

	tests := []struct {
		name             string
		schedule         *FanSchedule
		now              time.Time
		expectedAfter    time.Duration
		expectedActive   bool
		expectedIsActive bool
	}{
		{
			"AtMidnightActive",
			&FanSchedule{Duration: duration, Interval: interval},
			time.Date(2024, 1, 1, 0, 0, 0, 0, loc),
			30 * time.Minute,
			false,
			true,
		},
		{
			"DuringActivePeriod",
			&FanSchedule{Duration: duration, Interval: interval},
			time.Date(2024, 1, 1, 0, 15, 0, 0, loc),
			15 * time.Minute,
			false,
			true,
		},
		{
			"DuringOffPeriod",
			&FanSchedule{Duration: duration, Interval: interval},
			time.Date(2024, 1, 1, 1, 0, 0, 0, loc),
			1*time.Hour + 30*time.Minute,
			true,
			false,
		},
		{
			"JustBeforeNextActive",
			&FanSchedule{Duration: duration, Interval: interval},
			time.Date(2024, 1, 1, 2, 29, 0, 0, loc),
			1 * time.Minute,
			true,
			false,
		},
		{
			"LateNightOff",
			&FanSchedule{Duration: duration, Interval: interval},
			time.Date(2024, 1, 1, 23, 0, 0, 0, loc),
			2 * time.Hour,
			true,
			false,
		},
		{
			"LateNightDuringOff",
			&FanSchedule{Duration: duration, Interval: interval},
			time.Date(2024, 1, 1, 23, 45, 0, 0, loc),
			1*time.Hour + 15*time.Minute,
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextChange, willBeActive := tt.schedule.NextChange(tt.now)
			assert.WithinDuration(t, tt.now.Add(tt.expectedAfter), nextChange, time.Second)
			assert.Equal(t, tt.expectedActive, willBeActive)
			assert.Equal(t, tt.expectedIsActive, tt.schedule.IsActiveAtTime(tt.now))
		})
	}
}
