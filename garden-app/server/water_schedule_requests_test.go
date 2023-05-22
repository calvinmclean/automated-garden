package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestWaterScheduleRequest(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		pr   *WaterScheduleRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WaterSchedule fields",
		},
		{
			"EmptyError",
			&WaterScheduleRequest{},
			"missing required WaterSchedule fields",
		},
		{
			"EmptyIntervalError",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Duration: &pkg.Duration{Duration: time.Second},
				},
			},
			"missing required interval field",
		},
		{
			"EmptyDurationError",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval: &pkg.Duration{Duration: time.Hour * 24},
				},
			},
			"missing required duration field",
		},
		{
			"EmptyStartTimeError",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval: &pkg.Duration{Duration: time.Hour * 24},
					Duration: &pkg.Duration{Duration: time.Second},
				},
			},
			"missing required start_time field",
		},
		{
			"EmptyWeatherControlBaselineTemperature",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval:  &pkg.Duration{Duration: time.Hour * 24},
					Duration:  &pkg.Duration{Duration: time.Second},
					StartTime: &now,
					WeatherControl: &weather.Control{
						Temperature: &weather.ScaleControl{
							Factor: float32Pointer(0.5),
							Range:  float32Pointer(10),
						},
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: baseline_value",
		},
		{
			"EmptyWeatherControlFactor",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval:  &pkg.Duration{Duration: time.Hour * 24},
					Duration:  &pkg.Duration{Duration: time.Second},
					StartTime: &now,
					WeatherControl: &weather.Control{
						Temperature: &weather.ScaleControl{
							BaselineValue: float32Pointer(27),
							Range:         float32Pointer(10),
						},
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: factor",
		},
		{
			"EmptyWeatherControlRange",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval:  &pkg.Duration{Duration: time.Hour * 24},
					Duration:  &pkg.Duration{Duration: time.Second},
					StartTime: &now,
					WeatherControl: &weather.Control{
						Temperature: &weather.ScaleControl{
							BaselineValue: float32Pointer(27),
							Factor:        float32Pointer(0.5),
						},
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: range",
		},
		{
			"EmptyWeatherControlClientID",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval:  &pkg.Duration{Duration: time.Hour * 24},
					Duration:  &pkg.Duration{Duration: time.Second},
					StartTime: &now,
					WeatherControl: &weather.Control{
						Temperature: &weather.ScaleControl{
							BaselineValue: float32Pointer(27),
							Factor:        float32Pointer(0.5),
							Range:         float32Pointer(10),
						},
					},
				},
			},
			"error validating weather_control: error validating temperature_control: missing required field: client_id",
		},
		{
			"WeatherControlInvalidFactorBig",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval:  &pkg.Duration{Duration: time.Hour * 24},
					Duration:  &pkg.Duration{Duration: time.Second},
					StartTime: &now,
					WeatherControl: &weather.Control{
						Temperature: &weather.ScaleControl{
							BaselineValue: float32Pointer(27),
							Factor:        float32Pointer(2),
							Range:         float32Pointer(10),
						},
					},
				},
			},
			"error validating weather_control: error validating temperature_control: factor must be between 0 and 1",
		},
		{
			"WeatherControlInvalidFactorSmall",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval:  &pkg.Duration{Duration: time.Hour * 24},
					Duration:  &pkg.Duration{Duration: time.Second},
					StartTime: &now,
					WeatherControl: &weather.Control{
						Temperature: &weather.ScaleControl{
							BaselineValue: float32Pointer(27),
							Factor:        float32Pointer(-1),
							Range:         float32Pointer(10),
						},
					},
				},
			},
			"error validating weather_control: error validating temperature_control: factor must be between 0 and 1",
		},
		{
			"WeatherControlInvalidRange",
			&WaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					Interval:  &pkg.Duration{Duration: time.Hour * 24},
					Duration:  &pkg.Duration{Duration: time.Second},
					StartTime: &now,
					WeatherControl: &weather.Control{
						Temperature: &weather.ScaleControl{
							BaselineValue: float32Pointer(27),
							Factor:        float32Pointer(0.5),
							Range:         float32Pointer(-1),
						},
					},
				},
			},
			"error validating weather_control: error validating temperature_control: range must be a positive number",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &WaterScheduleRequest{
			WaterSchedule: &pkg.WaterSchedule{
				Duration:  &pkg.Duration{Duration: time.Second},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: &now,
				WeatherControl: &weather.Control{
					Temperature: &weather.ScaleControl{
						BaselineValue: float32Pointer(27),
						Factor:        float32Pointer(0.5),
						Range:         float32Pointer(10),
						ClientID:      xid.New(),
					},
				},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := pr.Bind(r)
		assert.NoError(t, err)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			assert.Error(t, err)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestUpdateWaterScheduleRequest(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	tests := []struct {
		name string
		pr   *UpdateWaterScheduleRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WaterSchedule fields",
		},
		{
			"EmptyWaterScheduleError",
			&UpdateWaterScheduleRequest{},
			"missing required WaterSchedule fields",
		},
		{
			"ManualSpecificationOfIDError",
			&UpdateWaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{ID: xid.New()},
			},
			"updating ID is not allowed",
		},
		{
			"StartTimeInPastError",
			&UpdateWaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					StartTime: &past,
				},
			},
			"unable to set start_time to time in the past",
		},
		{
			"EndDateError",
			&UpdateWaterScheduleRequest{
				WaterSchedule: &pkg.WaterSchedule{
					EndDate: &now,
				},
			},
			"to end-date a WaterSchedule, please use the DELETE endpoint",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		wsr := &UpdateWaterScheduleRequest{
			WaterSchedule: &pkg.WaterSchedule{
				Interval: &pkg.Duration{Duration: time.Hour},
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := wsr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading WaterScheduleRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading WaterScheduleRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
