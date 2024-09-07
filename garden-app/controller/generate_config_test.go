package controller

import (
	"os"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/stretchr/testify/assert"
)

func TestGenerateConfig(_ *testing.T) {
	GenerateConfig(Config{}, true, true, true, false, false)
	GenerateConfig(Config{}, true, true, true, false, false)
	os.RemoveAll("config.h")
}

func TestGenerateMainConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		expectedOutput string
	}{
		{
			"OneZoneNoSpecialFeatures",
			Config{
				NestedConfig: NestedConfig{
					Zones: []ZoneConfig{
						{
							PumpPin:  "GPIO_NUM_18",
							ValvePin: "GPIO_NUM_16",
						},
					},
					TopicPrefix:      "garden",
					DefaultWaterTime: 5 * time.Second,
				},
				MQTTConfig: mqtt.Config{
					Broker: "localhost",
					Port:   1883,
				},
			},
			`#ifndef config_h
#define config_h

#define TOPIC_PREFIX "garden"

#define QUEUE_SIZE 10

#define ENABLE_WIFI
#ifdef ENABLE_WIFI
#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define ENABLE_MQTT_LOGGING

#endif

#define NUM_ZONES 1
#define ZONES { { GPIO_NUM_18, GPIO_NUM_16, GPIO_NUM_MAX, GPIO_NUM_MAX } }
#define DEFAULT_WATER_TIME 5000

#endif
`,
		},
		{
			"OneZoneAllSpecialFeatures",
			Config{
				NestedConfig: NestedConfig{
					Zones: []ZoneConfig{
						{
							PumpPin:           "GPIO_NUM_18",
							ValvePin:          "GPIO_NUM_16",
							ButtonPin:         "GPIO_NUM_19",
							MoistureSensorPin: "GPIO_NUM_36",
						},
					},
					TopicPrefix:                 "garden",
					DefaultWaterTime:            5 * time.Second,
					LightPin:                    "GPIO_NUM_32",
					EnableButtons:               true,
					StopButtonPin:               "GPIO_NUM_23",
					EnableMoistureSensor:        true,
					MoistureInterval:            5 * time.Second,
					PublishHealth:               true,
					HealthInterval:              1 * time.Minute,
					PublishTemperatureHumidity:  true,
					TemperatureHumidityInterval: 5 * time.Minute,
					TemperatureHumidityPin:      "GPIO_NUM_27",
				},
				MQTTConfig: mqtt.Config{
					Broker: "localhost",
					Port:   1883,
				},
			},
			`#ifndef config_h
#define config_h

#define TOPIC_PREFIX "garden"

#define QUEUE_SIZE 10

#define ENABLE_WIFI
#ifdef ENABLE_WIFI
#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define ENABLE_MQTT_HEALTH

#define ENABLE_MQTT_LOGGING

#endif

#define NUM_ZONES 1
#define ZONES { { GPIO_NUM_18, GPIO_NUM_16, GPIO_NUM_19, GPIO_NUM_36 } }
#define DEFAULT_WATER_TIME 5000

#define LIGHT_PIN GPIO_NUM_32

#define ENABLE_BUTTONS
#ifdef ENABLE_BUTTONS
#define STOP_BUTTON_PIN GPIO_NUM_23
#endif

#ifdef ENABLE_MOISTURE_SENSORS AND ENABLE_WIFI
#define MOISTURE_SENSOR_AIR_VALUE 3415
#define MOISTURE_SENSOR_WATER_VALUE 1362
#define MOISTURE_SENSOR_INTERVAL 5000
#endif

#define ENABLE_DHT22
#ifdef ENABLE_DHT22
#define DHT22_PIN GPIO_NUM_27
#define DHT22_INTERVAL 300000
#endif
#endif
`,
		},
		{
			"OneZoneNoSpecialFeaturesDisableWatering",
			Config{
				NestedConfig: NestedConfig{
					Zones: []ZoneConfig{
						{
							PumpPin:  "GPIO_NUM_18",
							ValvePin: "GPIO_NUM_16",
						},
					},
					TopicPrefix:      "garden",
					DefaultWaterTime: 5 * time.Second,
					DisableWatering:  true,
				},
				MQTTConfig: mqtt.Config{
					Broker: "localhost",
					Port:   1883,
				},
			},
			`#ifndef config_h
#define config_h

#define TOPIC_PREFIX "garden"

#define QUEUE_SIZE 10

#define ENABLE_WIFI
#ifdef ENABLE_WIFI
#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define ENABLE_MQTT_LOGGING

#endif

#define DISABLE_WATERING
#define NUM_ZONES 1
#define ZONES { { GPIO_NUM_18, GPIO_NUM_16, GPIO_NUM_MAX, GPIO_NUM_MAX } }
#define DEFAULT_WATER_TIME 5000

#endif
`,
		},
		{
			"MultipleZonesNoSpecialFeatures",
			Config{
				NestedConfig: NestedConfig{
					Zones: []ZoneConfig{
						{
							PumpPin:  "GPIO_NUM_18",
							ValvePin: "GPIO_NUM_16",
						},
						{
							PumpPin:  "GPIO_NUM_18",
							ValvePin: "GPIO_NUM_16",
						},
						{
							PumpPin:  "GPIO_NUM_18",
							ValvePin: "GPIO_NUM_16",
						},
						{
							PumpPin:  "GPIO_NUM_18",
							ValvePin: "GPIO_NUM_16",
						},
					},
					TopicPrefix:      "garden",
					DefaultWaterTime: 5 * time.Second,
				},
				MQTTConfig: mqtt.Config{
					Broker: "localhost",
					Port:   1883,
				},
			},
			`#ifndef config_h
#define config_h

#define TOPIC_PREFIX "garden"

#define QUEUE_SIZE 10

#define ENABLE_WIFI
#ifdef ENABLE_WIFI
#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define ENABLE_MQTT_LOGGING

#endif

#define NUM_ZONES 4
#define ZONES { { GPIO_NUM_18, GPIO_NUM_16, GPIO_NUM_MAX, GPIO_NUM_MAX }, { GPIO_NUM_18, GPIO_NUM_16, GPIO_NUM_MAX, GPIO_NUM_MAX }, { GPIO_NUM_18, GPIO_NUM_16, GPIO_NUM_MAX, GPIO_NUM_MAX }, { GPIO_NUM_18, GPIO_NUM_16, GPIO_NUM_MAX, GPIO_NUM_MAX } }
#define DEFAULT_WATER_TIME 5000

#endif
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := generateMainConfig(tt.config, false)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, config)
		})
	}
}

func TestGenerateWifiConfig(t *testing.T) {
	config, err := generateWiFiConfig(WifiConfig{
		SSID:     "ssid",
		Password: "password",
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, `#ifndef wifi_config_h
#define wifi_config_h

#define SSID "ssid"
#define PASSWORD "password"

#endif
`, config)
}
