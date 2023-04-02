package configmap

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// GetWeatherClient reads the config from storage and then returns the actual Client to reduce code duplication in the
// worker code that calls it
func (c *Client) GetWeatherClient(id xid.ID) (weather.Client, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}

	// TODO: Do i need nil check? write test for it
	return weather.NewClient(c.data.WeatherClientConfigs[id], func(weatherClientOptions map[string]interface{}) error {
		// storageCallback will update the options with the value in the input and then save
		c.m.Lock()
		defer c.m.Unlock()

		c.data.WeatherClientConfigs[id].Options = weatherClientOptions
		return c.save()
	})
}

// GetWeatherClientConfig returns the WeatherClient's configuration value for the provided ID
func (c *Client) GetWeatherClientConfig(id xid.ID) (*weather.Config, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.WeatherClientConfigs[id], nil
}

// GetWeatherClientConfigs returns all of the WeatherClient configurations
func (c *Client) GetWeatherClientConfigs() ([]*weather.Config, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*weather.Config{}
	for _, wc := range c.data.WeatherClientConfigs {
		result = append(result, wc)
	}
	return result, nil
}

// SaveWeatherClientConfig saves the config
func (c *Client) SaveWeatherClientConfig(wc *weather.Config) error {
	c.m.Lock()
	defer c.m.Unlock()

	c.data.WeatherClientConfigs[wc.ID] = wc
	return c.save()
}

// DeleteWeatherClientConfig deletes the config
func (c *Client) DeleteWeatherClientConfig(id xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.WeatherClientConfigs, id)
	return c.save()
}
