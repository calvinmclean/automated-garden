package pkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFanSchedulePatch(t *testing.T) {
	power50 := uint(50)
	power75 := uint(75)
	activeTime := &Duration{Duration: 30 * time.Minute}
	offTime := &Duration{Duration: 2 * time.Hour}
	startTime := NewStartTime(time.Date(0, 1, 1, 8, 0, 0, 0, time.UTC))

	tests := []struct {
		name        string
		newSchedule *FanSchedule
	}{
		{
			"PatchActiveTime",
			&FanSchedule{ActiveTime: activeTime},
		},
		{
			"PatchOffTime",
			&FanSchedule{OffTime: offTime},
		},
		{
			"PatchPower",
			&FanSchedule{Power: &power50},
		},
		{
			"PatchOnlyWithLight",
			&FanSchedule{OnlyWithLight: true},
		},
		{
			"PatchStartTime",
			&FanSchedule{StartTime: startTime},
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
		fs.Patch(&FanSchedule{ActiveTime: activeTime, OffTime: offTime, Power: &power50, OnlyWithLight: true})
		assert.Equal(t, activeTime, fs.ActiveTime)
		assert.Equal(t, offTime, fs.OffTime)
		assert.Equal(t, &power50, fs.Power)
		assert.True(t, fs.OnlyWithLight)
	})

	t.Run("PatchDoesNotOverwriteUnspecifiedFields", func(t *testing.T) {
		fs := &FanSchedule{ActiveTime: activeTime, Power: &power50, OnlyWithLight: true}
		fs.Patch(&FanSchedule{Power: &power75, OnlyWithLight: true})
		assert.Equal(t, activeTime, fs.ActiveTime)
		assert.Equal(t, &power75, fs.Power)
		assert.True(t, fs.OnlyWithLight)
	})
}

func TestFanScheduleInterval(t *testing.T) {
	tests := []struct {
		name     string
		schedule *FanSchedule
		expected time.Duration
	}{
		{
			"ValidInterval",
			&FanSchedule{
				ActiveTime: &Duration{Duration: 30 * time.Minute},
				OffTime:    &Duration{Duration: 2 * time.Hour},
			},
			2*time.Hour + 30*time.Minute,
		},
		{
			"NilActiveTime",
			&FanSchedule{OffTime: &Duration{Duration: 2 * time.Hour}},
			0,
		},
		{
			"NilOffTime",
			&FanSchedule{ActiveTime: &Duration{Duration: 30 * time.Minute}},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.schedule.Interval())
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
	activeTime := &Duration{Duration: 30 * time.Minute}
	offTime := &Duration{Duration: 2 * time.Hour}

	startTime8AM := NewStartTime(time.Date(0, 1, 1, 8, 0, 0, 0, loc))

	tests := []struct {
		name             string
		schedule         *FanSchedule
		now              time.Time
		expectedAfter    time.Duration
		expectedActive   bool
		expectedIsActive bool
	}{
		{
			"BeforeStartTime",
			&FanSchedule{ActiveTime: activeTime, OffTime: offTime, StartTime: startTime8AM},
			time.Date(2024, 1, 1, 7, 0, 0, 0, loc),
			1 * time.Hour,
			true,
			false,
		},
		{
			"DuringActivePeriod",
			&FanSchedule{ActiveTime: activeTime, OffTime: offTime, StartTime: startTime8AM},
			time.Date(2024, 1, 1, 8, 15, 0, 0, loc),
			15 * time.Minute,
			false,
			true,
		},
		{
			"DuringOffPeriod",
			&FanSchedule{ActiveTime: activeTime, OffTime: offTime, StartTime: startTime8AM},
			time.Date(2024, 1, 1, 9, 0, 0, 0, loc),
			1*time.Hour + 30*time.Minute,
			true,
			false,
		},
		{
			"NextCycleActive",
			&FanSchedule{ActiveTime: activeTime, OffTime: offTime, StartTime: startTime8AM},
			time.Date(2024, 1, 1, 10, 45, 0, 0, loc),
			15 * time.Minute,
			false,
			true,
		},
		{
			"NoStartTimeDuringActive",
			&FanSchedule{ActiveTime: activeTime, OffTime: offTime},
			time.Date(2024, 1, 1, 0, 15, 0, 0, loc),
			15 * time.Minute,
			false,
			true,
		},
		{
			"NoStartTimeDuringOff",
			&FanSchedule{ActiveTime: activeTime, OffTime: offTime},
			time.Date(2024, 1, 1, 1, 0, 0, 0, loc),
			1*time.Hour + 30*time.Minute,
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
