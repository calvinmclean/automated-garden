package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// GetWaterSchedule ...
func (c *Client) GetWaterSchedule(id xid.ID) (*pkg.WaterSchedule, error) {
	return c.WaterSchedules.Get(id.String())
}

// GetWaterSchedules ...
func (c *Client) GetWaterSchedules(getEndDated bool) ([]*pkg.WaterSchedule, error) {
	return c.WaterSchedules.GetAll(getEndDated)
}

// SaveWaterSchedule ...
func (c *Client) SaveWaterSchedule(ws *pkg.WaterSchedule) error {
	return c.WaterSchedules.Set(ws)
}

// DeleteWaterSchedule ...
func (c *Client) DeleteWaterSchedule(id xid.ID) error {
	return c.WaterSchedules.Delete(id.String())
}

// GetMultipleWaterSchedules ...
func (c *Client) GetMultipleWaterSchedules(ids []xid.ID) ([]*pkg.WaterSchedule, error) {
	results := []*pkg.WaterSchedule{}
	for _, id := range ids {
		result, err := c.GetWaterSchedule(id)
		if err != nil {
			return nil, fmt.Errorf("error getting WaterSchedule: %w", err)
		}

		results = append(results, result)
	}

	return results, nil
}

// GetZonesUsingWaterSchedule will find all Zones that use this WaterSchedule and return the Zones along with the Gardens they belong to
func (c *Client) GetZonesUsingWaterSchedule(id xid.ID) ([]*pkg.ZoneAndGarden, error) {
	gardens, err := c.GetGardens(false)
	if err != nil {
		return nil, fmt.Errorf("unable to get all Gardens: %w", err)
	}

	results := []*pkg.ZoneAndGarden{}
	for _, g := range gardens {
		zones, err := c.GetZones(g.ID, false)
		if err != nil {
			return nil, fmt.Errorf("unable to get all Zones for Garden %q: %w", g.ID, err)
		}

		for _, z := range zones {
			for _, wsID := range z.WaterScheduleIDs {
				if wsID == id {
					results = append(results, &pkg.ZoneAndGarden{Zone: z, Garden: g})
				}
			}
		}
	}

	return results, nil
}
