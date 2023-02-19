# Garden App (Advanced)
This section provides more details on the features, code organization, and configurations available for the `garden-app`

The `garden-app`'s main functionality comes from the `server` command, which runs the REST API, scheduled actions, and connects to MQTT and InfluxDB.

Additional functionality comes from the following commands:
  - `controller`
  - `controller generate-config`

## Features
- User-friendly REST API for managing Gardens, Zones and Plants
- Reliable scheduled actions using [`gocron`](https://github.com/go-co-op/gocron) for watering and lighting
- Connect to MQTT to publish messages to distributed controllers
- Connect to InfluxDB to read time-series data that comes from controllers
- Integrated with external Weather APIs to influence watering based on recent data

## Server
This section goes into additional details of the main functionality of the `garden-app`, which is the `server` command.

### Usage
```shell
garden-app server --help
```

### Configuration
The [`server.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/server#Config) consists of:
  - [`server.WebConfig`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/server#WebConfig)
  - [`influxdb.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb#Config)
  - [`mqtt.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt#Config)
  - [`storage.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/storage#Config)
  - [`weather.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/weather#Config)

These encapsulated `Config` structs allow the `server` to easily create the various clients by passing those configs to the package.

Please see the [API reference](https://github.com/calvinmclean/automated-garden/blob/main/garden-app/api/openapi.yaml) for the most up-to-date information about configurations.

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

### Weather Client
`pkg/weather` defines a `Client` interface. Currently there is only an implementation for Netatmo weather stations which can be setup with a configuration like this:

```yaml
weather:
  type: "netatmo"
  options:
    # The following names are defaults if you have one of each type in your Netatmo account
    station_name: "Weather Station"
    rain_module_name: "Smart Rain Gauge"
    outdoor_module_name: "Outdoor Module"
    authentication:
      access_token: "<access_token>"
      refresh_token: "<refresh_token>"
    client_id: "<client_id>"
    client_secret: "<client_id>"
```

The `authentication`, `client_id`, and `client_secret` configuration values can be found by following [the official Netatmo authentication guide](https://dev.netatmo.com/apidocumentation/oauth).

If you would rather use precise device IDs or the default names d not work, you can explore the [Netatmo API](https://dev.netatmo.com/apidocumentation/weather). Configuration with device IDs looks like:
```yaml
station_id: "<station_mac_address>"
rain_module_id: "<rain_module_mac_address>"
outdoor_module_id: "<outdoor_module_mac_address>"
```

### Kubernetes
It is possible to run this project on Kubernetes and I highly recommend this because you can easily manage all services in the cluster and quickly redeploy the `garden-app` for updates. [K3s](https://k3s.io) is a simple single-node cluster that can be run on a Raspberry Pi.

All the necessary Kubernetes manifests are available in this repository at [`deploy/`](https://github.com/calvinmclean/automated-garden/tree/main/deploy/). The project uses [`kustomize`](https://kustomize.io) to easily deploy to multiple K8s environments.
```
kubectl apply -k deploy/dev
```
Available kustomizations are:
    - `base`: basic setup includes `garden-app` and all dependencies
    - `overlays/dev`: adds a `garden-controller` Deployment to test communication with a mock controller
    - `overlays/prod`: adds PersistentVolume for InfluxDB and Grafana
    - `overlays/staging`: extends `dev`, changing namespace to `staging` and changing `NodePorts` for all services

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
garden-app controller --help
```

### Configuration
The [`controller.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/controller#Config) struct consists of an [`mqtt.Config`](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt#Config) and all the command-line options (which could also be put in the config file directly).

Please see the [API reference](https://pkg.go.dev/github.com/calvinmclean/automated-garden/garden-app/controller#Config) for the most up-to-date information about configurations.
