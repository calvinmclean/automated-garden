package kv

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// GetZone ...
func (c *Client) GetZone(gardenID xid.ID, id xid.ID) (*pkg.Zone, error) {
	garden, err := c.GetGarden(gardenID)
	if err != nil {
		return nil, fmt.Errorf("error getting parent Garden %q for Zone %q: %w", gardenID, id, err)
	}

	return garden.Zones[id], nil
}

// GetZones ...
func (c *Client) GetZones(gardenID xid.ID, getEndDated bool) ([]*pkg.Zone, error) {
	garden, err := c.GetGarden(gardenID)
	if err != nil {
		return nil, fmt.Errorf("error getting parent Garden %q: %w", gardenID, err)
	}

	results := []*pkg.Zone{}
	for _, zone := range garden.Zones {
		if getEndDated || !zone.EndDated() {
			results = append(results, zone)
		}
	}

	return results, nil
}

// SaveZone ...
func (c *Client) SaveZone(gardenID xid.ID, zone *pkg.Zone) error {
	garden, err := c.GetGarden(gardenID)
	if err != nil {
		return fmt.Errorf("error getting parent Garden %q for Zone %q: %w", gardenID, zone.ID, err)
	}

	if garden.Zones == nil {
		garden.Zones = map[xid.ID]*pkg.Zone{}
	}
	garden.Zones[zone.ID] = zone

	return c.SaveGarden(garden)
}

// DeleteZone ...
func (c *Client) DeleteZone(gardenID xid.ID, id xid.ID) error {
	garden, err := c.GetGarden(gardenID)
	if err != nil {
		return fmt.Errorf("error getting parent Garden %q for Zone %q: %w", gardenID, id, err)
	}

	delete(garden.Zones, id)

	return c.SaveGarden(garden)
}
