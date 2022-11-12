package netatmo

import (
	"time"
)

// GetTotalRain returns the sum of all rainfall in millimeters in the given period
func (c *Client) GetTotalRain(since time.Duration) (float32, error) {
	beginDate := time.Now().Add(-since)
	rainData, err := c.getMeasure("sum_rain", "1day", beginDate, nil)
	if err != nil {
		return 0, err
	}

	return rainData.Total(), nil
}
