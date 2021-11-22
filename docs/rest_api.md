# REST API
The `garden-app server` provides a robust REST API for interacting with the application.

The OpenAPI specification can be found [here](https://github.com/calvinmclean/automated-garden/blob/main/garden-app/api/openapi.yaml).
Use [this link](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/calvinmclean/automated-garden/main/garden-app/api/openapi.yaml) to open in Swagger UI.

To use with [Insomnia HTTP Client](https://insomnia.rest), import [`Insomnia.yaml`](https://github.com/calvinmclean/automated-garden/blob/main/garden-app/api/Insomnia.yaml).

## Overview
This REST API is centered around two main resources: `Gardens` and `Plants`.

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
  - Stop watering by sending a `StopAction` to the `/action` endpoint
  - Access to a controller's health status using the `/health` endpoint to see if the controller has recently checked-in
  - Storage of a collection of Plants

#### Examples
<!-- tabs:start -->
#### **Garden JSON**
```json
{
  "name": "test-garden",
  "id": "c22tmvucie6n6gdrpal0",
  "created_at": "2021-08-03T19:53:14.816332-07:00",
  "plants": {
    "rel": "collection",
    "href": "/gardens/c22tmvucie6n6gdrpal0/plants"
  },
  "light_schedule": {
    "duration": "15h",
    "start_time": "23:00:00-07:00"
  },
  "links": [
    {
      "rel": "self",
      "href": "/gardens/c22tmvucie6n6gdrpal0"
    },
    {
      "rel": "health",
      "href": "/gardens/c22tmvucie6n6gdrpal0/health"
    },
    {
      "rel": "plants",
      "href": "/gardens/c22tmvucie6n6gdrpal0/plants"
    },
    {
      "rel": "action",
      "href": "/gardens/c22tmvucie6n6gdrpal0/action"
    }
  ]
}
```

#### **Garden JSON (end-dated)**
```json
{
  "name": "test-garden",
  "id": "c22tmvucie6n6gdrpal0",
  "created_at": "2021-08-03T19:53:14.816332-07:00",
  "end_date": "2021-11-22T16:10:03.104226-07:00",
  "plants": {
    "rel": "collection",
    "href": "/gardens/c22tmvucie6n6gdrpal0/plants"
  },
  "links": [
    {
      "rel": "self",
      "href": "/gardens/c22tmvucie6n6gdrpal0"
    }
  ]
}
```
<!-- tabs:end -->

### Plants
A `Plant` represents a resource that can be watered. In many cases, this will represent a plant in the physical world, but it isn't necessarily always one-to-one. For example, a simple deep water culture hydroponics system might have multiple plants, but only controls watering by circulating water with a single pump. A Plant provides the following functionalities:
  - Accessed at `/gardens/{GardenID}/plants/{PlantID}`
  - Scheduled control of watering using a `watering_strategy`:
    ```json
    "watering_strategy": {
        "watering_amount": 20000,
        "interval": "72h",
        "start_time": "23:00:00-07:00"
    }
    ```
  - Control of watering based on moisture using `minimum_moisture` in the `watering_strategy`. This sets the moisture percentage the plant's soil must drop below to enable watering
  - On-demand control of watering using a `WaterAction` to the `/action` endpoint
  - Access to a Plant's watering history from InfluxDB using `/history` endpoint

It is important to note that, although this doesn't necessarily correspond to one real plant, it must correspond directly to a Plant in the `garden-controller` `PLANT` configuration array. This is controlled by the `plant_position` field in the `Plant` which is the index in the `PLANT` configuration.

#### Examples
<!-- tabs:start -->
#### **Plant JSON**
```json
{
  "name": "Tom Thumb Lettuce",
  "details": {
    "description": "Dwarf lettuce variety",
    "notes": "Planted from seed",
    "time_to_harvest": "70 days",
    "count": 6
  },
  "id": "c3ucvu06n88pt1dom670",
  "garden_id": "c22tmvucie6n6gdrpal0",
  "plant_position": 0,
  "created_at": "2021-07-24T19:44:08.014997-07:00",
  "watering_strategy": {
    "watering_amount": 300000,
    "interval": "24h",
    "start_time": "19:00:00-07:00"
  },
  "next_watering_time": "2021-11-22T18:59:59.999998-07:00",
  "links": [
    {
      "rel": "self",
      "href": "/gardens/c22tmvucie6n6gdrpal0/plants/c3ucvu06n88pt1dom670"
    },
    {
      "rel": "garden",
      "href": "/gardens/c22tmvucie6n6gdrpal0"
    },
    {
      "rel": "action",
      "href": "/gardens/c22tmvucie6n6gdrpal0/plants/c3ucvu06n88pt1dom670/action"
    },
    {
      "rel": "history",
      "href": "/gardens/c22tmvucie6n6gdrpal0/plants/c3ucvu06n88pt1dom670/history"
    }
  ]
}
```

#### **Plant JSON (end-dated)**
```json
{
  "name": "Tom Thumb Lettuce",
  "details": {
    "description": "Dwarf lettuce variety",
    "notes": "Planted from seed",
    "time_to_harvest": "70 days",
    "count": 6
  },
  "id": "c3ucvu06n88pt1dom670",
  "garden_id": "c22tmvucie6n6gdrpal0",
  "plant_position": 0,
  "created_at": "2021-07-24T19:44:08.014997-07:00",
  "end_date": "2021-11-22T16:11:53.010698-07:00",
  "watering_strategy": {
    "watering_amount": 300000,
    "interval": "24h",
    "start_time": "19:00:00-07:00"
  },
  "links": [
    {
      "rel": "self",
      "href": "/gardens/c22tmvucie6n6gdrpal0/plants/c3ucvu06n88pt1dom670"
    },
    {
      "rel": "garden",
      "href": "/gardens/c22tmvucie6n6gdrpal0"
    }
  ]
}
```
<!-- tabs:end -->
