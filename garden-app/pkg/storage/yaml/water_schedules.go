package yaml

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// GetWaterSchedule ...
func (c *Client) GetWaterSchedule(id xid.ID) (*pkg.WaterSchedule, error) {
	c.m.Lock()
	defer c.m.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	return c.data.WaterSchedules[id], nil
}

// GetWaterSchedules returns all WaterSchedules
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

// GetMultipleWaterSchedules returns multiple WaterSchedules matching the slice of IDs
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

// SaveWaterSchedule ...
func (c *Client) SaveWaterSchedule(ws *pkg.WaterSchedule) error {
	c.m.Lock()
	defer c.m.Unlock()

	c.data.WaterSchedules[ws.ID] = ws
	return c.save()
}

// DeleteWaterSchedule ...
func (c *Client) DeleteWaterSchedule(id xid.ID) error {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.data.WaterSchedules, id)
	return c.save()
}
