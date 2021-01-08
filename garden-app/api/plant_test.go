package api

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	plantBytes := []byte(`{
		"name": "Cherry Tomato",
		"id": "D89406D6-884D-48D3-98A8-7A282CD210EB",
		"watering_amount": 15000,
		"interval": "24h",
		"start_date": "2021-01-15",
		"end_date": null,
		"valve_pin": 3,
		"pump_pin": 5
	}`)
	var actual Plant
	err := json.Unmarshal(plantBytes, &actual)
	if err != nil {
		t.Errorf("Unexpected error when Unmarshaling JSON: %s", err.Error())
	}

	expected := Plant{
		Name:           "Cherry Tomato",
		ID:             "D89406D6-884D-48D3-98A8-7A282CD210EB",
		WateringAmount: 15000,
		Interval:       "24h",
		StartDate:      "2021-01-15",
		ValvePin:       3,
		PumpPin:        5,
	}

	tests := []struct {
		fieldName string
		expected  interface{}
		actual    interface{}
	}{
		{"Name", expected.Name, actual.Name},
		{"ID", expected.ID, actual.ID},
		{"WateringAmount", expected.WateringAmount, actual.WateringAmount},
		{"Interval", expected.Interval, actual.Interval},
		{"StartDate", expected.StartDate, actual.StartDate},
		{"EndDate", expected.EndDate, actual.EndDate},
		{"ValvePin", expected.ValvePin, actual.ValvePin},
		{"PumpPin", expected.PumpPin, actual.PumpPin},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("%s was incorrect.\nExpected: %v, Actual: %v", tt.fieldName, tt.expected, tt.actual)
		}
	}
}
