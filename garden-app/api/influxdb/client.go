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

type moistureQueryData struct {
	Bucket        string
	Start         time.Duration
	PlantPosition int
	GardenTopic   string
}

func (q moistureQueryData) String() (string, error) {
	queryTemplate := template.Must(template.New("query").Parse(moistureQueryTemplate))
	var queryBytes bytes.Buffer
	err := queryTemplate.Execute(&queryBytes, q)
	if err != nil {
		return "", err
	}
	return queryBytes.String(), nil
}

type Config struct {
	Address string `mapstructure:"address"`
	Token   string `mapstructure:"token"`
	Org     string `mapstructure:"org"`
	Bucket  string `mapstructure:"bucket"`
}

type Client struct {
	influxdb2.Client
	Config
}

func NewClient() (*Client, error) {
	var c Config
	if err := viper.UnmarshalKey("influxdb", &c); err != nil {
		return &Client{}, err
	}

	return &Client{influxdb2.NewClient(c.Address, c.Token), c}, nil
}

func (client *Client) GetMoisture(ctx context.Context, plantPosition int, gardenTopic string) (result float64, err error) {
	queryString, err := moistureQueryData{
		Bucket:        client.Bucket,
		Start:         time.Minute * 15,
		PlantPosition: plantPosition,
		GardenTopic:   gardenTopic,
	}.String()
	if err != nil {
		return
	}

	queryAPI := client.QueryAPI(client.Org)
	queryResult, err := queryAPI.Query(ctx, queryString)
	if err != nil {
		return
	}
	// Iterate over query response
	// for queryResult.Next() {
	// 	// Access data
	// 	fmt.Printf("value: %v\n", queryResult.Record().Value())
	// 	result = queryResult.Record().Value().(float64)
	// }
	if queryResult.Next() {
		result = queryResult.Record().Value().(float64)
	}
	// check for an error
	err = queryResult.Err()
	return
}
