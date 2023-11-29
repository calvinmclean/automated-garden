package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name            string
		lastContactTime time.Time
		err             error
		expectedStatus  string
	}{
		{
			"GardenIsUp",
			time.Now(),
			nil,
			"UP",
		},
		{
			"GardenIsDown",
			time.Now().Add(-5 * time.Minute),
			nil,
			"DOWN",
		},
		{
			"InfluxDBError",
			time.Time{},
			errors.New("influxdb error"),
			"N/A",
		},
		{
			"ZeroTime",
			time.Time{},
			nil,
			"DOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "garden").Return(tt.lastContactTime, tt.err)

			g := Garden{TopicPrefix: "garden"}

			gardenHealth := g.Health(context.Background(), influxdbClient)
			if gardenHealth.Status != tt.expectedStatus {
				t.Errorf("Unexpected GardenHealth.Status: expected = %s, actual = %s", tt.expectedStatus, gardenHealth.Status)
			}
		})
	}
}

func TestGardenEndDated(t *testing.T) {
	pastDate := time.Now().Add(-1 * time.Minute)
	futureDate := time.Now().Add(time.Minute)
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
	now := time.Now()
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
				StartTime: "start time",
			}},
		},
		{
			"PatchLightSchedule.AdhocOnTime",
			&Garden{LightSchedule: &LightSchedule{
				AdhocOnTime: nil,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Garden{}

			err := g.Patch(tt.newGarden)
			require.NoError(t, err)

			if g.LightSchedule != nil && *g.LightSchedule != *tt.newGarden.LightSchedule {
				t.Errorf("Unexpected result for LightSchedule: expected=%v, actual=%v", tt.newGarden.LightSchedule, g.LightSchedule)
			}
			if g.Name != tt.newGarden.Name {
				t.Errorf("Unexpected result for Name: expected=%v, actual=%v", tt.newGarden.Name, g.Name)
			}
			if g.CreatedAt != tt.newGarden.CreatedAt {
				t.Errorf("Unexpected result for CreatedAt: expected=%v, actual=%v", tt.newGarden.CreatedAt, g.CreatedAt)
			}
		})
	}

	t.Run("RemoveLightSchedule", func(t *testing.T) {
		g := &Garden{
			LightSchedule: &LightSchedule{
				StartTime: "START TIME",
				Duration:  &Duration{2 * time.Hour, ""},
			},
		}
		err := g.Patch(&Garden{LightSchedule: &LightSchedule{}})
		require.NoError(t, err)

		if g.LightSchedule != nil {
			t.Errorf("Expected nil LightSchedule, but got: %v", g.LightSchedule)
		}
	})

	t.Run("PatchDoesNotAddEndDate", func(t *testing.T) {
		now := time.Now()
		g := &Garden{}

		err := g.Patch(&Garden{EndDate: &now})
		require.NoError(t, err)

		if g.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", g.EndDate)
		}
	})

	t.Run("PatchRemoveEndDate", func(t *testing.T) {
		now := time.Now()
		g := &Garden{
			EndDate: &now,
		}

		err := g.Patch(&Garden{})
		require.NoError(t, err)

		if g.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", g.EndDate)
		}
	})
}

func TestLightStateString(t *testing.T) {
	tests := []struct {
		name     string
		input    LightState
		expected string
	}{
		{
			"ON",
			LightStateOn,
			"ON",
		},
		{
			"OFF",
			LightStateOff,
			"OFF",
		},
		{
			"Toggle",
			LightStateToggle,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.String() != tt.expected {
				t.Errorf("Expected %v, but got: %v", tt.expected, tt.input)
			}
		})
	}
}

func TestLightStateUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LightState
	}{
		{
			"ON",
			`"ON"`,
			LightStateOn,
		},
		{
			"on",
			`"on"`,
			LightStateOn,
		},
		{
			"OFF",
			`"OFF"`,
			LightStateOff,
		},
		{
			"off",
			`"off"`,
			LightStateOff,
		},
		{
			"Toggle",
			`""`,
			LightStateToggle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l LightState
			err := json.Unmarshal([]byte(tt.input), &l)
			if err != nil {
				t.Errorf("Unexpected error when unmarshaling JSON: %v", err)
			}
			if l != tt.expected {
				t.Errorf("Expected %v, but got: %v", tt.expected, l.String())
			}
		})
	}

	t.Run("InvalidInput", func(t *testing.T) {
		var l LightState
		err := json.Unmarshal([]byte(`"invalid"`), &l)
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != `cannot unmarshal "invalid" into Go value of type *pkg.LightState` {
			t.Errorf("Unexpected error string: %v", err)
		}
	})
}

func TestLightStateMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    LightState
		expected string
	}{
		{
			"ON",
			LightStateOn,
			`"ON"`,
		},
		{
			"OFF",
			LightStateOff,
			`"OFF"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			if err != nil {
				t.Errorf("Unexpected error when marshaling JSON: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Expected %v, but got: %s", tt.expected, string(result))
			}
		})
	}

	t.Run("InvalidLightState", func(t *testing.T) {
		result, err := json.Marshal(LightState(3))
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != `json: error calling MarshalJSON for type pkg.LightState: cannot convert 3 to pkg.LightState` {
			t.Errorf("Unexpected error string: %v", err)
		}
		if string(result) != "" {
			t.Errorf("Expected empty string but got: %v", string(result))
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
