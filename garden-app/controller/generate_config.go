package controller

import (
	"bytes"
	"fmt"
	"html/template"
)

const (
	configTemplate = `#ifndef config_h
#define config_h

#define GARDEN_NAME "{{.Garden}}"

#define QUEUE_SIZE 10

#define ENABLE_WIFI
#ifdef ENABLE_WIFI
#define MQTT_ADDRESS "{{.MQTTConfig.Broker}}"
#define MQTT_PORT {{.MQTTConfig.Port}}
#define MQTT_CLIENT_NAME GARDEN_NAME
#define MQTT_WATER_TOPIC GARDEN_NAME"/command/water"
#define MQTT_STOP_TOPIC GARDEN_NAME"/command/stop"
#define MQTT_STOP_ALL_TOPIC GARDEN_NAME"/command/stop_all"
#define MQTT_LIGHT_TOPIC GARDEN_NAME"/command/light"
#define MQTT_LIGHT_DATA_TOPIC GARDEN_NAME"/data/light"
#define MQTT_WATER_DATA_TOPIC GARDEN_NAME"/data/water"

{{if .PublishHealth}}
#define ENABLE_MQTT_HEALTH
#ifdef ENABLE_MQTT_HEALTH
#define MQTT_HEALTH_DATA_TOPIC GARDEN_NAME"/data/health"
#define HEALTH_PUBLISH_INTERVAL {{.HealthInterval}}
#endif
{{end}}

#define ENABLE_MQTT_LOGGING
#ifdef ENABLE_MQTT_LOGGING
#define MQTT_LOGGING_TOPIC GARDEN_NAME"/data/logs"
#endif

#define JSON_CAPACITY 48
#endif

#define NUM_PLANTS {{.NumPlants}}
#define PLANTS { {{range $p := .PlantConfigs}}{ {{$p.PumpPin}}, {{$p.ValvePin}}, {{$p.ButtonPin}}, {{$p.MoistureSensorPin}} }{{end}} }
#define DEFAULT_WATER_TIME {{.DefaultWaterTime}}

#define LIGHT_PIN {{.LightPin}}

{{if .EnableButtons}}
#define ENABLE_BUTTONS
#ifdef ENABLE_BUTTONS
#define STOP_BUTTON_PIN {{.StopButtonPin}}
#endif
{{end}}

{{if .EnableMoistureSensor}}
#ifdef ENABLE_MOISTURE_SENSORS AND ENABLE_WIFI
#define MQTT_MOISTURE_DATA_TOPIC GARDEN_NAME"/data/moisture"
#define MOISTURE_SENSOR_AIR_VALUE 3415
#define MOISTURE_SENSOR_WATER_VALUE 1362
#define MOISTURE_SENSOR_INTERVAL {{.MoistureInterval}}
#endif
{{end}}
#endif
`
	wifiConfigTemplate = `#ifndef wifi_config_h
#define wifi_config_h

#define SSID "{{.NetworkName}}"
#define PASSWORD "{{.Password}}"

#endif
`
)

type WifiConfigData struct {
	NetworkName string
	Password    string
}

type ConfigData struct {
	Config
	PlantConfigs         []PlantConfig
	DefaultWaterTime     int
	EnableButtons        bool
	EnableMoistureSensor bool
	LightPin             string
	StopButtonPin        string
}

type PlantConfig struct {
	PumpPin           string
	ValvePin          string
	ButtonPin         string
	MoistureSensorPin string
}

func GenerateConfig(config Config) {
	t := template.Must(template.New("config.h").Parse(configTemplate))
	var result bytes.Buffer
	data := ConfigData{
		Config: config,
		PlantConfigs: []PlantConfig{
			{
				PumpPin:           "0",
				ValvePin:          "1",
				ButtonPin:         "2",
				MoistureSensorPin: "3",
			},
		},
		EnableMoistureSensor: true,
		DefaultWaterTime:     5000,
		LightPin:             "4",
		EnableButtons:        true,
		StopButtonPin:        "5",
	}
	err := t.Execute(&result, data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(result.String())
}
