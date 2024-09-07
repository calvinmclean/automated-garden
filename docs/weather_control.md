# Weather Control

Weather Control allows scaling watering duration based on recent weather data. This requires a Weather Client to be configured. Please see [Advanced Server Guide](./app_advanced.md#weather-client) for more details on setting up a Weather Client.

Example JSON (part of a Zone):
```json
{
    "water_schedule": {
        "duration": "1h0m0s",
        "interval": "72h",
        "start_time": "2023-01-01T15:00:00Z",
        "weather_control": {
            "rain_control": {
                "baseline_value": 0,
                "factor": 1,
                "range": 25.4
            },
            "temperature_control": {
                "baseline_value": 30,
                "factor": 0.5,
                "range": 10
            }
        }
    }
}
```

## Rain Control

Rain Control will scale down watering duration when total rainfall between now and the previously-scheduled watering (now - interval) is between configured values. Configuration uses millimeter units.

The above example will proportionally scale watering down to zero when there is up to 1 inch (25.4mm) of rain. If there is half an inch of rain, watering will be scaled by half (30m).

## Temperature Control

Temperature control usese the average daily high temperatures for scaling control and will scale watering both up and down based on recent temperatures. Units are in degrees Celsius.

In the above example, there is a baseline value of 30C (86F) and range of 10 degrees. If the average daily high temperatures in the last 3 days (72h) are >= 40C (104F), watering will be scaled to 1.5 (1h30m). If the average daily high is <= 20C (68F), watering is scaled to 0.5 (30m). The scaling is proportional between these values.

## Viewing Weather and Scaling Data

Sometimes it might be hard to know what the total rainfall was or the recent average highs and it would also be useful to see how exactly that data is going to impact the next watering. Luckily, this information is included in the Zone API. The following example shows these relevant parts of a Zone response:

```json
{
    "water_schedule": {
        "duration": "35m0s",
        "interval": "72h0m0s",
        "start_time": "2023-01-01T15:00:00Z",
        "weather_control": {
            "rain_control": {
                "baseline_value": 0,
                "factor": 1,
                "range": 25.4
            },
            "temperature_control": {
                "baseline_value": 30,
                "factor": 0.56,
                "range": 10
            }
        }
    },
    "weather_data": {
        "rain": {
            "mm": 0,
            "scale_factor": 1
        },
        "average_temperature": {
            "celsius": 23,
            "scale_factor": 0.60800004
        }
    },
    "next_water_time": "2023-02-20T15:00:00.003449354Z",
    "next_water_duration": "21m16.800139264s"
}
```

In this example, the default watering duration of 35 minutes is reduced since recent weather has an average of 23C (73.4F) which is lower than the baseline of 30C (86F). Keep in mind that this does not necessarily reflect the actual next watering duration because that may be a few days off and the weather can always change. Regardless, it is still useful for making sure things are working as expected and make an estimate of upcoming watering.
