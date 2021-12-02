# Sensors Only Example

## Details
This setup doesn't handle any watering and instead just attaches sensors. Although this can be useful on its own, it is really intended to be a second controller that is part of the same real-world Garden. For example, if you have a large outdoor Garden, you cannot realistically wire a moisture sensor to each Plant since the main controller will be located potentially far away, near the hose output. This allows you to connect separate controllers to collect data.

## Components

### Purchased
- Power adapter
- Circuit board
- ESP32 dev board
- Capactivie moisture sensors

### 3D Printed
- Electronics cases

### Circuit
This has a very basic setup since it just consists of the ESP32 and moisture sensors.

## Configurations
Only the `garden-controller` config is shown here because this is intended to be in addition to an existing Garden setup and won't require any additional changes to the `garden-app` setup.
<!-- tabs:start -->
#### **`garden-controller/config.h`**
```c
#ifndef config_h
#define config_h

#define GARDEN_NAME "garden"

#define QUEUE_SIZE 10

#define MQTT_ADDRESS "192.168.0.107"
#define MQTT_PORT 30002
#define MQTT_CLIENT_NAME GARDEN_NAME"-sensors"

#define JSON_CAPACITY 48

#define DISABLE_WATERING
#define NUM_PLANTS 3
#define PUMP_PIN GPIO_NUM_18
#define PLANT_1 { PUMP_PIN, GPIO_NUM_16, GPIO_NUM_19, GPIO_NUM_36 }
#define PLANT_2 { PUMP_PIN, GPIO_NUM_17, GPIO_NUM_21, GPIO_NUM_39 }
#define PLANT_3 { PUMP_PIN, GPIO_NUM_5, GPIO_NUM_22, GPIO_NUM_34 }
#define PLANTS { PLANT_1, PLANT_2, PLANT_3 }
#define DEFAULT_WATER_TIME 5000

#define ENABLE_MOISTURE_SENSORS
#ifdef ENABLE_MOISTURE_SENSORS
#define MQTT_MOISTURE_DATA_TOPIC GARDEN_NAME"/data/moisture"
#define MOISTURE_SENSOR_AIR_VALUE 3415
#define MOISTURE_SENSOR_WATER_VALUE 1362
#define MOISTURE_SENSOR_INTERVAL 5000
#endif

#endif
```
<!-- tabs:end -->
