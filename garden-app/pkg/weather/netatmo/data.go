package netatmo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type weatherData map[time.Time]float32

type weatherDataResponse struct {
	Body weatherData `json:"body"`
}

// UnmarshalJSON is used to custom unmarshal a response that uses epoch timestamps as keys for a list of floats. This unmarshals
// that data into a representation of the time to the first value in the list of data
func (d *weatherData) UnmarshalJSON(s []byte) error {
	var inputMap map[string][]float32
	err := json.Unmarshal(s, &inputMap)
	if err != nil {
		return err
	}

	result := weatherData{}

	for k, v := range inputMap {
		epochInt, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			return err
		}

		result[time.Unix(epochInt, 0)] = v[0]
	}

	*(*weatherData)(d) = result

	return nil
}

func (d *weatherData) Total() float32 {
	totalMM := float32(0)

	for _, data := range *d {
		totalMM += data
	}
	return totalMM
}

func (d *weatherData) Average() float32 {
	return d.Total() / float32(len(*d))
}

func (c *Client) getMeasure(dataType, scale string, beginDate time.Time, endDate *time.Time) (*weatherData, error) {
	err := c.refreshToken()
	if err != nil {
		return nil, err
	}

	measureURL := *c.baseURL
	measureURL.Path = "/api/getmeasure"

	moduleID := c.OutdoorModuleID
	if strings.Contains(dataType, "rain") {
		moduleID = c.RainModuleID
	}

	values := measureURL.Query()
	values.Add("device_id", c.StationID)
	values.Add("module_id", moduleID)
	values.Add("scale", scale)
	values.Add("optimize", "false")
	values.Add("real_time", "false")
	values.Add("type", dataType)
	values.Add("date_begin", fmt.Sprintf("%d", beginDate.Unix()))
	if endDate != nil {
		values.Add("date_end", fmt.Sprintf("%d", endDate.Unix()))
	}
	measureURL.RawQuery = values.Encode()

	req, err := http.NewRequest(http.MethodGet, measureURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Authentication.AccessToken))
	req.Header.Add("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body with status %d: %v", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received unexpected status %d with body: %s", resp.StatusCode, string(respBody))
	}

	var respData weatherDataResponse
	err = json.Unmarshal(respBody, &respData)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body '%s': %v", string(respBody), err)
	}

	return &respData.Body, err
}
