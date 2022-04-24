#ifndef config_h
#define config_h

#define TOPIC_PREFIX "garden"

// Size of FreeRTOS queues
#define QUEUE_SIZE 10

/**
 * Wifi and MQTT Configurations
 *   NOTE: Use "wifi_config.h" for Wifi SSID and password (ignored by git)
 *
 * MQTT_ADDRESS
 *   IP address or hostname for MQTT broker
 * MQTT_PORT
 *   Port for MQTT broker
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
#define MQTT_ADDRESS "192.168.0.107"
#define MQTT_PORT 30002
#define MQTT_CLIENT_NAME TOPIC_PREFIX
#define MQTT_WATER_TOPIC TOPIC_PREFIX"/command/water"
#define MQTT_STOP_TOPIC TOPIC_PREFIX"/command/stop"
#define MQTT_STOP_ALL_TOPIC TOPIC_PREFIX"/command/stop_all"
#define MQTT_LIGHT_TOPIC TOPIC_PREFIX"/command/light"
#define MQTT_LIGHT_DATA_TOPIC TOPIC_PREFIX"/data/light"
#define MQTT_WATER_DATA_TOPIC TOPIC_PREFIX"/data/water"

#define ENABLE_MQTT_HEALTH
#ifdef ENABLE_MQTT_HEALTH
#define MQTT_HEALTH_DATA_TOPIC TOPIC_PREFIX"/data/health"
#define HEALTH_PUBLISH_INTERVAL 60000
#endif

#define ENABLE_MQTT_LOGGING
#ifdef ENABLE_MQTT_LOGGING
#define MQTT_LOGGING_TOPIC TOPIC_PREFIX"/data/logs"
#endif

 // Size of JSON object calculated using Arduino JSON Assistant
#define JSON_CAPACITY 48

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

#define ENABLE_BUTTONS
#ifdef ENABLE_BUTTONS
#define STOP_BUTTON_PIN GPIO_NUM_23
#endif

// Currently, moisture sensing requires  MQTT because the logic for handling this data lives in the garden-app
// #define ENABLE_MOISTURE_SENSORS
#ifdef ENABLE_MOISTURE_SENSORS
#define MQTT_MOISTURE_DATA_TOPIC TOPIC_PREFIX"/data/moisture"
#define MOISTURE_SENSOR_AIR_VALUE 3415
#define MOISTURE_SENSOR_WATER_VALUE 1362
#define MOISTURE_SENSOR_INTERVAL 5000
#endif

#endif
