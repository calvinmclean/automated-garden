package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/xid"
)

func TestUnmarshalJSON(t *testing.T) {
	plantBytes := []byte(`{
		"name": "Cherry Tomato",
		"id": "9m4e2mr0ui3e8a215n4g",
		"watering_amount": 15000,
		"plant_position": 0,
		"interval": "24h",
		"start_date": "2020-01-15T00:00:00-07:00",
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
		Name:           "Cherry Tomato",
		ID:             id,
		WateringAmount: 15000,
		PlantPosition:  0,
		Interval:       "24h",
		StartDate:      &startDate,
	}

	tests := []struct {
		fieldName string
		expected  interface{}
		actual    interface{}
	}{
		{"Name", expected.Name, actual.Name},
		{"ID", expected.ID, actual.ID},
		{"WateringAmount", expected.WateringAmount, actual.WateringAmount},
		{"PlantPosition", expected.PlantPosition, actual.PlantPosition},
		{"Interval", expected.Interval, actual.Interval},
		{"StartDate", expected.StartDate.String(), actual.StartDate.String()},
		{"EndDate", expected.EndDate, actual.EndDate},
		{"SkipCount", expected.SkipCount, actual.SkipCount},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("%s was incorrect.\nExpected: %v, Actual: %v", tt.fieldName, tt.expected, tt.actual)
		}
	}
}
