# Automated Garden
> A complete system for managing real-world Gardens with a central server application and distributed IoT devices.

## What it is
The `automated-garden` allows you to manage on or more Gardens through a user-friendly REST API. The project is flexible and configurable, allowing you to easily build and manage different types of gardens all through one application. Here are a few ideas of different gardens:

Never worry about manually watering your plants or dealing with confusing irrigation timers again!

`garden-app` refers to the central server for managing Plants, Zones and Gardens.

`garden-controller` refers to the IoT devices attached to each physical Garden.

## Features
- Water zones on a schedule
- Control lighting for indoor gardens on a schedule
- Collect soil moisture data (can be used to control watering)
- Fully-configurable Arduino project allowing for easy and flexible setup without code changes
- Monitor health of connected gardens from the central server
- Message-queue based action ensure controlling your garden will succeed even if `garden-controller` connection is unreliable
- Collected logs of all events so you can make sure watering is never missed

## Core Technologies

- **Arduino/FreeRTOS**: used to manage all of the complex operations that must be handled by the individual `garden-controllers`
- **Go**: used to create a reliable and lightweight central server application and command-line interface
- **MQTT**: message queue designed specifically for IoT applications. This allows a single server to exercise control over many distributed devices without having to know specific details about each connected device
- **InfluxDB**: a time-series database that allows for logging event and sensor data
- **Telegraf**: a thin layer that adapts MQTT messages to InfluxDB data points. This allows the `garden-controller` to publish data to InfluxDB in a reliable and simple way
- **Grafana**: allows for nice visualizations of data stored in InfluxDB
