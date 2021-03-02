#ifndef config_h
#define config_h

#define GARDEN_NAME "garden"

/**
 * MQTT Configurations
 */
#define MQTT_ADDRESS "192.168.0.107"
#define MQTT_RETRY_DELAY 5000
#define MQTT_CLIENT_NAME GARDEN_NAME
#define MQTT_PORT 1883
#define MQTT_WATER_TOPIC GARDEN_NAME"/command/water"
#define MQTT_STOP_TOPIC GARDEN_NAME"/command/stop"
#define MQTT_STOP_ALL_TOPIC GARDEN_NAME"/command/stop_all"
#define MQTT_WATER_DATA_TOPIC GARDEN_NAME"/data/water"

/**
 * Garden Configurations
 *
 * - DEFAULT_WATER_TIME:
 *   Default time to water for if none is specified
 *   This is used by button and interval watering
 * - NUM_PLANTS:
 *   Number of plants in the PLANTS list
 * - PUMP_PIN:
 *   Optionally defined since most of the time plants will use the same pump
 * - PLANTS:
 *   List of plant pins in this format: { {PUMP_PIN, VALVE_PIN, BUTTON_PIN} }
 *   You can create multiple plants before creating the PLANTS list for
 *   improved readability (see example below)
 * - STOP_BUTTON_PIN:
 *   Pin used for the button that will stop all watering
 * - ENABLE_WATERING_INTERVAL:
 *   Determines if we should water the plant automatically
 * - INTERVAL:
 *   The time, in milliseconds, to wait between automatic watering
 *   Only used if ENABLE_WATERING_INTERVAL is defined.
 */
#define DEFAULT_WATER_TIME 15000
#define NUM_PLANTS 3
#define PUMP_PIN GPIO_NUM_18
#define PLANT_1 { PUMP_PIN, GPIO_NUM_16, GPIO_NUM_19 }
#define PLANT_2 { PUMP_PIN, GPIO_NUM_17, GPIO_NUM_21 }
#define PLANT_3 { PUMP_PIN, GPIO_NUM_5, GPIO_NUM_22 }
#define PLANTS { PLANT_1, PLANT_2, PLANT_3 }

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
