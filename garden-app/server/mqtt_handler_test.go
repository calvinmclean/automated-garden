package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseWaterMessage(t *testing.T) {
	tests := []struct {
		in            string
		expectedPos   int
		waterDuration time.Duration
	}{
		{
			"water,zone=1 millis=6000",
			1, 6000 * time.Millisecond,
		},
		{
			"water,zone=100 millis=1",
			100, 1 * time.Millisecond,
		},
		{
			"water,zone=0 millis=0",
			0, 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			zonePosition, waterDuration, err := parseWaterMessage([]byte(tt.in))
			require.NoError(t, err)
			require.Equal(t, tt.expectedPos, zonePosition)
			require.Equal(t, tt.waterDuration, waterDuration)
		})
	}
}
