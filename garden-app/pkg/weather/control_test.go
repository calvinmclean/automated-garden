package weather

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func float64Ptr(f float64) *float64 {
	return &f
}

func TestControlPatch(t *testing.T) {
	tests := []struct {
		name       string
		newControl *Control
	}{
		{
			"PatchRain.InputMin",
			&Control{
				Rain: &WeatherScaler{
					InputMin: float64Ptr(0.0),
				},
			},
		},
		{
			"PatchRain.InputMax",
			&Control{
				Rain: &WeatherScaler{
					InputMax: float64Ptr(25.4),
				},
			},
		},
		{
			"PatchRain.FactorMin",
			&Control{
				Rain: &WeatherScaler{
					FactorMin: float64Ptr(0.5),
				},
			},
		},
		{
			"PatchRain.FactorMax",
			&Control{
				Rain: &WeatherScaler{
					FactorMax: float64Ptr(1.0),
				},
			},
		},
		{
			"PatchRain.ClientID",
			&Control{
				Rain: &WeatherScaler{
					ClientID: xid.New(),
				},
			},
		},
		{
			"PatchTemperature.InputMin",
			&Control{
				Temperature: &WeatherScaler{
					InputMin: float64Ptr(20.0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"WithNilExisting", func(t *testing.T) {
			c := &Control{}
			c.Patch(tt.newControl)
			assert.Equal(t, tt.newControl, c)
		})
		t.Run(tt.name+"WithAllExisting", func(t *testing.T) {
			if tt.newControl.Rain == nil {
				tt.newControl.Rain = &WeatherScaler{}
			}
			if tt.newControl.Temperature == nil {
				tt.newControl.Temperature = &WeatherScaler{}
			}
			c := &Control{
				Rain:        &WeatherScaler{},
				Temperature: &WeatherScaler{},
			}
			c.Patch(tt.newControl)
			assert.Equal(t, tt.newControl, c)
		})
	}
}
