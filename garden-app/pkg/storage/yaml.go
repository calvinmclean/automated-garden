package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

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
	data, err := ioutil.ReadFile(client.filename)
	if err != nil {
		return client, err
	}
	err = yaml.Unmarshal(data, &client.gardens)
	if err != nil {
		return client, err
	}

	// Create start dates for Gardens and Plants if it is empty
	for _, garden := range client.gardens {
		now := time.Now().Add(1 * time.Minute)
		if garden.CreatedAt == nil {
			garden.CreatedAt = &now
			client.Save()
		}
		for _, plant := range garden.Plants {
			if plant.CreatedAt == nil {
				plant.CreatedAt = &now
				client.SavePlant(garden.ID, plant)
			}
		}
	}

	return client, err
}

// GetGarden returns the garden
func (c *YAMLClient) GetGarden(id xid.ID) (*pkg.Garden, error) {
	return c.gardens[id], nil
}

// GetGardens returns all the gardens
func (c *YAMLClient) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	result := []*pkg.Garden{}
	for _, g := range c.gardens {
		if getEndDated || (!getEndDated && g.EndDate == nil) {
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

// GetPlant just returns the request Plant from the map
func (c *YAMLClient) GetPlant(garden xid.ID, id xid.ID) (*pkg.Plant, error) {
	return c.gardens[garden].Plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *YAMLClient) GetPlants(garden xid.ID, getEndDated bool) ([]*pkg.Plant, error) {
	result := []*pkg.Plant{}
	for _, p := range c.gardens[garden].Plants {
		// Only return end-dated plants if specifically asked for
		if getEndDated || (!getEndDated && p.EndDate == nil) {
			result = append(result, p)
		}
	}
	return result, nil
}

// SavePlant saves a plant in the map and will write it back to the YAML file
func (c *YAMLClient) SavePlant(garden xid.ID, plant *pkg.Plant) error {
	c.gardens[garden].Plants[plant.ID] = plant
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
