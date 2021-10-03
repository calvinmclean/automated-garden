package pkg

import (
	"testing"
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
