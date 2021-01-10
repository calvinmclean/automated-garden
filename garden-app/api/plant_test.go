package api

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	plantBytes := []byte(`{
		"name": "Cherry Tomato",
		"id": "9m4e2mr0ui3e8a215n4g",
		"watering_amount": 15000,
		"plant_position": 0,
		"interval": "24h",
		"start_date": "2021-01-15",
		"end_date": null
	}`)
	var actual Plant
	err := json.Unmarshal(plantBytes, &actual)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	expected := Plant{
		Name:           "Cherry Tomato",
		ID:             "9m4e2mr0ui3e8a215n4g",
		WateringAmount: 15000,
		PlantPosition:  0,
		Interval:       "24h",
		StartDate:      "2021-01-15",
	}

	tests := []struct {
		fieldName string
		expected  interface{}
		actual    interface{}
	}{
		{"Name", expected.Name, actual.Name},
		{"ID", expected.ID, actual.ID},
		{"PlantPosition", expected.PlantPosition, actual.PlantPosition},
		{"WateringAmount", expected.WateringAmount, actual.WateringAmount},
		{"Interval", expected.Interval, actual.Interval},
		{"StartDate", expected.StartDate, actual.StartDate},
		{"EndDate", expected.EndDate, actual.EndDate},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("%s was incorrect.\nExpected: %v, Actual: %v", tt.fieldName, tt.expected, tt.actual)
		}
	}
}
