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

// TODO: These methods are duplicated in all WeatherClients because they just require using interface methods
// I can refactor this to be part of a BaseClient, but that is kind of annoying because it requires a lot of interfaces
// Another option is to just make them helper methods in the server package where they are used
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

func (c *Client) GetWaterSchedulesUsingWeatherClient(id xid.ID) ([]*pkg.WaterSchedule, error) {
	waterSchedules, err := c.GetWaterSchedules(false)
	if err != nil {
		return nil, fmt.Errorf("unable to get all WaterSchedules: %w", err)
	}

	results := []*pkg.WaterSchedule{}
	for _, ws := range waterSchedules {
		if ws.HasWeatherControl() {
			if ws.HasRainControl() {
				if ws.WeatherControl.Rain.ClientID == id {
					results = append(results, ws)
				}
			}
			if ws.HasTemperatureControl() {
				if ws.WeatherControl.Temperature.ClientID == id {
					results = append(results, ws)
				}
			}
		}
	}

	return results, nil
}
