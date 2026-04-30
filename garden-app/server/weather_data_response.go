package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/concurrent"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
)

const (
	// weatherAPITimeout is the timeout for external weather API calls.
	weatherAPITimeout = 5 * time.Second
)

// WeatherData is used to represent the data used for WeatherControl to a user
type WeatherData struct {
	Rain               *RainData               `json:"rain,omitempty"`
	Temperature        *TemperatureData        `json:"temperature,omitempty"`
	Evapotranspiration *EvapotranspirationData `json:"evapotranspiration,omitempty"`
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

// EvapotranspirationData shows the average daily evapotranspiration in both metric and imperial units
type EvapotranspirationData struct {
	MM     *float32 `json:"mm,omitempty"`
	Inches *float32 `json:"inches,omitempty"`
}

func getWeatherData(ctx context.Context, ws *pkg.WaterSchedule, storageClient *storage.Client, logger *slog.Logger) *WeatherData {
	weatherData := &WeatherData{}

	// Prepare tasks for concurrent rain and temperature data fetching
	tasks := []concurrent.TaskFunc{
		{
			Name: "rain-data",
			Fn: func(taskCtx context.Context) error {
				if !ws.HasRainControl() {
					return nil
				}
				logger.Debug("getting rain data for WaterSchedule")
				rainMM, err := getRainData(taskCtx, ws, storageClient)
				if err != nil || rainMM == nil {
					logger.Warn("unable to get rain data for WaterSchedule", "error", err)
					return err
				}
				inches := *rainMM * 0.0393701
				weatherData.Rain = &RainData{
					MM:     rainMM,
					Inches: &inches,
				}
				return nil
			},
		},
		{
			Name: "temperature-data",
			Fn: func(taskCtx context.Context) error {
				if !ws.HasTemperatureControl() {
					return nil
				}
				logger.Debug("getting average high temperature for WaterSchedule")
				celsius, err := getTemperatureData(taskCtx, ws, storageClient)
				if err != nil || celsius == nil {
					logger.Warn("unable to get average high temperature from weather client", "error", err)
					return err
				}
				weatherData.Temperature = &TemperatureData{
					Celsius:    *celsius,
					Fahrenheit: *celsius*1.8 + 32,
				}
				return nil
			},
		},
		{
			Name: "evapotranspiration-data",
			Fn: func(taskCtx context.Context) error {
				// Only fetch ET if temperature control is enabled (to get the weather client)
				if !ws.HasTemperatureControl() {
					return nil
				}
				logger.Debug("getting evapotranspiration data for WaterSchedule")
				etMM, err := getEvapotranspirationData(taskCtx, ws, storageClient)
				if err != nil || etMM == nil {
					// ET is optional, so don't warn on error
					return nil
				}
				inches := *etMM * 0.0393701
				weatherData.Evapotranspiration = &EvapotranspirationData{
					MM:     etMM,
					Inches: &inches,
				}
				return nil
			},
		},
	}

	// Execute weather data fetching concurrently with timeout
	errors := concurrent.RunFuncs(ctx, weatherAPITimeout, tasks)
	for taskName, err := range errors {
		logger.Warn("weather data task failed", "task", taskName, "error", err)
	}

	return weatherData
}

func getRainData(ctx context.Context, ws *pkg.WaterSchedule, storageClient *storage.Client) (*float32, error) {
	weatherClient, err := storageClient.GetWeatherClient(ws.WeatherControl.Rain.ClientID)
	if err != nil {
		return nil, fmt.Errorf("error getting WeatherClient for RainControl: %w", err)
	}

	totalRain, err := weatherClient.GetTotalRain(ctx, ws.Interval.Duration)
	if err != nil {
		return nil, fmt.Errorf("unable to get rain data from weather client: %w", err)
	}
	return &totalRain, nil
}

func getTemperatureData(ctx context.Context, ws *pkg.WaterSchedule, storageClient *storage.Client) (*float32, error) {
	weatherClient, err := storageClient.GetWeatherClient(ws.WeatherControl.Temperature.ClientID)
	if err != nil {
		return nil, fmt.Errorf("error getting WeatherClient for TemperatureControl: %w", err)
	}

	avgTemperature, err := weatherClient.GetAverageHighTemperature(ctx, ws.Interval.Duration)
	if err != nil {
		return nil, fmt.Errorf("unable to get average high temperature from weather client: %w", err)
	}
	return &avgTemperature, nil
}

func getEvapotranspirationData(ctx context.Context, ws *pkg.WaterSchedule, storageClient *storage.Client) (*float32, error) {
	weatherClient, err := storageClient.GetWeatherClient(ws.WeatherControl.Temperature.ClientID)
	if err != nil {
		return nil, fmt.Errorf("error getting WeatherClient for ET data: %w", err)
	}

	// Check if the client supports evapotranspiration
	etClient, ok := weatherClient.(weather.ETProvider)
	if !ok {
		return nil, nil // Client doesn't support ET, return nil without error
	}

	avgET, err := etClient.GetAverageEvapotranspiration(ctx, ws.Interval.Duration)
	if err != nil {
		return nil, fmt.Errorf("unable to get evapotranspiration data from weather client: %w", err)
	}
	return &avgET, nil
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
