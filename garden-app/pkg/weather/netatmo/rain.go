package netatmo

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

const minRainInterval = 24 * time.Hour

// GetTotalRain returns the sum of all rainfall in millimeters in the given period
func (c *Client) GetTotalRain(since time.Duration) (float32, error) {
	now := time.Now()
	cacheKey := fmt.Sprintf("total_rain_%d", now.Unix())
	cachedData, found := c.responseCache.Get(cacheKey)
	if found {
		return cachedData.(float32), nil
	}

	// Time to check from must always be at least 24 hours to get valid data
	if since < minRainInterval {
		since = minRainInterval
	}

	beginDate := now.Add(-since)
	rainData, err := c.getMeasure("sum_rain", "1day", beginDate, nil)
	if err != nil {
		return 0, err
	}

	c.responseCache.Set(cacheKey, rainData, cache.DefaultExpiration)

	return rainData.Total(), nil
}
