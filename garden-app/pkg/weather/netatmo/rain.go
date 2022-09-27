package netatmo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type rainData map[time.Time]float32

type rainDataResponse struct {
	Body rainData `json:"body"`
}

// UnmarshalJSON is used to custom unmarshal a response that uses epoch timestamps as keys for a list of floats. This unmarshals
// that data into a representation of the time to the first value in the list of data
func (d *rainData) UnmarshalJSON(s []byte) error {
	var rainInfo map[string][]float32
	err := json.Unmarshal(s, &rainInfo)
	if err != nil {
		return err
	}

	result := rainData{}

	for k, v := range rainInfo {
		epochInt, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			return err
		}

		result[time.Unix(epochInt, 0)] = v[0]
	}

	*(*rainData)(d) = result

	return nil
}

func (d *rainData) Total() float32 {
	totalMM := float32(0)

	for _, data := range *d {
		totalMM += data
	}
	return totalMM
}

// GetTotalRain returns the sum of all rainfall in millimeters in the given period
func (c *Client) GetTotalRain(since time.Duration) (float32, error) {
	beginDate := time.Now().Add(-since)
	rainDataJSON, err := c.getMeasure("sum_rain", "1day", beginDate)
	if err != nil {
		return 0, err
	}

	var rainDataResp rainDataResponse
	err = json.Unmarshal(rainDataJSON, &rainDataResp)
	if err != nil {
		return 0, fmt.Errorf("unable to read response body '%s': %v", string(rainDataJSON), err)
	}

	return rainDataResp.Body.Total(), nil
}
