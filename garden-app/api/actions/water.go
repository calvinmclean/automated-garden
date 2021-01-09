package actions

import (
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// WaterAction ...
type WaterAction struct {
	Duration int `json:"duration"`
}

// WaterMessage ...
type WaterMessage struct {
	Duration int    `json:"duration"`
	PlantID  string `json:"id"`
	ValvePin int    `json:"valve_pin"`
	PumpPin  int    `json:"pump_pin"`
}

// Execute ...
func (action *WaterAction) Execute(p api.Plant) error {
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
