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
					TopicPrefix: "garden",
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

#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define NUM_ZONES 1
#define VALVES { GPIO_NUM_16 }
#define PUMPS { GPIO_NUM_18 }

#define LIGHT_ENABLED false
#define LIGHT_PIN GPIO_NUM_MAX

#define ENABLE_DHT22 false
#define DHT22_PIN GPIO_NUM_MAX
#define DHT22_INTERVAL 0
#endif
`,
		},
		{
			"OneZoneAllSpecialFeatures",
			Config{
				NestedConfig: NestedConfig{
					Zones: []ZoneConfig{
						{
							PumpPin:  "GPIO_NUM_18",
							ValvePin: "GPIO_NUM_16",
						},
					},
					TopicPrefix:                 "garden",
					LightPin:                    "GPIO_NUM_32",
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

#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define NUM_ZONES 1
#define VALVES { GPIO_NUM_16 }
#define PUMPS { GPIO_NUM_18 }

#define LIGHT_ENABLED true
#define LIGHT_PIN GPIO_NUM_32

#define ENABLE_DHT22 true
#define DHT22_PIN GPIO_NUM_27
#define DHT22_INTERVAL 300000
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
					TopicPrefix: "garden",
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

#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define NUM_ZONES 1
#define VALVES { GPIO_NUM_16 }
#define PUMPS { GPIO_NUM_18 }

#define LIGHT_ENABLED false
#define LIGHT_PIN GPIO_NUM_MAX

#define ENABLE_DHT22 false
#define DHT22_PIN GPIO_NUM_MAX
#define DHT22_INTERVAL 0
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
					TopicPrefix: "garden",
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

#define MQTT_ADDRESS "localhost"
#define MQTT_PORT 1883

#define NUM_ZONES 4
#define VALVES { GPIO_NUM_16, GPIO_NUM_16, GPIO_NUM_16, GPIO_NUM_16 }
#define PUMPS { GPIO_NUM_18, GPIO_NUM_18, GPIO_NUM_18, GPIO_NUM_18 }

#define LIGHT_ENABLED false
#define LIGHT_PIN GPIO_NUM_MAX

#define ENABLE_DHT22 false
#define DHT22_PIN GPIO_NUM_MAX
#define DHT22_INTERVAL 0
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
