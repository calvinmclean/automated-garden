package yaml

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// GetZone just returns the request Zone from the map
func (c *Client) GetZone(garden xid.ID, id xid.ID) (*pkg.Zone, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.Gardens[garden].Zones[id], nil
}

// GetZones returns all zones from the map as a slice
func (c *Client) GetZones(garden xid.ID, getEndDated bool) ([]*pkg.Zone, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.Zone{}
	for _, p := range c.data.Gardens[garden].Zones {
		if getEndDated || !p.EndDated() {
			result = append(result, p)
		}
	}
	return result, nil
}

// SaveZone saves a zone in the map and will write it back to the YAML file
func (c *Client) SaveZone(gardenID xid.ID, zone *pkg.Zone) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.data.Gardens[gardenID].Zones == nil {
		c.data.Gardens[gardenID].Zones = map[xid.ID]*pkg.Zone{}
	}
	c.data.Gardens[gardenID].Zones[zone.ID] = zone
	return c.save()
}

// DeleteZone permanently deletes a zone and removes it from the YAML file
func (c *Client) DeleteZone(garden xid.ID, zone xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.Gardens[garden].Zones, zone)
	return c.save()
}
