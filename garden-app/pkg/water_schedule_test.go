package pkg

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/stretchr/testify/assert"
)

func TestWaterScheduleEndDated(t *testing.T) {
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
			ws := &WaterSchedule{EndDate: tt.endDate}
			if ws.EndDated() != tt.expected {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.expected, ws.EndDated())
			}
		})
	}
}

func TestWaterSchedulePatch(t *testing.T) {
	one := 1
	float := float32(1)
	now := time.Now()
	tests := []struct {
		name             string
		newWaterSchedule *WaterSchedule
	}{
		{
			"PatchDuration",
			&WaterSchedule{
				Duration: &Duration{time.Second, ""},
			},
		},
		{
			"PatchInterval",
			&WaterSchedule{
				Interval: &Duration{time.Hour * 2, ""},
			},
		},
		{
			"PatchWeatherControl.SoilMoisture.MinimumMoisture",
			&WaterSchedule{
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &one,
					},
				},
			},
		},
		{
			"PatchStartTime",
			&WaterSchedule{
				StartTime: &now,
			},
		},
		{
			"PatchWeatherControl.Temperature",
			&WaterSchedule{
				WeatherControl: &weather.Control{
					Rain: &weather.ScaleControl{
						BaselineValue: &float,
						Factor:        &float,
						Range:         &float,
					},
				},
			},
		},
		{
			"PatchWeatherControl.Temperature",
			&WaterSchedule{
				WeatherControl: &weather.Control{
					Temperature: &weather.ScaleControl{
						BaselineValue: &float,
						Factor:        &float,
						Range:         &float,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &WaterSchedule{}
			ws.Patch(tt.newWaterSchedule)
			assert.Equal(t, tt.newWaterSchedule, ws)
		})
	}

	t.Run("PatchDoesNotAddEndDate", func(t *testing.T) {
		now := time.Now()
		ws := &WaterSchedule{}

		ws.Patch(&WaterSchedule{EndDate: &now})

		if ws.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", ws.EndDate)
		}
	})

	t.Run("PatchRemoveEndDate", func(t *testing.T) {
		now := time.Now()
		ws := &WaterSchedule{
			EndDate: &now,
		}

		ws.Patch(&WaterSchedule{})

		if ws.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", ws.EndDate)
		}
	})
}
