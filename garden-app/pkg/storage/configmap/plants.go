package configmap

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// GetPlant just returns the request Plant from the map
func (c *Client) GetPlant(garden xid.ID, id xid.ID) (*pkg.Plant, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.Gardens[garden].Plants[id], nil
}

// GetPlants returns all plants from the map as a slice
func (c *Client) GetPlants(garden xid.ID, getEndDated bool) ([]*pkg.Plant, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Plant{}
	for _, p := range c.data.Gardens[garden].Plants {
		if getEndDated || !p.EndDated() {
			result = append(result, p)
		}
	}
	return result, nil
}

// SavePlant saves a plant in the map and will write it back to the YAML file
func (c *Client) SavePlant(gardenID xid.ID, plant *pkg.Plant) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.data.Gardens[gardenID].Plants == nil {
		c.data.Gardens[gardenID].Plants = map[xid.ID]*pkg.Plant{}
	}
	c.data.Gardens[gardenID].Plants[plant.ID] = plant
	return c.save()
}

// DeletePlant permanently deletes a plant and removes it from the YAML file
func (c *Client) DeletePlant(garden xid.ID, plant xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.Gardens[garden].Plants, plant)
	return c.save()
}
