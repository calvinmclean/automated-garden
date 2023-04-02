package yaml

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// GetGarden returns the garden
func (c *Client) GetGarden(id xid.ID) (*pkg.Garden, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.Gardens[id], nil
}

// GetGardens returns all the gardens
func (c *Client) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Garden{}
	for _, g := range c.data.Gardens {
		if getEndDated || !g.EndDated() {
			result = append(result, g)
		}
	}
	return result, nil
}

// SaveGarden saves a garden and writes it back to the YAML file
func (c *Client) SaveGarden(garden *pkg.Garden) error {
	c.m.Lock()
	defer c.m.Unlock()

	c.data.Gardens[garden.ID] = garden
	return c.save()
}

// DeleteGarden permanently deletes a garden and removes it from the YAML file
func (c *Client) DeleteGarden(garden xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.Gardens, garden)
	return c.save()
}
