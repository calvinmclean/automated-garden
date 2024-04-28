package notifications

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/babyapi"
)

// Client is used to interact with an external notification API. It has generic options to allow multiple Client implementations
type Client struct {
	ID      babyapi.ID     `json:"id" yaml:"id"`
	Type    string         `json:"type" yaml:"type"`
	Options map[string]any `json:"options" yaml:"options"`
}

// TestCreate will call the Client implementation's initialization function to make sure it is valid and able to connect
func (nc *Client) TestCreate() error {
	_, err := newClient(nc)
	return err
}

func (nc *Client) GetID() string {
	return nc.ID.String()
}

func (nc *Client) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (nc *Client) Bind(r *http.Request) error {
	if nc == nil {
		return errors.New("missing required NotificationClient fields")
	}

	err := nc.ID.Bind(r)
	if err != nil {
		return err
	}

	switch r.Method {
	case http.MethodPut, http.MethodPost:
		if nc.Type == "" {
			return errors.New("missing required type field")
		}
		if nc.Options == nil {
			return errors.New("missing required options field")
		}
	}

	return nil
}

// Patch allows modifying an existing Config with fields from a new one
func (nc *Client) Patch(newConfig *Client) *babyapi.ErrResponse {
	if newConfig.Type != "" {
		nc.Type = newConfig.Type
	}

	if nc.Options == nil && newConfig.Options != nil {
		nc.Options = map[string]any{}
	}
	for k, v := range newConfig.Options {
		nc.Options[k] = v
	}

	return nil
}

// EndDated allows this to satisfy an interface even though the resources does not have end-dates
func (*Client) EndDated() bool {
	return false
}

func (*Client) SetEndDate(_ time.Time) {}

// SendMessage will send a notification using the client created from this config
func (nc *Client) SendMessage(title, message string) error {
	client, err := newClient(nc)
	if err != nil {
		return fmt.Errorf("error initializing client: %w", err)
	}

	return client.SendMessage(title, message)
}
