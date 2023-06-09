package pkg

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestZoneEndDated(t *testing.T) {
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
			p := &Zone{EndDate: tt.endDate}
			if p.EndDated() != tt.expected {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.expected, p.EndDated())
			}
		})
	}
}

func TestZonePatch(t *testing.T) {
	zero := uint(0)
	three := 3
	now := time.Now()
	wsID := xid.New()
	tests := []struct {
		name    string
		newZone *Zone
	}{
		{
			"PatchName",
			&Zone{Name: "name"},
		},
		{
			"PatchPosition",
			&Zone{Position: &zero},
		},
		{
			"PatchCreatedAt",
			&Zone{CreatedAt: &now},
		},
		{
			"PatchWaterScheduleID",
			&Zone{WaterScheduleIDs: []xid.ID{wsID}},
		},
		{
			"PatchDetails.Description",
			&Zone{Details: &ZoneDetails{
				Description: "description",
			}},
		},
		{
			"PatchDetails.Notes",
			&Zone{Details: &ZoneDetails{
				Notes: "notes",
			}},
		},
		{
			"PatchSkipCount",
			&Zone{
				SkipCount: &three,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &Zone{}
			z.Patch(tt.newZone)
			assert.Equal(t, tt.newZone, z)
		})
	}

	t.Run("PatchDoesNotAddEndDate", func(t *testing.T) {
		now := time.Now()
		p := &Zone{}

		p.Patch(&Zone{EndDate: &now})

		if p.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", p.EndDate)
		}
	})

	t.Run("PatchRemoveEndDate", func(t *testing.T) {
		now := time.Now()
		p := &Zone{
			EndDate: &now,
		}

		p.Patch(&Zone{})

		if p.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", p.EndDate)
		}
	})
}
