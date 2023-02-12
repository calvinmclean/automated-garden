package cmd

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestViperAutomaticEnv(t *testing.T) {
	envVars := []struct {
		envVar    string
		configKey string
		value     string
	}{
		{
			"GARDEN_APP_INFLUXDB_BUCKET",
			"influxdb.bucket",
			"bucket",
		},
		{
			"GARDEN_APP_MQTT_BROKER",
			"mqtt.broker",
			"localhost",
		},
		{
			"GARDEN_APP_WEB_SERVER_PORT",
			"web_server.port",
			"8080",
		},
		{
			"GARDEN_APP_STORAGE_TYPE",
			"storage.type",
			"TYPE",
		},
		{
			"GARDEN_APP_WEATHER_OPTIONS_CLIENT_SECRET",
			"weather.options.client_secret",
			"WEATHER_SECRET",
		},
		{
			"GARDEN_APP_WEATHER_OPTIONS_CLIENT_ID",
			"weather.options.client_id",
			"CLIENT_ID",
		},
	}

	for _, v := range envVars {
		os.Setenv(v.envVar, v.value)
		assert.Equal(t, v.value, viper.Get(v.configKey))
	}
}
