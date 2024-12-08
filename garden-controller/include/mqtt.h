#ifndef mqtt_h
#define mqtt_h

#include <ArduinoJson.h>
#include <PubSubClient.h>

#include "config.h"
#include "garden_config.h"

#define MQTT_WATER_TOPIC "/command/water"
#define MQTT_STOP_TOPIC "/command/stop"
#define MQTT_STOP_ALL_TOPIC "/command/stop_all"
#define MQTT_LIGHT_TOPIC "/command/light"
#define MQTT_UPDATE_CONFIG_TOPIC "/command/update_config"

#define MQTT_LIGHT_DATA_TOPIC "/data/light"
#define MQTT_WATER_DATA_TOPIC "/data/water"
#define MQTT_LOGGING_TOPIC "/data/logs"
#define MQTT_HEALTH_DATA_TOPIC "/data/health"
#define MQTT_TEMPERATURE_DATA_TOPIC "/data/temperature"
#define MQTT_HUMIDITY_DATA_TOPIC "/data/humidity"

#define HEALTH_PUBLISH_INTERVAL 60000

extern PubSubClient client;

void setupMQTT();
void setupWifi();
void waterPublisherTask(void* parameters);
void lightPublisherTask(void* parameters);
void healthPublisherTask(void* parameters);
void mqttConnectTask(void* parameters);
void mqttLoopTask(void* parameters);
void processIncomingMessage(char* topic, byte* message, unsigned int length);

extern QueueHandle_t waterPublisherQueue;
extern QueueHandle_t lightPublisherQueue;

#endif
