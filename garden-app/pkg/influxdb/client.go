package influxdb

import (
	"bytes"
	"context"
	"text/template"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

const (
	// QueryTimeout is the default time to use for a query's context timeout
	QueryTimeout          = time.Millisecond * 1000
	moistureQueryTemplate = `from(bucket: "{{.Bucket}}")
|> range(start: -{{.Start}})
|> filter(fn: (r) => r["_measurement"] == "moisture")
|> filter(fn: (r) => r["_field"] == "value")
|> filter(fn: (r) => r["plant"] == "{{.PlantPosition}}")
|> filter(fn: (r) => r["topic"] == "{{.GardenName}}/data/moisture")
|> mean()`
	healthQueryTemplate = `from(bucket: "{{.Bucket}}")
|> range(start: -{{.Start}})
|> filter(fn: (r) => r["_measurement"] == "health")
|> filter(fn: (r) => r["_field"] == "garden")
|> filter(fn: (r) => r["_value"] == "{{.GardenName}}")
|> last()`
	wateringHistoryQueryTemplate = `from(bucket: "{{.Bucket}}")
|> range(start: -{{.Start}})
|> filter(fn: (r) => r["_measurement"] == "water")
|> filter(fn: (r) => r["topic"] == "{{.GardenName}}/data/water")
|> filter(fn: (r) => r["plant"] == "{{.PlantPosition}}")
|> yield(name: "last")`
)

// queryData is used to fill out any of the query templates
type queryData struct {
	Bucket        string
	Start         time.Duration
	PlantPosition int
	GardenName    string
}

// String executes the specified template with the queryData to create a string
func (q queryData) String(queryTemplate string) (string, error) {
	t := template.Must(template.New("query").Parse(queryTemplate))
	var queryBytes bytes.Buffer
	err := t.Execute(&queryBytes, q)
	if err != nil {
		return "", err
	}
	return queryBytes.String(), nil
}

// Client is an interface that allows querying InfluxDB for data
type Client interface {
	GetMoisture(context.Context, int, string) (float64, error)
	GetLastContact(context.Context, string) (time.Time, error)
	GetWateringHistory(context.Context, int, string, time.Duration) ([]map[string]interface{}, error)
	influxdb2.Client
}

// client wraps an InfluxDB2 Client and our custom config
type client struct {
	influxdb2.Client
	config Config
}

// Config holds configuration values for connecting the the InfluxDB server
type Config struct {
	Address string `mapstructure:"address"`
	Token   string `mapstructure:"token"`
	Org     string `mapstructure:"org"`
	Bucket  string `mapstructure:"bucket"`
}

// NewClient creates an InfluxDB client from the viper config
func NewClient(config Config) Client {
	return &client{
		influxdb2.NewClient(config.Address, config.Token),
		config,
	}
}

// GetMoisture returns the plant's average soil moisture in the last 15 minutes
func (client *client) GetMoisture(ctx context.Context, plantPosition int, gardenName string) (result float64, err error) {
	// Prepare query
	queryString, err := queryData{
		Bucket:        client.config.Bucket,
		Start:         time.Minute * 15,
		PlantPosition: plantPosition,
		GardenName:    gardenName,
	}.String(moistureQueryTemplate)
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

func (client *client) GetLastContact(ctx context.Context, gardenName string) (result time.Time, err error) {
	// Prepare query
	queryString, err := queryData{
		Bucket:     client.config.Bucket,
		Start:      time.Minute * 15,
		GardenName: gardenName,
	}.String(healthQueryTemplate)
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
		time := queryResult.Record().Time()
		result = time
	}
	err = queryResult.Err()
	return
}

// GetWateringHistory gets recent watering events for a specific Plant
func (client *client) GetWateringHistory(ctx context.Context, plantPosition int, gardenName string, timeRange time.Duration) ([]map[string]interface{}, error) {
	// Prepare query
	queryString, err := queryData{
		Bucket:        client.config.Bucket,
		Start:         timeRange,
		GardenName:    gardenName,
		PlantPosition: plantPosition,
	}.String(wateringHistoryQueryTemplate)
	if err != nil {
		return nil, err
	}

	// Query InfluxDB
	queryAPI := client.QueryAPI(client.config.Org)
	queryResult, err := queryAPI.Query(ctx, queryString)
	if err != nil {
		return nil, err
	}

	// Read and return the result as slice of maps
	result := []map[string]interface{}{}
	for queryResult.Next() {
		result = append(result, map[string]interface{}{
			"WateringAmount": int(queryResult.Record().Value().(float64)),
			"RecordTime":     queryResult.Record().Time(),
		})
	}
	return result, queryResult.Err()
}
