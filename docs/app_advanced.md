# Garden App (Advanced)
This section provides more details on the features, code organization, and configurations available for the `garden-app`

The `garden-app`'s main functionality comes from the `server` command, which runs the REST API, scheduled actions, and connects to MQTT and InfluxDB.

Additional functionality comes from the following commands:
  - `controller`

## Features
- User-friendly REST API for managing Gardens, Zones and Plants
- Reliable scheduled actions using [`gocron`](https://github.com/go-co-op/gocron) for watering and lighting
- Connect to MQTT to publish messages to distributed controllers
- Connect to InfluxDB to read time-series data that comes from controllers

## Code Organization
- `pkg`: contains models and other core code necessary for the application's functionality that may be used by other packages
    - `pkg/influxdb`: interacts with InfluxDB to make queries
    - `pkg/mqtt`: connects to MQTT for publish/subscribe
    - `pkg/storage`: provides a generic `Client` interface and various implementations for storing data
- `cmd`: this is the entrypoint to the application that contains code using the popular [`spf13/cobra` CLI library](https://github.com/spf13/cobra) to configure different commands and flags. Logic in this package is minimal and it will just configure CLI options and call the relevant package's startup function
- `server`: contains code for implementing the HTTP API and running scheduled actions
- `controller`: contains code for running the mock `garden-controller` that behaves as-if it is an embedded device

Configuration is at the core of the application. Each package provides its own `Config` struct that encapsulates all the necessary options. The config file is read by the command code and then passed into the actual startup code of relevant packages. This allows for easily keeping all configurations in a `config.yaml` file that is read at application startup.

## Server
This section goes into additional details of the main functionality of the `garden-app`, which is the `server` command.

### Usage
```shell
garden-app server
```
```shell
Usage:
  garden-app server [flags]

Aliases:
  server, run

Flags:
  -h, --help       help for server
      --port int   port to run Application server on (default 80)

Global Flags:
      --config string      path to config file (default "config.yaml")
  -l, --log-level string   level of logging to display (default "info")
```

### Configuration
The [`server.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/server#Config) consists of:
  - [`server.WebConfig`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/server#WebConfig)
  - [`influxdb.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb#Config)
  - [`mqtt.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt#Config)
  - [`storage.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/storage#Config)

These encapsulated `Config` structs allow the `server` to easily create the various clients by passing those configs to the package.

Please see the [API reference](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/server#Config) for the most up-to-date information about configurations.

Example YAML config file:
```yaml
web_server:
  port: 80
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "garden-app"
  water_topic: "{{.Garden}}/command/water"
  stop_topic: "{{.Garden}}/command/stop"
  stop_all_topic: "{{.Garden}}/command/stop_all"
  light_topic: "{{.Garden}}/command/light"
influxdb:
  address: "http://localhost:8086"
  token: "my-token"
  org: "garden"
  bucket: "garden"
storage:
  type: "YAML"
  options:
    filename: "gardens.yaml"
```

### Storage Client
The `pkg/storage` package defines a `Client` interface and multiple implementations of it. The `NewStorageClient` will create a client based on the configuration. The available clients are:
- `YAMLClient`
    - Writes objects to a YAML file on the local filesystem
    - Requires a filename to use
- `ConfigMapClient`
    - Write objects to a YAML file in a Kubernetes `ConfigMap`
    - Requires a `ConfigMap` name and key to access the data

This setup will allow for easily adding more storage clients in the future.

### Kubernetes
It is possible to run this project on Kubernetes and I highly recommend this because you can easily manage all services in the cluster and quickly redeploy the `garden-app` for updates. [K3s](https://k3s.io) is a simple single-node cluster that can be run on a Raspberry Pi.

All the necessary Kubernetes manifests are available in this repository at [`deploy/k8s`](https://github.com/calvinmclean/automated-garden/tree/main/deploy/k8s). You might need to make changes to `garden_app.yaml` to change the configuration and storage file.

#### Setup `PersistentVolumeClaim`
Adding a `PersistentVolumeClaim` will allow storing InfluxDB data and Grafana configurations on the local filesystem so you don't have to worry about losing that when Pods go down.

Setting up a PVC might require additional steps depending on your system, but editing `persistent_volume.yaml` to match your system's settings will be necessary.

#### `ConfigMap` Storage Client
Since your `garden-app` container won't have access to a filesystem, you can emulate it with `ConfigMap` and the built-in Storage Client. All you will need to do is enable write access to the Pods in the system:
```
kubectl create clusterrolebinding default --clusterrole=admin --serviceaccount=default:default
```
YAML configuration to use this Storage Client:
```yaml
storage:
    type: "ConfigMap"
    options:
        name: "garden-app-config"
        key: "gardens.yaml"
```

## Controller
The `controller` command behaves as a mock `garden-controller` that makes it easier to develop, test, and debug the `garden-app server` without using a standalone microcontroller. This has extensive options using flags to control different behaviors. In most cases, the defaults will work perfectly fine.

### Usage
```shell
garden-app controller
```
```shell
Usage:
  garden-app controller [flags]

Flags:
      --health-interval duration     Interval between health data publishing (default 1m0s)
  -h, --help                         help for controller
      --moisture-interval duration   Interval between moisture data publishing (default 10s)
      --moisture-strategy string     Strategy for creating moisture data (default "random")
      --moisture-value int           The value, or starting value, to use for moisture data publishing (default 100)
  -n, --name string                  Name of the garden-controller (helps determine which MQTT topic to subscribe to) (default "garden")
  -z, --zones int                    Number of Zones for which moisture data should be emulated
      --publish-health               Whether or not to publish health data every minute (default true)
      --publish-water-event          Whether or not water events should be published for logging (default true)

Global Flags:
      --config string      path to config file (default "config.yaml")
  -l, --log-level string   level of logging to display (default "info")
```

### Configuration
The [`controller.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/controller#Config) struct consists of an [`mqtt.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt#Config) and all the command-line options (which could also be put in the config file directly).

Please see the [API reference](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/controller#Config) for the most up-to-date information about configurations.
