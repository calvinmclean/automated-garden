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
const char* waterCommandTopic = MQTT_WATER_TOPIC;
const char* stopCommandTopic = MQTT_STOP_TOPIC;
const char* stopAllCommandTopic = MQTT_STOP_ALL_TOPIC;
const char* waterDataTopic = MQTT_WATER_DATA_TOPIC;
const char* healthDataTopic = MQTT_HEALTH_DATA_TOPIC;

/* FreeRTOS Queue and Task handlers */
QueueHandle_t publisherQueue;
TaskHandle_t mqttConnectTaskHandle;
TaskHandle_t mqttLoopTaskHandle;
TaskHandle_t publisherTaskHandle;
TaskHandle_t healthPublisherTaskHandle;

#endif
