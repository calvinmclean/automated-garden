package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

const weatherClientPrefix = "WeatherClient_"

func weatherClientKey(id xid.ID) string {
	return weatherClientPrefix + id.String()
}
func WeatherClientKey(id string) string {
	return weatherClientPrefix + id
}

// GetWeatherClient ...
func (c *Client) GetWeatherClient(id xid.ID) (weather.Client, error) {
	clientConfig, err := GetOne[weather.Config](c, weatherClientKey(id))
	if err != nil {
		return nil, fmt.Errorf("error getting weather client config: %w", err)
	}

	if clientConfig == nil {
		return nil, fmt.Errorf("weather client config not found")
	}

	return weather.NewClient(clientConfig, func(weatherClientOptions map[string]interface{}) error {
		clientConfig.Options = weatherClientOptions
		return c.SaveWeatherClientConfig(clientConfig)
	})
}

// GetWeatherClientConfig ...
func (c *Client) GetWeatherClientConfig(id xid.ID) (*weather.Config, error) {
	return GetOne[weather.Config](c, weatherClientKey(id))
}

// GetWeatherClientConfigs ...
func (c *Client) GetWeatherClientConfigs() ([]*weather.Config, error) {
	return GetMultiple[*weather.Config](c, true, weatherClientPrefix)
}

// SaveWeatherClientConfig ...
func (c *Client) SaveWeatherClientConfig(wc *weather.Config) error {
	return Save[*weather.Config](c, wc, weatherClientKey(wc.ID))
}

// DeleteWeatherClientConfig ...
func (c *Client) DeleteWeatherClientConfig(id xid.ID) error {
	return c.db.Delete(weatherClientKey(id))
}

// GetWaterSchedulesUsingWeatherClient will return all WaterSchedules that rely on this WeatherClient
func (c *Client) GetWaterSchedulesUsingWeatherClient(id string) ([]*pkg.WaterSchedule, error) {
	waterSchedules, err := c.GetWaterSchedules(false)
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
