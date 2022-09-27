package yaml

import (
	"fmt"
	"os"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
)

// Client implements the Client interface to use a YAML file as a storage mechanism
type Client struct {
	gardens  map[xid.ID]*pkg.Garden
	filename string
	Options  map[string]string
}

// NewClient will read the plants from the file and store them in a map
func NewClient(options map[string]string) (*Client, error) {
	if _, ok := options["filename"]; !ok {
		return nil, fmt.Errorf("missing config key 'filename'")
	}
	client := &Client{
		gardens:  map[xid.ID]*pkg.Garden{},
		filename: options["filename"],
		Options:  options,
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

func (c *Client) update() error {
	data, err := os.ReadFile(c.filename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &c.gardens)
	if err != nil {
		return err
	}
	return nil
}

// GetGarden returns the garden
func (c *Client) GetGarden(id xid.ID) (*pkg.Garden, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[id], nil
}

// GetGardens returns all the gardens
func (c *Client) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Garden{}
	for _, g := range c.gardens {
		if getEndDated || !g.EndDated() {
			result = append(result, g)
		}
	}
	return result, nil
}

// SaveGarden saves a garden and writes it back to the YAML file
func (c *Client) SaveGarden(garden *pkg.Garden) error {
	c.gardens[garden.ID] = garden
	return c.Save()
}

// DeleteGarden permanently deletes a garden and removes it from the YAML file
func (c *Client) DeleteGarden(garden xid.ID) error {
	delete(c.gardens, garden)
	return c.Save()
}

// GetZone just returns the request Zone from the map
func (c *Client) GetZone(garden xid.ID, id xid.ID) (*pkg.Zone, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[garden].Zones[id], nil
}

// GetZones returns all zones from the map as a slice
func (c *Client) GetZones(garden xid.ID, getEndDated bool) ([]*pkg.Zone, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Zone{}
	for _, p := range c.gardens[garden].Zones {
		if getEndDated || !p.EndDated() {
			result = append(result, p)
		}
	}
	return result, nil
}

// SaveZone saves a zone in the map and will write it back to the YAML file
func (c *Client) SaveZone(gardenID xid.ID, zone *pkg.Zone) error {
	if c.gardens[gardenID].Zones == nil {
		c.gardens[gardenID].Zones = map[xid.ID]*pkg.Zone{}
	}
	c.gardens[gardenID].Zones[zone.ID] = zone
	return c.Save()
}

// DeleteZone permanently deletes a zone and removes it from the YAML file
func (c *Client) DeleteZone(garden xid.ID, zone xid.ID) error {
	delete(c.gardens[garden].Zones, zone)
	return c.Save()
}

// GetPlant just returns the request Plant from the map
func (c *Client) GetPlant(garden xid.ID, id xid.ID) (*pkg.Plant, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[garden].Plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *Client) GetPlants(garden xid.ID, getEndDated bool) ([]*pkg.Plant, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Plant{}
	for _, p := range c.gardens[garden].Plants {
		if getEndDated || !p.EndDated() {
			result = append(result, p)
		}
	}
	return result, nil
}

// SavePlant saves a plant in the map and will write it back to the YAML file
func (c *Client) SavePlant(gardenID xid.ID, plant *pkg.Plant) error {
	if c.gardens[gardenID].Plants == nil {
		c.gardens[gardenID].Plants = map[xid.ID]*pkg.Plant{}
	}
	c.gardens[gardenID].Plants[plant.ID] = plant
	return c.Save()
}

// DeletePlant permanently deletes a plant and removes it from the YAML file
func (c *Client) DeletePlant(garden xid.ID, plant xid.ID) error {
	delete(c.gardens[garden].Plants, plant)
	return c.Save()
}

// Save saves the client's data back to a persistent source
func (c *Client) Save() error {
	// Marshal map to YAML bytes
	content, err := yaml.Marshal(c.gardens)
	if err != nil {
		return err
	}

	return os.WriteFile(c.filename, content, 0755)
}
