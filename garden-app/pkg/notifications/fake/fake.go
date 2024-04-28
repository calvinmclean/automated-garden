package fake

import (
	"errors"

	"github.com/mitchellh/mapstructure"
)

type Config struct {
	CreateError      string `mapstructure:"create_error"`
	SendMessageError string `mapstructure:"send_message_error"`
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

	if client.Config.CreateError != "" {
		return nil, errors.New(client.CreateError)
	}

	return client, nil
}

func (c *Client) SendMessage(string, string) error {
	if c.SendMessageError != "" {
		return errors.New(c.SendMessageError)
	}
	return nil
}
