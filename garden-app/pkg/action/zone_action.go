package action

import (
	"context"
	"encoding/json"
	"fmt"
	time "time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/rs/xid"
)

// ZoneAction collects all the possible actions for a Zone into a single struct so these can easily be
// received as one request
type ZoneAction struct {
	Water *WaterAction `json:"water"`
}

// Execute is responsible for performing the actual individual actions in this ZoneAction.
// The actions are executed in a deliberate order to be most intuitive for a user that wants
// to perform multiple actions with one request
func (action *ZoneAction) Execute(g *pkg.Garden, z *pkg.Zone, scheduler Scheduler) error {
	if action.Water != nil {
		if err := action.Water.Execute(g, z, scheduler); err != nil {
			return err
		}
	}
	return nil
}

// WaterAction is an action for watering a Zone for the specified amount of time
type WaterAction struct {
	Duration       int64 `json:"duration"`
	IgnoreMoisture bool  `json:"ignore_moisture"`
}

// WaterMessage is the message being sent over MQTT to the embedded garden controller
type WaterMessage struct {
	Duration int64  `json:"duration"`
	ZoneID   xid.ID `json:"id"`
	Position uint   `json:"position"`
}

// Execute sends the message over MQTT to the embedded garden controller. Before doing this, it
// will first check if watering is set to skip and if the moisture value is below the threshold
// if configured
func (action *WaterAction) Execute(g *pkg.Garden, z *pkg.Zone, scheduler Scheduler) error {
	if z.WaterSchedule.MinimumMoisture > 0 && !action.IgnoreMoisture {
		ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
		defer cancel()

		defer scheduler.InfluxDBClient().Close()
		moisture, err := scheduler.InfluxDBClient().GetMoisture(ctx, *z.Position, g.TopicPrefix)
		if err != nil {
			return fmt.Errorf("error getting Zone's moisture data: %v", err)
		}
		if moisture > float64(z.WaterSchedule.MinimumMoisture) {
			return fmt.Errorf("moisture value %.2f%% is above threshold %d%%", moisture, z.WaterSchedule.MinimumMoisture)
		}
	}

	if scheduler.WeatherClient() != nil && z.HasWeatherControl() {
		// Ignore weather errors and proceed with watering
		shouldWater, _ := action.shouldWater(g, z, scheduler)
		// TODO: Refactor to be able to return warnings so they can be logged without returning an error
		if !shouldWater {
			return fmt.Errorf("rain control determined that watering should be skipped")
		}
	}

	msg, err := json.Marshal(WaterMessage{
		Duration: action.Duration,
		ZoneID:   z.ID,
		Position: *z.Position,
	})
	if err != nil {
		return fmt.Errorf("unable to marshal WaterMessage to JSON: %v", err)
	}

	topic, err := scheduler.MQTTClient().WaterTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return scheduler.MQTTClient().Publish(topic, msg)
}

func (action *WaterAction) shouldWater(g *pkg.Garden, z *pkg.Zone, scheduler Scheduler) (bool, error) {
	intervalDuration, err := time.ParseDuration(z.WaterSchedule.Interval)
	if err != nil {
		return false, err
	}

	totalRain, err := scheduler.WeatherClient().GetTotalRain(intervalDuration)
	if err != nil {
		return false, err
	}

	// if rain < threshold, still water
	return totalRain < z.WaterSchedule.WeatherControl.Rain.Threshold, nil
}
