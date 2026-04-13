package openmeteo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/mitchellh/mapstructure"
)

// Config is specific to the OpenMeteo API and holds the necessary fields for location
type Config struct {
	Latitude  float32 `json:"latitude" yaml:"latitude" mapstructure:"latitude"`
	Longitude float32 `json:"longitude" yaml:"longitude" mapstructure:"longitude"`
}

// Client is used to interact with OpenMeteo API
type Client struct {
	*Config
	httpClient *http.Client
	baseURL    string
}

const (
	minRainInterval        = 24 * time.Hour
	minTemperatureInterval = 72 * time.Hour
	defaultBaseURL         = "https://api.open-meteo.com"
)

// openMeteoResponse represents the structure of the API response
type openMeteoResponse struct {
	Daily struct {
		Time             []string  `json:"time"`
		Temperature2mMax []float32 `json:"temperature_2m_max"`
		PrecipitationSum []float32 `json:"precipitation_sum"`
	} `json:"daily"`
}

// NewClient creates a new OpenMeteo API client from configuration
func NewClient(options map[string]any) (*Client, error) {
	return NewClientWithHTTPClient(options, http.DefaultClient)
}

// NewClientWithHTTPClient creates a new OpenMeteo API client with a custom HTTP client (used for testing)
func NewClientWithHTTPClient(options map[string]any, httpClient *http.Client) (*Client, error) {
	client := &Client{
		Config:     &Config{},
		httpClient: httpClient,
		baseURL:    defaultBaseURL,
	}

	err := mapstructure.WeakDecode(options, &client.Config)
	if err != nil {
		return nil, err
	}

	if client.Latitude == 0 || client.Longitude == 0 {
		return nil, errors.New("latitude and longitude must be provided")
	}

	return client, nil
}

// fetchData makes the API request to OpenMeteo and returns the parsed response
func (c *Client) fetchData(pastDays int, dailyVars ...string) (*openMeteoResponse, error) {
	u, err := url.Parse(c.baseURL + "/v1/forecast")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("latitude", fmt.Sprintf("%f", c.Latitude))
	q.Set("longitude", fmt.Sprintf("%f", c.Longitude))
	q.Set("past_days", fmt.Sprintf("%d", pastDays))
	q.Set("timezone", "auto")

	// Add daily variables
	for _, v := range dailyVars {
		q.Add("daily", v)
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var data openMeteoResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &data, nil
}

// GetTotalRain returns the sum of all precipitation in millimeters in the given period
func (c *Client) GetTotalRain(_ context.Context, since time.Duration) (float32, error) {
	// Time to check from must always be at least 24 hours to get valid data
	if since < minRainInterval {
		since = minRainInterval
	}

	// Calculate past days needed (round up)
	pastDays := int(since.Hours()/24) + 1

	data, err := c.fetchData(pastDays, "precipitation_sum")
	if err != nil {
		return 0, fmt.Errorf("error fetching precipitation data: %w", err)
	}

	if len(data.Daily.PrecipitationSum) == 0 {
		return 0, errors.New("no precipitation data returned")
	}

	// Sum all precipitation values
	var total float32
	for _, precip := range data.Daily.PrecipitationSum {
		total += precip
	}

	return total, nil
}

// GetAverageHighTemperature returns the average daily high temperature between the given time and the end of
// yesterday (since daily high can be misleading if queried mid-day)
func (c *Client) GetAverageHighTemperature(_ context.Context, since time.Duration) (float32, error) {
	// Time to check since must always be at least 3 days
	if since < minTemperatureInterval {
		since = minTemperatureInterval
	}

	// Calculate past days needed (round up)
	pastDays := int(since.Hours()/24) + 1

	data, err := c.fetchData(pastDays, "temperature_2m_max")
	if err != nil {
		return 0, fmt.Errorf("error fetching temperature data: %w", err)
	}

	if len(data.Daily.Temperature2mMax) == 0 {
		return 0, errors.New("no temperature data returned")
	}

	// Calculate average of daily max temperatures
	now := clock.Now()
	endOfYesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 0, time.Local)

	var sum float32
	var count int
	for i, t := range data.Daily.Time {
		date, err := time.Parse("2006-01-02", t)
		if err != nil {
			continue
		}
		// Only include days up to end of yesterday
		if date.Before(endOfYesterday) || date.Equal(endOfYesterday) {
			sum += data.Daily.Temperature2mMax[i]
			count++
		}
	}

	if count == 0 {
		return 0, errors.New("no valid temperature data for the specified period")
	}

	return sum / float32(count), nil
}
