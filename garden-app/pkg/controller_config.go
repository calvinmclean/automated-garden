package pkg

import (
	"github.com/calvinmclean/babyapi"
)

// ControllerConfig is the configuration used for an
type ControllerConfig struct {
	NumZones                    *uint  `json:"num_zones,omitempty"`
	ValvePins                   []uint `json:"valve_pins,omitempty"`
	PumpPins                    []uint `json:"pump_pins,omitempty"`
	LightEnabled                *bool  `json:"light,omitempty"`
	LightPin                    *uint  `json:"light_pin,omitempty"`
	TemperatureHumidityEnabled  *bool  `json:"temp_humidity,omitempty"`
	TemperatureHumidityPin      *uint  `json:"temp_humidity_pin,omitempty"`
	TemperatureHumidityInterval *uint  `json:"temp_humidity_interval,omitempty"`
}

func (c *ControllerConfig) Patch(newVal *ControllerConfig) *babyapi.ErrResponse {
	if newVal.NumZones != nil {
		c.NumZones = newVal.NumZones
	}
	if newVal.ValvePins != nil {
		c.ValvePins = make([]uint, len(newVal.ValvePins))
		copy(c.ValvePins, newVal.ValvePins)
	}
	if newVal.PumpPins != nil {
		c.PumpPins = make([]uint, len(newVal.PumpPins))
		copy(c.PumpPins, newVal.PumpPins)
	}
	if newVal.LightEnabled != nil {
		c.LightEnabled = newVal.LightEnabled
	}
	if newVal.LightPin != nil {
		c.LightPin = newVal.LightPin
	}
	if newVal.TemperatureHumidityEnabled != nil {
		c.TemperatureHumidityEnabled = newVal.TemperatureHumidityEnabled
	}
	if newVal.TemperatureHumidityPin != nil {
		c.TemperatureHumidityPin = newVal.TemperatureHumidityPin
	}
	if newVal.TemperatureHumidityInterval != nil {
		c.TemperatureHumidityInterval = newVal.TemperatureHumidityInterval
	}

	return nil
}
