#ifndef mqtt_h
#define mqtt_h

#include <WiFi.h>
#include <ArduinoJson.h>
#include <PubSubClient.h>

// Configure network name and password in this file
#include "wifi_config.h"
#include "config.h"

extern PubSubClient client;

void setupMQTT();
void setupWifi();
void waterPublisherTask(void* parameters);
void lightPublisherTask(void* parameters);
void healthPublisherTask(void* parameters);
void mqttConnectTask(void* parameters);
void mqttLoopTask(void* parameters);
void processIncomingMessage(char* topic, byte* message, unsigned int length);
void wifiDisconnectHandler(WiFiEvent_t event, WiFiEventInfo_t info);

/* FreeRTOS Queue and Task handlers */
extern TaskHandle_t mqttConnectTaskHandle;
extern TaskHandle_t mqttLoopTaskHandle;
extern TaskHandle_t healthPublisherTaskHandle;
extern TaskHandle_t waterPublisherTaskHandle;
extern QueueHandle_t waterPublisherQueue;
#ifdef LIGHT_PIN
extern QueueHandle_t lightPublisherQueue;
extern TaskHandle_t lightPublisherTaskHandle;
#endif

#endif
