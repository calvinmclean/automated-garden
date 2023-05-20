package yaml

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func (c *Client) GetWaterSchedule(xid.ID) (*pkg.WaterSchedule, error) {
	return nil, nil
}

func (c *Client) GetWaterSchedules(bool) ([]*pkg.WaterSchedule, error) {
	return nil, nil
}

func (c *Client) SaveWaterSchedule(*pkg.WaterSchedule) error {
	return nil
}

func (c *Client) DeleteWaterSchedule(xid.ID) error {
	return nil
}

func (c *Client) GetZonesUsingWaterSchedule(xid.ID) ([]*pkg.Zone, error) {
	return nil, nil
}

func (c *Client) GetWaterSchedulesUsingWeatherClient(xid.ID) ([]*pkg.WaterSchedule, error) {
	return nil, nil
}
