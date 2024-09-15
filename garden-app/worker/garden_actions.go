package worker

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
)

// ExecuteGardenAction will execute a GardenAction
func (w *Worker) ExecuteGardenAction(g *pkg.Garden, input *action.GardenAction) error {
	switch {
	case input.Light != nil:
		err := w.ExecuteLightAction(g, input.Light)
		if err != nil {
			return fmt.Errorf("unable to execute LightAction: %v", err)
		}
	case input.Stop != nil:
		err := w.ExecuteStopAction(g, input.Stop)
		if err != nil {
			return fmt.Errorf("unable to execute StopAction: %v", err)
		}
	case input.Update != nil:
		err := w.ExecuteUpdateAction(g, input.Update)
		if err != nil {
			return fmt.Errorf("unable to execute UpdateActin: %v", err)
		}
	}
	return nil
}

// ExecuteStopAction sends the message over MQTT to the embedded garden controller
func (w *Worker) ExecuteStopAction(g *pkg.Garden, input *action.StopAction) error {
	topicFunc := mqtt.StopTopic
	if input.All {
		topicFunc = mqtt.StopAllTopic
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

	topic, err := mqtt.LightTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish LightAction: %v", err)
	}

	// If this is a LightAction with specified duration, additional steps are necessary
	if input != nil && input.ForDuration != nil {
		err := w.ScheduleLightDelay(g, input)
		if err != nil {
			return fmt.Errorf("unable to handle light delay: %v", err)
		}
	}
	return nil
}

// ExecuteUpdateAction sends an MQTT message to the garden controller with the current configuration
func (w *Worker) ExecuteUpdateAction(g *pkg.Garden, input *action.UpdateAction) error {
	if !input.Config {
		return errors.New("update action must have config=true")
	}
	msg, err := json.Marshal(g.ControllerConfig)
	if err != nil {
		return fmt.Errorf("unable to marshal ControllerConfig to JSON: %v", err)
	}

	topic, err := mqtt.UpdateTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = w.mqttClient.Publish(topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish UpdateAction: %v", err)
	}

	return nil
}
