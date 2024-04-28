package notifications

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/pushover"

	"github.com/calvinmclean/babyapi"
)

// Client is an interface defining the possible methods used to interact with the notification APIs
type Client interface {
	SendMessage(title, message string) error
}

// Config is used to identify and configure a client type
type Config struct {
	ID      babyapi.ID             `json:"id" yaml:"id"`
	Type    string                 `json:"type" yaml:"type"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

func (nc *Config) GetID() string {
	return nc.ID.String()
}

func (nc *Config) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (nc *Config) Bind(r *http.Request) error {
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

// NewClient will use the config to create and return the correct type of notification client
func NewClient(c *Config) (Client, error) {
	var client Client
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

// Patch allows modifying an existing Config with fields from a new one
func (nc *Config) Patch(newConfig *Config) *babyapi.ErrResponse {
	if newConfig.Type != "" {
		nc.Type = newConfig.Type
	}

	if nc.Options == nil && newConfig.Options != nil {
		nc.Options = map[string]interface{}{}
	}
	for k, v := range newConfig.Options {
		nc.Options[k] = v
	}

	return nil
}

// EndDated allows this to satisfy an interface even though the resources does not have end-dates
func (*Config) EndDated() bool {
	return false
}

func (*Config) SetEndDate(_ time.Time) {}
