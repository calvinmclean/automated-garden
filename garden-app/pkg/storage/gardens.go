package storage

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// GetGarden ...
func (c *Client) GetGarden(id xid.ID) (*pkg.Garden, error) {
	return c.Gardens.Get(id.String())
}

// GetGardens ...
func (c *Client) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	return c.Gardens.GetAll(getEndDated)
}

// SaveGarden ...
func (c *Client) SaveGarden(g *pkg.Garden) error {
	return c.Gardens.Set(g)
}

// DeleteGarden ...
func (c *Client) DeleteGarden(id xid.ID) error {
	return c.Gardens.Delete(id.String())
}
