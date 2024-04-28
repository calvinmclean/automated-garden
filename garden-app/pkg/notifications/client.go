package notifications

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/pushover"
)

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
