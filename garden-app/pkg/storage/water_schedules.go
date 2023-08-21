package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

const waterSchedulePrefix = "WaterSchedule_"

func waterScheduleKey(id xid.ID) string {
	return waterSchedulePrefix + id.String()
}

// GetWaterSchedule ...
func (c *Client) GetWaterSchedule(id xid.ID) (*pkg.WaterSchedule, error) {
	return getOne[pkg.WaterSchedule](c, waterScheduleKey(id))
}

// GetWaterSchedules ...
func (c *Client) GetWaterSchedules(getEndDated bool) ([]*pkg.WaterSchedule, error) {
	return getMultiple[*pkg.WaterSchedule](c, getEndDated, waterSchedulePrefix)
}

// SaveWaterSchedule ...
func (c *Client) SaveWaterSchedule(ws *pkg.WaterSchedule) error {
	return save[*pkg.WaterSchedule](c, ws, waterScheduleKey(ws.ID))
}

// DeleteWaterSchedule ...
func (c *Client) DeleteWaterSchedule(id xid.ID) error {
	return c.db.Delete(waterScheduleKey(id))
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
