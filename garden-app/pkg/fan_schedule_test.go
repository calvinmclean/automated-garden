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
