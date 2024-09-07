# Garden Controller (Advanced)
This section provides more details on the features, code organization, and configurations available for the `garden-controller`.

## Features
- Highly configurable and flexible
- Number of connected pumps/valves is only limited by the number of pins on your controller (and memory)
- Connect to MQTT to publish periodic health check-ins, logs, and event data for watering and lighting

## Code Organization
This code is split up into different `.ino` and header files to improve organization and separate logic.

Arduino IDE will automatically combine all `.ino` files in alphabetical order, so some of the variables and specific configurations are split into header files so they could be included at the top.

C pre-processor features are used to reduce the amount of code when certain features are disabled.

## Configuration
This project is designed to be highly-configurable for all of your use-cases. All of the user-configurable options are located in `config.h` and `wifi_config.h` is used to hold sensitive WiFi password.

### Basic Options
These are the basic options that are required and do not fit in specific categories.

`TOPIC_PREFIX`: this is used as the prefix for MQTT topics so it is critical that this matches the Garden's `TopicPrefix` in `garden-app`

`QUEUE_SIZE`: maximum number of messages that can be queued in FreeRTOS queues. 10 is a sensible default that should never overflow unless you have a large number of Zones

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

`MQTT_WATER_TOPIC`: Topic to subscribe to for incoming commands to water a zone

`MQTT_STOP_TOPIC`: Topic to subscribe to for incoming command to stop watering a zone

`MQTT_STOP_ALL_TOPIC`: Topic to subscribe to for incoming command to stop watering a zone and clear the watering queue

`MQTT_LIGHT_TOPIC`: Topic to subscribe to for incoming command to change the state of an attached grow light

`MQTT_LIGHT_DATA_TOPIC`: Topic to publish LightEvents on

`MQTT_WATER_DATA_TOPIC`: Topic to publish watering metrics on

#### Health Publishing Options
These options are used for enabled/configuring publishing of health check-ins to MQTT.

`MQTT_HEALTH_DATA_TOPIC`: Topic to publish health check-ins on

`HEALTH_PUBLISH_INTERVAL`: Time, in milliseconds, to wait between publishing of health check-ins

### Zone Options
These options are related to the actual pins and other necessary information for watering zones.

`NUM_ZONES`: Number of zones connected to this Garden

`PUMP_PIN`: Optional configuration that makes organization better if you use the same pump for all zones

`ZONES`: This is a list of all zones managed by this controller. It contains the pin details for following format:
```
{ ZONE1_PIN, ZONE2_PIN, ... }
```

`LIGHT_PIN`: Pin used to control grow light relay
