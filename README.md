# Automated Garden

[![Go Report Card](https://goreportcard.com/badge/github.com/calvinmclean/automated-garden)](https://goreportcard.com/report/github.com/calvinmclean/automated-garden)

This project is a monorepo containing code for an ESP32-based microcontroller project and a Go backend for improving interactions with the garden.

![Garden](../assets/garden.jpeg?raw=true)


## Components

### Garden App
This Go application is contained in the [`garden-app`](./garden-app) directory and consists of a CLI and web backend for interacting with the garden controller. It implements the following features:
  - CRUD operations for Plants using RESTy API
  - Actions to water plants, skip next N waterings, and stop watering
  - Water plants automatically on a cron-based schedule


### Garden Controller
This is contained in the [`garden-controller`](./garden-controller) and is an Arduino/FreeRTOS firmware application intended to be used with an ESP32 microcontroller.

The microcontroller is able to send and receive messages using MQTT, a popular pub/sub message queue for IoT. When watering is completed, the elapsed time is published to MQTT with the intention of being picked up by Telegraf and inserted into InfluxDB. This allows me to monitor when and for how long my plants have been watered using a Grafana dashboard.

Capabilities:
  - Control pump and valves for multiple plants
  - Queue up watering events to water multiple plants one after the other
  - Water all plants every 24 hours
  - Respond to buttons to water individual plants and cancel watering
  - Listen on MQTT for watering a plant for specified amount of time, signal that the next watering event for a plant should be skipped (to avoid overwatering), cancel current watering, and clear current watering queue
  - Publish on MQTT for logging watering events in InfluxDB for visualization with Grafana


## Core Technologies
- Arduino/FreeRTOS
- Go
- MQTT
- InfluxDB
- Grafana
- Telegraf


## Parts
In addition to the software, this project consists of hardware that I either purchased or designed and 3D printed. The base of this is an [IKEA Billy bookcase](https://www.ikea.com/us/en/p/billy-bookcase-birch-veneer-40279788/). This was an affordable option that fit in my home and had adjustable shelves. At the bottom of the bookcase is a water container and submersible pump. This pumps water through an 8mm ID tube straight up a few shelves where the planters are located. A 3D printed splitter splits this 8mm tube to three 5mm ID tubes. These tubes each go to a separate solenoid valve which directs output to other plants. The pump and valves are controlled by an ESP32 and four-channel relay with 12V DC power.

3D printed parts:
  - Case for housing most of the electronics (ESP32, relays, and power connections)
  - Case and button caps for control buttons
  - Mount for the 3 water valve solenoids
  - 1-to-3 splitter for water coming from pump to the valves. This part was particularly interesting because I was able to print in the custom sizes that I needed. I also had to work with the slicer settings a bit to get a water-tight print. I configure this model with [this Customizer on Thingiverse](https://www.thingiverse.com/thing:158717)
  - Simple clips for organizing the tubing
  - Watering halo to evenly distribute water in the planters
  - Mounts for attaching the light bar to an IKEA bookshelf with variable height

Purchased parts:
  - Relays
  - ESP32
  - Circuit boards
  - Tubing (5mm and 8mm)
  - Submersible water pump
  - 12V DC power supply
  - Full spectrum grow light
  - Smart plug for light
  - Planters, seeds, soil, etc.

I don't really expect that anyone will try to recreate this project so I haven't provided a whole lot of details on the build, but if you would like any more information or access to STL files please don't hesitate to ask in an issue or email. Thanks! :)


## Images

#### Electronics Case and Buttons
![electronics](../assets/electronics.jpeg?raw=true)
<img src="../assets/electronics_open.jpeg?raw=true" alt="electronics_open" width="360" height="640"/>

### Watering System
<img src="../assets/water_tank.jpeg?raw=true" alt="water_tank" width="360" height="640"/> <img src="../assets/valves.jpeg?raw=true" alt="valves" width="360" height="640"/>
