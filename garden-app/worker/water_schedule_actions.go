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

// ExecuteScheduledWaterAction will get all of the Zones that use the schedule and execute WaterActions on them after
// scaling durations based on the Zone's configuration
func (w *Worker) ExecuteScheduledWaterAction(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule) error {
	duration, err := w.exerciseWeatherControl(g, z, ws)
	if err != nil {
		w.logger.Errorf("error executing weather controls, continuing to water: %v", err)
		duration = ws.Duration.Duration
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

func (w *Worker) exerciseWeatherControl(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule) (time.Duration, error) {
	if !ws.HasWeatherControl() {
		return ws.Duration.Duration, nil
	}

	skipMoisture, err := w.shouldMoistureSkip(g, z, ws)
	if err != nil {
		return 0, err
	}
	if skipMoisture {
		return 0, nil
	}

	duration, err := w.ScaleWateringDuration(ws)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

func (w *Worker) shouldMoistureSkip(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule) (bool, error) {
	if !ws.HasSoilMoistureControl() {
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
	return moisture > float64(*ws.WeatherControl.SoilMoisture.MinimumMoisture), nil
}

// ScaleWateringDuration returns a new watering duration based on weather scaling
func (w *Worker) ScaleWateringDuration(ws *pkg.WaterSchedule) (time.Duration, error) {
	scaleFactor := float32(1)

	if ws.HasTemperatureControl() {
		weatherClient, err := w.storageClient.GetWeatherClient(ws.WeatherControl.Temperature.ClientID)
		if err != nil {
			return 0, fmt.Errorf("error getting WeatherClient for TemperatureControl: %w", err)
		}

		avgHighTemp, err := weatherClient.GetAverageHighTemperature(ws.Interval.Duration)
		if err != nil {
			w.logger.WithError(err).Warn("error getting average high temperatures")
		} else {
			scaleFactor = ws.WeatherControl.Temperature.Scale(avgHighTemp)
			w.logger.Infof("weather client calculated %fC as the average daily high temperature over the last %s, resulting in scale factor of %f", avgHighTemp, ws.Interval.String(), scaleFactor)
		}
	}

	if ws.HasRainControl() {
		weatherClient, err := w.storageClient.GetWeatherClient(ws.WeatherControl.Rain.ClientID)
		if err != nil {
			return 0, fmt.Errorf("error getting WeatherClient for RainControl: %w", err)
		}

		totalRain, err := weatherClient.GetTotalRain(ws.Interval.Duration)
		if err != nil {
			w.logger.WithError(err).Warn("error getting rain data")
		} else {
			rainScaleFactor := ws.WeatherControl.Rain.InvertedScaleDownOnly(totalRain)
			w.logger.Infof("weather client recorded %fmm of rain in the last %s, resulting in scale factor of %f", totalRain, ws.Interval.String(), rainScaleFactor)
			scaleFactor *= rainScaleFactor
		}
	}

	w.logger.Infof("compounded scale factor: %f", scaleFactor)

	return time.Duration(float32(ws.Duration.Duration) * scaleFactor), nil
}
