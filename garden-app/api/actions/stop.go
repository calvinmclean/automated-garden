package actions

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// StopAction ...
type StopAction struct {
	All bool `json:"all"`
}

// Execute ...
func (action *StopAction) Execute(p api.Plant) error {
	fmt.Printf("Stopping watering plant (all=%t)\n", action.All)
	topic := "garden/command/stop"
	if action.All {
		topic = "garden/command/stop_all"
	}
	token := mqttClient.Publish(topic, 0, false, "no message")
	token.Wait()
	return token.Error()
}
