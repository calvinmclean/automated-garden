# Garden Controller
The `garden-controller` is an Arduino/FreeRTOS firmware built for ESP32 to control a single real-world Garden.

A majority of the functionality relies on a connection to MQTT to receive commands and publish data, but it is still usable as a standalone device.

## Getting Started
This is going to assume you have some familiarity with the ESP32 and Arduino IDE. If that is not true, please look at the [official repository for `arduino-esp32`](https://github.com/espressif/arduino-esp32) to get that setup.

1. Make any necessary changes to `config.h` to fit your setup
    - Read comments in the file to see the different available configuration options or [see docs here](controller_advanced.md)
    - To enable automated watering without using the Go `garden-app`, use `ENABLE_WATERING_INTERVAL` setting
    - You can also look at Examples in this documentation to see specific configs
1. Copy `wifi_config.h.example` to `wifi_config.h` and configure your network access information
1. Compile and upload to your ESP32 using Arduino IDE

It is more difficult to provide a comprehensive quickstart for the `garden-controller` because it is so closely related to the hardware. You will have to wire everything up to the device and that depends on your individual skill level, hardware components, and use-case. Please see the Examples section for more detailed explanations on hardware setups and corresponding controller configs.

## Advanced
See the [advanced section](controller_advanced.md) for more detailed documentation.
