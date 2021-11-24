package pkg

import (
	"testing"
	"time"
)

func TestWateringEvent(t *testing.T) {
	plant := Plant{
		WateringStrategy: &WateringStrategy{
			WateringAmount: 15000,
			Interval:       "24h",
		},
	}
	action := plant.WateringAction()
	if action.Duration != 15000 {
		t.Errorf("Unexpected Duration in WaterAction: Expected: %v, Actual: %v", 15000, action.Duration)
	}
}

func TestPlantEndDated(t *testing.T) {
	pastDate := time.Now().Add(-1 * time.Minute)
	futureDate := time.Now().Add(time.Minute)
	tests := []struct {
		name     string
		endDate  *time.Time
		expected bool
	}{
		{"NilEndDateFalse", nil, false},
		{"EndDateFutureEndDateFalse", &futureDate, false},
		{"EndDatePastEndDateTrue", &pastDate, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plant{EndDate: tt.endDate}
			if p.EndDated() != tt.expected {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.expected, p.EndDated())
			}
		})
	}
}

func TestPatch(t *testing.T) {
	zero := 0
	now := time.Now()
	tests := []struct {
		name     string
		newPlant *Plant
	}{
		{
			"PatchName",
			&Plant{Name: "name"},
		},
		{
			"PatchPlantPosition",
			&Plant{PlantPosition: &zero},
		},
		{
			"PatchCreatedAt",
			&Plant{CreatedAt: &now},
		},
		{
			"PatchEndDate",
			&Plant{EndDate: &now},
		},
		{
			"PatchSkipCount",
			&Plant{SkipCount: &zero},
		},
		{
			"PatchWateringStrategy.WateringAmount",
			&Plant{WateringStrategy: &WateringStrategy{
				WateringAmount: 1000,
			}},
		},
		{
			"PatchWateringStrategy.Interval",
			&Plant{WateringStrategy: &WateringStrategy{
				Interval: "2h",
			}},
		},
		{
			"PatchWateringStrategy.MinimumMoisture",
			&Plant{WateringStrategy: &WateringStrategy{
				MinimumMoisture: 1,
			}},
		},
		{
			"PatchWateringStrategy.StartTime",
			&Plant{WateringStrategy: &WateringStrategy{
				StartTime: "start time",
			}},
		},
		{
			"PatchDetails.Description",
			&Plant{Details: &Details{
				Description: "description",
			}},
		},
		{
			"PatchDetails.Notes",
			&Plant{Details: &Details{
				Notes: "notes",
			}},
		},
		{
			"PatchDetails.TimeToHarvest",
			&Plant{Details: &Details{
				TimeToHarvest: "TimeToHarvest",
			}},
		},
		{
			"PatchDetails.Count",
			&Plant{Details: &Details{
				Count: 1,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plant{}
			p.Patch(tt.newPlant)
			if p.WateringStrategy != nil && *p.WateringStrategy != *tt.newPlant.WateringStrategy {
				t.Errorf("Unexpected result for WateringStrategy: expected=%v, actual=%v", tt.newPlant, p)
			}
			p.WateringStrategy = nil
			tt.newPlant.WateringStrategy = nil
			if p.Details != nil && *p.Details != *tt.newPlant.Details {
				t.Errorf("Unexpected result for Details: expected=%v, actual=%v", tt.newPlant, p)
			}
			p.Details = nil
			tt.newPlant.Details = nil
			if *p != *tt.newPlant {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.newPlant, p)
			}
		})
	}
}
