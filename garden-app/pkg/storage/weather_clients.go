package storage

import (
	"context"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// GetWeatherClient retrieves a WeatherClient by ID and initializes it
func (c *Client) GetWeatherClient(id xid.ID) (weather.Client, error) {
	clientConfig, err := c.WeatherClientConfigs.Get(context.Background(), id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting weather client config: %w", err)
	}

	if clientConfig == nil {
		return nil, fmt.Errorf("weather client config not found")
	}

	return weather.NewClient(clientConfig, func(weatherClientOptions map[string]any) error {
		clientConfig.Options = weatherClientOptions
		return c.WeatherClientConfigs.Set(context.Background(), clientConfig)
	})
}
