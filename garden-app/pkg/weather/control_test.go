package weather

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScale(t *testing.T) {
	baseline := float32(90)
	factor := float32(0.5)
	r := float32(30)
	sc := ScaleControl{
		BaselineTemperature: &baseline,
		Factor:              &factor,
		Range:               &r,
	}

	tests := []struct {
		name             string
		input            float32
		expectedFactor   float32
		expectedDuration time.Duration
	}{
		{
			"ScaleUpABit",
			100,
			1 + 1.0/6,
			35 * time.Minute,
		},
		{
			"MaxScaleUp",
			120,
			1.5,
			45 * time.Minute,
		},
		{
			"BeyondMaxScaleUp",
			130,
			1.5,
			45 * time.Minute,
		},
		{
			"ScaleDownABit",
			80,
			1 - 1.0/6,
			25 * time.Minute,
		},
		{
			"MaxScaleDown",
			60,
			0.5,
			15 * time.Minute,
		},
		{
			"BeyondMaxScaleDown",
			50,
			0.5,
			15 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scale := sc.Scale(tt.input)
			assert.Equal(t, tt.expectedFactor, scale)
			baseDuration := time.Minute * 30
			scaledDuration := time.Duration(int64(float32(baseDuration) * scale)).Round(time.Second)
			assert.Equal(t, tt.expectedDuration, scaledDuration)
		})
	}
}
