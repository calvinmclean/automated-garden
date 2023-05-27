package kv

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/madflojo/hord"
	"github.com/rs/xid"
)

const weatherclientPrefix = "WeatherClient_"

// GetWeatherClient ...
func (c *Client) GetWeatherClient(id xid.ID) (weather.Client, error) {
	clientConfig, err := c.getWeatherClientConfig(weatherclientPrefix + id.String())
	if err != nil {
		return nil, fmt.Errorf("error getting weather client config: %w", err)
	}

	return weather.NewClient(clientConfig, func(weatherClientOptions map[string]interface{}) error {
		clientConfig.Options = weatherClientOptions
		return c.SaveWeatherClientConfig(clientConfig)
	})
}

// GetWeatherClientConfig ...
func (c *Client) GetWeatherClientConfig(id xid.ID) (*weather.Config, error) {
	return c.getWeatherClientConfig(weatherclientPrefix + id.String())
}

// GetWeatherClientConfigs ...
func (c *Client) GetWeatherClientConfigs() ([]*weather.Config, error) {
	keys, err := c.db.Keys()
	if err != nil {
		return nil, fmt.Errorf("error getting keys: %w", err)
	}

	results := []*weather.Config{}
	for _, key := range keys {
		if !strings.HasPrefix(key, weatherclientPrefix) {
			continue
		}

		result, err := c.getWeatherClientConfig(key)
		if err != nil {
			return nil, fmt.Errorf("error getting keys: %w", err)
		}

		results = append(results, result)
	}

	return results, nil
}

// SaveWeatherClientConfig ...
func (c *Client) SaveWeatherClientConfig(wc *weather.Config) error {
	asBytes, err := c.marshal(wc)
	if err != nil {
		return fmt.Errorf("error marshalling WeatherClient: %w", err)
	}

	err = c.db.Set(weatherclientPrefix+wc.ID.String(), asBytes)
	if err != nil {
		return fmt.Errorf("error writing WeatherClient to database: %w", err)
	}

	return nil
}

// DeleteWeatherClientConfig ...
func (c *Client) DeleteWeatherClientConfig(id xid.ID) error {
	return c.db.Delete(weatherclientPrefix + id.String())
}

func (c *Client) getWeatherClientConfig(key string) (*weather.Config, error) {
	dataBytes, err := c.db.Get(key)
	if err != nil {
		if errors.Is(hord.ErrNil, err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting WeatherClient: %w", err)
	}

	var result weather.Config
	err = c.unmarshal(dataBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing WeatherClient data: %w", err)
	}

	return &result, nil
}
