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
		BaselineValue: &baseline,
		Factor:        &factor,
		Range:         &r,
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

func TestInvertedScaleDownOnly(t *testing.T) {
	sc := ScaleControl{
		BaselineValue: float32Pointer(25.4),
		Factor:        float32Pointer(0.5),
		Range:         float32Pointer(12.7),
	}

	// Any amount of rain will scale watering down, 2 inches will stop all watering
	fullRangeScale := ScaleControl{
		BaselineValue: float32Pointer(0),
		Factor:        float32Pointer(0),
		Range:         float32Pointer(50),
	}

	tests := []struct {
		name             string
		sc               ScaleControl
		input            float32
		expectedFactor   float32
		expectedDuration time.Duration
	}{
		{
			"ValueBelowThresholdNoChange",
			sc,
			20,
			1,
			30 * time.Minute,
		},
		{
			"1/2RangePastBaselineScales75%",
			sc,
			25.4 + 6.35,
			0.75,
			22*time.Minute + 30*time.Second,
		},
		{
			"FullRangePastBaselineScales50%",
			sc,
			25.4 + 12.7,
			0.5,
			15 * time.Minute,
		},
		{
			"BeyondRangeMaxesScaleFactor",
			sc,
			50,
			0.5,
			15 * time.Minute,
		},
		{
			"ScaleToZero",
			ScaleControl{
				BaselineValue: float32Pointer(25.4),
				Factor:        float32Pointer(0),
				Range:         float32Pointer(10),
			},
			35.4,
			0,
			0,
		},
		{
			"FullRangeNoScale",
			fullRangeScale,
			0,
			1,
			30 * time.Minute,
		},
		{
			"FullRangeHalfway",
			fullRangeScale,
			25,
			0.5,
			15 * time.Minute,
		},
		{
			"FullRangeScaleToZero",
			fullRangeScale,
			50,
			0,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scale := tt.sc.InvertedScaleDownOnly(tt.input)
			assert.Equal(t, tt.expectedFactor, scale)
			baseDuration := time.Minute * 30
			scaledDuration := time.Duration(int64(float32(baseDuration) * scale)).Round(time.Second)
			assert.Equal(t, tt.expectedDuration, scaledDuration)
		})
	}
}

func float32Pointer(n float64) *float32 {
	f := float32(n)
	return &f
}
