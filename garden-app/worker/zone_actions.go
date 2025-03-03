package worker

import (
	"encoding/json"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/rs/xid"
)

// CreateNewID can be overridden for mocking
var CreateNewID = xid.New

// ExecuteZoneAction will execute a ZoneAction
func (w *Worker) ExecuteZoneAction(g *pkg.Garden, z *pkg.Zone, input *action.ZoneAction) error {
	if input.Water != nil {
		err := w.ExecuteWaterAction(g, z, input.Water)
		if err != nil {
			return fmt.Errorf("unable to execute WaterAction: %w", err)
		}
	}
	return nil
}

// ExecuteWaterAction sends the message over MQTT to the embedded garden controller. This is used for a directly-requested
// WaterAction and does not perform any of the watering checks that are usuall done for a scheduled watering
func (w *Worker) ExecuteWaterAction(g *pkg.Garden, z *pkg.Zone, input *action.WaterAction) error {
	if input.Duration.Duration == 0 {
		w.logger.Info("weather control determined that watering should be skipped")
		return nil
	}

	eventID := CreateNewID().String()
	msg, err := json.Marshal(action.WaterMessage{
		Duration: input.Duration.Duration.Milliseconds(),
		ZoneID:   z.GetID(),
		Position: *z.Position,
		EventID:  eventID,
	})
	if err != nil {
		return fmt.Errorf("unable to marshal WaterMessage to JSON: %w", err)
	}

	topic, err := mqtt.WaterTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %w", err)
	}

	w.logger.Info("publishing WaterMessage", "event_id", eventID)

	return w.mqttClient.Publish(topic, msg)
}
