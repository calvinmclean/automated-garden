package api

import (
	"bytes"
	"context"
	"text/template"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api/influxdb"
	"github.com/rs/xid"
)

// Plant is the representation of the most important resource for this application, a Plant.
// This includes some general information like name and unique ID, a start and end date to show when
// the Plant was in the system, plus some information for watering like the duration to water for, how
// often to water, and the PlantPosition field will tell the microcontroller which plant to water
type Plant struct {
	Name             string           `json:"name" yaml:"name,omitempty"`
	ID               xid.ID           `json:"id" yaml:"id,omitempty"`
	Garden           string           `json:"garden" yaml:"garden,omitempty"`
	PlantPosition    int              `json:"plant_position" yaml:"plant_position"`
	StartDate        *time.Time       `json:"start_date" yaml:"start_date,omitempty"`
	EndDate          *time.Time       `json:"end_date,omitempty" yaml:"end_date,omitempty"`
	SkipCount        int              `json:"skip_count,omitempty" yaml:"skip_count,omitempty"`
	WateringStrategy WateringStrategy `json:"watering_strategy,omitempty" yaml:"watering_strategy,omitempty"`
}

// WateringStrategy allows the user to have more control over how the Plant is watered using an Interval
// and optional MinimumMoisture which acts as the threshold the Plant's soil should be above
type WateringStrategy struct {
	WateringAmount  int    `json:"watering_amount" yaml:"watering_amount"`
	Interval        string `json:"interval" yaml:"interval"`
	MinimumMoisture int    `json:"minimum_moisture,omitempty" yaml:"minimum_moisture,omitempty"`
}

// Topic is used to populate and return a MQTT Topic string from a template string input
func (p *Plant) Topic(topic string) (string, error) {
	t := template.Must(template.New("topic").Parse(topic))
	var result bytes.Buffer
	err := t.Execute(&result, p)
	return result.String(), err
}

// WateringAction creates the default/basic WateringAction for this Plant
func (p *Plant) WateringAction() *WaterAction {
	return &WaterAction{Duration: p.WateringStrategy.WateringAmount}
}

// GetMoisture will read the most recent moisture data from InfluxDB for this Plant
func (p *Plant) GetMoisture(config influxdb.Config) (float64, error) {
	client := influxdb.NewClient(config)

	ctx, cancel := context.WithTimeout(context.Background(), influxdb.QueryTimeout)
	defer cancel()
	return client.GetMoisture(ctx, p.PlantPosition, p.Garden)
}
