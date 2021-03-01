package actions

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/rs/xid"
)

var plant = api.Plant{
	Name:           "Cherry Tomato",
	Garden:         "garden",
	WateringAmount: 15000,
	PlantPosition:  0,
	Interval:       "24h",
}

func init() {
	id, _ := xid.FromString("9m4e2mr0ui3e8a215n4g")
	startDate, _ := time.Parse(time.RFC3339, "2020-01-15T00:00:00-07:00")
	plant.StartDate = &startDate
	plant.ID = id
}

func TestUnmarshalWaterEvent(t *testing.T) {
	waterActionBytes := []byte(`{"water": {"duration": 15000}}`)
	var waterAction AggregateAction
	err := json.Unmarshal(waterActionBytes, &waterAction)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	if waterAction.Water == nil {
		t.Error("Water was unexpectedly nil")
	}
	if waterAction.Water.Duration != 15000 {
		t.Errorf("Duration was incorrect.\nExpected: %d, Actual: %d", 15000, waterAction.Water.Duration)
	}
}

func TestUnmarshalAllEvent(t *testing.T) {
	actionBytes := []byte(`{"water": {"duration": 15000}}`)
	var action AggregateAction
	err := json.Unmarshal(actionBytes, &action)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	if action.Water == nil {
		t.Error("Water was unexpectedly nil")
	}
	if action.Water.Duration != 15000 {
		t.Errorf("Duration was incorrect.\nExpected: %d, Actual: %d", 15000, action.Water.Duration)
	}
}
