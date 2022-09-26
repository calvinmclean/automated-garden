package worker

import (
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
)

// ExecuteGardenAction will execute a GardenAction
func (w *Worker) ExecuteGardenAction(g *pkg.Garden, input *action.GardenAction) error {
	if input.Light != nil {
		err := w.ExecuteLightAction(g, input.Light)
		if err != nil {
			return fmt.Errorf("unable to execute LightAction: %v", err)
		}
	}
	if input.Stop != nil {
		err := w.ExecuteStopAction(g, input.Stop)
		if err != nil {
			return fmt.Errorf("unable to execute StopAction: %v", err)
		}
	}
	return nil
}

// ExecuteStopAction sends the message over MQTT to the embedded garden controller
func (w *Worker) ExecuteStopAction(g *pkg.Garden, input *action.StopAction) error {
	topicFunc := w.mqttClient.StopTopic
	if input.All {
		topicFunc = w.mqttClient.StopAllTopic
	}
	topic, err := topicFunc(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return w.mqttClient.Publish(topic, []byte("no message"))
}

// ExecuteLightAction sends an MQTT message to the garden controller to change the state of the light
func (w *Worker) ExecuteLightAction(g *pkg.Garden, input *action.LightAction) error {
	msg, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("unable to marshal LightAction to JSON: %v", err)
	}

	topic, err := w.mqttClient.LightTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish LightAction: %v", err)
	}

	// If this is a LightAction with specified duration, additional steps are necessary
	if input != nil && input.ForDuration != "" {
		err := w.ScheduleLightDelay(g, input)
		if err != nil {
			return fmt.Errorf("unable to handle light delay: %v", err)
		}
	}
	return nil
}
