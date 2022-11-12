package netatmo

import (
	"time"
)

const minRainInterval = 24 * time.Hour

// GetTotalRain returns the sum of all rainfall in millimeters in the given period
func (c *Client) GetTotalRain(since time.Duration) (float32, error) {
	// Time to check from must always be at least 24 hours to get valid data
	if since < minRainInterval {
		since = minRainInterval
	}

	beginDate := time.Now().Add(-since)
	rainData, err := c.getMeasure("sum_rain", "1day", beginDate, nil)
	if err != nil {
		return 0, err
	}

	return rainData.Total(), nil
}
