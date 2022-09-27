package fake

import (
	"time"

	"github.com/mitchellh/mapstructure"
)

// Config is specific to the Fake API and holds all of the necessary fields for configuring fake data responses.
type Config struct {
	RainMM       float32 `mapstructure:"rain_mm"`
	RainInterval string  `mapstructure:"rain_interval"`
	rainInterval time.Duration
}

// Client ...
type Client struct {
	*Config
}

// NewClient creates a new client that will return fake data based on configuration. This is intended for testing purposes only
func NewClient(options map[string]interface{}) (*Client, error) {
	client := &Client{}

	err := mapstructure.Decode(options, &client.Config)
	if err != nil {
		return nil, err
	}

	client.rainInterval, err = time.ParseDuration(client.RainInterval)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetTotalRain calculates and returns the configured amount of rain for the given period
func (c *Client) GetTotalRain(since time.Duration) (float32, error) {
	numIntervals := float32(since.Hours() / c.rainInterval.Hours())
	return numIntervals * c.RainMM, nil
}
