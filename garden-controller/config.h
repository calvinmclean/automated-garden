#ifndef config_h
#define config_h

#define GARDEN_NAME "garden"

/**
 * Wifi and MQTT Configurations
 *   NOTE: Use "wifi_config.h" for Wifi SSID and password (ignored by git)
 *
 * ENABLE_WIFI
 *   Should Wifi and MQTT features be used
 * MQTT_ADDRESS
 *   IP address or hostname for MQTT broker
 * MQTT_PORT
 *   Port for MQTT broker
 * MQTT_CLIENT_NAME
 *   Name to use when connecting to MQTT broker. By default this is GARDEN_NAME
 * MQTT_WATER_TOPIC
 *   Topic to subscribe to for incoming commands to water a plant
 * MQTT_STOP_TOPIC
 *   Topic to subscribe to for incoming command to stop watering a plant
 * MQTT_STOP_ALL_TOPIC
 *   Topic to subscribe to for incoming command to stop watering a plant and clear the watering queue
 * MQTT_WATER_DATA_TOPIC
 *   Topic to publish watering metrics on
 */
#define ENABLE_WIFI
#ifdef ENABLE_WIFI
#define MQTT_ADDRESS "192.168.0.107"
#define MQTT_PORT 1883
#define MQTT_CLIENT_NAME GARDEN_NAME
#define MQTT_WATER_TOPIC GARDEN_NAME"/command/water"
#define MQTT_STOP_TOPIC GARDEN_NAME"/command/stop"
#define MQTT_STOP_ALL_TOPIC GARDEN_NAME"/command/stop_all"
#define MQTT_WATER_DATA_TOPIC GARDEN_NAME"/data/water"
#endif

/**
 * Garden Configurations
 *
 * NUM_PLANTS
 *   Number of plants in the PLANTS list
 * PLANTS
 *   List of plant pins in this format: { {PUMP_PIN, VALVE_PIN, BUTTON_PIN} }
 *   You can create multiple plants before creating the PLANTS list for improved readability (see example below)
 * DEFAULT_WATER_TIME
 *   Default time to water for if none is specified. This is used by button and interval watering
 * ENABLE_BUTTONS
 *   Configure if there are any hardware buttons corresponding to plants
 * DEBOUNCE_DELAY
 *   Delay to use when handling button debounce
 * STOP_BUTTON_PIN
 *   Pin used for the button that will stop all watering
 * ENABLE_WATERING_INTERVAL
 *   Determines if we should water the plant automatically
 * INTERVAL
 *   The time, in milliseconds, to wait between automatic watering. Only used if ENABLE_WATERING_INTERVAL is defined.
 */
#define NUM_PLANTS 3
#define PUMP_PIN GPIO_NUM_18
#define PLANT_1 { PUMP_PIN, GPIO_NUM_16, GPIO_NUM_19 }
#define PLANT_2 { PUMP_PIN, GPIO_NUM_17, GPIO_NUM_21 }
#define PLANT_3 { PUMP_PIN, GPIO_NUM_5, GPIO_NUM_22 }
#define PLANTS { PLANT_1, PLANT_2, PLANT_3 }
#define DEFAULT_WATER_TIME 15000

#define ENABLE_BUTTONS
#ifdef ENABLE_BUTTONS
#define DEBOUNCE_DELAY 50
#define STOP_BUTTON_PIN GPIO_NUM_23
#endif

// #define ENABLE_WATERING_INTERVAL
#ifdef ENABLE_WATERING_INTERVAL
#define INTERVAL 86400000 // 24 hours
#endif

#endif
