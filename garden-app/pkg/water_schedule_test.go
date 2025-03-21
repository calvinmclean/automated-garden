package pkg

import (
	"fmt"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaterScheduleEndDated(t *testing.T) {
	pastDate := clock.Now().Add(-1 * time.Minute)
	futureDate := clock.Now().Add(time.Minute)
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
	float := float32(1)
	now := clock.Now()
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
			"PatchStartTime",
			&WaterSchedule{
				StartTime: NewStartTime(now),
			},
		},
		{
			"PatchStartDate",
			&WaterSchedule{
				StartDate: &now,
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
			err := ws.Patch(tt.newWaterSchedule)
			require.Nil(t, err)
			assert.Equal(t, tt.newWaterSchedule, ws)
		})
	}

	t.Run("PatchDoesNotAddEndDate", func(t *testing.T) {
		now := clock.Now()
		ws := &WaterSchedule{}

		err := ws.Patch(&WaterSchedule{EndDate: &now})
		require.Nil(t, err)

		if ws.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", ws.EndDate)
		}
	})

	t.Run("PatchRemoveEndDate", func(t *testing.T) {
		now := clock.Now()
		ws := &WaterSchedule{
			EndDate: &now,
		}

		err := ws.Patch(&WaterSchedule{})
		require.Nil(t, err)

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
		{
			"InvalidSameStartEnd",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "January",
			},
			`StartMonth and EndMonth must be different`,
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

func TestWaterScheduleIsActive(t *testing.T) {
	tests := []struct {
		name         string
		currentMonth string
		ap           *ActivePeriod
		expected     bool
	}{
		{
			"AllYear",
			"February",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "December",
			},
			true,
		},
		{
			"CurrentlyStartMonth",
			"February",
			&ActivePeriod{
				StartMonth: "February",
				EndMonth:   "December",
			},
			true,
		},
		{
			"CurrentlyEndMonth",
			"February",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "February",
			},
			true,
		},
		{
			"CurrentlyOneMonthBeforeEnd",
			"February",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "March",
			},
			true,
		},
		{
			"CurrentlyOneMonthAfterStart",
			"February",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "December",
			},
			true,
		},
		{
			"CurrentlyOneMonthBeforeStart",
			"February",
			&ActivePeriod{
				StartMonth: "March",
				EndMonth:   "December",
			},
			false,
		},
		{
			"CurrentlyOneMonthAfterEnd",
			"March",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "February",
			},
			false,
		},
		{
			"WrapAroundYearTrue",
			"February",
			&ActivePeriod{
				StartMonth: "December",
				EndMonth:   "July",
			},
			true,
		},
		{
			"WrapAroundYearStartMonth",
			"December",
			&ActivePeriod{
				StartMonth: "December",
				EndMonth:   "July",
			},
			true,
		},
		{
			"WrapAroundYearEndMonth",
			"July",
			&ActivePeriod{
				StartMonth: "December",
				EndMonth:   "July",
			},
			true,
		},
		{
			"WrapAroundYearBefore",
			"November",
			&ActivePeriod{
				StartMonth: "December",
				EndMonth:   "July",
			},
			false,
		},
		{
			"WrapAroundYearAfter",
			"August",
			&ActivePeriod{
				StartMonth: "December",
				EndMonth:   "July",
			},
			false,
		},
		{
			"FullYear",
			"January",
			&ActivePeriod{
				StartMonth: "January",
				EndMonth:   "December",
			},
			true,
		},
		{
			"WraparoundWhereCurrentMonthAfterStart",
			"November",
			&ActivePeriod{
				StartMonth: "October",
				EndMonth:   "May",
			},
			true,
		},
		{
			"WraparoundWhereNextYearButBeforeEnd",
			"January",
			&ActivePeriod{
				StartMonth: "October",
				EndMonth:   "May",
			},
			true,
		},
		{
			"WraparoundFalse",
			"June",
			&ActivePeriod{
				StartMonth: "October",
				EndMonth:   "May",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse current month and set to this year
			currentMonth, err := time.Parse("January", tt.currentMonth)
			assert.NoError(t, err)

			currentTime := currentMonth.AddDate(clock.Now().Year(), 0, 0)

			// Check every day in this month
			for currentTime.Month() == currentMonth.Month() {
				t.Run(fmt.Sprintf("Day_%d", currentTime.Day()), func(t *testing.T) {
					assert.Equal(t, tt.expected, (&WaterSchedule{ActivePeriod: tt.ap}).IsActive(currentTime))
				})
				currentTime = currentTime.AddDate(0, 0, 1)
			}
		})
	}

	t.Run("NoActivePeriod", func(t *testing.T) {
		assert.Equal(t, true, (&WaterSchedule{}).IsActive(clock.Now()))
	})
}
