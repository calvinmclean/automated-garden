package storage

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
)

// YAMLClient implements the Client interface to use a YAML file as a storage mechanism
type YAMLClient struct {
	gardens  map[xid.ID]*pkg.Garden
	filename string
	Config   Config
}

// NewYAMLClient will read the plants from the file and store them in a map
func NewYAMLClient(config Config) (*YAMLClient, error) {
	if _, ok := config.Options["filename"]; !ok {
		return nil, fmt.Errorf("missing config key 'filename'")
	}
	client := &YAMLClient{
		gardens:  map[xid.ID]*pkg.Garden{},
		filename: config.Options["filename"],
		Config:   config,
	}

	// If file does not exist, that is fine and we will just have an empty map
	_, err := os.Stat(client.Config.Options["filename"])
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

func (c *YAMLClient) update() error {
	data, err := ioutil.ReadFile(c.filename)
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
func (c *YAMLClient) GetGarden(id xid.ID) (*pkg.Garden, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[id], nil
}

// GetGardens returns all the gardens
func (c *YAMLClient) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
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
func (c *YAMLClient) SaveGarden(garden *pkg.Garden) error {
	c.gardens[garden.ID] = garden
	return c.Save()
}

// DeleteGarden permanently deletes a garden and removes it from the YAML file
func (c *YAMLClient) DeleteGarden(garden xid.ID) error {
	delete(c.gardens, garden)
	return c.Save()
}

// GetPlant just returns the request Plant from the map
func (c *YAMLClient) GetPlant(garden xid.ID, id xid.ID) (*pkg.Plant, error) {
	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.gardens[garden].Plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *YAMLClient) GetPlants(garden xid.ID, getEndDated bool) ([]*pkg.Plant, error) {
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
func (c *YAMLClient) SavePlant(plant *pkg.Plant) error {
	if c.gardens[plant.GardenID].Plants == nil {
		c.gardens[plant.GardenID].Plants = map[xid.ID]*pkg.Plant{}
	}
	c.gardens[plant.GardenID].Plants[plant.ID] = plant
	return c.Save()
}

// DeletePlant permanently deletes a plant and removes it from the YAML file
func (c *YAMLClient) DeletePlant(garden xid.ID, plant xid.ID) error {
	delete(c.gardens[garden].Plants, plant)
	return c.Save()
}

// Save saves the client's data back to a persistent source
func (c *YAMLClient) Save() error {
	// Marshal map to YAML bytes
	content, err := yaml.Marshal(c.gardens)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.filename, content, 0755)
}
