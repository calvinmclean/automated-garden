# REST API
The `garden-app server` provides a robust REST API for interacting with the application.

The OpenAPI specification can be found [here](https://github.com/calvinmclean/automated-garden/blob/main/garden-app/api/openapi.yaml).
Use [this link](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/calvinmclean/automated-garden/main/garden-app/api/openapi.yaml) to open in Swagger UI.

To use with [Insomnia HTTP Client](https://insomnia.rest), import [`Insomnia.yaml`](https://github.com/calvinmclean/automated-garden/blob/main/garden-app/api/Insomnia.yaml).

## Overview
This REST API is centered around three main resources: `Gardens`, `Zones` and `Plants`.

### Gardens
A `Garden` represents a garden in the physical world and should correspond one-to-one with an IoT device running `garden-controller` firmware. A Garden provides the following functionalities:
  - Accessed at `/gardens/{GardenID}`
  - Scheduled control of a grow light using a `light_schedule`:
    ```json
    "light_schedule": {
        "duration": "15h",
        "start_time": "23:00:00-07:00"
    }
    ```
  - On-demand control of a light using a `LightAction` to the `/action` endpoint
    - Using the `for_duration` field of the action with `state=OFF` allows turning a light off or delaying the light from turning on for a specific duration. This is useful if an indoor garden's light turning on would be disruptive
  - Stop watering by sending a `StopAction` to the `/action` endpoint
  - Access to a controller's health status using the `/health` endpoint to see if the controller has recently checked-in
  - Storage of a collection of Plants and Zones

#### Examples
<!-- tabs:start -->
#### **Garden JSON**
```json
{
	"name": "Test Garden",
	"topic_prefix": "test-garden",
	"id": "c9i98glvqc7km2vasfig",
	"max_zones": 1,
	"created_at": "2022-04-23T17:05:22.245112-07:00",
	"light_schedule": {
		"duration": "14h",
		"start_time": "08:00:00-07:00"
	},
	"next_light_action": {
		"time": "2022-04-23T22:00:00-07:00",
		"state": "OFF"
	},
	"num_plants": 0,
	"num_zones": 1,
	"plants": {
		"rel": "collection",
		"href": "/gardens/c9i98glvqc7km2vasfig/plants"
	},
	"zones": {
		"rel": "collection",
		"href": "/gardens/c9i98glvqc7km2vasfig/zones"
	},
	"links": [
		{
			"rel": "self",
			"href": "/gardens/c9i98glvqc7km2vasfig"
		},
		{
			"rel": "health",
			"href": "/gardens/c9i98glvqc7km2vasfig/health"
		},
		{
			"rel": "plants",
			"href": "/gardens/c9i98glvqc7km2vasfig/plants"
		},
		{
			"rel": "zones",
			"href": "/gardens/c9i98glvqc7km2vasfig/zones"
		},
		{
			"rel": "action",
			"href": "/gardens/c9i98glvqc7km2vasfig/action"
		}
	]
}
```
<!-- tabs:end -->

### Zones
A `Zone` represents a resource that can be watered. It may contain zero or more Plants. In the real world, this might be a single raised bed, a section of the yard, or a lawn. A Zone provides the following functionalities:
  - Accessed at `/gardens/{GardenID}/zones/{ZoneID}`
  - Scheduled control of watering using a `water_schedule`:
    ```json
    "water_schedule": {
        "duration": "20s",
        "interval": "72h",
        "start_time": "2021-07-24T19:00:00-07:00"
    }
    ```
  - Control of watering based on moisture using `minimum_moisture` in the `water_schedule`. This sets the moisture percentage the zone's soil must drop below to enable watering
  - On-demand control of watering using a `WaterAction` to the `/action` endpoint
  - Access to a Zone's watering history from InfluxDB using `/history` endpoint

It is important to note that it must correspond directly to a Zone in the `garden-controller` `ZONES` configuration array. This is controlled by the `position` field in the `Zone` which is the index in the `ZONES` configuration.


#### Examples
<!-- tabs:start -->
#### **Zone JSON**
```json
{
	"name": "Zone 1",
	"id": "c9i99otvqc7kmt8hjio0",
	"position": 0,
	"created_at": "2022-04-23T17:08:03.441727-07:00",
	"water_schedule": {
		"duration": "30s",
		"interval": "24h",
		"start_time": "2022-04-23T08:00:00-07:00"
	},
	"next_water_time": "2022-04-24T08:00:00-07:00",
	"links": [
		{
			"rel": "self",
			"href": "/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0"
		},
		{
			"rel": "garden",
			"href": "/gardens/c9i98glvqc7km2vasfig"
		},
		{
			"rel": "action",
			"href": "/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0/action"
		},
		{
			"rel": "history",
			"href": "/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0/history"
		}
	]
}
```
<!-- tabs:end -->

### Plants
A `Plant` represents an actual Plant in the real world. It doesn't have any special characteristics to interact with, like a Zone or Garden. This is just used to track Plants that exist in certain Gardens and Zones and is completely optional. It allows to easily keep track of planting details such as number of plants, time to harvest, and planting date.

#### Examples
<!-- tabs:start -->
#### **Plant JSON**
```json
{
	"name": "Plant",
	"details": {
		"description": "My favorite plant",
		"notes": "Planted from seed",
		"time_to_harvest": "70 days",
		"count": 1
	},
	"id": "c9i9jl5vqc7l7e3ikkgg",
	"zone_id": "c9i99otvqc7kmt8hjio0",
	"created_at": "2022-04-23T17:29:08.526638-07:00",
	"links": [
		{
			"rel": "self",
			"href": "/gardens/c9i98glvqc7km2vasfig/plants/c9i9jl5vqc7l7e3ikkgg"
		},
		{
			"rel": "garden",
			"href": "/gardens/c9i98glvqc7km2vasfig"
		},
		{
			"rel": "zone",
			"href": "/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0"
		}
	]
}
```
<!-- tabs:end -->
