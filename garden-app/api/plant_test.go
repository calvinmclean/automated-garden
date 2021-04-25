package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/xid"
)

func TestTopic(t *testing.T) {
	plant := &Plant{Garden: "garden"}
	topic := "{{.Garden}}/topic"
	expected := "garden/topic"
	result, err := plant.Topic(topic)
	if err != nil {
		t.Errorf("Unexpected error when getting template result: %s", err.Error())
	}
	if result != expected {
		t.Errorf("Unexpected topic result: expected=%s, actual=%s", expected, result)
	}
}

func TestWateringEvent(t *testing.T) {
	plant := Plant{
		WateringStrategy: WateringStrategy{
			WateringAmount: 15000,
			Interval:       "24h",
		},
	}
	action := plant.WateringAction()
	if action.Duration != 15000 {
		t.Errorf("Unexpected Duration in WaterAction: Expected: %v, Actual: %v", 15000, action.Duration)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	plantBytes := []byte(`{
		"name": "Cherry Tomato",
		"id": "9m4e2mr0ui3e8a215n4g",
		"plant_position": 0,
		"watering_strategy": {
			"watering_amount": 15000,
			"interval": "24h"
		},
		"created_at": "2020-01-15T00:00:00-07:00",
		"end_date": null
	}`)
	var actual Plant
	err := json.Unmarshal(plantBytes, &actual)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	id, _ := xid.FromString("9m4e2mr0ui3e8a215n4g")
	startDate, _ := time.Parse(time.RFC3339, "2020-01-15T00:00:00-07:00")

	expected := Plant{
		Name:          "Cherry Tomato",
		ID:            id,
		PlantPosition: 0,
		CreatedAt:     &startDate,
		WateringStrategy: WateringStrategy{
			WateringAmount: 15000,
			Interval:       "24h",
		},
	}

	tests := []struct {
		fieldName string
		expected  interface{}
		actual    interface{}
	}{
		{"Name", expected.Name, actual.Name},
		{"ID", expected.ID, actual.ID},
		{"PlantPosition", expected.PlantPosition, actual.PlantPosition},
		{"CreatedAt", expected.CreatedAt.String(), actual.CreatedAt.String()},
		{"EndDate", expected.EndDate, actual.EndDate},
		{"SkipCount", expected.SkipCount, actual.SkipCount},
		{"WateringAmount", expected.WateringStrategy.WateringAmount, actual.WateringStrategy.WateringAmount},
		{"Interval", expected.WateringStrategy.Interval, actual.WateringStrategy.Interval},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("%s was incorrect.\nExpected: %v, Actual: %v", tt.fieldName, tt.expected, tt.actual)
		}
	}
}
