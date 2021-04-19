package api

import (
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/api/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/api/mqtt"
	"github.com/rs/xid"
)

// ActionExecutor is an interface used to create generic actions that the CLI or webserver
// can execute without knowing much detail about what the action is really doing
type ActionExecutor interface {
	Execute(*Plant, mqtt.Config, influxdb.Config) error
}

// AggregateAction collects all the possible actions into a single struct/request so one
// or more action can be performed from a single request
type AggregateAction struct {
	Water *WaterAction `json:"water"`
	Stop  *StopAction  `json:"stop"`
}

// Execute is responsible for performing the actual individual actions in this aggregate.
// The actions are executed in a deliberate order to be most intuitive for a user that wants
// to perform multiple actions with one request
func (action *AggregateAction) Execute(p *Plant, mqttConfig mqtt.Config, influxdbConfig influxdb.Config) error {
	if action.Stop != nil {
		if err := action.Stop.Execute(p, mqttConfig, influxdbConfig); err != nil {
			return err
		}
	}
	if action.Water != nil {
		if err := action.Water.Execute(p, mqttConfig, influxdbConfig); err != nil {
			return err
		}
	}
	return nil
}

// StopAction is an action for stopping watering of a Plant and optionally clearing
// the queue of Plants to water
// TODO: currently this will just stop whatever watering is currently happening, not
//       only for a specific Plant
type StopAction struct {
	All bool `json:"all"`
}

// Execute sends the message over MQTT to the embedded garden controller
func (action *StopAction) Execute(p *Plant, mqttConfig mqtt.Config, influxdbConfig influxdb.Config) error {
	mqttClient, err := mqtt.NewMQTTClient(mqttConfig)
	if err != nil {
		return fmt.Errorf("unable to create MQTT Client: %v", err)
	}
	defer mqttClient.Disconnect(0)

	templateString := mqttClient.StopTopic
	if action.All {
		templateString = mqttClient.StopAllTopic
	}
	topic, err := p.Topic(templateString)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return mqttClient.Publish(topic, []byte("no message"))
}

// WaterAction is an action for watering a Plant for the specified amount of time
type WaterAction struct {
	Duration       int  `json:"duration"`
	IgnoreMoisture bool `json:"ignore_moisture"`
}

// WaterMessage is the message being sent over MQTT to the embedded garden controller
type WaterMessage struct {
	Duration      int    `json:"duration"`
	PlantID       xid.ID `json:"id"`
	PlantPosition int    `json:"plant_position"`
}

// Execute sends the message over MQTT to the embedded garden controller. Before doing this, it
// will first check if watering is set to skip and if the moisture value is below the threshold
// if configured
func (action *WaterAction) Execute(p *Plant, mqttConfig mqtt.Config, influxdbConfig influxdb.Config) error {
	if p.WateringStrategy.MinimumMoisture > 0 && !action.IgnoreMoisture {
		moisture, err := p.GetMoisture(influxdbConfig)
		if err != nil {
			return fmt.Errorf("error getting Plant's moisture data: %v", err)
		}

		if moisture > float64(p.WateringStrategy.MinimumMoisture) {
			return fmt.Errorf("moisture value %f%% is above threshold %d%%", moisture, p.WateringStrategy.MinimumMoisture)
		}
	}
	if p.SkipCount > 0 {
		p.SkipCount--
		return fmt.Errorf("plant %s is configured to skip watering", p.ID)
	}

	msg, err := json.Marshal(WaterMessage{
		Duration:      action.Duration,
		PlantID:       p.ID,
		PlantPosition: p.PlantPosition,
	})
	if err != nil {
		return fmt.Errorf("unable to marshal WaterMessage to JSON: %v", err)
	}

	mqttClient, err := mqtt.NewMQTTClient(mqttConfig)
	if err != nil {
		return fmt.Errorf("unable to create MQTT Client: %v", err)
	}
	defer mqttClient.Disconnect(0)

	topic, err := p.Topic(mqttClient.WateringTopic)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return mqttClient.Publish(topic, msg)
}
