package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func TestPlantRequest(t *testing.T) {
	pp := 0
	tests := []struct {
		name string
		pr   *PlantRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required Plant fields",
		},
		{
			"EmptyPlantError",
			&PlantRequest{},
			"missing required Plant fields",
		},
		{
			"EmptyPlantPositionError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name: "plant",
				},
			},
			"missing required plant_position field",
		},
		{
			"EmptyWateringStrategyError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name:          "plant",
					PlantPosition: &pp,
				},
			},
			"missing required watering_strategy field",
		},
		{
			"EmptyWateringStrategyIntervalError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name:          "plant",
					PlantPosition: &pp,
					WateringStrategy: &pkg.WateringStrategy{
						WateringAmount: 1000,
					},
				},
			},
			"missing required watering_strategy.interval field",
		},
		{
			"EmptyWateringStrategyWateringAmountError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name:          "plant",
					PlantPosition: &pp,
					WateringStrategy: &pkg.WateringStrategy{
						Interval: "24h",
					},
				},
			},
			"missing required watering_strategy.watering_amount field",
		},
		{
			"EmptyWateringStrategyStartTimeError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name:          "plant",
					PlantPosition: &pp,
					WateringStrategy: &pkg.WateringStrategy{
						Interval:       "24h",
						WateringAmount: 1000,
					},
				},
			},
			"missing required watering_strategy.start_time field",
		},
		{
			"InvalidWateringStrategyStartTimeError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name:          "plant",
					PlantPosition: &pp,
					WateringStrategy: &pkg.WateringStrategy{
						Interval:       "24h",
						WateringAmount: 1000,
						StartTime:      "NOT A TIME",
					},
				},
			},
			"invalid time format for watering_strategy.start_time: NOT A TIME",
		},
		{
			"EmptyNameError",
			&PlantRequest{
				Plant: &pkg.Plant{
					PlantPosition: &pp,
					WateringStrategy: &pkg.WateringStrategy{
						Interval:       "24h",
						WateringAmount: 1000,
						StartTime:      "19:00:00-07:00",
					},
				},
			},
			"missing required name field",
		},
		{
			"ManualSpecificationOfGardenIDError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name:          "garden",
					PlantPosition: &pp,
					GardenID:      xid.New(),
					WateringStrategy: &pkg.WateringStrategy{
						Interval:       "24h",
						WateringAmount: 1000,
						StartTime:      "19:00:00-07:00",
					},
				},
			},
			"manual specification of garden ID is not allowed",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &PlantRequest{
			Plant: &pkg.Plant{
				Name:          "plant",
				PlantPosition: &pp,
				WateringStrategy: &pkg.WateringStrategy{
					WateringAmount: 1000,
					Interval:       "24h",
					StartTime:      "19:00:00-07:00",
				},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := pr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading PlantRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading PlantRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}

func TestUpdatePlantRequest(t *testing.T) {
	pp := 0
	now := time.Now()
	tests := []struct {
		name string
		pr   *UpdatePlantRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required Plant fields",
		},
		{
			"EmptyPlantError",
			&UpdatePlantRequest{},
			"missing required Plant fields",
		},
		{
			"ManualSpecificationOfGardenIDError",
			&UpdatePlantRequest{
				Plant: &pkg.Plant{
					GardenID: xid.New(),
				},
			},
			"updating garden ID is not allowed",
		},
		{
			"ManualSpecificationOfIDError",
			&UpdatePlantRequest{
				Plant: &pkg.Plant{ID: xid.New()},
			},
			"updating ID is not allowed",
		},
		{
			"InvalidWateringStrategyStartTimeError",
			&UpdatePlantRequest{
				Plant: &pkg.Plant{
					WateringStrategy: &pkg.WateringStrategy{
						StartTime: "NOT A TIME",
					},
				},
			},
			"invalid time format for watering_strategy.start_time: NOT A TIME",
		},
		{
			"EndDateError",
			&UpdatePlantRequest{
				Plant: &pkg.Plant{
					EndDate: &now,
				},
			},
			"to end-date a Plant, please use the DELETE endpoint",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &UpdatePlantRequest{
			Plant: &pkg.Plant{
				Name:          "plant",
				PlantPosition: &pp,
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := pr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading PlantRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading PlantRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}

func TestPlantActionRequest(t *testing.T) {
	tests := []struct {
		name string
		ar   *PlantActionRequest
		err  string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required action fields",
		},
		{
			"EmptyActionError",
			&PlantActionRequest{},
			"missing required action fields",
		},
		{
			"EmptyPlantActionError",
			&PlantActionRequest{
				PlantAction: &pkg.PlantAction{},
			},
			"missing required action fields",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		ar := &PlantActionRequest{
			PlantAction: &pkg.PlantAction{
				Water: &pkg.WaterAction{},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := ar.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading PlantActionRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.ar.Bind(r)
			if err == nil {
				t.Error("Expected error reading PlantActionRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
