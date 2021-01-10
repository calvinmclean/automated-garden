package storage

import (
	"io/ioutil"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"gopkg.in/yaml.v3"
)

// YAMLClient implements the Client interface to use a YAML file as a storage mechanism
type YAMLClient struct {
	filename string
	plants   map[string]*api.Plant
}

// NewYAMLClient will read the plants from the file and store them in a map
func NewYAMLClient(filename string) (*YAMLClient, error) {
	client := &YAMLClient{filename: filename}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return client, err
	}
	err = yaml.Unmarshal(data, &client.plants)
	if err != nil {
		return client, err
	}
	return client, err
}

// GetPlant just returns the request Plant from the map
func (c *YAMLClient) GetPlant(id string) *api.Plant {
	return c.plants[id]
}

// GetPlants returns all plants from the map as a slice
func (c *YAMLClient) GetPlants() []*api.Plant {
	result := []*api.Plant{}
	for _, p := range c.plants {
		result = append(result, p)
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

// EndDatePlant is in place of Delete method and will just mark the end date and
// and save it
// TODO: implement this
func (c *YAMLClient) EndDatePlant(plant *api.Plant) error {
	return nil
}
