package pkg

import (
	"testing"

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
			"ValvePinsEmpty",
			&ControllerConfig{ValvePins: []uint{}},
		},
		{
			"ValvePinsNotEmpty",
			&ControllerConfig{ValvePins: []uint{1, 2, 3}},
		},
		{
			"PumpPinsEmpty",
			&ControllerConfig{PumpPins: []uint{}},
		},
		{
			"PumpPinsNotEmpty",
			&ControllerConfig{PumpPins: []uint{1, 2, 3}},
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

	t.Run("RemoveValvePins", func(t *testing.T) {
		c := &ControllerConfig{
			ValvePins: []uint{1, 2, 3},
		}

		err := c.Patch(&ControllerConfig{ValvePins: []uint{5}})
		require.Nil(t, err)

		assert.ElementsMatch(t, []uint{5}, c.ValvePins)
	})

	t.Run("RemovePumpPins", func(t *testing.T) {
		c := &ControllerConfig{
			PumpPins: []uint{1, 2, 3},
		}

		err := c.Patch(&ControllerConfig{PumpPins: []uint{5}})
		require.Nil(t, err)

		assert.ElementsMatch(t, []uint{5}, c.PumpPins)
	})
}
