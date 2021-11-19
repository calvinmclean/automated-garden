# Garden Controller

This is an Arduino/FreeRTOS firmware application for controlling a home garden


## Getting Started
This is going to assume you have some familiarity with the ESP32 and Arduino IDE. 
1. Make any necessary changes to `config.h` to fit your setup
    - Read comments in the file to see the different available configuration options
    - To enable automated watering without using the Go `garden-app`, use `ENABLE_WATERING_INTERVAL` setting
2. Copy `wifi_config.h.example` to `wifi_config.h` and configure your network access information
3. Compile and upload to your ESP32 using Arduino IDE


## Design Choices

### Code Organization
This code is split up into different `.ino` and header files to improve organization and separate logic.

Arduino IDE will automatically combine all `.ino` files in alphabetical order, so some of the variables and specific configurations are split into header files so they could be included at the top.


### FreeRTOS Tasks and Queues
This project uses FreeRTOS features to take better advantage of the ESP32's two cores and allows interruptible long-running delays when watering a plant. The separate tasks also make it easier to organize logic of all the different things this controller needs to handle.

This is still an Arduino project since it is more approachable and easier to compile.


## Connectivity
Since the ESP32 has built-in WiFi capabilities, this was perfect for allowing remote control of the device. To achieve this, the controller uses MQTT. The device listens for incoming JSON messages containing commands. 

In addition to listening for incoming commands, the device will use MQTT to send information to InfluxDB. Telegraf is used as an in-between step to listen on the queue and insert data into the database.
