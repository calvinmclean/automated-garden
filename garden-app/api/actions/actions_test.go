package actions

import (
	"encoding/json"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

var plant = api.Plant{
	Name:           "Cherry Tomato",
	ID:             "9m4e2mr0ui3e8a215n4g",
	WateringAmount: 15000,
	PlantPosition:  0,
	Interval:       "24h",
	StartDate:      "2021-01-15",
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
	if waterAction.Skip != nil {
		t.Errorf("Skip was unexpectedly not nil: %v", waterAction.Skip)
	}
	if waterAction.Water.Duration != 15000 {
		t.Errorf("Duration was incorrect.\nExpected: %d, Actual: %d", 15000, waterAction.Water.Duration)
	}
}

func TestUnmarshalSkipEvent(t *testing.T) {
	skipActionBytes := []byte(`{"skip": {"count": 1}}`)
	var skipAction AggregateAction
	err := json.Unmarshal(skipActionBytes, &skipAction)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	if skipAction.Skip == nil {
		t.Error("Skip was unexpectedly nil")
	}
	if skipAction.Water != nil {
		t.Errorf("Water was unexpectedly not nil: %v", skipAction.Water)
	}
	if skipAction.Skip.Count != 1 {
		t.Errorf("Count was incorrect.\nExpected: %d, Actual: %d", 1, skipAction.Skip.Count)
	}
}

func TestUnmarshalAllEvent(t *testing.T) {
	actionBytes := []byte(`{"skip": {"count": 1}, "water": {"duration": 15000}}`)
	var action AggregateAction
	err := json.Unmarshal(actionBytes, &action)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	if action.Skip == nil {
		t.Error("Skip was unexpectedly nil")
	}
	if action.Water == nil {
		t.Error("Water was unexpectedly nil")
	}
	if action.Water.Duration != 15000 {
		t.Errorf("Duration was incorrect.\nExpected: %d, Actual: %d", 15000, action.Water.Duration)
	}
	if action.Skip.Count != 1 {
		t.Errorf("Count was incorrect.\nExpected: %d, Actual: %d", 1, action.Skip.Count)
	}
}
