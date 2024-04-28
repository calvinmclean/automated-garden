package pushover

import (
	"errors"

	"github.com/gregdel/pushover"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	AppToken       string `json:"app_token,omitempty" yaml:"app_token,omitempty" mapstructure:"app_token,omitempty"`
	RecipientToken string `json:"recipient_token,omitempty" yaml:"recipient_token,omitempty" mapstructure:"recipient_token,omitempty"`
}

type Client struct {
	*Config
	app       *pushover.Pushover
	recipient *pushover.Recipient
}

func NewClient(options map[string]interface{}) (*Client, error) {
	client := &Client{}

	err := mapstructure.Decode(options, &client.Config)
	if err != nil {
		return nil, err
	}

	if client.AppToken == "" {
		return nil, errors.New("missing required app_token")
	}
	if client.RecipientToken == "" {
		return nil, errors.New("missing required recipient_token")
	}

	client.app = pushover.New(client.AppToken)
	client.recipient = pushover.NewRecipient(client.RecipientToken)

	return client, nil
}

func (c *Client) SendMessage(title, message string) error {
	msg := pushover.NewMessageWithTitle(message, title)
	_, err := c.app.SendMessage(msg, c.recipient)
	return err
}
