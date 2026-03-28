package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
)

// WeatherData is used to represent the data used for WeatherControl to a user
type WeatherData struct {
	Rain        *RainData        `json:"rain,omitempty"`
	Temperature *TemperatureData `json:"temperature,omitempty"`
}

// RainData shows the total rain in both metric and imperial units
type RainData struct {
	MM     *float32 `json:"mm,omitempty"`
	Inches *float32 `json:"inches,omitempty"`
}

// TemperatureData shows the average high temperatures in both Celsius and Fahrenheit
type TemperatureData struct {
	Celsius    float32 `json:"celsius,omitempty"`
	Fahrenheit float32 `json:"fahrenheit,omitempty"`
}

func getWeatherData(ctx context.Context, ws *pkg.WaterSchedule, storageClient *storage.Client) *WeatherData {
	logger := babyapi.GetLoggerFromContext(ctx).With(waterScheduleIDLogField, ws.ID.String())
	weatherData := &WeatherData{}

	if ws.HasRainControl() {
		logger.Debug("getting rain data for WaterSchedule")
		rainMM, err := getRainData(ws, storageClient)
		if err != nil || rainMM == nil {
			logger.Warn("unable to get rain data for WaterSchedule", "error", err)
		} else {
			inches := *rainMM * 0.0393701
			weatherData.Rain = &RainData{
				MM:     rainMM,
				Inches: &inches,
			}
		}
	}

	if ws.HasTemperatureControl() {
		logger.Debug("getting average high temperature for WaterSchedule")
		celsius, err := getTemperatureData(ws, storageClient)
		if err != nil || celsius == nil {
			logger.Warn("unable to get average high temperature from weather client", "error", err)
		} else {
			weatherData.Temperature = &TemperatureData{
				Celsius:    *celsius,
				Fahrenheit: *celsius*1.8 + 32,
			}
		}
	}
	return weatherData
}

func getRainData(ws *pkg.WaterSchedule, storageClient *storage.Client) (*float32, error) {
	weatherClient, err := storageClient.GetWeatherClient(ws.WeatherControl.Rain.ClientID)
	if err != nil {
		return nil, fmt.Errorf("error getting WeatherClient for RainControl: %w", err)
	}

	totalRain, err := weatherClient.GetTotalRain(ws.Interval.Duration)
	if err != nil {
		return nil, fmt.Errorf("unable to get rain data from weather client: %w", err)
	}
	return &totalRain, nil
}

func getTemperatureData(ws *pkg.WaterSchedule, storageClient *storage.Client) (*float32, error) {
	weatherClient, err := storageClient.GetWeatherClient(ws.WeatherControl.Temperature.ClientID)
	if err != nil {
		return nil, fmt.Errorf("error getting WeatherClient for TemperatureControl: %w", err)
	}

	avgTemperature, err := weatherClient.GetAverageHighTemperature(ws.Interval.Duration)
	if err != nil {
		return nil, fmt.Errorf("unable to get average high temperature from weather client: %w", err)
	}
	return &avgTemperature, nil
}

func getUnitsFromRequest(r *http.Request) string {
	units := r.URL.Query().Get("units")
	if units != "imperial" {
		return "metric"
	}
	return units
}

func getDurationFromRequest(r *http.Request) time.Duration {
	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		return 72 * time.Hour
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return 72 * time.Hour
	}
	return duration
}
