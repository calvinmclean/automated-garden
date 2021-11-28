package pkg

import (
	"testing"
	"time"
)

func TestWateringEvent(t *testing.T) {
	plant := Plant{
		WaterSchedule: &WaterSchedule{
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

func TestPlantPatch(t *testing.T) {
	zero := uint(0)
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
			"PatchSkipCount",
			&Plant{SkipCount: &zero},
		},
		{
			"PatchWaterSchedule.WateringAmount",
			&Plant{WaterSchedule: &WaterSchedule{
				WateringAmount: 1000,
			}},
		},
		{
			"PatchWaterSchedule.Interval",
			&Plant{WaterSchedule: &WaterSchedule{
				Interval: "2h",
			}},
		},
		{
			"PatchWaterSchedule.MinimumMoisture",
			&Plant{WaterSchedule: &WaterSchedule{
				MinimumMoisture: 1,
			}},
		},
		{
			"PatchWaterSchedule.StartTime",
			&Plant{WaterSchedule: &WaterSchedule{
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
			if p.WaterSchedule != nil && *p.WaterSchedule != *tt.newPlant.WaterSchedule {
				t.Errorf("Unexpected result for WaterSchedule: expected=%v, actual=%v", tt.newPlant, p)
			}
			p.WaterSchedule = nil
			tt.newPlant.WaterSchedule = nil
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

	t.Run("PatchDoesNotAddEndDate", func(t *testing.T) {
		now := time.Now()
		p := &Plant{}

		p.Patch(&Plant{EndDate: &now})

		if p.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", p.EndDate)
		}
	})

	t.Run("PatchRemoveEndDate", func(t *testing.T) {
		now := time.Now()
		p := &Plant{
			EndDate: &now,
		}

		p.Patch(&Plant{})

		if p.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", p.EndDate)
		}
	})
}
