package yaml

import (
	"fmt"
	"os"
	"sync"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
)

// Client implements the Client interface to use a YAML file as a storage mechanism
type Client struct {
	data     clientData
	filename string
	Options  map[string]string

	m *sync.Mutex
}

type clientData struct {
	Gardens        map[xid.ID]*pkg.Garden     `yaml:"gardens"`
	WeatherClients map[xid.ID]*weather.Config `yaml:"weather_clients"`
}

// NewClient will read the plants from the file and store them in a map
func NewClient(options map[string]string) (*Client, error) {
	if _, ok := options["filename"]; !ok {
		return nil, fmt.Errorf("missing config key 'filename'")
	}
	client := &Client{
		data: clientData{
			Gardens:        map[xid.ID]*pkg.Garden{},
			WeatherClients: map[xid.ID]*weather.Config{},
		},
		filename: options["filename"],
		Options:  options,
		m:        &sync.Mutex{},
	}

	// If file does not exist, that is fine and we will just have an empty map
	_, err := os.Stat(client.filename)
	if os.IsNotExist(err) {
		return client, nil
	}

	// If file exists, continue by reading its contents to the map
	err = client.update()
	if err != nil {
		return client, err
	}

	return client, err
}

// save saves the client's data back to a persistent source. This is unexported and should only be used when a RWLock is already acquired
func (c *Client) save() error {
	// Marshal map to YAML bytes
	content, err := yaml.Marshal(c.data)
	if err != nil {
		return err
	}

	return os.WriteFile(c.filename, content, 0755)
}

// update will refresh from the file in case something was changed externally. Although it is mostly used prior to reads, it
// still modifies the map and should only be used while an RWLock is acquired
func (c *Client) update() error {
	data, err := os.ReadFile(c.filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &c.data)
	if err != nil {
		return err
	}
	return nil
}
