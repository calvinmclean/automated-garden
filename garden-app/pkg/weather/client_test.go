package weather

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPatch(t *testing.T) {
	tests := []struct {
		name      string
		newConfig *Config
	}{
		{
			"PatchName",
			&Config{Name: "new_name"},
		},
		{
			"PatchType",
			&Config{Type: "other_type"},
		},
		{
			"PatchOptions",
			&Config{Options: map[string]any{
				"key": "value",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			err := c.Patch(tt.newConfig)
			require.Nil(t, err)
			assert.Equal(t, tt.newConfig, c)
		})
	}
}

func TestNewWeatherClientInvalidType(t *testing.T) {
	_, err := NewClient(&Config{Name: "test", Type: "DNE"}, func(m map[string]any) error { return nil })
	assert.Error(t, err)
	assert.Equal(t, "invalid type 'DNE'", err.Error())
}

func TestNewWeatherClientOpenMeteo(t *testing.T) {
	client, err := NewClient(&Config{
		Name: "test",
		Type: "openmeteo",
		Options: map[string]any{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
	}, func(m map[string]any) error { return nil })
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewWeatherClientOpenMeteoInvalidConfig(t *testing.T) {
	_, err := NewClient(&Config{
		Name:    "test",
		Type:    "openmeteo",
		Options: map[string]any{},
	}, func(m map[string]any) error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "latitude and longitude must be provided")
}

func TestCachedWeatherClient(t *testing.T) {
	client, err := NewClient(&Config{
		Name: "test",
		Type: "fake",
		Options: map[string]any{
			"rain_mm":              25.4,
			"rain_interval":        "24h",
			"avg_high_temperature": 40,
		},
	}, func(m map[string]any) error { return nil })
	assert.NoError(t, err)
	assert.NotNil(t, client)

	t.Run("GetTotalRain", func(t *testing.T) {
		rain, err := client.GetTotalRain(context.Background(), 24*time.Hour)
		assert.NoError(t, err)

		rainFromCache, err := client.GetTotalRain(context.Background(), 24*time.Hour)
		assert.NoError(t, err)

		assert.Equal(t, rain, rainFromCache)
	})

	t.Run("GetAverageHighTemperature", func(t *testing.T) {
		temp, err := client.GetAverageHighTemperature(context.Background(), 24*time.Hour)
		assert.NoError(t, err)

		tempFromCache, err := client.GetAverageHighTemperature(context.Background(), 24*time.Hour)
		assert.NoError(t, err)

		assert.Equal(t, temp, tempFromCache)
	})
}

func TestEndDated(t *testing.T) {
	assert.False(t, (&Config{}).EndDated())
}
