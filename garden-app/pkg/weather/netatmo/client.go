package netatmo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

const (
	baseURI = "https://api.netatmo.com"

	StationDeviceType       = "NAMain"
	OutdoorModuleDeviceType = "NAModule1"
	RainModuleDeviceType    = "NAModule3"
)

type Config struct {
	StationID      string     `mapstructure:"station_id"`
	RainModuleID   string     `mapstructure:"rain_module_id"`
	StationName    string     `mapstructure:"station_name"`
	RainModuleName string     `mapstructure:"rain_module_name"`
	Authentication *TokenData `mapstructure:"authentication"`
	ClientID       string     `mapstructure:"client_id"`
	ClientSecret   string     `mapstructure:"client_secret"`
}

type TokenData struct {
	AccessToken    string `mapstructure:"access_token" json:"access_token"`
	RefreshToken   string `mapstructure:"refresh_token" json:"refresh_token"`
	ExpiresIn      int    `mapstructure:"expires_in" json:"expires_in"`
	ExpirationDate time.Time
}

type Client struct {
	*Config
	*http.Client
	baseURL *url.URL
}

func NewClient(options map[string]interface{}) (*Client, error) {
	client := &Client{Client: http.DefaultClient}

	err := mapstructure.Decode(options, &client.Config)
	if err != nil {
		return nil, err
	}

	client.baseURL, err = url.Parse(baseURI)
	if err != nil {
		return nil, err
	}

	if client.StationID == "" || client.RainModuleID == "" {
		err = client.setDeviceIDs()
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

type StationDataResponse struct {
	Body struct {
		Devices []Station `json:"devices"`
	} `json:"body"`
}

type Station struct {
	ID         string `json:"_id"`
	Name       string `json:"station_name"`
	ModuleName string `json:"module_name"`
	Modules    []struct {
		ID   string `json:"_id"`
		Name string `json:"module_name"`
	} `json:"modules"`
}

func (c *Client) getStationData() (StationDataResponse, error) {
	err := c.refreshToken()
	if err != nil {
		return StationDataResponse{}, err
	}

	stationDataURL := *c.baseURL
	stationDataURL.Path = "/api/getstationsdata"

	values := stationDataURL.Query()
	if c.StationID != "" {
		values.Add("device_id", c.StationID)
	}
	values.Add("get_favorites", "false")
	stationDataURL.RawQuery = values.Encode()

	req, err := http.NewRequest(http.MethodGet, stationDataURL.String(), nil)
	if err != nil {
		return StationDataResponse{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Authentication.AccessToken))
	req.Header.Add("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return StationDataResponse{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return StationDataResponse{}, fmt.Errorf("error reading response body with status %d: %v", resp.StatusCode, err)
	}

	var respData StationDataResponse
	err = json.Unmarshal(respBody, &respData)
	if err != nil {
		return StationDataResponse{}, err
	}

	return respData, nil
}

func (c *Client) setDeviceIDs() error {
	if c.StationID == "" && c.StationName == "" {
		return errors.New("station_id or station_name must be provided")
	}
	if c.RainModuleID == "" && c.RainModuleName == "" {
		return errors.New("rain_module_id or rain_module_name must be provided")
	}

	stationData, err := c.getStationData()
	if err != nil {
		return err
	}

	// Find Station ID if not provided
	var targetStation Station
	if c.StationID == "" {
		for _, s := range stationData.Body.Devices {
			if s.ModuleName == c.StationName {
				targetStation = s
			}
		}
	}
	if targetStation.ID == "" {
		return fmt.Errorf("no station found with name %q", c.StationName)
	}
	c.StationID = targetStation.ID

	// Find module ID if not provided
	if c.RainModuleID == "" {
		for _, m := range targetStation.Modules {
			if m.Name == c.RainModuleName {
				c.RainModuleID = m.ID
			}
		}
	}
	if c.RainModuleID == "" {
		return fmt.Errorf("no rain module found with name %q", c.RainModuleName)
	}

	return nil
}

func (c *Client) getMeasure(dataType, scale string, beginDate time.Time) ([]byte, error) {
	err := c.refreshToken()
	if err != nil {
		return nil, err
	}

	measureURL := *c.baseURL
	measureURL.Path = "/api/getmeasure"

	values := measureURL.Query()
	values.Add("device_id", c.StationID)
	values.Add("module_id", c.RainModuleID)
	values.Add("scale", scale)
	values.Add("optimize", "false")
	values.Add("real_time", "false")
	values.Add("type", dataType)
	values.Add("date_begin", fmt.Sprintf("%d", beginDate.Unix()))
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

	return respBody, err
}

func (c *Client) refreshToken() error {
	formData := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.Authentication.RefreshToken},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}

	req, err := http.NewRequest("POST", "https://api.netatmo.com/oauth2/token", strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body with status %d: %v", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received unexpected status %d with body: %s", resp.StatusCode, string(respBody))
	}

	err = json.Unmarshal(respBody, c.Authentication)
	if err != nil {
		return err
	}
	c.Authentication.ExpirationDate = time.Now().Add(time.Duration(c.Authentication.ExpiresIn) * time.Second)

	return nil
}
