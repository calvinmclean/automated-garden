#ifndef mqtt_h
#define mqtt_h

#include <WiFi.h>
#include <ArduinoJson.h>
#include <PubSubClient.h>

// Configure network name and password in this file
#include "wifi_config.h"

WiFiClient wifiClient;

/* MQTT variables */
unsigned long lastConnectAttempt = 0;
PubSubClient client(wifiClient);

#ifdef DISABLE_WATERING
const char* waterCommandTopic = "";
const char* stopCommandTopic = "";
const char* stopAllCommandTopic = "";
const char* waterDataTopic = "";
#else
const char* waterCommandTopic = MQTT_WATER_TOPIC;
const char* stopCommandTopic = MQTT_STOP_TOPIC;
const char* stopAllCommandTopic = MQTT_STOP_ALL_TOPIC;
const char* waterDataTopic = MQTT_WATER_DATA_TOPIC;
#endif

#ifdef LIGHT_PIN
const char* lightCommandTopic = MQTT_LIGHT_TOPIC;
const char* lightDataTopic = MQTT_LIGHT_DATA_TOPIC;
#else
const char* lightCommandTopic = "";
const char* lightDataTopic = "";
#endif

#ifdef ENABLE_MQTT_HEALTH
const char* healthDataTopic = MQTT_HEALTH_DATA_TOPIC;
#else
const char* healthDataTopic = "";
#endif

/* FreeRTOS Queue and Task handlers */
TaskHandle_t mqttConnectTaskHandle;
TaskHandle_t mqttLoopTaskHandle;
TaskHandle_t healthPublisherTaskHandle;
TaskHandle_t waterPublisherTaskHandle;
QueueHandle_t waterPublisherQueue;
#ifdef LIGHT_PIN
QueueHandle_t lightPublisherQueue;
TaskHandle_t lightPublisherTaskHandle;
#endif

#endif
