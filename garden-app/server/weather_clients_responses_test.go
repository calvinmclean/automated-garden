package server

import (
	"net/http/httptest"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestWeatherClientRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *WeatherClientRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WeatherClient fields",
		},
		{
			"EmptyError",
			&WeatherClientRequest{},
			"missing required WeatherClient fields",
		},
		{
			"EmptyTypeError",
			&WeatherClientRequest{
				Config: &weather.Config{},
			},
			"missing required type field",
		},
		{
			"EmptyOptionsError",
			&WeatherClientRequest{
				Config: &weather.Config{
					Type: "fake",
				},
			},
			"missing required options field",
		},
		{
			"ErrorCreatingClientWithConfigs",
			&WeatherClientRequest{
				Config: &weather.Config{
					Type: "fake",
					Options: map[string]interface{}{
						"key": "value",
					},
				},
			},
			"failed to create valid client using config: time: invalid duration \"\"",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		req := &WeatherClientRequest{
			Config: createExampleWeatherClientConfig(),
		}
		r := httptest.NewRequest("", "/", nil)
		err := req.Bind(r)
		assert.NoError(t, err)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.req.Bind(r)
			assert.Error(t, err)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestUpdateWeatherClientRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *UpdateWeatherClientRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required WeatherClient fields",
		},
		{
			"EmptyWeatherClientError",
			&UpdateWeatherClientRequest{},
			"missing required WeatherClient fields",
		},
		{
			"ManualSpecificationOfIDError",
			&UpdateWeatherClientRequest{
				Config: &weather.Config{ID: xid.New()},
			},
			"updating ID is not allowed",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		wsr := &UpdateWeatherClientRequest{
			Config: &weather.Config{
				Type: "fake",
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := wsr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading WeatherClientRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.req.Bind(r)
			if err == nil {
				t.Error("Expected error reading WeatherClientRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
