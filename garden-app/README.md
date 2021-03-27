# Garden App

This is a Go application with a CLI and web backend for working with the garden controller.


## Getting Started
WIP


### Watering Strategies
Watering is configured in the `WateringStrategy` property of a `Plant`. This consists of an interval, watering amount, and optionally a minimum moisture. Whenever the interval time elapses, the plant will be watered for the configured time. If the minimum moisture is configured, the InfluxDB moisture data is checked and the plant will only be watered if the moisture is below the threshold. The moisture value will actually be the average over the last 15 minutes to avoid outlier data causing unnecessary watering.

The moisture-based watering feature was designed to still use the interval rather than continuously reading from the stream of data because this offloads the complexity of data streaming to the Telegraf/InfluxDB setup. Additionally, this reduces the complexity of the `WateringStrategy` configuration by only adding a single optional field. It will also prevent the watering from being triggered by outlier data.

YAML example:
```yaml
watering_strategy:
    watering_amount: 10000
    interval: 24h
    minimum_moisture: 50
```


## Design Choices

### Code Organization
The base of this project is a [Cobra](https://github.com/spf13/cobra) CLI application. It is used to start up a [`go-chi`](https://github.com/go-chi/chi) web application.

Currently, this consists of 3 base packages:
- `api`: contains models and other core code necessary for the application's functionality
- `cmd`: contains Cobra commands for working with the other packages
- `server`: contains most of the `go-chi` parts of the application for implementing the HTTP API

This approach allows me to focus on the application's core functionality separate from how the user will interact with it through the CLI or HTTP API.
