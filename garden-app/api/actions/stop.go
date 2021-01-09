package actions

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// StopAction is an action for stopping watering of a Plant and optionally clearing
// the queue of Plants to water
// TODO: currently this will just stop whatever watering is currently happening, not
//       only for a specific Plant
type StopAction struct {
	All bool `json:"all"`
}

// Execute sends the message over MQTT to the embedded garden controller
func (action *StopAction) Execute(p *api.Plant) error {
	fmt.Printf("Stopping watering plant (all=%t)\n", action.All)
	topic := "garden/command/stop"
	if action.All {
		topic = "garden/command/stop_all"
	}
	token := mqttClient.Publish(topic, 0, false, "no message")
	token.Wait()
	return token.Error()
}
