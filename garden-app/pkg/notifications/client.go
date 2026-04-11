package notifications

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/babyapi"
	"github.com/containrrr/shoutrrr"
	shoutrrrTypes "github.com/containrrr/shoutrrr/pkg/types"
)

const fakeScheme = "fake"

type Client struct {
	ID   babyapi.ID `json:"id" yaml:"id"`
	Name string     `json:"name" yaml:"name"`
	URL  string     `json:"url" yaml:"url"`
}

func (nc *Client) TestCreate() error {
	if nc.URL == "" {
		return errors.New("missing required url field")
	}

	if strings.HasPrefix(nc.URL, fakeScheme+"://") {
		_, err := newFakeClient(nc.URL)
		return err
	}

	_, err := shoutrrr.CreateSender(nc.URL)
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
		if nc.URL == "" {
			return errors.New("missing required url field")
		}
	}

	return nil
}

func (nc *Client) Patch(newConfig *Client) *babyapi.ErrResponse {
	if newConfig.Name != "" {
		nc.Name = newConfig.Name
	}
	if newConfig.URL != "" {
		nc.URL = newConfig.URL
	}

	return nil
}

func (*Client) EndDated() bool {
	return false
}

func (*Client) SetEndDate(_ time.Time) {}

func (nc *Client) SendMessage(title, message string) error {
	if strings.HasPrefix(nc.URL, fakeScheme+"://") {
		fc, err := newFakeClient(nc.URL)
		if err != nil {
			return fmt.Errorf("error initializing fake client: %w", err)
		}
		return fc.SendMessage(title, message)
	}

	sender, err := shoutrrr.CreateSender(nc.URL)
	if err != nil {
		return fmt.Errorf("error initializing shoutrrr sender: %w", err)
	}

	errs := sender.Send(message, &shoutrrrTypes.Params{"title": title})
	var actualErrs []error
	for _, e := range errs {
		if e != nil {
			actualErrs = append(actualErrs, e)
		}
	}
	if len(actualErrs) > 0 {
		return fmt.Errorf("error sending notification: %v", actualErrs)
	}

	return nil
}

func newFakeClient(urlStr string) (*fake.Client, error) {
	options, err := parseFakeURL(urlStr)
	if err != nil {
		return nil, err
	}

	return fake.NewClient(options)
}

func parseFakeURL(urlStr string) (map[string]any, error) {
	if !strings.HasPrefix(urlStr, fakeScheme+"://") {
		return nil, errors.New("invalid fake URL scheme")
	}

	result := make(map[string]any)
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing fake URL: %w", err)
	}

	for key, values := range u.Query() {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}

	return result, nil
}
