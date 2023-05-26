package kv

import (
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

const waterSchedulePrefix = "WaterSchedule_"

func (c *Client) GetWaterSchedule(id xid.ID) (*pkg.WaterSchedule, error) {
	return c.getWaterSchedule(waterSchedulePrefix + id.String())
}

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

func (c *Client) SaveWaterSchedule(g *pkg.WaterSchedule) error {
	asBytes, err := c.marshal(g)
	if err != nil {
		return fmt.Errorf("error marshalling WaterSchedule: %w", err)
	}

	err = c.db.Set(waterSchedulePrefix+g.ID.String(), asBytes)
	if err != nil {
		return fmt.Errorf("error writing WaterSchedule to database: %w", err)
	}

	return nil
}

func (c *Client) DeleteWaterSchedule(id xid.ID) error {
	return c.db.Delete(waterSchedulePrefix + id.String())
}

func (c *Client) getWaterSchedule(key string) (*pkg.WaterSchedule, error) {
	dataBytes, err := c.db.Get(key)
	if err != nil {
		return nil, fmt.Errorf("error getting WaterSchedule: %w", err)
	}

	var result pkg.WaterSchedule
	err = c.unmarshal(dataBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing WaterSchedule data: %w", err)
	}

	return &result, nil
}

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
