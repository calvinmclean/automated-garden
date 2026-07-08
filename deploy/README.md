# Deploy

This directory contains the necessary files for running all of the services locally with Docker Compose.

## Docker Compose

`docker-compose.yml` defines the services used by the application. Each subdirectory under `configs/` contains configuration files that are mounted as volumes for different services. By default, the `garden-app` is commented out in `docker-compose.yml` since it is often easier to use Docker Compose for services and run the `garden-app` with `go run` when developing locally.

### Profiles

- `test`: run just the services required for integration testing
- `run-local`: run required services + extras like Grafana and Prometheus
- `demo`: run everything, including an instance of `garden-app` and `garden-controller`
