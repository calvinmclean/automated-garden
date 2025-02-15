package controller

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

func configPrompts(config *Config) error {
	err := mqttPrompts(config)
	if err != nil {
		return fmt.Errorf("error completing MQTT prompts: %w", err)
	}

	err = zonePrompts(config)
	if err != nil {
		return fmt.Errorf("error completing zone prompts: %w", err)
	}

	err = survey.AskOne(&survey.Input{
		Message: "Light pin (optional)",
		Default: config.LightPin,
		Help:    "this is the identifier for the pin that controls a relay attached to a light source",
	}, &config.LightPin)
	if err != nil {
		return fmt.Errorf("error completing light pin prompt: %w", err)
	}

	err = temperatureHumidityPrompts(config)
	if err != nil {
		return fmt.Errorf("error completing temperature and humidity prompts: %w", err)
	}

	return nil
}

func mqttPrompts(config *Config) error {
	qs := []*survey.Question{
		{
			Name: "topic_prefix",
			Prompt: &survey.Input{
				Message: "Topic Prefix",
				Default: config.TopicPrefix,
				Help: "this prefix will be used for all MQTT pub/sub topics. It is used to associate data " +
					"and commands with this particular controller. It is also used for the client ID and must be unique",
			},
			Validate: survey.Required,
		},
		{
			Name: "mqtt_address",
			Prompt: &survey.Input{
				Message: "MQTT Address",
				Default: config.MQTTConfig.Broker,
				Help:    "IP address of the MQTT broker",
			},
			Validate: survey.Required,
		},
		{
			Name: "mqtt_port",
			Prompt: &survey.Input{
				Message: "MQTT Port",
				Default: fmt.Sprintf("%d", config.MQTTConfig.Port),
				Help:    "port of the MQTT broker",
			},
			Validate: survey.Required,
		},
	}
	err := survey.Ask(qs, config)
	if err != nil {
		return fmt.Errorf("error in survey response: %w", err)
	}

	config.MQTTConfig.Broker = config.MQTTAddress
	config.MQTTConfig.Port = config.MQTTPort

	if config.PublishHealth {
		err = survey.AskOne(&survey.Input{
			Message: "Health publishing interval",
			Default: config.HealthInterval.String(),
			Help:    "how often to publish health message. Use the default unless you have good reason not to",
		}, &config.HealthInterval)
		if err != nil {
			return fmt.Errorf("error in survey response: %w", err)
		}
	}

	return nil
}

func temperatureHumidityPrompts(config *Config) error {
	err := survey.AskOne(&survey.Input{
		Message: "Enable temperature and humidity (DHT22) sensor",
		Default: fmt.Sprintf("%t", config.PublishTemperatureHumidity),
		Help:    "enable temperature and humidity publishing",
	}, &config.PublishTemperatureHumidity)
	if err != nil {
		return err
	}

	if !config.PublishTemperatureHumidity {
		return nil
	}

	qs := []*survey.Question{
		{
			Name: "temperature_humidity_interval",
			Prompt: &survey.Input{
				Message: "Temperature and humidity read/publish interval",
				Default: config.TemperatureHumidityInterval.String(),
				Help:    "how often to read and publish temperature and humidity data",
			},
		},
		{
			Name: "temperature_humidity_pin",
			Prompt: &survey.Input{
				Message: "Temperature and humidity sensor (DHT22) pin",
				Default: config.TemperatureHumidityPin,
				Help:    "pin identifier for a DHT22 sensor",
			},
		},
	}
	return survey.Ask(qs, config)
}

func zonePrompts(config *Config) error {
	addAnotherZone := true
	for addAnotherZone {
		err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("You currently have %d Zones configured. Would you like to add another?", len(config.Zones)),
		}, &addAnotherZone)
		if err != nil {
			return err
		}

		if !addAnotherZone {
			break
		}

		qs := []*survey.Question{
			{
				Name: "pump_pin",
				Prompt: &survey.Input{
					Message: "\tPump pin",
					Help:    "pin identifier for the relay controlling a pump or main valve. Use the same value as the valve if no pump is used",
				},
				Validate: survey.Required,
			},
			{
				Name: "valve_pin",
				Prompt: &survey.Input{
					Message: "\tValve pin",
					Help:    "pin identifier used for controlling a valve",
				},
				Validate: survey.Required,
			},
		}

		var zc ZoneConfig
		err = survey.Ask(qs, &zc)
		if err != nil {
			return err
		}
		config.Zones = append(config.Zones, zc)
	}

	return nil
}
