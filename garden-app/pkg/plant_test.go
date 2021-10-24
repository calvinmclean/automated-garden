package pkg

import (
	"testing"
	"time"
)

func TestWateringEvent(t *testing.T) {
	plant := Plant{
		WateringStrategy: WateringStrategy{
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
