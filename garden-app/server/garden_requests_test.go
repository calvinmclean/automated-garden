package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
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
				GardenAction: &action.GardenAction{},
			},
			"missing required action fields",
		},
	}

	t.Run("SuccessfulLightAction", func(t *testing.T) {
		ar := &GardenActionRequest{
			GardenAction: &action.GardenAction{
				Light: &action.LightAction{
					State: pkg.LightStateOn,
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
			GardenAction: &action.GardenAction{
				Stop: &action.StopAction{},
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
			"MissingTopicPrefixErrorError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name: "garden",
				},
			},
			"missing required topic_prefix field",
		},
		{
			"InvalidTopicPrefixError$",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden$",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError#",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden#",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError*",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden*",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError>",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden>",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError+",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden+",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError/",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden/",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"MissingMaxZonesError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden",
				},
			},
			"missing required max_zones field",
		},
		{
			"MaxZonesZeroError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden",
					MaxZones:    &zero,
				},
			},
			"max_zones must not be 0",
		},
		{
			"CreatingPlantsNotAllowedError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden",
					MaxZones:    &one,
					Plants: map[xid.ID]*pkg.Plant{
						xid.New(): {},
					},
				},
			},
			"cannot add or modify Plants with this request",
		},
		{
			"CreatingZonesNotAllowedError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden",
					MaxZones:    &one,
					Zones: map[xid.ID]*pkg.Zone{
						xid.New(): {},
					},
				},
			},
			"cannot add or modify Zones with this request",
		},
		{
			"EmptyLightScheduleDurationError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden",
					MaxZones:    &one,
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
					Name:        "garden",
					TopicPrefix: "garden",
					MaxZones:    &one,
					LightSchedule: &pkg.LightSchedule{
						Duration: &pkg.Duration{Duration: time.Minute},
					},
				},
			},
			"missing required light_schedule.start_time field",
		},
		{
			"DurationGreaterThanOrEqualTo24HoursError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden",
					MaxZones:    &one,
					LightSchedule: &pkg.LightSchedule{
						Duration: &pkg.Duration{Duration: 25 * time.Hour},
					},
				},
			},
			"invalid light_schedule.duration >= 24 hours: 25h0m0s",
		},
		{
			"BadStartTimeError",
			&GardenRequest{
				Garden: &pkg.Garden{
					Name:        "garden",
					TopicPrefix: "garden",
					MaxZones:    &one,
					LightSchedule: &pkg.LightSchedule{
						Duration:  &pkg.Duration{Duration: time.Minute},
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
				TopicPrefix: "garden",
				Name:        "garden",
				MaxZones:    &one,
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
			"InvalidTopicPrefixError$",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					TopicPrefix: "garden$",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError#",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					TopicPrefix: "garden#",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError*",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					TopicPrefix: "garden*",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError>",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					TopicPrefix: "garden>",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError+",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					TopicPrefix: "garden+",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError/",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					TopicPrefix: "garden/",
				},
			},
			"one or more invalid characters in Garden topic_prefix",
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
			"DurationGreaterThanOrEqualTo24HoursError",
			&UpdateGardenRequest{
				Garden: &pkg.Garden{
					LightSchedule: &pkg.LightSchedule{
						Duration: &pkg.Duration{Duration: 25 * time.Hour},
					},
				},
			},
			"invalid light_schedule.duration >= 24 hours: 25h0m0s",
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
					MaxZones: &zero,
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
