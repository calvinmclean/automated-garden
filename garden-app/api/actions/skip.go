package actions

import (
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// SkipAction is an action for skipping the next watering event for a Plant
// TODO: currently "count" doesn't do anything and only next watering will be skipped
type SkipAction struct {
	Count int `json:"count"`
}

// SkipMessage is the message being sent over MQTT to the embedded garden controller
type SkipMessage struct {
	PlantID  string `json:"id"`
	ValvePin int    `json:"valve_pin"`
}

// Execute sends the message over MQTT to the embedded garden controller
func (action *SkipAction) Execute(p *api.Plant) error {
	fmt.Printf("Skipping next %d waterings for plant %s\n", action.Count, p.ID)

	msg, err := json.Marshal(SkipMessage{
		PlantID:  p.ID,
		ValvePin: p.ValvePin,
	})
	if err != nil {
		panic(err)
	}

	token := mqttClient.Publish("garden/command/skip", 0, false, msg)
	token.Wait()
	return token.Error()
}
