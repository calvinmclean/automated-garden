package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/api/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/api/mqtt"
	"github.com/rs/xid"
)

// ActionExecutor is an interface used to create generic actions that the CLI or webserver
// can execute without knowing much detail about what the action is really doing
type ActionExecutor interface {
	Execute(*Plant) error
}

// AggregateAction collects all the possible actions into a single struct/request so one
// or more action can be performed from a single request
type AggregateAction struct {
	Water *WaterAction `json:"water"`
	Stop  *StopAction  `json:"stop"`
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *AggregateAction) Bind(r *http.Request) error {
	// a.AggregateAction is nil if no AggregateAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || (action.Water == nil && action.Stop == nil) {
		return errors.New("missing required action fields")
	}

	return nil
}

// Execute is responsible for performing the actual individual actions in this aggregate.
// The actions are executed in a deliberate order to be most intuitive for a user that wants
// to perform multiple actions with one request
func (action *AggregateAction) Execute(p *Plant) error {
	if action.Stop != nil {
		if err := action.Stop.Execute(p); err != nil {
			return err
		}
	}
	if action.Water != nil {
		if err := action.Water.Execute(p); err != nil {
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
func (action *StopAction) Execute(p *Plant) error {
	mqttClient, err := mqtt.NewMQTTClient()
	if err != nil {
		panic(err)
	}

	templateString := mqttClient.StopTopic
	if action.All {
		templateString = mqttClient.StopAllTopic
	}
	topic, err := p.Topic(templateString)
	if err != nil {
		panic(err)
	}

	defer mqttClient.Disconnect(0)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	token := mqttClient.Publish(topic, 0, false, "no message")
	token.Wait()
	return token.Error()
}

// WaterAction is an action for watering a Plant for the specified amount of time
type WaterAction struct {
	Duration int `json:"duration"`
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
func (action *WaterAction) Execute(p *Plant) error {
	if p.SkipCount > 0 {
		p.SkipCount--
		return fmt.Errorf("plant %s is configured to skip watering", p.ID)
	}
	if p.WateringStrategy.MinimumMoisture > 0 {
		client, err := influxdb.NewClient()
		if err != nil {
			return fmt.Errorf("unable to initialize influxdb client: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
		defer cancel()
		moisture, err := client.GetMoisture(ctx, p.PlantPosition, p.Garden)
		if err != nil {
			return fmt.Errorf("unable to query influxdb for moisture data: %v", err)
		}

		if moisture > float64(p.WateringStrategy.MinimumMoisture) {
			return fmt.Errorf("moisture value %f%% is above threshold %d%%", moisture, p.WateringStrategy.MinimumMoisture)
		}
	}

	msg, err := json.Marshal(WaterMessage{
		Duration:      action.Duration,
		PlantID:       p.ID,
		PlantPosition: p.PlantPosition,
	})
	if err != nil {
		panic(err)
	}

	mqttClient, err := mqtt.NewMQTTClient()
	if err != nil {
		panic(err)
	}

	topic, err := p.Topic(mqttClient.WateringTopic)
	if err != nil {
		panic(err)
	}

	defer mqttClient.Disconnect(0)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	token := mqttClient.Publish(topic, 0, false, msg)
	token.Wait()
	return token.Error()
}
