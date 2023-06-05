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
)

// Config is specific to the Netatmo API and holds all of the necessary fields for interacting with the API.
// If StationID is not provided, StationName is used to get it from the API
// If RainModuleID is not provided, RainModuleName is used to get it from the API
// For Authentication, AccessToken, RefreshToken, ClientID and ClientSecret are required
type Config struct {
	StationID   string `json:"station_id,omitempty" yaml:"station_id,omitempty" mapstructure:"station_id,omitempty"`
	StationName string `json:"station_name,omitempty" yaml:"station_name,omitempty" mapstructure:"station_name,omitempty"`

	RainModuleID   string `json:"rain_module_id,omitempty" yaml:"rain_module_id,omitempty" mapstructure:"rain_module_id,omitempty"`
	RainModuleName string `json:"rain_module_name,omitempty" yaml:"rain_module_name,omitempty" mapstructure:"rain_module_name,omitempty"`

	OutdoorModuleID   string `json:"outdoor_module_id,omitempty" yaml:"outdoor_module_id,omitempty" mapstructure:"outdoor_module_id,omitempty"`
	OutdoorModuleName string `json:"outdoor_module_name,omitempty" yaml:"outdoor_module_name,omitempty" mapstructure:"outdoor_module_name,omitempty"`

	Authentication *TokenData `json:"authentication,omitempty" yaml:"authentication,omitempty" mapstructure:"authentication,omitempty"`
	ClientID       string     `json:"client_id,omitempty" yaml:"client_id,omitempty" mapstructure:"client_id,omitempty"`
	ClientSecret   string     `json:"client_secret,omitempty" yaml:"client_secret,omitempty" mapstructure:"client_secret,omitempty"`
}

// TokenData contains information returned by Netatmo auth API
type TokenData struct {
	AccessToken    string    `json:"access_token,omitempty" yaml:"access_token,omitempty" mapstructure:"access_token,omitempty"`
	RefreshToken   string    `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty" mapstructure:"refresh_token,omitempty"`
	ExpiresIn      int       `json:"expires_in,omitempty" yaml:"expires_in,omitempty" mapstructure:"expires_in,omitempty"`
	ExpirationDate time.Time `json:"expiration_date,omitempty" yaml:"expiration_date,omitempty" mapstructure:"expiration_date,omitempty"`
}

// Client is used to interact with Netatmo API
type Client struct {
	*Config
	*http.Client
	baseURL         *url.URL
	storageCallback func(map[string]interface{}) error
}

// NewClient creates a new Netatmo API client from configuration
// If StationID is not provided, StationName is used to get it from the API
// If RainModuleID is not provided, RainModuleName is used to get it from the API
// For Authentication, AccessToken, RefreshToken, ClientID and ClientSecret are required
func NewClient(options map[string]interface{}, storageCallback func(map[string]interface{}) error) (*Client, error) {
	client := &Client{Client: http.DefaultClient, storageCallback: storageCallback}

	err := mapstructure.Decode(options, &client.Config)
	if err != nil {
		return nil, err
	}

	client.baseURL, err = url.Parse(baseURI)
	if err != nil {
		return nil, err
	}

	if client.StationID == "" || client.RainModuleID == "" || client.OutdoorModuleID == "" {
		err = client.setDeviceIDs()
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

type stationDataResponse struct {
	Body struct {
		Devices []station `json:"devices"`
	} `json:"body"`
}

type station struct {
	ID         string `json:"_id"`
	Name       string `json:"station_name"`
	ModuleName string `json:"module_name"`
	Modules    []struct {
		ID   string `json:"_id"`
		Name string `json:"module_name"`
	} `json:"modules"`
}

func (c *Client) getStationData() (stationDataResponse, error) {
	err := c.refreshToken()
	if err != nil {
		return stationDataResponse{}, err
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
		return stationDataResponse{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Authentication.AccessToken))
	req.Header.Add("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return stationDataResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return stationDataResponse{}, fmt.Errorf("error reading response body with status %d: %v", resp.StatusCode, err)
	}

	var respData stationDataResponse
	err = json.Unmarshal(respBody, &respData)
	if err != nil {
		return stationDataResponse{}, err
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
	if c.OutdoorModuleID == "" && c.OutdoorModuleName == "" {
		return errors.New("outdoor_module_id or outdoor_module_name must be provided")
	}

	stationData, err := c.getStationData()
	if err != nil {
		return err
	}

	// Find Station ID if not provided
	var targetStation station
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
	if c.RainModuleID == "" || c.OutdoorModuleID == "" {
		for _, m := range targetStation.Modules {
			if c.RainModuleID == "" && m.Name == c.RainModuleName {
				c.RainModuleID = m.ID
			}
			if c.OutdoorModuleID == "" && m.Name == c.OutdoorModuleName {
				c.OutdoorModuleID = m.ID
			}
		}
	}
	if c.RainModuleID == "" {
		return fmt.Errorf("no rain module found with name %q", c.RainModuleName)
	}
	if c.OutdoorModuleID == "" {
		return fmt.Errorf("no outdoor module found with name %q", c.OutdoorModuleName)
	}

	return nil
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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body with status %d: %v", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received unexpected status %d with body: %s", resp.StatusCode, string(respBody))
	}

	err = json.Unmarshal(respBody, c.Authentication)
	if err != nil {
		return fmt.Errorf("unable to unmarshal refresh token response body: %w", err)
	}
	c.Authentication.ExpirationDate = time.Now().Add(time.Duration(c.Authentication.ExpiresIn) * time.Second)

	// Use storage callback to save new authentication details
	err = c.storageCallback(map[string]interface{}{
		"station_id":          c.Config.StationID,
		"station_name":        c.Config.StationName,
		"rain_module_id":      c.Config.RainModuleID,
		"rain_module_name":    c.Config.RainModuleName,
		"outdoor_module_id":   c.Config.OutdoorModuleID,
		"outdoor_module_name": c.Config.OutdoorModuleName,
		"authentication": map[string]interface{}{
			"access_token":    c.Config.Authentication.AccessToken,
			"refresh_token":   c.Config.Authentication.RefreshToken,
			"expires_in":      c.Config.Authentication.ExpiresIn,
			"expiration_date": c.Config.Authentication.ExpirationDate,
		},
		"client_id":     c.Config.ClientID,
		"client_secret": c.Config.ClientSecret,
	})
	if err != nil {
		return fmt.Errorf("error executing storage callback to store new tokens: %w", err)
	}

	return nil
}
