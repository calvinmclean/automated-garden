package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
)

// ExecuteScheduledWaterAction will run ExecuteWaterAction after checking SkipCount
func (w *Worker) ExecuteScheduledWaterAction(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule, duration time.Duration) error {
	if z.SkipCount != nil && *z.SkipCount > 0 {
		*z.SkipCount--
		err := w.storageClient.Zones.Set(context.Background(), z)
		if err != nil {
			return fmt.Errorf("unable to save Zone after decrementing SkipCount: %w", err)
		}

		w.logger.Info("skipping watering Zone because of SkipCount", "zone_id", z.GetID())
		return nil
	}

	if duration == 0 {
		w.logger.Info("skipping watering Zone because duration is 0")
		return nil
	}

	if ws.GetNotificationClientID() != "" {
		w.sendDownNotification(g, ws.GetNotificationClientID(), "Water")
	}

	return w.ExecuteWaterAction(g, z, &action.WaterAction{
		Duration: &pkg.Duration{Duration: duration},
		Source:   action.SourceSchedule,
	})
}

// CalculateETDuration calculates watering duration based on ET data using the citrus tree formula.
// Returns (duration, true) if ET calculation succeeds, (0, false) otherwise.
// This completely overrides the configured duration when successful.
func (w *Worker) CalculateETDuration(ws *pkg.WaterSchedule) (time.Duration, bool) {
	if !ws.HasEvapotranspirationControl() {
		return 0, false
	}

	etConfig := ws.WeatherControl.Evapotranspiration

	// Validate config
	if err := etConfig.Validate(); err != nil {
		w.logger.Warn("invalid ET configuration", "error", err)
		return 0, false
	}

	// Get weather client
	weatherClient, err := w.storageClient.GetWeatherClient(etConfig.ClientID)
	if err != nil {
		w.logger.Warn("error getting WeatherClient for ET control", "error", err)
		return 0, false
	}

	// Check if client supports ET
	etProvider, ok := weatherClient.(weather.ETProvider)
	if !ok {
		w.logger.Error("weather client does not support evapotranspiration")
		return 0, false
	}

	// Fetch average ET over the interval (minimum 24h enforced by client)
	avgET, err := etProvider.GetAverageEvapotranspiration(context.Background(), ws.Interval.Duration)
	if err != nil {
		w.logger.Warn("error getting evapotranspiration data", "error", err)
		return 0, false
	}

	// Calculate duration using citrus formula
	duration, err := etConfig.CalculateETDuration(avgET, ws.Interval.Duration, time.Now())
	if err != nil {
		w.logger.Warn("error calculating ET-based duration", "error", err)
		return 0, false
	}

	w.logger.Info("calculated ET-based watering duration",
		"duration", duration,
		"avg_et", avgET,
		"species", etConfig.Species,
		"canopy_diameter_feet", etConfig.CanopyDiameterFeet)

	return duration, true
}

// ScaleWateringDuration returns a new watering duration based on weather scaling. It will not return
// any errors if they are encountered because there are multiple factors impacting watering.
// If ET control is configured and succeeds, it provides the base duration which can then be scaled
// by temperature and rain controls if they are also configured.
func (w *Worker) ScaleWateringDuration(ws *pkg.WaterSchedule) (time.Duration, bool) {
	baseDuration := ws.Duration.Duration
	scaleFactor := 1.0
	hadError := false

	if ws.HasEvapotranspirationControl() {
		etDuration, ok := w.CalculateETDuration(ws)
		if ok {
			baseDuration = etDuration
			w.logger.Info("using ET-calculated duration as base", "et_duration", etDuration)
		}
	}

	if ws.HasTemperatureControl() {
		weatherClient, err := w.storageClient.GetWeatherClient(ws.WeatherControl.Temperature.ClientID)
		if err != nil {
			hadError = true
			w.logger.Warn("error getting WeatherClient for TemperatureControl", "error", err)
		} else {
			avgHighTemp, err := weatherClient.GetAverageHighTemperature(context.Background(), ws.Interval.Duration)
			if err != nil {
				hadError = true
				w.logger.Warn("error getting average high temperatures", "error", err)
			} else {
				tempScaleFactor := ws.WeatherControl.Temperature.Scale(float64(avgHighTemp))
				scaleFactor *= tempScaleFactor
				w.logger.With(
					"avg_high_temp", avgHighTemp,
					"time_period", ws.Interval.String(),
					"scale_factor", tempScaleFactor,
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
			totalRain, err := weatherClient.GetTotalRain(context.Background(), ws.Interval.Duration)
			if err != nil {
				hadError = true
				w.logger.Warn("error getting rain data", "error", err)
			} else {
				rainScaleFactor := ws.WeatherControl.Rain.Scale(float64(totalRain))
				w.logger.With(
					"total_rain", totalRain,
					"time_period", ws.Interval.String(),
					"scale_factor", rainScaleFactor,
				).Info("weather client detected rain and resulting scale factor")
				scaleFactor *= rainScaleFactor
			}
		}
	}

	w.logger.Info("compounded scale factor", "compound_scale_factor", scaleFactor, "base_duration", baseDuration)

	result := time.Duration(float64(baseDuration) * scaleFactor)
	if result.Milliseconds() == 0 {
		return 0, hadError
	}
	return result, hadError
}
