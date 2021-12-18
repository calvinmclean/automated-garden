# Garden Controller (Advanced)
This section provides more details on the features, code organization, and configurations available for the `garden-controller`.

## Features
- Highly configurable and flexible
- Number of connected pumps/valves is only limited by the number of pins on your controller (and memory)
- Optionally control watering with connected buttons
- Collect moisture data from connected sensors
- Connect to MQTT to publish periodic health check-ins, moisture sensor data, logs, and event data for watering and lighting

## Code Organization
This code is split up into different `.ino` and header files to improve organization and separate logic.

Arduino IDE will automatically combine all `.ino` files in alphabetical order, so some of the variables and specific configurations are split into header files so they could be included at the top.

C pre-processor features are used to reduce the amount of code when certain features are disabled.

## Configuration
This project is designed to be highly-configurable for all of your use-cases. All of the user-configurable options are located in `config.h` and `wifi_config.h` is used to hold sensitive WiFi password.

### Basic Options
These are the basic options that are required and do not fit in specific categories.

`TOPIC_PREFIX`: this is used as the prefix for MQTT topics so it is critical that this matches the Garden's `TopicPrefix` in `garden-app`

`QUEUE_SIZE`: maximum number of messages that can be queued in FreeRTOS queues. 10 is a sensible default that should never overflow unless you have a large number of Plants

`JSON_CAPACITY`: Size of JSON object calculated using Arduino JSON Assistant. This should not be changed

### MQTT/WiFi Options
These are all the configurations for setting up MQTT publish/subscribe.

`SSID`: 2.4GHz WiFI name (located in `wifi_config.h`)

`PASSWORD`: WiFI password (located in `wifi_config.h`)

`ENABLE_WIFI`: WiFi and MQTT are enabled if this is defined

`MQTT_ADDRESS`: IP address or hostname for MQTT broker

`MQTT_PORT`: Port for MQTT broker

#### Additional MQTT Options
The following options should be left as defaults, unless you have a good reason to change them.

`MQTT_CLIENT_NAME`: Name to use when connecting to MQTT broker. By default this is `TOPIC_PREFIX`. It is important that this is unique

`MQTT_WATER_TOPIC`: Topic to subscribe to for incoming commands to water a plant

`MQTT_STOP_TOPIC`: Topic to subscribe to for incoming command to stop watering a plant

`MQTT_STOP_ALL_TOPIC`: Topic to subscribe to for incoming command to stop watering a plant and clear the watering queue

`MQTT_LIGHT_TOPIC`: Topic to subscribe to for incoming command to change the state of an attached grow light

`MQTT_LIGHT_DATA_TOPIC`: Topic to publish LightingEvents on

`MQTT_WATER_DATA_TOPIC`: Topic to publish watering metrics on

#### Health Publishing Options
These options are used for enabled/configuring publishing of health check-ins to MQTT.

`ENABLE_MQTT_HEALTH`: Enables periodic publishing of health check-ins when defined

`MQTT_HEALTH_DATA_TOPIC`: Topic to publish health check-ins on

`HEALTH_PUBLISH_INTERVAL`: Time, in milliseconds, to wait between publishing of health check-ins

### Plant Options
These options are related to the actual pins and other necessary information for watering plants.

`DISABLE_WATERING`: Allows disabling Pump/Valve pins and doesn't listen on relevant MQTT topics. This allows a sensor-only Garden. If you are running this alongside a separate `garden-controller` that handles watering, please remember to change the `MQTT_CLIENT_NAME` to be different

`NUM_PLANTS`: Number of plants connected to this Garden

`PUMP_PIN`: Optional configuration that makes organization better if you use the same pump for all plants

`PLANT_1`, `PLANT_2`, ..., `PLANT_N`: These are optional configurations that will be included in `PLANTS` below, but make it a bit easier to organize the configuration. Use the following format:
```
{PUMP_PIN, VALVE_PIN, BUTTON_PIN, MOISTURE_SENSOR_PIN}
```

`PLANTS`: This is a list of all plants managed by this controller. It contains the pin details for pump, valve, button, and moisture sensor. The button and sensor pins are ignored if not enabled (see sections below). If you are not using a pump, or not using a valve, just use the same pin for both. Use the following format:
```
{ {PUMP_PIN, VALVE_PIN, BUTTON_PIN, MOISTURE_SENSOR_PIN} }
```
**note**: Use `GPIO_NUM_MAX` to disable moisture sensing for only certain Plants.

`DEFAULT_WATER_TIME`: The default amount of time to water for, in milliseconds, if one is not defined in the command. This is also used to determine how long button-presses will water for

`LIGHT_PIN`: Pin used to control grow light relay

#### Button Options
These options allow optionally enabling button control. The buttons pins are defined as a part of the plants configuration.

`ENABLE_BUTTONS`: Enables reading input from buttons when defined

`STOP_BUTTON_PIN`: Button pins are usually defined for each individual plant, but this is a separate button that will cancel in-progress watering

#### Moisture Sensor Options
These options allow optionally enabling moisture data publishing. WiFi + MQTT are also required for this since the data must be published for storage. The value configurations below are used for calibrating the sensor. The moisture sensor pins are configured as part fo the plants configuration.

`ENABLE_MOISTURE_SENSORS`: Enables moisture sensors when defined

`MQTT_MOISTURE_DATA_TOPIC`: MQTT topic to publish moisture data to

`MOISTURE_SENSOR_AIR_VALUE`: Value to use for a dry sensor

`MOISTURE_SENSOR_WATER_VALUE`: Value to use for a fully-submerged sensor

`MOISTURE_SENSOR_INTERVAL`: Time, in milliseconds, to wait between sensor readings
