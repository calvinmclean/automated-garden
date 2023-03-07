package yaml

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

func (c *Client) GetWeatherClient(id xid.ID) (*weather.Config, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.WeatherClients[id], nil
}

func (c *Client) GetWeatherClients(getEndDated bool) ([]*weather.Config, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*weather.Config{}
	for _, wc := range c.data.WeatherClients {
		if getEndDated || !wc.EndDated() {
			result = append(result, wc)
		}
	}
	return result, nil
}

func (c *Client) SaveWeatherClient(wc *weather.Config) error {
	c.m.Lock()
	defer c.m.Unlock()

	c.data.WeatherClients[wc.ID] = wc
	return c.save()
}

func (c *Client) DeleteWeatherClient(id xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.WeatherClients, id)
	return c.save()
}
