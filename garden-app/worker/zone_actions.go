package worker

import (
	"context"
	"encoding/json"
	"errors"
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
		duration = input.Duration.Duration
	}
	if duration == 0 {
		w.logger.Info("weather control determined that watering should be skipped")
		return nil
	}

	msg, err := json.Marshal(action.WaterMessage{
		Duration: duration.Milliseconds(),
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

func (w *Worker) exerciseWeatherControl(g *pkg.Garden, z *pkg.Zone, input *action.WaterAction) (time.Duration, error) {
	if !z.HasWeatherControl() || input.IgnoreWeather {
		return input.Duration.Duration, nil
	}

	skipMoisture, err := w.shouldMoistureSkip(g, z)
	if err != nil {
		return 0, err
	}
	if skipMoisture {
		return 0, nil
	}

	duration, err := w.ScaleWateringDuration(z.WaterSchedule, input.Duration.Duration)
	if err != nil {
		return 0, err
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

// ScaleWateringDuration returns a new watering duration based on weather scaling
func (w *Worker) ScaleWateringDuration(ws *pkg.WaterSchedule, duration time.Duration) (time.Duration, error) {
	if w.weatherClient == nil {
		return 0, errors.New("unable to scale watering duration with no weather.Client")
	}

	scaleFactor := float32(1)

	if ws.HasTemperatureControl() {
		avgHighTemp, err := w.weatherClient.GetAverageHighTemperature(ws.Interval.Duration)
		if err != nil {
			w.logger.WithError(err).Warn("error getting average high temperatures")
		} else {
			scaleFactor = ws.WeatherControl.Temperature.Scale(avgHighTemp)
			w.logger.Infof("weather client calculated %fC as the average daily high temperature over the last %s, resulting in scale factor of %f", avgHighTemp, ws.Interval.String(), scaleFactor)
		}
	}

	if ws.HasRainControl() {
		totalRain, err := w.weatherClient.GetTotalRain(ws.Interval.Duration)
		if err != nil {
			w.logger.WithError(err).Warn("error getting rain data")
		} else {
			rainScaleFactor := ws.WeatherControl.Rain.InvertedScaleDownOnly(totalRain)
			w.logger.Infof("weather client recorded %fmm of rain in the last %s, resulting in scale factor of %f", totalRain, ws.Interval.String(), rainScaleFactor)
			scaleFactor *= rainScaleFactor
		}
	}

	w.logger.Infof("compounded scale factor: %f", scaleFactor)

	return time.Duration(float32(duration) * scaleFactor), nil
}
