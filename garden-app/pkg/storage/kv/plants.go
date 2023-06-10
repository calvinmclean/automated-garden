package kv

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// SavePlant ...
func (c *Client) SavePlant(gardenID xid.ID, plant *pkg.Plant) error {
	garden, err := c.GetGarden(gardenID)
	if err != nil {
		return fmt.Errorf("error getting parent Garden %q for Plant %q: %w", gardenID, plant.ID, err)
	}

	if garden.Plants == nil {
		garden.Plants = map[xid.ID]*pkg.Plant{}
	}
	garden.Plants[plant.ID] = plant

	return c.SaveGarden(garden)
}

// DeletePlant ...
func (c *Client) DeletePlant(gardenID xid.ID, id xid.ID) error {
	garden, err := c.GetGarden(gardenID)
	if err != nil {
		return fmt.Errorf("error getting parent Garden %q for Plant %q: %w", gardenID, id, err)
	}

	delete(garden.Plants, id)

	return c.SaveGarden(garden)
}
