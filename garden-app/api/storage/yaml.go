package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
)

// YAMLClient implements the Client interface to use a YAML file as a storage mechanism
type YAMLClient struct {
	plants   map[xid.ID]*api.Plant
	filename string
	Config   Config
}

// NewYAMLClient will read the plants from the file and store them in a map
func NewYAMLClient(config Config) (*YAMLClient, error) {
	if _, ok := config.Options["filename"]; !ok {
		return nil, fmt.Errorf("missing config key 'filename'")
	}
	client := &YAMLClient{
		plants:   map[xid.ID]*api.Plant{},
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
	err = yaml.Unmarshal(data, &client.plants)
	if err != nil {
		return client, err
	}

	// Create start dates for Plants if it is empty
	for _, plant := range client.plants {
		if plant.CreatedAt == nil {
			now := time.Now().Add(1 * time.Minute)
			plant.CreatedAt = &now
			client.SavePlant(plant)
		}
	}

	return client, err
}

// GetPlant just returns the request Plant from the map
func (c *YAMLClient) GetPlant(id xid.ID) (*api.Plant, error) {
	return c.plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *YAMLClient) GetPlants(getEndDated bool) []*api.Plant {
	result := []*api.Plant{}
	for _, p := range c.plants {
		// Only return end-dated plants if specifically asked for
		if getEndDated || (!getEndDated && p.EndDate == nil) {
			result = append(result, p)
		}
	}
	return result
}

// SavePlant saves a plant in the map and will write it back to the YAML file
func (c *YAMLClient) SavePlant(plant *api.Plant) error {
	c.plants[plant.ID] = plant

	// Marshal map to YAML bytes
	content, err := yaml.Marshal(c.plants)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.filename, content, 0755)
}
