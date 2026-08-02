# Integration Tests

The integration tests run garden-app against MQTT, InfluxDB, and Telegraf from
`deploy/docker-compose.yml`. They are separate from the short unit-test suite.

## Mock Controller Tests

`task integration-test` (or `task it`) starts the Docker test profile, runs
`garden-app/integration_tests`, and removes the test services and InfluxDB volume
afterward. `TestIntegration` starts garden-app and the Go mock controller in the
same process. It covers garden, zone, water-schedule, controller-action, history,
and controller-startup-notification behavior.

Run it with:

```shell
task integration-test
```

## Manual Test Data

`task manual-test-up` starts the Docker test profile, seeds `manual_test.db` with
a garden, zone, controller telemetry, and water history, then runs garden-app for
interactive testing. Stop the server with Ctrl-C and remove the containers and
seeded database with `task manual-test-down`.

```shell
task manual-test-up
task manual-test-down
```

The seed test is tagged `manual` and can also be run directly with
`go test -tags=manual ./integration_tests` after the Docker test profile starts.

## Physical Controller Test

`task controller-integration-test` (or `task cit`) exercises a real ESP32. It
starts the Docker test profile, starts garden-app in-process, configures the
controller through `ControllerSetupAction`, publishes the controller pin config,
then verifies completed and cancelled water events through the history API.

### Prerequisites

1. Flash the current `garden-controller` firmware to an ESP32.
2. Connect the ESP32 to the same WiFi network as the development machine using
   WiFiManager.
3. In WiFiManager, set the topic prefix to `controller-integration`, or to the
   value of `CONTROLLER_INTEGRATION_TOPIC_PREFIX`. The test sends the MQTT broker
   host and port through the controller setup action, so those WiFiManager fields
   do not need to be configured manually.
4. The test configures four valve/pump pairs on GPIO 16, 17, 5, and 18; a light
   on GPIO 32; a fan on GPIO 23; a DHT22 on GPIO 4; and a DS18B20 on GPIO 19.
   Ensure connected loads are safe before running the test.

Run it with:

```shell
task controller-integration-test
```

The test normally opens `http://<topic-prefix>.local/paramsave` to configure the
controller. If mDNS does not resolve, set `CONTROLLER_INTEGRATION_CONTROLLER_IP`
to its LAN IP address. A port may be included for a non-default HTTP port.

`CONTROLLER_INTEGRATION_MQTT_SERVER` overrides automatic LAN IPv4 detection when
the development machine has multiple active network interfaces. Its value must be
reachable from the ESP32. `CONTROLLER_INTEGRATION_TOPIC_PREFIX` overrides the
default topic prefix and must match the value saved through WiFiManager.

The physical-controller test has a 90-second connection/reconnection timeout and
is not suitable for CI without attached hardware.
