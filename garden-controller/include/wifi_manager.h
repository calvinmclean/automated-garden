#ifndef wifi_manager_h
#define wifi_manager_h

#include <WiFi.h>
#include <WiFiManager.h>
#include <ArduinoJson.h>
#include <LittleFS.h>
#include <ESPmDNS.h>

extern WiFiManager wifiManager;

extern char mqtt_server[41];
extern char mqtt_topic_prefix[41];
extern int mqtt_port;

void setupWifiManager();

 #endif
