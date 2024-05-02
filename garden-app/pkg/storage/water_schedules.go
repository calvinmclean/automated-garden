package storage

import (
	"context"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
)

// GetZonesUsingWaterSchedule will find all Zones that use this WaterSchedule and return the Zones along with the Gardens they belong to
func (c *Client) GetZonesUsingWaterSchedule(id string) ([]*pkg.ZoneAndGarden, error) {
	gardens, err := c.Gardens.GetAll(context.Background(), babyapi.EndDatedQueryParam(false))
	if err != nil {
		return nil, fmt.Errorf("unable to get all Gardens: %w", err)
	}

	results := []*pkg.ZoneAndGarden{}
	for _, g := range gardens {
		zones, err := c.Zones.GetAll(context.Background(), nil)
		if err != nil {
			return nil, fmt.Errorf("unable to get all Zones for Garden %q: %w", g.ID, err)
		}
		zones = babyapi.FilterFunc[*pkg.Zone](func(z *pkg.Zone) bool {
			if z.GardenID != g.ID.ID || z.EndDated() {
				return false
			}
			for _, wsID := range z.WaterScheduleIDs {
				if wsID.String() == id {
					return true
				}
			}
			return false
		}).Filter(zones)

		for _, z := range zones {
			results = append(results, &pkg.ZoneAndGarden{Zone: z, Garden: g})
		}
	}

	return results, nil
}
