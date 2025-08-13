package storage

import (
	"context"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
)

// GetWeatherClient ...
func (c *Client) GetWeatherClient(id xid.ID) (weather.Client, error) {
	clientConfig, err := c.WeatherClientConfigs.Get(context.Background(), id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting weather client config: %w", err)
	}

	if clientConfig == nil {
		return nil, fmt.Errorf("weather client config not found")
	}

	return weather.NewClient(clientConfig, func(weatherClientOptions map[string]interface{}) error {
		clientConfig.Options = weatherClientOptions
		return c.WeatherClientConfigs.Set(context.Background(), clientConfig)
	})
}

// GetWaterSchedulesUsingWeatherClient will return all WaterSchedules that rely on this WeatherClient
func (c *Client) GetWaterSchedulesUsingWeatherClient(id string) ([]*pkg.WaterSchedule, error) {
	waterSchedules, err := c.WaterSchedules.Search(context.Background(), "", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get all WaterSchedules: %w", err)
	}
	waterSchedules = babyapi.FilterFunc[*pkg.WaterSchedule](func(ws *pkg.WaterSchedule) bool {
		if !ws.HasWeatherControl() {
			return false
		}
		if ws.HasRainControl() && ws.WeatherControl.Rain.ClientID.String() == id {
			return true
		}
		if ws.HasTemperatureControl() && ws.WeatherControl.Temperature.ClientID.String() == id {
			return true
		}
		return false
	}).Filter(waterSchedules)

	return waterSchedules, nil
}
