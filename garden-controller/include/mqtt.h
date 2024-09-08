#ifndef mqtt_h
#define mqtt_h

#include <ArduinoJson.h>
#include <PubSubClient.h>

#include "config.h"

/**
 * MQTT_CLIENT_NAME
 *   Name to use when connecting to MQTT broker. By default this is TOPIC_PREFIX
 * MQTT_WATER_TOPIC
 *   Topic to subscribe to for incoming commands to water a zone
 * MQTT_STOP_TOPIC
 *   Topic to subscribe to for incoming command to stop watering a zone
 * MQTT_STOP_ALL_TOPIC
 *   Topic to subscribe to for incoming command to stop watering a zone and clear the watering queue
 * MQTT_LIGHT_TOPIC
 *   Topic to subscribe to for incoming command to change the state of an attached grow light
 * MQTT_LIGHT_DATA_TOPIC
 *   Topic to publish LightEvents on
 * MQTT_WATER_DATA_TOPIC
 *   Topic to publish watering metrics on
 */
#define MQTT_WATER_TOPIC "/command/water"
#define MQTT_STOP_TOPIC "/command/stop"
#define MQTT_STOP_ALL_TOPIC "/command/stop_all"
#define MQTT_LIGHT_TOPIC "/command/light"
#define MQTT_LIGHT_DATA_TOPIC "/data/light"
#define MQTT_WATER_DATA_TOPIC "/data/water"

#define MQTT_LOGGING_TOPIC "/data/logs"

#define MQTT_HEALTH_DATA_TOPIC "/data/health"
#define HEALTH_PUBLISH_INTERVAL 60000

#define MQTT_TEMPERATURE_DATA_TOPIC "/data/temperature"
#define MQTT_HUMIDITY_DATA_TOPIC "/data/humidity"

extern PubSubClient client;

void setupMQTT();
void setupWifi();
void waterPublisherTask(void* parameters);
void lightPublisherTask(void* parameters);
void healthPublisherTask(void* parameters);
void mqttConnectTask(void* parameters);
void mqttLoopTask(void* parameters);
void processIncomingMessage(char* topic, byte* message, unsigned int length);

/* FreeRTOS Queue and Task handlers */
extern TaskHandle_t mqttConnectTaskHandle;
extern TaskHandle_t mqttLoopTaskHandle;
extern TaskHandle_t healthPublisherTaskHandle;
extern TaskHandle_t waterPublisherTaskHandle;
extern QueueHandle_t waterPublisherQueue;
extern QueueHandle_t lightPublisherQueue;
extern TaskHandle_t lightPublisherTaskHandle;

#endif
