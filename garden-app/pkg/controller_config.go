// Package pkg provides domain models and utilities for the garden application
package pkg

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calvinmclean/babyapi"
)

// SensorConfig represents a configured physical sensor on the controller.
// The index in ControllerConfig.Sensors is used as the sensor_id in MQTT messages.
type SensorConfig struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`     // "DHT22" or "DS18B20"
	Pin      uint     `json:"pin"`
	Interval Duration `json:"interval"` // user-facing duration; converted to ms for firmware
}

// SensorCapabilities returns the measurement capabilities for a sensor type.
func SensorCapabilities(sensorType string) []string {
	switch strings.ToUpper(sensorType) {
	case "DHT22":
		return []string{"temperature", "humidity"}
	case "DS18B20":
		return []string{"temperature"}
	default:
		return nil
	}
}

// ControllerConfig is the configuration used for an
type ControllerConfig struct {
	ValvePins []uint         `json:"valve_pins,omitempty"`
	PumpPins  []uint         `json:"pump_pins,omitempty"`
	LightPin  *uint          `json:"light_pin,omitempty"`
	FanPin    *uint          `json:"fan_pin,omitempty"`
	Sensors   []SensorConfig `json:"sensors,omitempty"`
}

// ControllerConfigMessage is similar to ControllerConfig, but is the actual value published
// to the controller. This allows the actual user-facing config to be simplified.
// This is defined here instead of where it's used because this makes it easier to keep consistent
// with the ControllerConfig type
type ControllerConfigMessage struct {
	NumZones  uint                  `json:"num_zones"`
	ValvePins []uint                `json:"valve_pins"`
	PumpPins  []uint                `json:"pump_pins"`
	LightEnabled bool                `json:"light"`
	LightPin  uint                  `json:"light_pin"`
	FanEnabled bool                 `json:"fan"`
	FanPin    uint                  `json:"fan_pin"`
	Sensors   []SensorConfigMessage `json:"sensors"`
}

// SensorConfigMessage is the firmware-facing representation of a SensorConfig.
// The index in the Sensors array is used as the sensor_id.
type SensorConfigMessage struct {
	Type     string `json:"type"`
	Pin      uint   `json:"pin"`
	Interval uint   `json:"interval"` // ms
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

	if c.FanPin != nil {
		message.FanEnabled = true
		message.FanPin = *c.FanPin
	}

	message.Sensors = make([]SensorConfigMessage, len(c.Sensors))
	for i, s := range c.Sensors {
		message.Sensors[i] = SensorConfigMessage{
			Type: s.Type,
			Pin:  s.Pin,
		}
		if s.Interval.Duration > 0 {
			//nolint:gosec
			message.Sensors[i].Interval = uint(s.Interval.Duration.Milliseconds())
		} else {
			message.Sensors[i].Interval = 5000
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
	if newVal.FanPin != nil {
		c.FanPin = newVal.FanPin
	}
	if newVal.Sensors != nil {
		c.Sensors = make([]SensorConfig, len(newVal.Sensors))
		copy(c.Sensors, newVal.Sensors)
	}

	if len(c.PumpPins) != len(c.ValvePins) {
		return babyapi.ErrInvalidRequest(errors.New("pump_pins and valve_pins must be the same length"))
	}

	dht22Pins := map[uint]struct{}{}
	for i, s := range c.Sensors {
		switch strings.ToUpper(s.Type) {
		case "DHT22", "DS18B20":
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("sensor %d has unsupported type %q", i, s.Type))
		}
		if s.Type == "DHT22" {
			if _, exists := dht22Pins[s.Pin]; exists {
				return babyapi.ErrInvalidRequest(fmt.Errorf("multiple DHT22 sensors cannot share pin %d", s.Pin))
			}
			dht22Pins[s.Pin] = struct{}{}
		}
	}

	return nil
}

// ValvePin gets the config's valve pin at a specific index, if it exists, as a string representation for templating
func (c *ControllerConfig) ValvePin(i uint) string {
	if c == nil || uint(len(c.ValvePins)) <= i {
		return ""
	}
	return fmt.Sprint(c.ValvePins[i])
}

// PumpPin gets the config's pump pin at a specific index, if it exists, as a string representation for templating
func (c *ControllerConfig) PumpPin(i uint) string {
	if c == nil || uint(len(c.PumpPins)) <= i {
		return ""
	}
	return fmt.Sprint(c.PumpPins[i])
}

// SensorIntervalMillis gets the sensor interval as a time.Duration, defaulting to 5s.
func (s *SensorConfig) SensorIntervalMillis() time.Duration {
	if s.Interval.Duration > 0 {
		return s.Interval.Duration
	}
	return 5 * time.Second
}
