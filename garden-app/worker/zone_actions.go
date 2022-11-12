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
			return fmt.Errorf("unable to execute WaterAction: %w", err)
		}
	}
	return nil
}

// ExecuteWaterAction sends the message over MQTT to the embedded garden controller. Before doing this, it
// will first check if watering is set to skip and if the moisture value is below the threshold
// if configured
func (w *Worker) ExecuteWaterAction(g *pkg.Garden, z *pkg.Zone, input *action.WaterAction) error {
	duration, err := w.exerciseWeatherControl(g, z, input)
	if err != nil {
		w.logger.Errorf("error executing weather controls, continuing to water: %v", err)
		duration = input.Duration
	}
	if duration == 0 {
		w.logger.Info("weather control determined that watering should be skipped")
		return nil
	}

	msg, err := json.Marshal(action.WaterMessage{
		Duration: duration,
		ZoneID:   z.ID,
		Position: *z.Position,
	})
	if err != nil {
		return fmt.Errorf("unable to marshal WaterMessage to JSON: %w", err)
	}

	topic, err := w.mqttClient.WaterTopic(g.TopicPrefix)
	if err != nil {
		return fmt.Errorf("unable to fill MQTT topic template: %w", err)
	}

	return w.mqttClient.Publish(topic, msg)
}

func (w *Worker) exerciseWeatherControl(g *pkg.Garden, z *pkg.Zone, input *action.WaterAction) (int64, error) {
	if !z.HasWeatherControl() || input.IgnoreWeather {
		return input.Duration, nil
	}

	skipMoisture, err := w.shouldMoistureSkip(g, z)
	if err != nil {
		return input.Duration, err
	}
	if skipMoisture {
		return 0, nil
	}

	duration, err := w.scaleWateringDuration(z.WaterSchedule, input.Duration)
	if err != nil {
		return input.Duration, err
	}

	return duration, nil
}

func (w *Worker) shouldMoistureSkip(g *pkg.Garden, z *pkg.Zone) (bool, error) {
	if !z.WaterSchedule.HasSoilMoistureControl() {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
	defer cancel()

	defer w.influxdbClient.Close()
	moisture, err := w.influxdbClient.GetMoisture(ctx, *z.Position, g.TopicPrefix)
	if err != nil {
		return false, fmt.Errorf("error getting Zone's moisture data: %w", err)
	}
	w.logger.Infof("soil moisture is %f%%", moisture)

	// if moisture > minimum, skip watering
	return moisture > float64(z.WaterSchedule.WeatherControl.SoilMoisture.MinimumMoisture), nil
}

func (w *Worker) scaleWateringDuration(ws *pkg.WaterSchedule, duration int64) (int64, error) {
	scaleFactor := float32(1)

	interval, err := time.ParseDuration(ws.Interval)
	if err != nil {
		return duration, fmt.Errorf("error parsing WaterSchedule.Interval as duration: %w", err)
	}

	if ws.HasTemperatureControl() {
		avgHighTemp, err := w.weatherClient.GetAverageHighTemperature(interval)
		if err != nil {
			w.logger.WithError(err).Warn("error getting average high temperatures, continuing")
		} else {
			scaleFactor = ws.WeatherControl.Temperature.Scale(avgHighTemp)
			w.logger.Infof("weather client calculated %fC as the average daily high temperature over the last %s, resulting in scale factor of %f", avgHighTemp, interval.String(), scaleFactor)
		}
	}

	if ws.HasRainControl() {
		totalRain, err := w.weatherClient.GetTotalRain(interval)
		if err != nil {
			w.logger.WithError(err).Warn("error getting rain data")
		} else {
			rainScaleFactor := ws.WeatherControl.Rain.InvertedScaleDownOnly(totalRain)
			w.logger.Infof("weather client recorded %fmm of rain in the last %s, resulting in scale factor of %f", totalRain, interval.String(), rainScaleFactor)
			scaleFactor *= rainScaleFactor
		}
	}

	w.logger.Infof("compounded scale factor: %f", scaleFactor)

	return int64(float32(duration) * scaleFactor), nil
}
