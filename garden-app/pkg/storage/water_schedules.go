package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/madflojo/hord"
	"github.com/rs/xid"
)

const waterSchedulePrefix = "WaterSchedule_"

// GetWaterSchedule ...
func (c *Client) GetWaterSchedule(id xid.ID) (*pkg.WaterSchedule, error) {
	return c.getWaterSchedule(waterSchedulePrefix + id.String())
}

// GetWaterSchedules ...
func (c *Client) GetWaterSchedules(getEndDated bool) ([]*pkg.WaterSchedule, error) {
	keys, err := c.db.Keys()
	if err != nil {
		return nil, fmt.Errorf("error getting keys: %w", err)
	}

	results := []*pkg.WaterSchedule{}
	for _, key := range keys {
		if !strings.HasPrefix(key, waterSchedulePrefix) {
			continue
		}

		result, err := c.getWaterSchedule(key)
		if err != nil {
			return nil, fmt.Errorf("error getting keys: %w", err)
		}

		if getEndDated || !result.EndDated() {
			results = append(results, result)
		}
	}

	return results, nil
}

// SaveWaterSchedule ...
func (c *Client) SaveWaterSchedule(ws *pkg.WaterSchedule) error {
	asBytes, err := c.marshal(ws)
	if err != nil {
		return fmt.Errorf("error marshalling WaterSchedule: %w", err)
	}

	err = c.db.Set(waterSchedulePrefix+ws.ID.String(), asBytes)
	if err != nil {
		return fmt.Errorf("error writing WaterSchedule to database: %w", err)
	}

	return nil
}

// DeleteWaterSchedule ...
func (c *Client) DeleteWaterSchedule(id xid.ID) error {
	return c.db.Delete(waterSchedulePrefix + id.String())
}

func (c *Client) getWaterSchedule(key string) (*pkg.WaterSchedule, error) {
	dataBytes, err := c.db.Get(key)
	if err != nil {
		if errors.Is(hord.ErrNil, err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting WaterSchedule: %w", err)
	}

	var result pkg.WaterSchedule
	err = c.unmarshal(dataBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing WaterSchedule data: %w", err)
	}

	return &result, nil
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
