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

// Enable publishing health status to MQTT
#define ENABLE_MQTT_HEALTH

// Enable logging messages to MQTT
#define ENABLE_MQTT_LOGGING

/**
 * Garden Configurations
 *
 * DISABLE_WATERING
 *   Allows disabling Pump/Valve pins and doesn't listen on relevant MQTT topics. This allows a sensor-only Garden
 * NUM_ZONES
 *   Number of zones in the ZONES list
 * ZONES
 *   List of zone pins in this format: { {PUMP_PIN, VALVE_PIN, BUTTON_PIN, MOISTURE_SENSOR_PIN} }
 *   You can create multiple zones before creating the ZONES list for improved readability (see example below)
 * DEFAULT_WATER_TIME
 *   Default time to water for if none is specified. This is used by buttons
 * ENABLE_BUTTONS
 *   Configure if there are any hardware buttons corresponding to zones
 * STOP_BUTTON_PIN
 *   Pin used for the button that will stop all watering
 * LIGHT_PIN
 *   The pin used to control a grow light relay
 */
 // #define DISABLE_WATERING
#define NUM_ZONES 3
#define PUMP_PIN GPIO_NUM_18
#define ZONE_1 { PUMP_PIN, GPIO_NUM_16, GPIO_NUM_19, GPIO_NUM_36 }
#define ZONE_2 { PUMP_PIN, GPIO_NUM_17, GPIO_NUM_21, GPIO_NUM_39 }
#define ZONE_3 { PUMP_PIN, GPIO_NUM_5, GPIO_NUM_22, GPIO_NUM_34 }
#define ZONES { ZONE_1, ZONE_2, ZONE_3 }
#define DEFAULT_WATER_TIME 5000

#define LIGHT_PIN GPIO_NUM_32

// #define ENABLE_BUTTONS
#ifdef ENABLE_BUTTONS
#define STOP_BUTTON_PIN GPIO_NUM_23
#endif

// Currently, moisture sensing requires  MQTT because the logic for handling this data lives in the garden-app
// #define ENABLE_MOISTURE_SENSORS
#ifdef ENABLE_MOISTURE_SENSORS
#define MOISTURE_SENSOR_AIR_VALUE 3415
#define MOISTURE_SENSOR_WATER_VALUE 1362
#define MOISTURE_SENSOR_INTERVAL 5000
#endif

// DHT22 Temperature and Humidity sensor
#define ENABLE_DHT22
#ifdef ENABLE_DHT22
#define DHT22_PIN GPIO_NUM_27
#define DHT22_INTERVAL 5000
#endif

#endif
