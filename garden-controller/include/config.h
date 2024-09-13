#ifndef config_h
#define config_h

// Unique prefix for this controller. It is used for the root of MQTT topics and as the MQTT ClientID
#define TOPIC_PREFIX "test-garden"

// Size of FreeRTOS queues
#define QUEUE_SIZE 10

/**
 * MQTT Configurations
 *   NOTE: Use "wifi_config.h" for Wifi SSID and password (ignored by git)
 *
 * MQTT_ADDRESS
 *   IP address or hostname for MQTT broker
 * MQTT_PORT
 *   Port for MQTT broker
 */
#define MQTT_ADDRESS "192.168.0.107"
#define MQTT_PORT 30002

/**
 * Garden Configurations
 *
 * NUM_ZONES
 *   Number of zones in the ZONES list
 * VALVES
 *   List of valve pins
 * PUMPS
 *   List of pump pins. If a pump is not being used for this Zone, just use the same pin as the valve.
 * LIGHT_PIN
 *   The pin used to control a grow light relay
 */
#define NUM_ZONES 3
#define VALVES { GPIO_NUM_16, GPIO_NUM_17, GPIO_NUM_5 }
#define PUMPS { GPIO_NUM_18, GPIO_NUM_18, GPIO_NUM_18 }

#define LIGHT_ENABLED true
#define LIGHT_PIN GPIO_NUM_32

// DHT22 Temperature and Humidity sensor
#define ENABLE_DHT22 true
#define DHT22_PIN GPIO_NUM_27
#define DHT22_INTERVAL 5000

#endif
