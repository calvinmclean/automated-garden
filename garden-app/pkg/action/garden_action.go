package action

import (
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// GardenAction collects all the possible actions for a Garden into a single struct so these can easily be
// received as one request
type GardenAction struct {
	Light *LightAction `json:"light"`
	Stop  *StopAction  `json:"stop"`
}

// Execute is responsible for performing the actual individual actions in this GardenAction.
// The actions are executed in a deliberate order to be most intuitive for a user that wants
// to perform multiple actions with one request
func (action *GardenAction) Execute(g *pkg.Garden, scheduler Scheduler) error {
	if action.Stop != nil {
		if err := action.Stop.Execute(g, scheduler); err != nil {
			return err
		}
	}
	if action.Light != nil {
		if err := action.Light.Execute(g, scheduler); err != nil {
			return err
		}
	}
	return nil
}

// LightAction is an action for turning on or off a light for the Garden. The State field is optional and it will just toggle
// the current state if left empty.
type LightAction struct {
	State       pkg.LightState `json:"state"`
	ForDuration string         `json:"for_duration"`
}

// Execute sends an MQTT message to the garden controller to change the state of the light
func (action *LightAction) Execute(g *pkg.Garden, scheduler Scheduler) error {
	msg, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("unable to marshal LightAction to JSON: %v", err)
	}

	topic, err := scheduler.MQTTClient().LightTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	err = scheduler.MQTTClient().Publish(topic, msg)
	if err != nil {
		return fmt.Errorf("unable to publish LightAction: %v", err)
	}

	// If this is a LightAction with specified duration, additional steps are necessary
	if action != nil && action.ForDuration != "" {
		err := scheduler.ScheduleLightDelay(g, action)
		if err != nil {
			return fmt.Errorf("unable to handle light delay: %v", err)
		}
	}
	return nil
}

// StopAction is an action for stopping watering of a Plant. It doesn't stop watering a specific Plant, only what is
// currently watering and optionally clearing the queue of Plants to water.
type StopAction struct {
	All bool `json:"all"`
}

// Execute sends the message over MQTT to the embedded garden controller
func (action *StopAction) Execute(g *pkg.Garden, scheduler Scheduler) error {
	topicFunc := scheduler.MQTTClient().StopTopic
	if action.All {
		topicFunc = scheduler.MQTTClient().StopAllTopic
	}
	topic, err := topicFunc(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return scheduler.MQTTClient().Publish(topic, []byte("no message"))
}
