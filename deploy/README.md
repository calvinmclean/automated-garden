# Deploy

This directory contains all the necessary files for running all of the services locally (with Docker or `skaffold`) or in a Kubernetes cluster taking advantage of persistent volume claims for data persistence.

## Docker

This contains necessary files for running all the services locally with `docker-compose`. Each subdirectory contains configuration files that are mounted as volumes for different services. By default, the `garden-app` is commented out in `docker-compose.yml` since it is often easier to use `docker-compose` for services and run the `garden-app` with `go run` when developing locally.

## K8s

This contains all of the necessary Kubernetes manifests for deploying the Garden App and dependent services. Most of the variables are in `shared_env.yaml` so this is the main place to go for making changes. It might also be necessary to make changes to `garden-app` configuration and garden/plant storage YAML files in `base/configs/`.

This app uses [`kustomize`](https://kustomize.io) to manage different deployment environments without having to duplicate a bunch of Kubernetes manifests. These are the available environments:
    - `base`: basic setup includes `garden-app` and all dependencies
    - `overlays/dev`: adds a `garden-controller` Deployment to test communication with a mock controller
    - `overlays/prod`: adds PersistentVolume for InfluxDB and Grafana
    - `overlays/staging`: extends `dev`, changing namespace to `staging` and changing `NodePorts` for all services

```shell
kubectl apply -k deploy/dev
```

## Skaffold

This mostly uses the same files as the K8s directory through links, but has a few differences since it doesn't include the persistent volume claims.

## Kind

The `kind-cluster.yaml` can be used to start a Kind Cluster with the necessary ports mapped for these services and configured `NodePorts`.

```shell
kind create cluster --config kind-cluster.yaml
```

## Loki

In order to use Loki, install using Helm with the `loki-stack-values.yaml` values file:

```shell
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install loki grafana/loki-stack -n loki --create-namespace -f loki-stack-values.yml
```
