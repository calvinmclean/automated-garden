package server

import (
	"context"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
)

// WeatherData is used to represent the data used for WeatherControl to a user
type WeatherData struct {
	Rain                *RainData        `json:"rain,omitempty"`
	Temperature         *TemperatureData `json:"average_temperature,omitempty"`
	SoilMoisturePercent *float64         `json:"soil_moisture_percent,omitempty"`
}

// RainData shows the total rain in the last watering interval and the scaling factor it would result in
type RainData struct {
	MM          float32 `json:"mm"`
	ScaleFactor float32 `json:"scale_factor"`
}

// TemperatureData shows the average high temperatures in the last watering interval and the scaling factor it would result in
type TemperatureData struct {
	Celcius     float32 `json:"celcius"`
	ScaleFactor float32 `json:"scale_factor"`
}

func getWeatherData(ctx context.Context, ws *pkg.WaterSchedule, storageClient storage.Client) *WeatherData {
	logger := getLoggerFromContext(ctx).WithField(waterScheduleIDLogField, ws.ID.String())
	weatherData := &WeatherData{}
	var err error

	if ws.HasRainControl() {
		logger.Debug("getting rain data for WaterSchedule")
		weatherData.Rain = &RainData{}
		weatherData.Rain.MM, err = getRainData(ws, storageClient)
		if err != nil {
			logger.WithError(err).Warn("unable to get rain data for WaterSchedule")
		} else {
			weatherData.Rain.ScaleFactor = ws.WeatherControl.Rain.InvertedScaleDownOnly(weatherData.Rain.MM)
		}
	}

	if ws.HasTemperatureControl() {
		logger.Debug("getting average high temperature for WaterSchedule")
		weatherData.Temperature = &TemperatureData{}
		weatherData.Temperature.Celcius, err = getTemperatureData(ws, storageClient)
		if err != nil {
			logger.WithError(err).Warn("unable to get average high temperature from weather client")
		} else {
			weatherData.Temperature.ScaleFactor = ws.WeatherControl.Temperature.Scale(weatherData.Temperature.Celcius)
		}
	}
	return weatherData
}

func getRainData(ws *pkg.WaterSchedule, storageClient storage.Client) (float32, error) {
	weatherClient, err := storageClient.GetWeatherClient(ws.WeatherControl.Rain.ClientID)
	if err != nil {
		return 0, fmt.Errorf("error getting WeatherClient for RainControl: %w", err)
	}

	totalRain, err := weatherClient.GetTotalRain(ws.Interval.Duration)
	if err != nil {
		return 0, fmt.Errorf("unable to get rain data from weather client: %w", err)
	}
	return totalRain, nil
}

func getTemperatureData(ws *pkg.WaterSchedule, storageClient storage.Client) (float32, error) {
	weatherClient, err := storageClient.GetWeatherClient(ws.WeatherControl.Temperature.ClientID)
	if err != nil {
		return 0, fmt.Errorf("error getting WeatherClient for TemperatureControl: %w", err)
	}

	avgTemperature, err := weatherClient.GetAverageHighTemperature(ws.Interval.Duration)
	if err != nil {
		return 0, fmt.Errorf("unable to get average high temperature from weather client: %w", err)
	}
	return avgTemperature, nil
}
