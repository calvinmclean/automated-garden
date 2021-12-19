package action

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/rs/xid"
)

// PlantAction collects all the possible actions for a Plant into a single struct so these can easily be
// received as one request
type PlantAction struct {
	Water *WaterAction `json:"water"`
}

// Execute is responsible for performing the actual individual actions in this PlantAction.
// The actions are executed in a deliberate order to be most intuitive for a user that wants
// to perform multiple actions with one request
func (action *PlantAction) Execute(g *pkg.Garden, p *pkg.Plant, mqttClient mqtt.Client, influxdbClient influxdb.Client) error {
	if action.Water != nil {
		if err := action.Water.Execute(g, p, mqttClient, influxdbClient); err != nil {
			return err
		}
	}
	return nil
}

// WaterAction is an action for watering a Plant for the specified amount of time
type WaterAction struct {
	Duration       int64 `json:"duration"`
	IgnoreMoisture bool  `json:"ignore_moisture"`
}

// WaterMessage is the message being sent over MQTT to the embedded garden controller
type WaterMessage struct {
	Duration      int64  `json:"duration"`
	PlantID       xid.ID `json:"id"`
	PlantPosition uint   `json:"plant_position"`
}

// Execute sends the message over MQTT to the embedded garden controller. Before doing this, it
// will first check if watering is set to skip and if the moisture value is below the threshold
// if configured
func (action *WaterAction) Execute(g *pkg.Garden, p *pkg.Plant, mqttClient mqtt.Client, influxdbClient influxdb.Client) error {
	if p.WaterSchedule.MinimumMoisture > 0 && !action.IgnoreMoisture {
		ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
		defer cancel()

		defer influxdbClient.Close()
		moisture, err := influxdbClient.GetMoisture(ctx, *p.PlantPosition, g.TopicPrefix)
		if err != nil {
			return fmt.Errorf("error getting Plant's moisture data: %v", err)
		}
		if moisture > float64(p.WaterSchedule.MinimumMoisture) {
			return fmt.Errorf("moisture value %.2f%% is above threshold %d%%", moisture, p.WaterSchedule.MinimumMoisture)
		}
	}

	msg, err := json.Marshal(WaterMessage{
		Duration:      action.Duration,
		PlantID:       p.ID,
		PlantPosition: *p.PlantPosition,
	})
	if err != nil {
		return fmt.Errorf("unable to marshal WaterMessage to JSON: %v", err)
	}

	topic, err := mqttClient.WateringTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return mqttClient.Publish(topic, msg)
}
