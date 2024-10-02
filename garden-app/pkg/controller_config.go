package pkg

import (
	"errors"

	"github.com/calvinmclean/babyapi"
)

// ControllerConfig is the configuration used for an
type ControllerConfig struct {
	ValvePins                   []uint    `json:"valve_pins,omitempty"`
	PumpPins                    []uint    `json:"pump_pins,omitempty"`
	LightPin                    *uint     `json:"light_pin,omitempty"`
	TemperatureHumidityPin      *uint     `json:"temperature_humidity_pin,omitempty"`
	TemperatureHumidityInterval *Duration `json:"temperature_humidity_interval,omitempty"`
}

// ControllerConfigMessage is similar to ControllerConfig, but is the actual value published
// to the controller. This allows the actual user-facing config to be simplified.
// This is defined here instead of where it's used because this makes it easier to keep consistent
// with the ControllerConfig type
type ControllerConfigMessage struct {
	NumZones                    uint   `json:"num_zones"`
	ValvePins                   []uint `json:"valve_pins"`
	PumpPins                    []uint `json:"pump_pins"`
	LightEnabled                bool   `json:"light"`
	LightPin                    uint   `json:"light_pin"`
	TemperatureHumidityEnabled  bool   `json:"temp_humidity"`
	TemperatureHumidityPin      uint   `json:"temp_humidity_pin"`
	TemperatureHumidityInterval uint   `json:"temp_humidity_interval"`
}

// ToMessage converts ControllerConfig to a struct compatible with the controller
func (c *ControllerConfig) ToMessage() ControllerConfigMessage {
	message := ControllerConfigMessage{}

	message.NumZones = uint(len(c.ValvePins))

	message.ValvePins = make([]uint, len(c.ValvePins))
	copy(message.ValvePins, c.ValvePins)

	message.PumpPins = make([]uint, len(c.PumpPins))
	copy(message.PumpPins, c.PumpPins)

	if c.LightPin != nil {
		message.LightEnabled = true
		message.LightPin = *c.LightPin
	}

	if c.TemperatureHumidityPin != nil {
		message.TemperatureHumidityEnabled = true
		message.TemperatureHumidityPin = *c.TemperatureHumidityPin

		if c.TemperatureHumidityInterval != nil {
			//nolint:gosec
			message.TemperatureHumidityInterval = uint(c.TemperatureHumidityInterval.Duration.Milliseconds())
		} else {
			message.TemperatureHumidityInterval = 5000
		}
	}

	return message
}

func (c *ControllerConfig) Patch(newVal *ControllerConfig) *babyapi.ErrResponse {
	if newVal.ValvePins != nil {
		c.ValvePins = make([]uint, len(newVal.ValvePins))
		copy(c.ValvePins, newVal.ValvePins)
	}
	if newVal.PumpPins != nil {
		c.PumpPins = make([]uint, len(newVal.PumpPins))
		copy(c.PumpPins, newVal.PumpPins)
	}
	if newVal.LightPin != nil {
		c.LightPin = newVal.LightPin
	}
	if newVal.TemperatureHumidityPin != nil {
		c.TemperatureHumidityPin = newVal.TemperatureHumidityPin
	}
	if newVal.TemperatureHumidityInterval != nil {
		c.TemperatureHumidityInterval = newVal.TemperatureHumidityInterval
	}

	if len(c.PumpPins) != len(c.ValvePins) {
		return babyapi.ErrInvalidRequest(errors.New("pump_pins and valve_pins must be the same length"))
	}

	return nil
}
