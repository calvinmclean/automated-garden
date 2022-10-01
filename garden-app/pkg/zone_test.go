package pkg

import (
	"reflect"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
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
	now := time.Now()
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
			"PatchWaterSchedule.Duration",
			&Zone{WaterSchedule: &WaterSchedule{
				Duration: "1000ms",
			}},
		},
		{
			"PatchWaterSchedule.Interval",
			&Zone{WaterSchedule: &WaterSchedule{
				Interval: "2h",
			}},
		},
		{
			"PatchWaterSchedule.WeatherControl.SoilMoisture.MinimumMoisture",
			&Zone{WaterSchedule: &WaterSchedule{
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: 1,
					},
				},
			}},
		},
		{
			"PatchWaterSchedule.StartTime",
			&Zone{WaterSchedule: &WaterSchedule{
				StartTime: &now,
			}},
		},
		{
			"PatchWaterSchedule.WeatherControl.Rain.Threshold",
			&Zone{WaterSchedule: &WaterSchedule{
				WeatherControl: &weather.Control{
					Rain: &weather.RainControl{
						Threshold: 25.4,
					},
				},
			}},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Zone{}
			p.Patch(tt.newZone)
			if p.WaterSchedule != nil && !reflect.DeepEqual(*p.WaterSchedule, *tt.newZone.WaterSchedule) {
				t.Errorf("Unexpected result for WaterSchedule: expected=%v, actual=%v", tt.newZone, p)
			}
			p.WaterSchedule = nil
			tt.newZone.WaterSchedule = nil
			if p.Details != nil && *p.Details != *tt.newZone.Details {
				t.Errorf("Unexpected result for Details: expected=%v, actual=%v", tt.newZone, p)
			}
			p.Details = nil
			tt.newZone.Details = nil
			if *p != *tt.newZone {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.newZone, p)
			}
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
