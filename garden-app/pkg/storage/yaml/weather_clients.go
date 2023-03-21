package yaml

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
	return weather.NewClient(c.data.WeatherClientConfigs[id])
}

func (c *Client) GetWeatherClientConfig(id xid.ID) (*weather.Config, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.WeatherClientConfigs[id], nil
}

func (c *Client) GetWeatherClientConfigs(getEndDated bool) ([]*weather.Config, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*weather.Config{}
	for _, wc := range c.data.WeatherClientConfigs {
		if getEndDated || !wc.EndDated() {
			result = append(result, wc)
		}
	}
	return result, nil
}

func (c *Client) SaveWeatherClientConfig(wc *weather.Config) error {
	c.m.Lock()
	defer c.m.Unlock()

	c.data.WeatherClientConfigs[wc.ID] = wc
	return c.save()
}

func (c *Client) DeleteWeatherClientConfig(id xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.WeatherClientConfigs, id)
	return c.save()
}
