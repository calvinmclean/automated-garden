package api

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalWaterEvent(t *testing.T) {
	waterActionBytes := []byte(`{"duration": 15000, "ignore_moisture": true}`)
	var waterAction WaterAction
	err := json.Unmarshal(waterActionBytes, &waterAction)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	if waterAction.Duration != 15000 {
		t.Errorf("Duration was incorrect.\nExpected: %d, Actual: %d", 15000, waterAction.Duration)
	}
	if waterAction.IgnoreMoisture != true {
		t.Errorf("IgnoreMoisture was incorrect.\nExpected: %t, Actual: %t", true, waterAction.IgnoreMoisture)
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
