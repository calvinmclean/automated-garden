package pkg

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGardenEndDated(t *testing.T) {
	pastDate := clock.Now().Add(-1 * time.Minute)
	futureDate := clock.Now().Add(time.Minute)
	tests := []struct {
		name     string
		endDate  *time.Time
		expected bool
	}{
		{"NilEndDateFalse", nil, false},
		{"EndDateFutureEndDateFalse", &futureDate, false},
		{"EndDatePastEndDateTrue", &pastDate, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Garden{EndDate: tt.endDate}
			if g.EndDated() != tt.expected {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.expected, g.EndDated())
			}
		})
	}
}

func TestGardenPatch(t *testing.T) {
	now := clock.Now()
	ten := uint(10)
	trueBool := true
	falseBool := false

	tests := []struct {
		name      string
		newGarden *Garden
	}{
		{
			"PatchName",
			&Garden{Name: "name"},
		},
		{
			"PatchTopicPrefix",
			&Garden{TopicPrefix: "topic"},
		},
		{
			"PatchMaxZones",
			&Garden{MaxZones: &ten},
		},
		{
			"PatchCreatedAt",
			&Garden{CreatedAt: &now},
		},
		{
			"PatchLightSchedule.Duration",
			&Garden{LightSchedule: &LightSchedule{
				Duration: &Duration{2 * time.Hour, ""},
			}},
		},
		{
			"PatchLightSchedule.StartTime",
			&Garden{LightSchedule: &LightSchedule{
				StartTime: NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.FixedZone("", 0))),
			}},
		},
		{
			"PatchTemperatureHumiditySensorTrue",
			&Garden{TemperatureHumiditySensor: &trueBool},
		},
		{
			"PatchTemperatureHumiditySensorFalse",
			&Garden{TemperatureHumiditySensor: &falseBool},
		},
		{
			"ControllerConfig",
			&Garden{ControllerConfig: &ControllerConfig{
				ValvePins:              []uint{1, 2},
				PumpPins:               []uint{3, 4},
				LightPin:               pointer(uint(1)),
				TemperatureHumidityPin: pointer(uint(1)),
			}},
		},
		{
			"PatchNotificationSettings",
			&Garden{NotificationSettings: &NotificationSettings{
				ControllerStartup: true,
				LightSchedule:     true,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Garden{}

			err := g.Patch(tt.newGarden)
			require.Nil(t, err)

			if g.LightSchedule != nil && *g.LightSchedule != *tt.newGarden.LightSchedule {
				t.Errorf("Unexpected result for LightSchedule: expected=%v, actual=%v", tt.newGarden.LightSchedule, g.LightSchedule)
			}
			if g.Name != tt.newGarden.Name {
				t.Errorf("Unexpected result for Name: expected=%v, actual=%v", tt.newGarden.Name, g.Name)
			}
			if g.CreatedAt != tt.newGarden.CreatedAt {
				t.Errorf("Unexpected result for CreatedAt: expected=%v, actual=%v", tt.newGarden.CreatedAt, g.CreatedAt)
			}
			assert.EqualValues(t, tt.newGarden.ControllerConfig, g.ControllerConfig)
		})
	}

	t.Run("RemoveLightSchedule", func(t *testing.T) {
		g := &Garden{
			LightSchedule: &LightSchedule{
				StartTime: NewStartTime(time.Date(0, 1, 1, 15, 4, 0, 0, time.FixedZone("", 0))),
				Duration:  &Duration{2 * time.Hour, ""},
			},
		}
		err := g.Patch(&Garden{LightSchedule: &LightSchedule{}})
		require.Nil(t, err)

		if g.LightSchedule != nil {
			t.Errorf("Expected nil LightSchedule, but got: %v", g.LightSchedule)
		}
	})

	t.Run("PatchDoesNotAddEndDate", func(t *testing.T) {
		now := clock.Now()
		g := &Garden{}

		err := g.Patch(&Garden{EndDate: &now})
		require.Nil(t, err)

		if g.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", g.EndDate)
		}
	})

	t.Run("PatchRemoveEndDate", func(t *testing.T) {
		now := clock.Now()
		g := &Garden{
			EndDate: &now,
		}

		err := g.Patch(&Garden{})
		require.Nil(t, err)

		if g.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", g.EndDate)
		}
	})
}

func TestHasTemperatureHumiditySensor(t *testing.T) {
	trueBool := true
	falseBool := false
	tests := []struct {
		val      *bool
		expected bool
	}{
		{nil, false},
		{&trueBool, true},
		{&falseBool, false},
	}

	for _, tt := range tests {
		g := &Garden{TemperatureHumiditySensor: tt.val}
		assert.Equal(t, tt.expected, g.HasTemperatureHumiditySensor())
	}
}
