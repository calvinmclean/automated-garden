package pkg

import (
	"errors"
	"testing"
	"time"

	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pointer[T any](v T) *T {
	return &v
}

func TestControllerConfigPatch(t *testing.T) {
	tests := []struct {
		name      string
		newConfig *ControllerConfig
	}{
		{
			"LightPin",
			&ControllerConfig{LightPin: pointer(uint(1))},
		},
		{
			"TemperatureHumidityPin",
			&ControllerConfig{TemperatureHumidityPin: pointer(uint(1))},
		},
		{
			"TemperatureHumidityInterval",
			&ControllerConfig{TemperatureHumidityInterval: &Duration{Duration: 1 * time.Second}},
		},
		{
			"ValvePinsEmpty",
			&ControllerConfig{ValvePins: []uint{}},
		},
		{
			"PumpPinsEmpty",
			&ControllerConfig{PumpPins: []uint{}},
		},
		{
			"ValvePinsPumpPinsNotEmpty",
			&ControllerConfig{ValvePins: []uint{1, 2, 3}, PumpPins: []uint{1, 2, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ControllerConfig{}

			err := c.Patch(tt.newConfig)
			require.Nil(t, err)

			assert.EqualValues(t, tt.newConfig, c)
		})
	}

	t.Run("RemoveValvePinsErrorNotEqual", func(t *testing.T) {
		c := &ControllerConfig{
			ValvePins: []uint{1, 2, 3},
		}

		err := c.Patch(&ControllerConfig{ValvePins: []uint{5}})
		require.Error(t, err)

		var babyapiErr *babyapi.ErrResponse
		errors.As(err, &babyapiErr)
		require.Equal(t, "pump_pins and valve_pins must be the same length", babyapiErr.Err.Error())

		assert.ElementsMatch(t, []uint{5}, c.ValvePins)
	})

	t.Run("RemovePumpPinsValvePins", func(t *testing.T) {
		c := &ControllerConfig{
			ValvePins: []uint{1, 2, 3},
			PumpPins:  []uint{1, 2, 3},
		}

		err := c.Patch(&ControllerConfig{PumpPins: []uint{5}, ValvePins: []uint{5}})
		require.Nil(t, err)

		assert.ElementsMatch(t, []uint{5}, c.PumpPins)
	})
}

func TestToMessage(t *testing.T) {
	tests := []struct {
		name     string
		config   *ControllerConfig
		expected ControllerConfigMessage
	}{
		{
			"FullConfig",
			&ControllerConfig{
				ValvePins:                   []uint{1},
				PumpPins:                    []uint{1},
				LightPin:                    pointer(uint(1)),
				TemperatureHumidityPin:      pointer(uint(1)),
				TemperatureHumidityInterval: &Duration{Duration: time.Second},
			},
			ControllerConfigMessage{
				NumZones:                    1,
				ValvePins:                   []uint{1},
				PumpPins:                    []uint{1},
				LightEnabled:                true,
				LightPin:                    uint(1),
				TemperatureHumidityEnabled:  true,
				TemperatureHumidityPin:      uint(1),
				TemperatureHumidityInterval: 1000,
			},
		},
		{
			"DefaultTemperatureHumidityInterval",
			&ControllerConfig{
				ValvePins:              []uint{1},
				PumpPins:               []uint{1},
				LightPin:               pointer(uint(1)),
				TemperatureHumidityPin: pointer(uint(1)),
			},
			ControllerConfigMessage{
				NumZones:                    1,
				ValvePins:                   []uint{1},
				PumpPins:                    []uint{1},
				LightEnabled:                true,
				LightPin:                    uint(1),
				TemperatureHumidityEnabled:  true,
				TemperatureHumidityPin:      uint(1),
				TemperatureHumidityInterval: 5000,
			},
		},
		{
			"NoLightPin",
			&ControllerConfig{
				ValvePins:                   []uint{1},
				PumpPins:                    []uint{1},
				TemperatureHumidityPin:      pointer(uint(1)),
				TemperatureHumidityInterval: &Duration{Duration: time.Second},
			},
			ControllerConfigMessage{
				NumZones:                    1,
				ValvePins:                   []uint{1},
				PumpPins:                    []uint{1},
				LightEnabled:                false,
				TemperatureHumidityEnabled:  true,
				TemperatureHumidityPin:      uint(1),
				TemperatureHumidityInterval: 1000,
			},
		},
		{
			"NoTemperatureHumidityPin",
			&ControllerConfig{
				ValvePins: []uint{1},
				PumpPins:  []uint{1},
			},
			ControllerConfigMessage{
				NumZones:                   1,
				ValvePins:                  []uint{1},
				PumpPins:                   []uint{1},
				LightEnabled:               false,
				TemperatureHumidityEnabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.config.ToMessage()
			require.Equal(t, tt.expected, out)
		})
	}
}
