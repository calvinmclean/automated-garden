package server

import (
	"net/http/httptest"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func TestPlantRequest(t *testing.T) {
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
			"EmptyWateringStrategyError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name: "plant",
				},
			},
			"missing required watering_strategy field",
		},

		{
			"EmptyWateringStrategyIntervalError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name: "plant",
					WateringStrategy: pkg.WateringStrategy{
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
					Name: "plant",
					WateringStrategy: pkg.WateringStrategy{
						Interval: "24h",
					},
				},
			},
			"missing required watering_strategy.watering_amount field",
		},
		{
			"EmptyNameError",
			&PlantRequest{
				Plant: &pkg.Plant{
					WateringStrategy: pkg.WateringStrategy{
						Interval:       "24h",
						WateringAmount: 1000,
					},
				},
			},
			"missing required name field",
		},
		{
			"ManualSpecificationOfGardenIDError",
			&PlantRequest{
				Plant: &pkg.Plant{
					Name:     "garden",
					GardenID: xid.New(),
					WateringStrategy: pkg.WateringStrategy{
						Interval:       "24h",
						WateringAmount: 1000,
					},
				},
			},
			"manual specification of garden ID is not allowed",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &PlantRequest{
			Plant: &pkg.Plant{
				Name: "plant",
				WateringStrategy: pkg.WateringStrategy{
					WateringAmount: 1000,
					Interval:       "24h",
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

func TestGardenActionRequest(t *testing.T) {
	tests := []struct {
		name string
		ar   *GardenActionRequest
		err  string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required action fields",
		},
		{
			"EmptyActionError",
			&GardenActionRequest{},
			"missing required action fields",
		},
		{
			"EmptyGardenActionError",
			&GardenActionRequest{
				GardenAction: &pkg.GardenAction{},
			},
			"missing required action fields",
		},
		{
			"InvalidLightActionState",
			&GardenActionRequest{
				GardenAction: &pkg.GardenAction{
					Light: &pkg.LightAction{
						State: "WOW",
					},
				},
			},
			`invalid "state" provided: "WOW"`,
		},
	}

	t.Run("Successful", func(t *testing.T) {
		ar := &GardenActionRequest{
			GardenAction: &pkg.GardenAction{
				Light: &pkg.LightAction{
					State: "ON",
				},
				Stop: &pkg.StopAction{},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := ar.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading GardenActionRequest JSON: %v", err)
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

func TestGardenRequest(t *testing.T) {
	tests := []struct {
		name string
		gr   *GardenRequest
		err  string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required Garden fields",
		},
		{
			"EmptyGardenError",
			&GardenRequest{},
			"missing required Garden fields",
		},
		{
			"MissingNameErrorError",
			&GardenRequest{
				Garden: &pkg.Garden{},
			},
			"missing required name field",
		},
		{
			"InvalidNameErrorError$",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden$",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError#",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden#",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError*",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden*",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError>",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden>",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError+",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden+",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError/",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden/",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"CreatingPlantsNotAllowedError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden",
					Plants: map[xid.ID]*pkg.Plant{
						xid.New(): {},
					},
				},
			},
			"cannot add or modify Plants with this request",
		},
		{
			"EmptyLightScheduleIntervalError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden",
					LightSchedule: &pkg.LightSchedule{
						StartTime: "22:00:01-07:00",
					},
				},
			},
			"missing required light_schedule.interval field",
		},
		{
			"EmptyLightScheduleStartTimeError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden",
					LightSchedule: &pkg.LightSchedule{
						Interval: "1m",
					},
				},
			},
			"missing required light_schedule.start_time field",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		gr := &GardenRequest{
			Garden: &pkg.Garden{
				Name: "garden",
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := gr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading GardenRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.gr.Bind(r)
			if err == nil {
				t.Error("Expected error reading GardenRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
