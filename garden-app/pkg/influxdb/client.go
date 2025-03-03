package influxdb

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// QueryTimeout is the default time to use for a query's context timeout
	QueryTimeout        = time.Millisecond * 1000
	healthQueryTemplate = `from(bucket: "{{.Bucket}}")
|> range(start: -{{.Start}})
|> filter(fn: (r) => r["_measurement"] == "health")
|> filter(fn: (r) => r["_field"] == "garden")
|> filter(fn: (r) => r["_value"] == "{{.TopicPrefix}}")
|> drop(columns: ["host"])
|> last()`
	waterHistoryQueryTemplate = `from(bucket: "{{.Bucket}}")
|> range(start: -{{.Start}})
|> filter(fn: (r) => r["_measurement"] == "water")
|> filter(fn: (r) => r["topic"] == "{{.TopicPrefix}}/data/water")
|> filter(fn: (r) => r["zone_id"] == "{{.ZoneID}}")
|> filter(fn: (r) => r["status"] == "complete")
|> drop(columns: ["host"])
|> sort(columns: ["_time"], desc: true)
{{- if .Limit }}
|> limit(n: {{.Limit}})
{{- end }}
|> yield(name: "last")`
	temperatureAndHumidityQueryTemplate = `from(bucket: "{{.Bucket}}")
|> range(start: -{{.Start}})
|> filter(fn: (r) => r["_measurement"] == "temperature" or r["_measurement"] == "humidity")
|> filter(fn: (r) => r["_field"] == "value")
|> filter(fn: (r) => r["topic"] == "{{.TopicPrefix}}/data/temperature" or r["topic"] == "{{.TopicPrefix}}/data/humidity")
|> drop(columns: ["host"])
|> mean()`
)

func init() {
	sync.OnceFunc(func() {
		prometheus.MustRegister(influxDBClientSummary)
	})()
}

var influxDBClientSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Namespace: "garden_app",
	Name:      "influxdb_client_duration_seconds",
	Help:      "summary of influxdb client calls",
}, []string{"function"})

// Client is an interface that allows querying InfluxDB for data
type Client interface {
	GetLastContact(context.Context, string) (time.Time, error)
	GetWaterHistory(context.Context, string, string, time.Duration, uint64) ([]pkg.WaterHistory, error)
	GetTemperatureAndHumidity(context.Context, string) (float64, float64, error)
	influxdb2.Client
}

var _ Client = &client{}

// Config holds configuration values for connecting the the InfluxDB server
type Config struct {
	Address string `mapstructure:"address"`
	Token   string `mapstructure:"token"`
	Org     string `mapstructure:"org"`
	Bucket  string `mapstructure:"bucket"`
}

// queryData is used to fill out any of the query templates
type queryData struct {
	Bucket      string
	Start       time.Duration
	ZoneID      string
	TopicPrefix string
	Limit       uint64
}

// Render executes the specified template with the queryData to create a string
func (q queryData) Render(queryTemplate string) (string, error) {
	t := template.Must(template.New("query").Parse(queryTemplate))
	var queryBytes bytes.Buffer
	err := t.Execute(&queryBytes, q)
	if err != nil {
		return "", err
	}
	return queryBytes.String(), nil
}

// client wraps an InfluxDB2 Client and our custom config
type client struct {
	influxdb2.Client
	config Config
}

// NewClient creates an InfluxDB client from the viper config
func NewClient(config Config) Client {
	return &client{
		influxdb2.NewClient(config.Address, config.Token),
		config,
	}
}

func (client *client) GetLastContact(ctx context.Context, topicPrefix string) (time.Time, error) {
	timer := prometheus.NewTimer(influxDBClientSummary.WithLabelValues("GetLastContact"))
	defer timer.ObserveDuration()

	// Prepare query
	queryString, err := queryData{
		Bucket:      client.config.Bucket,
		Start:       time.Minute * 15,
		TopicPrefix: topicPrefix,
	}.Render(healthQueryTemplate)
	if err != nil {
		return time.Time{}, err
	}

	// Query InfluxDB
	queryAPI := client.QueryAPI(client.config.Org)
	queryResult, err := queryAPI.Query(ctx, queryString)
	if err != nil {
		return time.Time{}, err
	}

	// Read and return the result
	var result time.Time
	if queryResult.Next() {
		time := queryResult.Record().Time()
		result = time
	}

	return result, queryResult.Err()
}

// GetWaterHistory gets recent water events for a specific Zone
func (client *client) GetWaterHistory(ctx context.Context, zoneID string, topicPrefix string, timeRange time.Duration, limit uint64) ([]pkg.WaterHistory, error) {
	timer := prometheus.NewTimer(influxDBClientSummary.WithLabelValues("GetWaterHistory"))
	defer timer.ObserveDuration()

	// Prepare query
	queryString, err := queryData{
		Bucket:      client.config.Bucket,
		Start:       timeRange,
		TopicPrefix: topicPrefix,
		ZoneID:      zoneID,
		Limit:       limit,
	}.Render(waterHistoryQueryTemplate)
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
	result := []pkg.WaterHistory{}
	for queryResult.Next() {
		h := pkg.WaterHistory{}

		millis := queryResult.Record().Value()
		durationVal, ok := millis.(float64)
		if !ok {
			return nil, fmt.Errorf("unexpected type for duration millis: %T", millis)
		}
		h.Duration = (time.Duration(durationVal) * time.Millisecond).String()

		h.RecordTime = queryResult.Record().Time()

		eventID := queryResult.Record().ValueByKey("id")
		h.EventID, ok = eventID.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected type for event ID: %T", eventID)
		}

		result = append(result, h)
	}
	return result, queryResult.Err()
}

// GetTemperatureAndHumidity gets the recent temperature and humidity data for a Garden
func (client *client) GetTemperatureAndHumidity(ctx context.Context, topicPrefix string) (float64, float64, error) {
	timer := prometheus.NewTimer(influxDBClientSummary.WithLabelValues("GetTemperatureAndHumidity"))
	defer timer.ObserveDuration()

	queryString, err := queryData{
		Bucket:      client.config.Bucket,
		Start:       time.Minute * 15,
		TopicPrefix: topicPrefix,
	}.Render(temperatureAndHumidityQueryTemplate)
	if err != nil {
		return 0, 0, err
	}

	queryAPI := client.QueryAPI(client.config.Org)
	queryResult, err := queryAPI.Query(ctx, queryString)
	if err != nil {
		return 0, 0, err
	}

	var temperature float64
	var humidity float64
	for queryResult.Next() {
		switch queryResult.Record().Measurement() {
		case "temperature":
			temperature = queryResult.Record().Value().(float64)
		case "humidity":
			humidity = queryResult.Record().Value().(float64)
		}
	}

	return temperature, humidity, queryResult.Err()
}
