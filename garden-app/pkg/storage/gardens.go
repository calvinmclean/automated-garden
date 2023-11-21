package storage

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

const gardenPrefix = "Garden_"

func gardenKey(id xid.ID) string {
	return gardenPrefix + id.String()
}

// GetGarden ...
func (c *Client) GetGarden(id xid.ID) (*pkg.Garden, error) {
	return GetOne[pkg.Garden](c, gardenKey(id))
}

// GetGardens ...
func (c *Client) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	return GetMultiple[*pkg.Garden](c, getEndDated, gardenPrefix)
}

// SaveGarden ...
func (c *Client) SaveGarden(g *pkg.Garden) error {
	return Save[*pkg.Garden](c, g, gardenKey(g.ID))
}

// DeleteGarden ...
func (c *Client) DeleteGarden(id xid.ID) error {
	return c.db.Delete(gardenKey(id))
}
