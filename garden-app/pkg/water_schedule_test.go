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
			"PatchName",
			&WaterSchedule{
				Name: "new name",
			},
		},
		{
			"PatchDescription",
			&WaterSchedule{
				Description: "description",
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
		{
			"PatchActivePeriod.StartMonth",
			&WaterSchedule{
				ActivePeriod: &ActivePeriod{
					StartMonth: "new month",
				},
			},
		},
		{
			"PatchActivePeriod.EndMonth",
			&WaterSchedule{
				ActivePeriod: &ActivePeriod{
					EndMonth: "new month",
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

func TestActivePeriodValidate(t *testing.T) {
	tests := []struct {
		name        string
		input       *ActivePeriod
		expectedErr string
	}{
		{
			"ValidLongNames",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "February",
			},
			"",
		},
		{
			"InvalidStart",
			&ActivePeriod{
				StartMonth: "anuary",
				EndMonth:   "February",
			},
			`invalid StartMonth: parsing time "anuary" as "January": cannot parse "anuary" as "January"`,
		},
		{
			"InvalidEnd",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "ebruary",
			},
			`invalid EndMonth: parsing time "ebruary" as "January": cannot parse "ebruary" as "January"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err.Error())
			}
		})
	}
}

// These tests have some potential to fail depending on what time of year it is right now, but I'll fix it if it happens
func TestWaterScheduleIsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		ap       *ActivePeriod
		expected bool
	}{
		{
			"AllYear",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "December",
			},
			true,
		},
		{
			"CurrentlyStartMonth",
			&ActivePeriod{
				StartMonth: now.Month().String(),
				EndMonth:   "December",
			},
			true,
		},
		{
			"CurrentlyEndMonth",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   now.Month().String(),
			},
			true,
		},
		{
			"CurrentlyOneMonthBeforeEnd",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   now.AddDate(0, 1, 0).Month().String(),
			},
			true,
		},
		{
			"CurrentlyOneMonthAfterStart",
			&ActivePeriod{
				StartMonth: now.AddDate(0, -1, 0).Month().String(),
				EndMonth:   "December",
			},
			true,
		},
		{
			"CurrentlyOneMonthBeforeStart",
			&ActivePeriod{
				StartMonth: now.AddDate(0, 1, 0).Month().String(),
				EndMonth:   "December",
			},
			false,
		},
		{
			"CurrentlyOneMonthAfterEnd",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   now.AddDate(0, -1, 0).Month().String(),
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, (&WaterSchedule{ActivePeriod: tt.ap}).IsActive())
		})
	}

	t.Run("NoActivePeriod", func(t *testing.T) {
		assert.Equal(t, true, (&WaterSchedule{}).IsActive())
	})
}
