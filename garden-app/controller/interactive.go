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

	err = wateringPrompts(config)
	if err != nil {
		return fmt.Errorf("error completing watering prompts: %w", err)
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

	err = buttonPrompts(config)
	if err != nil {
		return fmt.Errorf("error completing button prompts: %w", err)
	}

	err = moisturePrompts(config)
	if err != nil {
		return fmt.Errorf("error completing moisture prompts: %w", err)
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
		{
			Name: "publish_health",
			Prompt: &survey.Input{
				Message: "Enable health publishing?",
				Default: fmt.Sprintf("%t", config.PublishHealth),
				Help:    "control whether or not healh publishing is enabled. Enable it unless you have a good reason not to",
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

func buttonPrompts(config *Config) error {
	err := survey.AskOne(&survey.Input{
		Message: "Enable buttons",
		Default: fmt.Sprintf("%t", config.EnableButtons),
		Help:    "allow the use of buttons for controlling watering using the default water time",
	}, &config.EnableButtons)
	if err != nil {
		return err
	}

	if !config.EnableButtons {
		return nil
	}

	return survey.AskOne(&survey.Input{
		Message: "Stop watering button pin",
		Default: config.StopButtonPin,
		Help:    "pin identifier of the button to use for stopping current watering",
	}, &config.StopButtonPin)
}

func moisturePrompts(config *Config) error {
	err := survey.AskOne(&survey.Input{
		Message: "Enable moisture sensor",
		Default: fmt.Sprintf("%t", config.EnableMoistureSensor),
		Help:    "enable moisture data publishing",
	}, &config.EnableMoistureSensor)
	if err != nil {
		return err
	}

	if !config.EnableMoistureSensor {
		return nil
	}

	qs := []*survey.Question{
		{
			Name: "moisture_interval",
			Prompt: &survey.Input{
				Message: "Moisture reading interval",
				Default: config.MoistureInterval.String(),
				Help:    "how often to read and publish moisture data for each configured sensor",
			},
		},
	}
	return survey.Ask(qs, config)
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

func wateringPrompts(config *Config) error {
	qs := []*survey.Question{
		{
			Name: "disable_watering",
			Prompt: &survey.Input{
				Message: "Disable watering",
				Default: fmt.Sprintf("%t", config.DisableWatering),
				Help:    "do not allow watering. Only used by sensor-only gardens",
			},
			Validate: survey.Required,
		},
		{
			Name: "default_water_time",
			Prompt: &survey.Input{
				Message: "Default water time",
				Default: config.DefaultWaterTime.String(),
				Help:    "default time (in milliseconds) to use for watering if button is used or command is missing value",
			},
			Validate: survey.Required,
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
					Help:    "pin identifier for the relay controlling a pump or main valve",
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
			{
				Name: "button_pin",
				Prompt: &survey.Input{
					Message: "\tButton pin",
					Default: "GPIO_NUM_MAX",
					Help:    "pin identifier for a button that controls this zone (GPIO_NUM_MAX to disable)",
				},
			},
			{
				Name: "moisture_sensor_pin",
				Prompt: &survey.Input{
					Message: "\tMoisture sensor pin",
					Default: "GPIO_NUM_MAX",
					Help:    "pin identifier for a moisture sensor that corresponds to this zone (GPIO_NUM_MAX to disable)",
				},
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
