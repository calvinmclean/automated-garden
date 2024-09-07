package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
)

// ExecuteScheduledWaterAction will run ExecuteWaterAction after checking SkipCount and scaling based on weather data
func (w *Worker) ExecuteScheduledWaterAction(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule) error {
	if z.SkipCount != nil && *z.SkipCount > 0 {
		*z.SkipCount--
		err := w.storageClient.Zones.Set(context.Background(), z)
		if err != nil {
			return fmt.Errorf("unable to save Zone after decrementing SkipCount: %w", err)
		}

		w.logger.Info("skipping watering Zone because of SkipCount", "zone_id", z.GetID())
		return nil
	}
	duration, err := w.exerciseWeatherControl(g, z, ws)
	if err != nil {
		w.logger.Error("error executing weather controls, continuing to water", "error", err)
		duration = ws.Duration.Duration
	}
	if duration == 0 {
		w.logger.Info("weather control determined that watering should be skipped")
		return nil
	}

	if ws.GetNotificationClientID() != "" {
		w.sendDownNotification(g, ws.GetNotificationClientID(), "Water")
	}

	return w.ExecuteWaterAction(g, z, &action.WaterAction{
		Duration: &pkg.Duration{Duration: duration},
	})
}

func (w *Worker) exerciseWeatherControl(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule) (time.Duration, error) {
	if !ws.HasWeatherControl() {
		return ws.Duration.Duration, nil
	}

	duration, _ := w.ScaleWateringDuration(ws)
	return duration, nil
}

// ScaleWateringDuration returns a new watering duration based on weather scaling. It will not return
// any errors if they are encountered because there are multiple factors impacting watering
func (w *Worker) ScaleWateringDuration(ws *pkg.WaterSchedule) (time.Duration, bool) {
	scaleFactor := float32(1)
	hadError := false

	if ws.HasTemperatureControl() {
		weatherClient, err := w.storageClient.GetWeatherClient(ws.WeatherControl.Temperature.ClientID)
		if err != nil {
			hadError = true
			w.logger.Warn("error getting WeatherClient for TemperatureControl", "error", err)
		} else {
			avgHighTemp, err := weatherClient.GetAverageHighTemperature(ws.Interval.Duration)
			if err != nil {
				hadError = true
				w.logger.Warn("error getting average high temperatures", "error", err)
			} else {
				scaleFactor = ws.WeatherControl.Temperature.Scale(avgHighTemp)
				w.logger.With(
					"avg_high_temp", avgHighTemp,
					"time_period", ws.Interval.String(),
					"scale_factor", scaleFactor,
				).Info("weather client calculated the average daily high temperature and resulting scale factor")
			}
		}
	}

	if ws.HasRainControl() {
		weatherClient, err := w.storageClient.GetWeatherClient(ws.WeatherControl.Rain.ClientID)
		if err != nil {
			hadError = true
			w.logger.Warn("error getting WeatherClient for RainControl", "error", err)
		} else {
			totalRain, err := weatherClient.GetTotalRain(ws.Interval.Duration)
			if err != nil {
				hadError = true
				w.logger.Warn("error getting rain data", "error", err)
			} else {
				rainScaleFactor := ws.WeatherControl.Rain.InvertedScaleDownOnly(totalRain)
				w.logger.With(
					"total_rain", totalRain,
					"time_period", ws.Interval.String(),
					"scale_factor", rainScaleFactor,
				).Info("weather client detected rain and resulting scale factor")
				scaleFactor *= rainScaleFactor
			}
		}
	}

	w.logger.Info("compounded scale factor", "compound_scale_factor", scaleFactor)

	return time.Duration(float32(ws.Duration.Duration) * scaleFactor), hadError
}
