#ifndef wifi_manager_h
#define wifi_manager_h

#include <WiFiManager.h>
#include <ArduinoJson.h>
#include <LittleFS.h>

#define FORMAT_LITTLEFS_IF_FAILED true

extern WiFiManagerParameter custom_mqtt_server;
extern WiFiManagerParameter custom_mqtt_topic_prefix;
extern WiFiManagerParameter custom_mqtt_port;

extern WiFiManager wifiManager;

// TODO: use these variables for MQTT setup
// It looks like it will be difficult to refactor everything to use the new MQTT configuration.
// My options are to ditch that feature for now since I am just using WifiManager for Wifi + OTA
// and don't need to immediately connect to MQTT yet since I am not doing OTA or configs over MQTT
extern char* mqtt_server;
extern char* mqtt_topic_prefix;
extern int mqtt_port;

void setupWifiManager();
void mqttLoopTask(void* parameters);

extern TaskHandle_t wifiManagerLoopTaskHandle;

#endif
