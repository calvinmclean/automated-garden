# Deploy

This directory contains all the necessary files for running all of the services locally (with Docker or `skaffold`) or in a Kubernetes cluster taking advantage of persistent volume claims for data persistence.

## Docker

This contains necessary files for running all the services locally with `docker-compose`. Each subdirectory contains configuration files that are mounted as volumes for different services. By default, the `garden-app` is commented out in `docker-compose.yml` since it is often easier to use `docker-compose` for services and run the `garden-app` with `go run` when developing locally.

## K8s

This contains all of the necessary Kubernetes manifests for deploying the Garden App and dependent services. Most of the variables are in `shared_env.yaml` so this is the main place to go for making changes. It might also be necessary to make changes to `garden-app` configuration and garden/plant storage ConfigMap in `garden_app.yaml`.

## Skaffold

This mostly uses the same files as the K8s directory through links, but has a few differences since it doesn't include the persistent volume claims.
