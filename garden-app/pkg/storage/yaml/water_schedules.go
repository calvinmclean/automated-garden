package yaml

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func (c *Client) GetWaterSchedule(id xid.ID) (*pkg.WaterSchedule, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.WaterSchedules[id], nil
}

func (c *Client) GetWaterSchedules(getEndDated bool) ([]*pkg.WaterSchedule, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.WaterSchedule{}
	for _, ws := range c.data.WaterSchedules {
		if getEndDated || !ws.EndDated() {
			result = append(result, ws)
		}
	}
	return result, nil
}

func (c *Client) GetMultipleWaterSchedules(ids []xid.ID) ([]*pkg.WaterSchedule, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	result := []*pkg.WaterSchedule{}
	for _, id := range ids {
		ws, ok := c.data.WaterSchedules[id]
		if ok {
			result = append(result, ws)
		}
	}
	return result, nil
}

func (c *Client) SaveWaterSchedule(ws *pkg.WaterSchedule) error {
	c.m.Lock()
	defer c.m.Unlock()

	c.data.WaterSchedules[ws.ID] = ws
	return c.save()
}

func (c *Client) DeleteWaterSchedule(id xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.WaterSchedules, id)
	return c.save()
}

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
