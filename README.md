# Automated Garden

[![Go Report Card](https://goreportcard.com/badge/github.com/calvinmclean/automated-garden)](https://goreportcard.com/report/github.com/calvinmclean/automated-garden)
[![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/calvinmclean/automated-garden?filename=garden-app%2Fgo.mod)](https://github.com/calvinmclean/automated-garden/blob/main/garden-app/go.mod)
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/calvinmclean/automated-garden/CI)
[![License](https://img.shields.io/github/license/calvinmclean/automated-garden)](https://github.com/calvinmclean/automated-garden/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/calvinmclean/automated-garden/garden-app.svg)](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app)

This project is a monorepo containing code for an ESP32-based microcontroller project and a Go backend for improving interactions with multiple gardens.

![Garden](../assets/garden.jpeg?raw=true)

This system is designed to be flexible for all types of gardening. Here are a few examples of different possible setups:
  - Indoor garden with a pump and multiple valves to control watering plants, plus a grow light
  - Indoor seedling germination system with a pump and grow light (fan and heating pad control possibly coming soon)
  - Hydroponics system with one circulation or aeration pump
  - Outdoor garden with hose spout providing water pressure and multiple valves controlling output to beds

## Components
This project consists of two main components and supporting services. A centralized Go server is used to control a distributed system of microcontrollers, each representing a Garden with one or more plants.

![Garden](../assets/FlowDiagram.png?raw=true)

### Garden App
This Go application is contained in the [`garden-app`](./garden-app) directory and consists of a CLI and web backend for interacting with the garden controller. It implements the following features:
  - CRUD operations for Gardens and Plants using REST API
  - Actions to water plants, turn on/off light, and stop watering
  - Water plants automatically on a cron-based schedule
  - Monitor health of connected `garden-controllers`

### Garden Controller
This is contained in the [`garden-controller`](./garden-controller) and is an Arduino/FreeRTOS firmware application intended to be used with an ESP32 microcontroller.

The microcontroller is able to send and receive messages using MQTT, a pub/sub message queue for IoT. When watering is completed, the elapsed time is published to MQTT with the intention of being picked up by Telegraf and inserted into InfluxDB. This allows me to monitor when and for how long my plants have been watered using a Grafana dashboard.

Capabilities:
  - Control pump and valves for one or more plants (only limited by number of output pins)
  - Queue up watering events to water multiple plants one after the other
  - Water all plants on an interval (optional for use without `garden-app`)
  - Publish moisture data to InfluxDB via Telegraf + MQTT
  - Respond to buttons to water individual plants and cancel watering
  - Listen on MQTT for watering a plant for specified amount of time, cancel current watering, and clear current watering queue
  - Publish on MQTT for logging watering events in InfluxDB for visualization with Grafana

## Core Technologies
- Arduino/FreeRTOS
- Go
- MQTT
- InfluxDB
- Grafana (optional for visualization of InfluxDB data)
- Telegraf

## Hardware
In addition to the software, this project consists of hardware for interacting with the real world. Some of this can be 3D printed while the most important parts must be purchased.

3D printed parts:
  - Case for housing most of the electronics (ESP32, relays, and power connections)
  - Case and button caps for control buttons
  - Mount for the 3 water valve solenoids
  - 1-to-3 splitter for water coming from pump to the valves
  - Simple clips for organizing the tubing
  - Watering rings to evenly distribute water in the planters
  - Mounts for grow light

Purchased parts:
  - Relays
  - ESP32
  - Circuit boards
  - Tubing (5mm and 8mm)
  - Submersible water pump
  - Water tank
  - 12V DC power supply
  - Full spectrum grow light
  - Planters, seeds, soil, etc.

I don't really expect that anyone will try to recreate this project so I haven't provided a whole lot of details on the build, but if you would like any more information or access to STL files please don't hesitate to ask in an issue or email. Thanks! :)
