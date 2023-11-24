package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// GetWeatherClient ...
func (c *Client) GetWeatherClient(id xid.ID) (weather.Client, error) {
	clientConfig, err := c.WeatherClientConfigs.Get(id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting weather client config: %w", err)
	}

	if clientConfig == nil {
		return nil, fmt.Errorf("weather client config not found")
	}

	return weather.NewClient(clientConfig, func(weatherClientOptions map[string]interface{}) error {
		clientConfig.Options = weatherClientOptions
		return c.WeatherClientConfigs.Set(clientConfig)
	})
}

// GetWaterSchedulesUsingWeatherClient will return all WaterSchedules that rely on this WeatherClient
func (c *Client) GetWaterSchedulesUsingWeatherClient(id string) ([]*pkg.WaterSchedule, error) {
	waterSchedules, err := c.WaterSchedules.GetAll(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get all WaterSchedules: %w", err)
	}

	results := []*pkg.WaterSchedule{}
	for _, ws := range waterSchedules {
		if ws.HasWeatherControl() {
			if ws.HasRainControl() {
				if ws.WeatherControl.Rain.ClientID.String() == id {
					results = append(results, ws)
				}
			}
			if ws.HasTemperatureControl() {
				if ws.WeatherControl.Temperature.ClientID.String() == id {
					results = append(results, ws)
				}
			}
		}
	}

	return results, nil
}
