# Hydroponics Example

## Details
This is a simple hydroponics setup built from parts of an Aerogarden that I found at a thrift store. It didn't have the built-in light and the pump was broken. I replaced the pump with a 12V DC fountain pump and hooked that up to a single relay.

This Garden has just one Zone that circulates the water with the pump for 5 minutes in every hour. It is not necessary to add any Plants.

## Components

### Purchased
- Water pump
- Relay
- Power adapter
- Circuit board
- ESP32 dev board
- 12V -> 5V converter

### 3D Printed
- Electronics cases

### Circuit
This has a very basic setup since it just consists of the ESP32 and a single relay. A 12V power supply is plugged in and the power goes to a 5V converter and to the relay.

## Configurations
<!-- tabs:start -->
#### **`garden-controller/config.h`**
```c
WIP
```

#### **`garden-app/config.yaml`**
```yaml
web_server:
  port: 80
mqtt:
  broker: "mosquitto"
  port: 1883
  client_id: "garden-app"
influxdb:
  address: "http://influxdb:8086"
  token: "my-secret-token"
  org: "garden"
  bucket: "garden"
storage:
  type: "ConfigMap"
  options:
    name: "garden-app-config"
    key: "gardens.yaml"
```

#### **Garden JSON**
```json
{
    "name": "Aerogarden",
    "topic_prefix": "aerogarden",
    "id": "c5ma39fphd1c514hk48g",
    "max_zones": 1,
    "created_at": "2021-10-13T02:55:13.046499731Z",
    "num_zones": 0,
    "num_plants": 1,
    "plants": {
        "rel": "collection",
        "href": "/gardens/c5ma39fphd1c514hk48g/plants"
    },
    "zones": {
		"rel": "collection",
		"href": "/gardens/c5ma39fphd1c514hk48g/zones"
	},
    "links": [
        {
            "rel": "self",
            "href": "/gardens/c5ma39fphd1c514hk48g"
        },
        {
            "rel": "health",
            "href": "/gardens/c5ma39fphd1c514hk48g/health"
        },
        {
            "rel": "plants",
            "href": "/gardens/c5ma39fphd1c514hk48g/plants"
        },
        {
			"rel": "zones",
			"href": "/gardens/c5ma39fphd1c514hk48g/zones"
		},
        {
            "rel": "action",
            "href": "/gardens/c5ma39fphd1c514hk48g/action"
        }
    ]
}
```

#### **Zone JSON**
```json
{
    "name": "Circulation Pump",
    "id": "c5ma39fphd1c514hk48g",
    "position": 0,
    "created_at": "2021-10-17T22:35:35.205085596Z",
    "water_schedule": {
        "duration": "5m",
        "interval": "1h",
        "start_time": "2021-07-24T19:00:00-07:00"
    },
    "next_water_time": "2021-11-24T19:00:00.000386496Z",
    "links": [
        {
            "rel": "self",
            "href": "/gardens/c5ma39fphd1c514hk48g/plants/c5ma39fphd1c514hk48g"
        },
        {
            "rel": "garden",
            "href": "/gardens/c5ma39fphd1c514hk48g"
        },
        {
            "rel": "action",
            "href": "/gardens/c5ma39fphd1c514hk48g/plants/c5ma39fphd1c514hk48g/action"
        },
        {
            "rel": "history",
            "href": "/gardens/c5ma39fphd1c514hk48g/plants/c5ma39fphd1c514hk48g/history"
        }
    ]
}
```
<!-- tabs:end -->

## Images
![Aerogarden](_images/hydroponics_example/aerogarden.jpeg)
![Electronics](_images/hydroponics_example/electronics.jpeg)
