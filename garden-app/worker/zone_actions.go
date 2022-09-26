package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
)

// ExecuteZoneAction will execute a ZoneAction
func (w *Worker) ExecuteZoneAction(g *pkg.Garden, z *pkg.Zone, input *action.ZoneAction) error {
	if input.Water != nil {
		err := w.ExecuteWaterAction(g, z, input.Water)
		if err != nil {
			return fmt.Errorf("unable to execute WaterAction: %v", err)
		}
	}
	return nil
}

// ExecuteWaterAction sends the message over MQTT to the embedded garden controller. Before doing this, it
// will first check if watering is set to skip and if the moisture value is below the threshold
// if configured
func (w *Worker) ExecuteWaterAction(g *pkg.Garden, z *pkg.Zone, input *action.WaterAction) error {
	if z.WaterSchedule.MinimumMoisture > 0 && !input.IgnoreMoisture {
		ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
		defer cancel()

		defer w.influxdbClient.Close()
		moisture, err := w.influxdbClient.GetMoisture(ctx, *z.Position, g.TopicPrefix)
		if err != nil {
			return fmt.Errorf("error getting Zone's moisture data: %v", err)
		}
		if moisture > float64(z.WaterSchedule.MinimumMoisture) {
			return fmt.Errorf("moisture value %.2f%% is above threshold %d%%", moisture, z.WaterSchedule.MinimumMoisture)
		}
	}

	if w.weatherClient != nil && z.HasWeatherControl() && !input.IgnoreWeather {
		// Ignore weather errors and proceed with watering
		shouldWater, _ := w.shouldWaterZone(z)
		// TODO: Refactor to be able to return warnings so they can be logged without returning an error
		if !shouldWater {
			return fmt.Errorf("rain control determined that watering should be skipped")
		}
	}

	msg, err := json.Marshal(action.WaterMessage{
		Duration: input.Duration,
		ZoneID:   z.ID,
		Position: *z.Position,
	})
	if err != nil {
		return fmt.Errorf("unable to marshal WaterMessage to JSON: %v", err)
	}

	topic, err := w.mqttClient.WaterTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %v", err)
	}

	return w.mqttClient.Publish(topic, msg)
}

func (w *Worker) shouldWaterZone(z *pkg.Zone) (bool, error) {
	intervalDuration, err := time.ParseDuration(z.WaterSchedule.Interval)
	if err != nil {
		return false, err
	}

	totalRain, err := w.weatherClient.GetTotalRain(intervalDuration)
	if err != nil {
		return false, err
	}

	// if rain < threshold, still water
	return totalRain < z.WaterSchedule.WeatherControl.Rain.Threshold, nil
}
