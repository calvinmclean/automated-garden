#ifndef config_h
#define config_h

// All values in this file are optional. If configured, they are only used when SSID/PASSWORD is also
// set for wifi connection. If these aren't set, WifiManager's captive portal is used instead.

// Unique prefix for this controller. It is used for the root of MQTT topics and as the MQTT ClientID
// #define TOPIC_PREFIX "test-garden"

/**
 * MQTT Configurations
 *   NOTE: Use "wifi_config.h" for Wifi SSID and password (ignored by git)
 *
 * MQTT_ADDRESS
 *   IP address or hostname for MQTT broker
 * MQTT_PORT
 *   Port for MQTT broker
 */
// #define MQTT_ADDRESS "192.168.0.32"
// #define MQTT_PORT 1883

#endif
