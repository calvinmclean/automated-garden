package pkg

import (
	"testing"
	"time"
)

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
			"PatchCreatedAt",
			&Plant{CreatedAt: &now},
		},
		{
			"PatchDetails.Description",
			&Plant{Details: &PlantDetails{
				Description: "description",
			}},
		},
		{
			"PatchDetails.Notes",
			&Plant{Details: &PlantDetails{
				Notes: "notes",
			}},
		},
		{
			"PatchDetails.TimeToHarvest",
			&Plant{Details: &PlantDetails{
				TimeToHarvest: "TimeToHarvest",
			}},
		},
		{
			"PatchDetails.Count",
			&Plant{Details: &PlantDetails{
				Count: 1,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plant{}
			p.Patch(tt.newPlant)
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
