package actions

import (
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// WaterAction is an action for watering a Plant for the specified amount of time
type WaterAction struct {
	Duration int `json:"duration"`
}

// WaterMessage is the message being sent over MQTT to the embedded garden controller
type WaterMessage struct {
	Duration int    `json:"duration"`
	PlantID  string `json:"id"`
	ValvePin int    `json:"valve_pin"`
	PumpPin  int    `json:"pump_pin"`
}

// Execute sends the message over MQTT to the embedded garden controller
func (action *WaterAction) Execute(p *api.Plant) error {
	fmt.Printf("Watering plant %s for %dms\n", p.ID, action.Duration)

	msg, err := json.Marshal(WaterMessage{
		Duration: action.Duration,
		PlantID:  p.ID,
		ValvePin: p.ValvePin,
		PumpPin:  p.PumpPin,
	})
	if err != nil {
		panic(err)
	}

	token := mqttClient.Publish("garden/command/water", 0, false, msg)
	token.Wait()
	return token.Error()
}
