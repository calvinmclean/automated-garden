package netatmo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/units"
)

type RainData map[time.Time]float32

type RainDataResponse struct {
	Body RainData `json:"body"`
}

func (d *RainData) UnmarshalJSON(s []byte) error {
	var rainInfo struct {
		Body map[string][]float32 `json:"body"`
	}
	err := json.Unmarshal(s, &rainInfo)
	if err != nil {
		return err
	}

	result := RainData{}

	for k, v := range rainInfo.Body {
		epochInt, err := strconv.ParseInt(k, 10, 64)
		fmt.Println(epochInt)
		if err != nil {
			return err
		}

		result[time.Unix(epochInt, 0)] = v[0]
	}

	*(*RainData)(d) = result

	return nil
}

func (d *RainData) Total(unit units.RainUnit) float32 {
	totalMM := float32(0)

	for _, data := range *d {
		totalMM += data
	}

	switch unit {
	case units.RainUnitInch:
		return totalMM / 25.4
	default:
		return totalMM
	}
}

func (c *Client) GetTotalRain(since time.Time, unit units.RainUnit) (float32, error) {
	rainDataJSON, err := c.getMeasure("sum_rain", "1day", since)
	if err != nil {
		return 0, err
	}

	var rainData RainData
	err = json.Unmarshal(rainDataJSON, &rainData)
	if err != nil {
		return 0, fmt.Errorf("unable to read response body '%s': %v", string(rainDataJSON), err)
	}

	return rainData.Total(unit), nil
}
