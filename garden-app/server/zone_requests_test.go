package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/rs/xid"
)

func TestZoneRequest(t *testing.T) {
	pos := uint(0)
	now := time.Now()
	tests := []struct {
		name string
		pr   *ZoneRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required Zone fields",
		},
		{
			"EmptyZoneError",
			&ZoneRequest{},
			"missing required Zone fields",
		},
		{
			"EmptyPositionError",
			&ZoneRequest{
				Zone: &pkg.Zone{
					Name: "zone",
				},
			},
			"missing required zone_position field",
		},
		{
			"EmptyWaterScheduleError",
			&ZoneRequest{
				Zone: &pkg.Zone{
					Name:     "zone",
					Position: &pos,
				},
			},
			"missing required water_schedule field",
		},
		{
			"EmptyWaterScheduleIntervalError",
			&ZoneRequest{
				Zone: &pkg.Zone{
					Name:     "zone",
					Position: &pos,
					WaterSchedule: &pkg.WaterSchedule{
						Duration: "1000ms",
					},
				},
			},
			"missing required water_schedule.interval field",
		},
		{
			"EmptyWaterScheduleDurationError",
			&ZoneRequest{
				Zone: &pkg.Zone{
					Name:     "zone",
					Position: &pos,
					WaterSchedule: &pkg.WaterSchedule{
						Interval: "24h",
					},
				},
			},
			"missing required water_schedule.duration field",
		},
		{
			"EmptyWaterScheduleStartTimeError",
			&ZoneRequest{
				Zone: &pkg.Zone{
					Name:     "zone",
					Position: &pos,
					WaterSchedule: &pkg.WaterSchedule{
						Interval: "24h",
						Duration: "1000ms",
					},
				},
			},
			"missing required water_schedule.start_time field",
		},
		{
			"InvalidDurationStringError",
			&ZoneRequest{
				Zone: &pkg.Zone{
					Name:     "zone",
					Position: &pos,
					WaterSchedule: &pkg.WaterSchedule{
						Interval:  "24h",
						Duration:  "NOT A DURATION",
						StartTime: &now,
					},
				},
			},
			"invalid duration format for water_schedule.duration: NOT A DURATION",
		},
		{
			"EmptyNameError",
			&ZoneRequest{
				Zone: &pkg.Zone{
					Position: &pos,
					WaterSchedule: &pkg.WaterSchedule{
						Interval:  "24h",
						Duration:  "1000ms",
						StartTime: &now,
					},
				},
			},
			"missing required name field",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &ZoneRequest{
			Zone: &pkg.Zone{
				Name:     "zone",
				Position: &pos,
				WaterSchedule: &pkg.WaterSchedule{
					Duration:  "1000ms",
					Interval:  "24h",
					StartTime: &now,
				},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := pr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading ZoneRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading ZoneRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}

func TestUpdateZoneRequest(t *testing.T) {
	pp := uint(0)
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	tests := []struct {
		name string
		pr   *UpdateZoneRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required Zone fields",
		},
		{
			"EmptyZoneError",
			&UpdateZoneRequest{},
			"missing required Zone fields",
		},
		{
			"ManualSpecificationOfIDError",
			&UpdateZoneRequest{
				Zone: &pkg.Zone{ID: xid.New()},
			},
			"updating ID is not allowed",
		},
		{
			"InvalidWaterScheduleDurationError",
			&UpdateZoneRequest{
				Zone: &pkg.Zone{
					WaterSchedule: &pkg.WaterSchedule{
						Duration: "NOT A DURATION",
					},
				},
			},
			"invalid duration format for water_schedule.duration: NOT A DURATION",
		},
		{
			"StartTimeInPastError",
			&UpdateZoneRequest{
				Zone: &pkg.Zone{
					WaterSchedule: &pkg.WaterSchedule{
						StartTime: &past,
					},
				},
			},
			"unable to set water_schedule.start_time to time in the past",
		},
		{
			"EndDateError",
			&UpdateZoneRequest{
				Zone: &pkg.Zone{
					EndDate: &now,
				},
			},
			"to end-date a Zone, please use the DELETE endpoint",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &UpdateZoneRequest{
			Zone: &pkg.Zone{
				Name:     "zone",
				Position: &pp,
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := pr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading ZoneRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading ZoneRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}

func TestZoneActionRequest(t *testing.T) {
	tests := []struct {
		name string
		ar   *ZoneActionRequest
		err  string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required action fields",
		},
		{
			"EmptyActionError",
			&ZoneActionRequest{},
			"missing required action fields",
		},
		{
			"EmptyZoneActionError",
			&ZoneActionRequest{
				ZoneAction: &action.ZoneAction{},
			},
			"missing required action fields",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		ar := &ZoneActionRequest{
			ZoneAction: &action.ZoneAction{
				Water: &action.WaterAction{},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := ar.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading ZoneActionRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.ar.Bind(r)
			if err == nil {
				t.Error("Expected error reading ZoneActionRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}