# Garden App
The `garden-app` is a Go server application that provides a REST API for managing Gardens and Plants.

## Getting Started
1. Clone the repository
    ```shell
    git clone https://github.com/calvinmclean/automated-garden.git
    ```
1. Go to `automated-garden/deploy/docker` directory and start up all services
    ```shell
    cd automated-garden/deploy/docker
    docker compose up -d
    ```
1. Navigate to `automated-garden/garden-app`
    ```shell
    cd ../../garden-app
    ```
1. Create a `config.yaml` file from the provided example
    ```shell
    cp config.yaml.example config.yaml
    ```
1. Optional: Create a `gardens.yaml` file from the provided example (skip this step for a fresh start)
    ```shell
    cp gardens.yaml.example gardens.yaml
    ```
1. Run the server:
    ```shell
    go run main.go server
    ```
    or
    ```shell
    go install .
    garden-app server
    ```

## Use a Mock Controller
The `garden-app` includes a `controller` subcommand that makes it easier to test without an acual `garden-controller`. This can also be useful to try things out without having to setup the embedded hardware!

This guide assumes you already completed the preceding section to run the `garden-app server` and used the included `garden.yaml.example`.

1. If not done already, install the `garden-app` command:
    ```shell
    go install .
    ```
1. Run the mock controller with the topic prefix `test-garden`
    ```shell
    garden-app controller --topic test-garden
    ```
1. Water the Plant to see output in the mock controller
    ```shell
    curl --request POST \
        --url http://localhost/gardens/c22tmvucie6n6gdrpal0/plants/c3ucvu06n88pt1dom670/action \
        --header 'Content-Type: application/json' \
        --data '{
            "water": {
                "duration": 2000
            }
        }'
    ```

## Advanced
See the [advanced section](app_advanced.md) for more detailed documentation and instructions for running in Kubernetes.
