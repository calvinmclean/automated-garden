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
			"FanPin",
			&ControllerConfig{FanPin: pointer(uint(1))},
		},
		{
			"Sensors",
			&ControllerConfig{Sensors: []SensorConfig{
				{Name: "Ambient", Type: "DHT22", Pin: 21, Interval: Duration{Duration: 5 * time.Second}},
				{Name: "Reservoir", Type: "DS18B20", Pin: 22, Interval: Duration{Duration: 5 * time.Second}},
			}},
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

			if tt.name == "Sensors" {
				require.Len(t, c.Sensors, len(tt.newConfig.Sensors))
				for i := range c.Sensors {
					assert.NotEmpty(t, c.Sensors[i].ID)
					assert.Equal(t, tt.newConfig.Sensors[i].Name, c.Sensors[i].Name)
					assert.Equal(t, tt.newConfig.Sensors[i].Type, c.Sensors[i].Type)
					assert.Equal(t, tt.newConfig.Sensors[i].Pin, c.Sensors[i].Pin)
					assert.Equal(t, tt.newConfig.Sensors[i].Interval, c.Sensors[i].Interval)
				}
			} else {
				assert.EqualValues(t, tt.newConfig, c)
			}
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

	t.Run("InvalidSensorType", func(t *testing.T) {
		c := &ControllerConfig{}
		err := c.Patch(&ControllerConfig{Sensors: []SensorConfig{
			{Name: "Bad", Type: "BMP280", Pin: 1},
		}})
		require.Error(t, err)
	})

	t.Run("DuplicateDHT22Pin", func(t *testing.T) {
		c := &ControllerConfig{}
		err := c.Patch(&ControllerConfig{Sensors: []SensorConfig{
			{Name: "A", Type: "DHT22", Pin: 21},
			{Name: "B", Type: "DHT22", Pin: 21},
		}})
		require.Error(t, err)
	})

	t.Run("PreserveExistingSensorIDs", func(t *testing.T) {
		c := &ControllerConfig{Sensors: []SensorConfig{
			{ID: "existing1", Name: "Ambient", Type: "DHT22", Pin: 21},
		}}

		err := c.Patch(&ControllerConfig{Sensors: []SensorConfig{
			{ID: "existing1", Name: "Ambient Renamed", Type: "DHT22", Pin: 22},
			{Name: "New", Type: "DS18B20", Pin: 23},
		}})
		require.Nil(t, err)
		require.Len(t, c.Sensors, 2)
		assert.Equal(t, "existing1", c.Sensors[0].ID)
		assert.Equal(t, "Ambient Renamed", c.Sensors[0].Name)
		assert.Equal(t, uint(22), c.Sensors[0].Pin)
		assert.NotEmpty(t, c.Sensors[1].ID)
		assert.NotEqual(t, "existing1", c.Sensors[1].ID)
	})

	t.Run("UnknownSensorIDRejected", func(t *testing.T) {
		c := &ControllerConfig{Sensors: []SensorConfig{
			{ID: "existing1", Name: "Ambient", Type: "DHT22", Pin: 21},
		}}

		err := c.Patch(&ControllerConfig{Sensors: []SensorConfig{
			{ID: "unknown", Name: "Hacker", Type: "DHT22", Pin: 21},
		}})
		require.Error(t, err)
	})

	t.Run("DuplicateSensorIDRejected", func(t *testing.T) {
		c := &ControllerConfig{}

		err := c.Patch(&ControllerConfig{Sensors: []SensorConfig{
			{ID: "dup", Name: "A", Type: "DHT22", Pin: 21},
			{ID: "dup", Name: "B", Type: "DS18B20", Pin: 22},
		}})
		require.Error(t, err)
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
				ValvePins: []uint{1},
				PumpPins:  []uint{1},
				LightPin:  pointer(uint(1)),
				FanPin:    pointer(uint(2)),
				Sensors: []SensorConfig{
					{ID: "sensor1", Name: "Ambient", Type: "DHT22", Pin: 21, Interval: Duration{Duration: time.Second}},
					{ID: "sensor2", Name: "Reservoir", Type: "DS18B20", Pin: 22, Interval: Duration{Duration: 2 * time.Second}},
				},
			},
			ControllerConfigMessage{
				NumZones:     1,
				ValvePins:    []uint{1},
				PumpPins:     []uint{1},
				LightEnabled: true,
				LightPin:     uint(1),
				FanEnabled:   true,
				FanPin:       uint(2),
				Sensors: []SensorConfigMessage{
					{ID: "sensor1", Type: "DHT22", Pin: 21, Interval: 1000},
					{ID: "sensor2", Type: "DS18B20", Pin: 22, Interval: 2000},
				},
			},
		},
		{
			"DefaultSensorInterval",
			&ControllerConfig{
				ValvePins: []uint{1},
				PumpPins:  []uint{1},
				LightPin:  pointer(uint(1)),
				FanPin:    pointer(uint(2)),
				Sensors: []SensorConfig{
					{ID: "sensor1", Name: "Ambient", Type: "DHT22", Pin: 21},
				},
			},
			ControllerConfigMessage{
				NumZones:     1,
				ValvePins:    []uint{1},
				PumpPins:     []uint{1},
				LightEnabled: true,
				LightPin:     uint(1),
				FanEnabled:   true,
				FanPin:       uint(2),
				Sensors: []SensorConfigMessage{
					{ID: "sensor1", Type: "DHT22", Pin: 21, Interval: 5000},
				},
			},
		},
		{
			"NoSensors",
			&ControllerConfig{
				ValvePins: []uint{1},
				PumpPins:  []uint{1},
			},
			ControllerConfigMessage{
				NumZones:     1,
				ValvePins:    []uint{1},
				PumpPins:     []uint{1},
				LightEnabled: false,
				FanEnabled:   false,
				Sensors:      []SensorConfigMessage{},
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

func TestValvePin(t *testing.T) {
	tests := []struct {
		name     string
		config   *ControllerConfig
		input    uint
		expected string
	}{
		{
			"NilConfig",
			nil,
			3,
			"",
		},
		{
			"EmptyPins",
			&ControllerConfig{},
			3,
			"",
		},
		{
			"IndexOutOfBounds",
			&ControllerConfig{
				ValvePins: []uint{1, 2},
			},
			3,
			"",
		},
		{
			"0",
			&ControllerConfig{
				ValvePins: []uint{1, 2},
			},
			0,
			"1",
		},
		{
			"1",
			&ControllerConfig{
				ValvePins: []uint{1, 2},
			},
			1,
			"2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.config.ValvePin(tt.input)
			require.Equal(t, tt.expected, out)
		})
	}
}

func TestPumpPin(t *testing.T) {
	tests := []struct {
		name     string
		config   *ControllerConfig
		input    uint
		expected string
	}{
		{
			"NilConfig",
			nil,
			3,
			"",
		},
		{
			"EmptyPins",
			&ControllerConfig{},
			3,
			"",
		},
		{
			"IndexOutOfBounds",
			&ControllerConfig{
				PumpPins: []uint{1, 2},
			},
			3,
			"",
		},
		{
			"0",
			&ControllerConfig{
				PumpPins: []uint{1, 2},
			},
			0,
			"1",
		},
		{
			"1",
			&ControllerConfig{
				PumpPins: []uint{1, 2},
			},
			1,
			"2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.config.PumpPin(tt.input)
			require.Equal(t, tt.expected, out)
		})
	}
}
