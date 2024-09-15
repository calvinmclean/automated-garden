package pkg

import (
	"github.com/calvinmclean/babyapi"
)

// ControllerConfig is the configuration used for an
type ControllerConfig struct {
	ValvePins              []uint `json:"valve_pins,omitempty"`
	PumpPins               []uint `json:"pump_pins,omitempty"`
	LightPin               *uint  `json:"light_pin,omitempty"`
	TemperatureHumidityPin *uint  `json:"temperature_humidity_pin,omitempty"`
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
	return nil
}
