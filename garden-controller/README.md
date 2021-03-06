# Garden Controller

This is an Arduino/FreeRTOS firmware application for controlling a home garden

**Currently being used in "production" and reliably watering my plants.**


## Getting Started
This is going to assume you have some familiarity with the ESP32 and Arduino IDE. 
1. Start by getting the supporting services running from the [`docker`](../docker) directory
2. Optionally run the Go application from [`garden-app`](../garden-app) directory
3. Make any necessary changes to `config.h` to fit your setup
4. Copy `wifi_config.h.example` to `wifi_config.h` and configure your network access information
5. Compile and upload to your ESP32 using Arduino IDE


## Design Choices

### Code Organization
Since there's starting to be a lot of code in this project, and I made it pretty configurable, I wanted to improve the organization of everything by splitting it into multiple files. The code isn't really setup to be split into Arduino libraries, so I just split it into multiple `.ino` files. Arduino IDE will automatically combine all `.ino` files in alphabetical order, so I had to split some of the variables into header files so they could be included at the top.


### Arduino vs FreeRTOS
This started out as an Arduino sketch, but I eventually wanted to make use of the dual-core capabilities of the ESP32 so I started learning about FreeRTOS. In the transition to making this a full FreeRTOS C++ project rather than an Arduino sketch, I chose to give up some of the niceties of Arduino such as `pinMode` and `digitalWrite`. However, I will probably leave this as an Arduino project since it is more approachable.


### FreeRTOS Tasks and Queues
In order to take advantage of FreeRTOS features, I created Tasks for most things that need to be done by the application and communicate using Queues. For example, to water a plant, the program will add a `WateringEvent` to a Queue that is being processed by the `WaterPlantTask`. This allows me to leave scheduling up to the system rather than the previous setup I had using `millis()` to manually schedule different "tasks." 


### MQTT Message Format
Since the device is listening for commands on an MQTT topic, I had to choose how i would format those messages. The most obvious options were JSON, Protobuf, or some simple customized setup. I chose JSON since it was the easiest to get started with and is standard, unlike making up my own format. Now that I am also writing a Go program, I might try to use Protobuf since it can easily be cross-platform and will be more efficient on the ESP32 instead of JSON.


## Connectivity
Since the ESP32 has built-in WiFi capabilities, this was perfect for allowing remote control of the device. To achieve this, I am using MQTT. The device listens for incoming JSON messages containing commands. 

In addition to listening for incoming commands, the device will use MQTT to send information to InfluxDB. Telegraf is used as an in-between step to listen on the queue and insert data into the database. I originally thought of this to publish sensor data, but I haven't added any sensors to the system yet. Currently I am using it to log watering events so I can easily track how much water my plants are getting over time.
