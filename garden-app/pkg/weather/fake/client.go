// Package fake provides a fake weather client for testing
package fake

import (
	"context"
	"errors"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Config is specific to the Fake API and holds all of the necessary fields for configuring fake data responses.
type Config struct {
	RainMM       float32 `mapstructure:"rain_mm"`
	RainInterval string  `mapstructure:"rain_interval"`
	rainInterval time.Duration

	AverageHighTemperature float32 `mapstructure:"avg_high_temperature"`

	Error      string `mapstructure:"error"`
	ErrorCount int    `mapstructure:"error_count"`

	errorCalls int
}

// Client ...
type Client struct {
	*Config
}

// NewClient creates a new client that will return fake data based on configuration.
// This is intended for testing purposes only and should be used in a staging environment
// or integration tests, not as a mock in unit tests
func NewClient(options map[string]any) (*Client, error) {
	client := &Client{}

	err := mapstructure.WeakDecode(options, &client.Config)
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
func (c *Client) GetTotalRain(_ context.Context, since time.Duration) (float32, error) {
	if c.shouldError() {
		return 0, errors.New(c.Error)
	}

	numIntervals := float32(since.Hours() / c.rainInterval.Hours())
	return numIntervals * c.RainMM, nil
}

// GetAverageHighTemperature returns the configured value
func (c *Client) GetAverageHighTemperature(_ context.Context, _ time.Duration) (float32, error) {
	if c.shouldError() {
		return 0, errors.New(c.Error)
	}

	return c.AverageHighTemperature, nil
}

// shouldError returns true if the fake client should return an error for this call.
// When ErrorCount is greater than zero, it returns true for the first ErrorCount
// calls and then succeeds. When ErrorCount is zero or negative, it returns true for
// every call if Error is set.
func (c *Client) shouldError() bool {
	if c.Error == "" {
		return false
	}
	if c.ErrorCount > 0 {
		if c.errorCalls < c.ErrorCount {
			c.errorCalls++
			return true
		}
		return false
	}
	return true
}
