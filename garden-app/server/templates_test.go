package server

import (
	"bytes"
	"html/template"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"

	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in       time.Duration
		expected string
	}{
		{0, "0s"},
		{100 * time.Millisecond, "0s"},
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

func TestNextWaterDetailZeroDuration(t *testing.T) {
	r := httptest.NewRequest("GET", "/water_schedules?refresh=true", nil)
	tmpl, err := template.New("zones.html").Funcs(templateFuncs(r)).ParseFS(templates, "templates/zones.html")
	require.NoError(t, err)
	nextTime := time.Date(2026, time.October, 16, 5, 0, 0, 0, time.UTC)
	var output bytes.Buffer
	err = tmpl.ExecuteTemplate(&output, "nextWaterDetail", NextWaterDetails{
		Time:     &nextTime,
		Duration: &pkg.Duration{},
	})
	require.NoError(t, err)
	require.Contains(t, output.String(), "Watering will be skipped")
	require.NotContains(t, output.String(), "Watering for")
	require.Contains(t, output.String(), `datetime="2026-10-16T05:00:00Z"`)
}
