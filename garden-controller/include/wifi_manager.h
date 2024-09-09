#ifndef wifi_manager_h
#define wifi_manager_h

#include <WiFi.h>
#include <WiFiManager.h>
#include <ArduinoJson.h>
#include <LittleFS.h>
#include <ESPmDNS.h>

#define FORMAT_LITTLEFS_IF_FAILED true

extern WiFiManager wifiManager;

extern char* mqtt_server;
extern char* mqtt_topic_prefix;
extern int mqtt_port;

void setupWifiManager();

 #endif
