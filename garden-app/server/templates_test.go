package server

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"

	"github.com/stretchr/testify/require"
)

func TestFormatUpcomingDate(t *testing.T) {
	defer clock.Reset()
	clock.MockTime()

	tests := []struct {
		name     string
		date     *time.Time
		expected string
	}{
		{
			name:     "today",
			date:     ptr(time.Date(2023, time.August, 23, 15, 30, 0, 0, time.UTC)),
			expected: "at 3:30PM",
		},
		{
			name:     "tomorrow",
			date:     ptr(time.Date(2023, time.August, 24, 15, 30, 0, 0, time.UTC)),
			expected: "on Thursday, 24 Aug at 3:30PM",
		},
		{
			name:     "nil date",
			date:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, formatUpcomingDate(tt.date))
		})
	}
}

func TestFormatUntilDate(t *testing.T) {
	defer clock.Reset()
	clock.MockTime()

	tests := []struct {
		name     string
		date     *time.Time
		expected string
	}{
		{
			name:     "today",
			date:     ptr(time.Date(2023, time.August, 23, 15, 30, 0, 0, time.UTC)),
			expected: "3:30PM",
		},
		{
			name:     "tomorrow",
			date:     ptr(time.Date(2023, time.August, 24, 15, 30, 0, 0, time.UTC)),
			expected: "Thursday, 24 Aug at 3:30PM",
		},
		{
			name:     "nil date",
			date:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, formatUntilDate(tt.date))
		})
	}
}

func ptr(t time.Time) *time.Time {
	return &t
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in       time.Duration
		expected string
	}{
		{
			10 * time.Second,
			"10s",
		},
		{
			10 * time.Minute,
			"10m",
		},
		{
			10*time.Minute + 10*time.Second,
			"10m10s",
		},
		{
			5 * time.Hour,
			"5h",
		},
		{
			5*time.Hour + 10*time.Minute,
			"5h10m",
		},
		{
			5*time.Hour + 10*time.Minute + 10*time.Second,
			"5h10m10s",
		},
		{
			48 * time.Hour,
			"2 days",
		},
		{
			48*time.Hour + 10*time.Second,
			"2 days and 10s",
		},
		{
			48*time.Hour + 10*time.Minute,
			"2 days and 10m",
		},
		{
			48*time.Hour + 10*time.Minute + 10*time.Second,
			"2 days and 10m10s",
		},
		{
			48*time.Hour + 5*time.Hour,
			"2 days and 5h",
		},
		{
			48*time.Hour + 5*time.Hour + 10*time.Minute,
			"2 days and 5h10m",
		},
		{
			48*time.Hour + 5*time.Hour + 10*time.Minute + 10*time.Second,
			"2 days and 5h10m10s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.in.String(), func(t *testing.T) {
			out := formatDuration(&pkg.Duration{Duration: tt.in})
			require.Equal(t, tt.expected, out)
		})
	}
}
