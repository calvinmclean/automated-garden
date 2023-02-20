# Garden Controller
The `garden-controller` is an Arduino/FreeRTOS firmware built for ESP32 to control a single real-world Garden.

A majority of the functionality relies on a connection to MQTT to receive commands and publish data.

## Getting Started
This is going to assume you have some familiarity with the ESP32 and Arduino IDE. If that is not true, please look at the [official repository for `arduino-esp32`](https://github.com/espressif/arduino-esp32) to get that setup.

1. Make any necessary changes to `config.h` to fit your setup
    - Read comments in the file to see the different available configuration options or [see docs here](controller_advanced.md)
    - You can also look at Examples in this documentation to see specific configs
1. Copy `wifi_config.h.example` to `wifi_config.h` and configure your network access information
1. Compile and upload to your ESP32 using Arduino IDE

It is more difficult to provide a comprehensive quickstart for the `garden-controller` because it is so closely related to the hardware. You will have to wire everything up to the device and that depends on your individual skill level, hardware components, and use-case. Please see the Examples section for more detailed explanations on hardware setups and corresponding controller configs.

## Configuring
The easiest way to get started is to use the default configurations. The next best thing is creating the Arduino configurations using the `garden-app controller generate-config` command.

You can use an interactive mode to generate a configuration or create a `config.yaml` file with the desired values. You can also combine the options to use the `config.yaml` as defaults for the prompts.

Run `garden-app controller generate-config --help` to see all command line options, but some useful ones are:
  - `-w`/`--write`: write configs to `config.h` and `wifi_config.h` instead of stdout
  - `-f`/`--force`: overwrite files if they already exist
  - `-i`/`--interactive`: use interactive CLI to prompt for configuration values

### Interactive mode
```shell
garden-app controller generate-config -i
```

In this interactive mode, the CLI will walk you through each required configuration value and generate the config files based on the answers.

### Using config file
```shell
garden-app controller generate-config --config config.yaml
```

The following `config.yaml` file creates the necessary configuration for a 3-zone garden with moisture sensing, buttons, and light control:

```YAML
mqtt:
  broker: "localhost"
  port: 1883

controller:
  wifi:
    ssid: "My Wifi Network"
  zones:
    - pump_pin: GPIO_NUM_18
      valve_pin: GPIO_NUM_16
      button_pin: GPIO_NUM_19
      moisture_sensor_pin: GPIO_NUM_36
    - pump_pin: GPIO_NUM_18
      valve_pin: GPIO_NUM_17
      button_pin: GPIO_NUM_21
      moisture_sensor_pin: GPIO_NUM_39
    - pump_pin: GPIO_NUM_18
      valve_pin: GPIO_NUM_5
      button_pin: GPIO_NUM_22
      moisture_sensor_pin: GPIO_NUM_34
  enable_moisture_sensor: true
  enable_buttons: true
  stop_water_button: GPIO_NUM_23
  light_pin: GPIO_NUM_32
  topic_prefix: "garden"
  default_water_time: 5s
  publish_health: true
  health_interval: 1m
  moisture_interval: 5s
```

## Advanced
See the [advanced section](controller_advanced.md) for more detailed documentation.
