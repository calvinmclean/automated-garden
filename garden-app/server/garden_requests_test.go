package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

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

	t.Run("SuccessfulLightAction", func(t *testing.T) {
		ar := &GardenActionRequest{
			GardenAction: &pkg.GardenAction{
				Light: &pkg.LightAction{
					State: pkg.StateOn,
				},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := ar.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading GardenActionRequest JSON: %v", err)
		}
	})
	t.Run("SuccessfulStopAction", func(t *testing.T) {
		ar := &GardenActionRequest{
			GardenAction: &pkg.GardenAction{
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
	zero := uint(0)
	one := uint(1)
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
			"MissingMaxPlantsError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden",
				},
			},
			"missing required max_plants field",
		},
		{
			"MaxPlantsZeroError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:      "garden",
					MaxPlants: &zero,
				},
			},
			"max_plants must not be 0",
		},
		{
			"CreatingPlantsNotAllowedError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:      "garden",
					MaxPlants: &one,
					Plants: map[xid.ID]*pkg.Plant{
						xid.New(): {},
					},
				},
			},
			"cannot add or modify Plants with this request",
		},
		{
			"EmptyLightScheduleDurationError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:      "garden",
					MaxPlants: &one,
					LightSchedule: &pkg.LightSchedule{
						StartTime: "22:00:01-07:00",
					},
				},
			},
			"missing required light_schedule.duration field",
		},
		{
			"EmptyLightScheduleStartTimeError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:      "garden",
					MaxPlants: &one,
					LightSchedule: &pkg.LightSchedule{
						Duration: "1m",
					},
				},
			},
			"missing required light_schedule.start_time field",
		},
		{
			"BadDurationError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:      "garden",
					MaxPlants: &one,
					LightSchedule: &pkg.LightSchedule{
						Duration: "NOT A DURATION",
					},
				},
			},
			"invalid duration format for light_schedule.duration: NOT A DURATION",
		},
		{
			"DurationGreaterThanOrEqualTo24HoursError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:      "garden",
					MaxPlants: &one,
					LightSchedule: &pkg.LightSchedule{
						Duration: "25h",
					},
				},
			},
			"invalid light_schedule.duration >= 24 hours: 25h",
		},
		{
			"BadStartTimeError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:      "garden",
					MaxPlants: &one,
					LightSchedule: &pkg.LightSchedule{
						Duration:  "1m",
						StartTime: "NOT A TIME",
					},
				},
			},
			"invalid time format for light_schedule.start_time: NOT A TIME",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		gr := &GardenRequest{
			Garden: &pkg.Garden{
				Name:      "garden",
				MaxPlants: &one,
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

func TestUpdateGardenRequest(t *testing.T) {
	now := time.Now()
	zero := uint(0)
	tests := []struct {
		name string
		gr   *UpdateGardenRequest
		err  string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required Garden fields",
		},
		{
			"EmptyGardenError",
			&UpdateGardenRequest{},
			"missing required Garden fields",
		},
		{
			"InvalidNameErrorError$",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					Name: "garden$",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError#",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					Name: "garden#",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError*",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					Name: "garden*",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError>",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					Name: "garden>",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError+",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					Name: "garden+",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"InvalidNameErrorError/",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					Name: "garden/",
				},
			},
			"one or more invalid characters in Garden name",
		},
		{
			"CreatingPlantsNotAllowedError",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					Plants: map[xid.ID]*pkg.Plant{
						xid.New(): {},
					},
				},
			},
			"cannot add or modify Plants with this request",
		},
		{
			"BadDurationError",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					LightSchedule: &pkg.LightSchedule{
						Duration: "NOT A DURATION",
					},
				},
			},
			"invalid duration format for light_schedule.duration: NOT A DURATION",
		},
		{
			"DurationGreaterThanOrEqualTo24HoursError",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					LightSchedule: &pkg.LightSchedule{
						Duration: "25h",
					},
				},
			},
			"invalid light_schedule.duration >= 24 hours: 25h",
		},
		{
			"InvalidLightScheduleStartTimeError",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					LightSchedule: &pkg.LightSchedule{
						StartTime: "NOT A TIME",
					},
				},
			},
			"invalid time format for light_schedule.start_time: NOT A TIME",
		},
		{
			"EndDateError",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					EndDate: &now,
				},
			},
			"to end-date a Garden, please use the DELETE endpoint",
		},
		{
			"MaxPlantsZeroError",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					MaxPlants: &zero,
				},
			},
			"max_plants must not be 0",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		gr := &UpdateGardenRequest{
			Garden: &pkg.Garden{
				Name: "garden",
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := gr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading UpdateGardenRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.gr.Bind(r)
			if err == nil {
				t.Error("Expected error reading UpdateGardenRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
