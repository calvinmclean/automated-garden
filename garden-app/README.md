# Garden App
This is a Go application with a CLI and web backend for working with the garden controller.

## Getting Started
1. Go to `/deploy` directory and start up all services
    ```shell
    docker compose --profile demo up
    ```
2. Create a `config.yaml` file from the provided example
    ```shell
    cp config.yaml.example config.yaml
    ```
3. Create a `gardens.yaml` file from the provided example
    ```shell
    cp gardens.yaml.example gardens.yaml
    ```
4. Run the server:
    ```shell
    go run main.go server --config config.yaml
    ```

To run this in a more long-term setup, I recommend using [K3s](https://k3s.io) and deploying the manifests from `/deploy/k8s`.

Don't forget to update the `config.yaml` and `gardens.yaml` in the `ConfigMap`. Also, to use `ConfigMap` storage client you will have to enable the correct permissions in your cluster with this command:
```shell
kubectl create clusterrolebinding default --clusterrole=admin --serviceaccount=default:default
```

## Additional Usage Details

### Server
The `server` command is the main program that runs the webserver backend for managing Gardens.

#### Storage Clients
The `storage` package defines a `Client` interface and multiple implementations of it. The `NewStorageClient` will create a client based on the configuration. The available clients are:
- `YAMLClient`
    - Writes objects to a YAML file on the local filesystem
    - Requires a filename to use
- `ConfigMapClient`
    - Write objects to a YAML file in a Kubernetes `ConfigMap`
    - Requires a `ConfigMap` name and key to access the data
    - Additional setup may be required to enable the `garden-app` Pod to write to the `ConfigMap`:
        ```shell
        kubectl create clusterrolebinding default --clusterrole=admin --serviceaccount=default:default
        ```

### Controller
The `controller` command behaves as a mock `garden-controller` that makes it easier to develop, test, and debug without using a standalone microcontroller. This has extensive options using flags to control different behaviors. In most cases, the defaults will work perfectly fine.
