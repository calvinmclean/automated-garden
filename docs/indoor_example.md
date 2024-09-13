# Indoor Herb Garden Example

## Details
This setup consists of three small/medium pots and a grow light on a bookshelf. A 12V DC fountain pump moves water from a container at the base to a 3D printed splitter. The splitter delivers water to three 12V DC valves that route to a 3D printed watering ring on the pot.

This small garden has worked really well for growing herbs or even cherry tomatoes.

## Components

### Purchased
- Water pump
- Water valves/solenoids
- 8mm and 6mm Tubing
- Relays
- Grow lights
- Power adapter
- Circuit board
- ESP32 dev board
- 12V -> 5V converter

### 3D Printed
- Electronics case
- Hose splitter
- Hose routing clips
- Grow light mounts

### Circuit
The circuitry for this setup is pretty straightforward. There is a 12V input that provides power to the 5V converter and the 4-channel relay. The 5V output is used to power the ESP32, the relay control, and the buttons. The 12V power is connected to four pairs of 2-pin JST connectors. Each pair has one connection to a 12V device (pump/valve) and one connection to the relay power. The power supply's ground goes to the ground wire of the 12V device and the power wire goes to the relay plug. The relay is used to bridge the 12V power to the connected device.

Then there are two 6-pin JST connectors that provide power (ground/5V) and 4 data wires for the buttons and for the relay control. These are simply wired up to the ESP32 pins as shown in the configurations.

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
    "name": "Garden",
    "topic_prefix": "garden",
    "id": "c5ma3cvphd1c514hk490",
    "max_zones": 3,
    "created_at": "2021-10-13T02:55:13.025436541Z",
    "light_schedule": {
        "duration": "13h",
        "start_time": "23:00:00-07:00"
    },
    "next_light_action": {
        "time": "2021-11-24T18:59:59.999998625Z",
        "state": "OFF"
    },
    "num_plants": 3,
    "num_zones": 3,
    "plants": {
        "rel": "collection",
        "href": "/gardens/c5ma3cvphd1c514hk490/plants"
    },
    "zones": {
		"rel": "collection",
		"href": "/gardens/c5ma3cvphd1c514hk490/zones"
	},
    "links": [
        {
            "rel": "self",
            "href": "/gardens/c5ma3cvphd1c514hk490"
        },
        {
            "rel": "health",
            "href": "/gardens/c5ma3cvphd1c514hk490/health"
        },
        {
            "rel": "plants",
            "href": "/gardens/c5ma3cvphd1c514hk490/plants"
        },
        {
			"rel": "zones",
			"href": "/gardens/c5ma3cvphd1c514hk490/zones"
		},
        {
            "rel": "action",
            "href": "/gardens/c5ma3cvphd1c514hk490/action"
        }
    ]
}
```

#### **Example Zone JSON**
```json
{
    "name": "Planter 1",
    "id": "c5maco7phd1ftefa6ol0",
    "position": 1,
    "created_at": "2021-10-17T22:40:32.474585191Z",
    "water_schedule": {
        "duration": "20s",
        "interval": "72h",
        "start_time": "2021-07-24T19:00:00-07:00"
    },
    "next_water_time": "2021-11-26T04:59:59.99999675Z",
    "links": [
        {
            "rel": "self",
            "href": "/gardens/c5ma3cvphd1c514hk490/plants/c5maco7phd1ftefa6ol0"
        },
        {
            "rel": "garden",
            "href": "/gardens/c5ma3cvphd1c514hk490"
        },
        {
            "rel": "action",
            "href": "/gardens/c5ma3cvphd1c514hk490/plants/c5maco7phd1ftefa6ol0/action"
        },
        {
            "rel": "history",
            "href": "/gardens/c5ma3cvphd1c514hk490/plants/c5maco7phd1ftefa6ol0/history"
        }
    ]
}
```

#### **Example Plant JSON**
```json
{
    "name": "Chives",
    "details": {
        "notes": "Planted by seed",
        "count": 4
    },
    "id": "c9i9jl5vqc7l7e3ikkgg",
    "zone_id": "c5maco7phd1ftefa6ol0",
    "created_at": "2021-10-17T22:40:32.474585191Z",
    "links": [
        {
            "rel": "self",
            "href": "/gardens/c5ma3cvphd1c514hk490/plants/c5maco7phd1ftefa6ol0"
        },
        {
            "rel": "garden",
            "href": "/gardens/c5ma3cvphd1c514hk490"
        },
        {
            "rel": "zone",
            "href": "/gardens/c5ma3cvphd1c514hk490/zones/c5maco7phd1ftefa6ol0"
        },
    ]
}
```
<!-- tabs:end -->

## Images
![Plants](_images/indoor_example/plants.jpeg)
![Electronics](_images/indoor_example/electronics.jpeg)
![Water Splitter](_images/indoor_example/water_splitter.jpeg)
