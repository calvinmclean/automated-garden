package influxdb

import (
	"bytes"
	"context"
	"text/template"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/spf13/viper"
)

const (
	// QueryTimeout is the default time to use for a query's context timeout
	QueryTimeout          = time.Millisecond * 1000
	moistureQueryTemplate = `from(bucket: "{{.Bucket}}")
|> range(start: -{{.Start}})
|> filter(fn: (r) => r["_measurement"] == "moisture")
|> filter(fn: (r) => r["_field"] == "value")
|> filter(fn: (r) => r["plant"] == "{{.PlantPosition}}")
|> filter(fn: (r) => r["topic"] == "{{.GardenTopic}}/data/moisture")
|> mean()`
)

// moistureQueryData is used to fill out the moistureQueryTemplate
type moistureQueryData struct {
	Bucket        string
	Start         time.Duration
	PlantPosition int
	GardenTopic   string
}

// String executes the moistureQueryTemplate with the moistureQueryData to create a string
func (q moistureQueryData) String() (string, error) {
	queryTemplate := template.Must(template.New("query").Parse(moistureQueryTemplate))
	var queryBytes bytes.Buffer
	err := queryTemplate.Execute(&queryBytes, q)
	if err != nil {
		return "", err
	}
	return queryBytes.String(), nil
}

// Client wraps an InfluxDB2 Client and our custom config
type Client struct {
	influxdb2.Client
	config *struct {
		Address string `mapstructure:"address"`
		Token   string `mapstructure:"token"`
		Org     string `mapstructure:"org"`
		Bucket  string `mapstructure:"bucket"`
	}
}

// NewClient creates an InfluxDB client from the viper config
func NewClient() (*Client, error) {
	var client Client
	if err := viper.UnmarshalKey("influxdb", &client.config); err != nil {
		return &client, err
	}

	client.Client = influxdb2.NewClient(client.config.Address, client.config.Token)
	return &client, nil
}

// GetMoisture returns the plant's average soil moisture in the last 15 minutes
func (client *Client) GetMoisture(ctx context.Context, plantPosition int, gardenTopic string) (result float64, err error) {
	// Prepare query
	queryString, err := moistureQueryData{
		Bucket:        client.config.Bucket,
		Start:         time.Minute * 15,
		PlantPosition: plantPosition,
		GardenTopic:   gardenTopic,
	}.String()
	if err != nil {
		return
	}

	// Query InfluxDB
	queryAPI := client.QueryAPI(client.config.Org)
	queryResult, err := queryAPI.Query(ctx, queryString)
	if err != nil {
		return
	}

	// Read and return the result
	if queryResult.Next() {
		result = queryResult.Record().Value().(float64)
	}
	err = queryResult.Err()
	return
}
