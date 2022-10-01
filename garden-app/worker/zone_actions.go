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
	if z.HasWeatherControl() {
		shouldSkip, err := w.shouldSkipWatering(g, z, input)
		// Ignore weather errors and proceed with watering
		if err != nil {
			w.logger.Errorf("unable to determine if watering should be skipped, continuing to water: %v", err)
		}
		if shouldSkip {
			w.logger.Info("weather control determined that watering should be skipped")
			return nil
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

func (w *Worker) shouldSkipWatering(g *pkg.Garden, z *pkg.Zone, input *action.WaterAction) (bool, error) {
	if w.weatherClient != nil && z.WaterSchedule.WeatherControl.Rain != nil && !input.IgnoreWeather {
		skipRain, err := w.shouldRainSkip(z)
		if err != nil {
			return false, err
		}
		if skipRain {
			return true, nil
		}
	}

	if z.WaterSchedule.WeatherControl.SoilMoisture != nil && z.WaterSchedule.WeatherControl.SoilMoisture.MinimumMoisture > 0 && !input.IgnoreMoisture {
		skipMoisture, err := w.shouldMoistureSkip(g, z)
		if err != nil {
			return false, err
		}
		if skipMoisture {
			return true, nil
		}
	}

	return false, nil
}

func (w *Worker) shouldRainSkip(z *pkg.Zone) (bool, error) {
	intervalDuration, err := time.ParseDuration(z.WaterSchedule.Interval)
	if err != nil {
		return true, err
	}

	totalRain, err := w.weatherClient.GetTotalRain(intervalDuration)
	if err != nil {
		return true, err
	}
	w.logger.Infof("weather client recorded %fmm of rain in the last %s", totalRain, intervalDuration.String())

	// if rain >= threshold, skip watering
	return totalRain >= z.WaterSchedule.WeatherControl.Rain.Threshold, nil
}

func (w *Worker) shouldMoistureSkip(g *pkg.Garden, z *pkg.Zone) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
	defer cancel()

	defer w.influxdbClient.Close()
	moisture, err := w.influxdbClient.GetMoisture(ctx, *z.Position, g.TopicPrefix)
	if err != nil {
		return false, fmt.Errorf("error getting Zone's moisture data: %v", err)
	}
	w.logger.Infof("soil moisture is %f%%", moisture)

	// if moisture > minimum, skip watering
	return moisture > float64(z.WaterSchedule.WeatherControl.SoilMoisture.MinimumMoisture), nil
}
