package fake

import (
	"errors"

	"github.com/mitchellh/mapstructure"
)

type Config struct {
	Error string `mapstructure:"error"`
}

type Client struct {
	*Config
}

func NewClient(options map[string]interface{}) (*Client, error) {
	client := &Client{}

	err := mapstructure.Decode(options, &client.Config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) SendMessage(string, string) error {
	if c.Error != "" {
		return errors.New(c.Error)
	}
	return nil
}
