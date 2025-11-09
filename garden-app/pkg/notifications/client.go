package notifications

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/pushover"
	"github.com/calvinmclean/babyapi"
)

// Client is used to interact with an external notification API. It has generic options to allow multiple Client implementations
type Client struct {
	ID      babyapi.ID     `json:"id" yaml:"id"`
	Name    string         `json:"name" yaml:"name"`
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

func (nc *Client) ParentID() string {
	return ""
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
		if nc.Name == "" {
			return errors.New("missing required name field")
		}
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
	if newConfig.Name != "" {
		nc.Name = newConfig.Name
	}
	if newConfig.Type != "" {
		nc.Type = newConfig.Type
	}

	if nc.Options == nil && newConfig.Options != nil {
		nc.Options = map[string]any{}
	}
	maps.Copy(nc.Options, newConfig.Options)

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

// client is an interface defining the possible methods used to interact with the notification APIs
type client interface {
	SendMessage(title, message string) error
}

// newClient will use the config to create and return the correct type of notification client
func newClient(c *Client) (client, error) {
	var client client
	var err error
	switch c.Type {
	case "pushover":
		client, err = pushover.NewClient(c.Options)
	case "fake":
		client, err = fake.NewClient(c.Options)
	default:
		err = fmt.Errorf("invalid type '%s'", c.Type)
	}

	return client, err
}
