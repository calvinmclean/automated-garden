# Automated Garden

[![Go Report Card](https://goreportcard.com/badge/github.com/calvinmclean/automated-garden)](https://goreportcard.com/report/github.com/calvinmclean/automated-garden)
[![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/calvinmclean/automated-garden?filename=garden-app%2Fgo.mod)](https://github.com/calvinmclean/automated-garden/blob/main/garden-app/go.mod)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/calvinmclean/automated-garden/main.yml?branch=main)
[![License](https://img.shields.io/github/license/calvinmclean/automated-garden)](https://github.com/calvinmclean/automated-garden/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/calvinmclean/automated-garden/garden-app.svg)](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app)

This project is a monorepo containing code for an ESP32-based microcontroller project and a Go backend for improving interactions with multiple irrigation systems.

**If you have any question at all, please reach out and I would love to chat about the project!**

[See additional documentation here](https://calvinmclean.github.io/automated-garden).

This system is designed to be flexible for all types of gardening. Here are a few examples of different possible setups:
  - Traditional irrigation system using 24V AC valves
  - Backyard vegetable garden
  - Indoor garden with a pump and multiple valves to control watering plants, plus a grow light
  - Indoor seedling germination system with a pump and grow light (fan and heating pad control possibly coming soon)
  - Hydroponics system with one circulation or aeration pump

## How It Works

![Garden](docs/_images/FlowDiagram.png?raw=true)

### Garden Server
This Go application is contained in the [`garden-app`](./garden-app) directory and consists of a CLI and web backend for interacting with the garden controller. It implements all logic and orchestrates the separate systems.

Key features include:
  - Intuitive REST API
  - Actions to water plants and toggle a light
  - Water plants automatically on a schedule
  - Aggregate data from connected services and previous actions

### Garden Controller
This Arduino-compatible firmware is contained in the [`garden-controller`](./garden-controller) and is intended to be used with an ESP32. It implements the most basic functionality to control hardware based on messages from the Go application received over MQTT.

Key features include:
  - Control valves or devices (only limited by number of output pins)
  - Queue up water events to water multiple zones one after the other
  - Publish data and logs to InfluxDB via Telegraf + MQTT
  - Respond to buttons to water individual zones and cancel watering

## Core Technologies
- Arduino/FreeRTOS
- Go
- MQTT
- InfluxDB
- Grafana (optional for visualization of InfluxDB data)
- Telegraf
- Netatmo Weather (optional for weather-based watering)
