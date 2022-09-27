package fake

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTotalRain(t *testing.T) {
	tests := []struct {
		name           string
		rainMM         float32
		rainInterval   string
		inputHours     int
		expectedResult float32
	}{
		{
			"OneInchDaily_24h",
			25.4,
			"24h",
			24,
			25.4,
		},
		{
			"OneInchDaily_48h",
			25.4,
			"24h",
			48,
			50.8,
		},
		{
			"OneInchDaily_36h",
			25.4,
			"24h",
			36,
			38.1,
		},
		{
			"OneInchEveryOtherDay_24h",
			25.4,
			"48h",
			24,
			12.7,
		},
		{
			"OneInchEveryOtherDay_48h",
			25.4,
			"48h",
			48,
			25.4,
		},
		{
			"OneInchEveryOtherDay_36h",
			25.4,
			"48h",
			36,
			19.05,
		},
		{
			"OneInchEveryOtherDay_72h",
			25.4,
			"48h",
			72,
			38.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(map[string]interface{}{
				"rain_mm":       tt.rainMM,
				"rain_interval": tt.rainInterval,
			})
			assert.NoError(t, err)

			totalRain, err := client.GetTotalRain(time.Duration(tt.inputHours) * time.Hour)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, totalRain)
		})
	}
}
