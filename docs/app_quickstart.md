# Garden App
The `garden-app` is a Go server application that provides a REST API for managing Gardens, Zones and Plants.

## Quickstart/Demo

Use Docker Compose to easily run everything and try it out! This will run all services the `garden-app` depends on, plus an instance of the `garden-app` and a mock `garden-controller`.

1. Clone this repository
  ```shell
  git clone https://github.com/calvinmclean/automated-garden.git
  cd automated-garden
  ```

2. Run Docker Compose and wait a bit for everything to start up
  ```shell
  docker compose -f deploy/docker-compose.yml --profile demo up
  ```

3. Try out some `curl` commands to see what is available in the API
  ```shell
  # list all Gardens
  curl -s localhost:8080/gardens | jq

  # get a specific Garden
  curl -s localhost:8080/gardens/c9i98glvqc7km2vasfig | jq

  # get all Zones that are a part of this Garden
  curl -s localhost:8080/gardens/c9i98glvqc7km2vasfig/zones | jq
  ```
  - You may notice that these responses all contain a `links` array with the full API routes for other endpoints related to the resources. Go ahead and follow some of these links to learn more about the available API!

4. Water a Zone for 3 seconds
  ```shell
  curl -s localhost:8080/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0/action \
    -d '{"water": {"duration": 3000}}'
  ```

5. Now access Grafana dashboards at http://localhost:3000 and login as `admin/adminadmin`
  - The "Garden App" dashboard contains application metrics for resource usage, HTTP stats, and others
  - The "Garden Dashboard" dashboard contains more interesting data that comes from the `garden-controller` to show uptime and a watering history. You should see the recent 3 second watering event here

And that's it! I encourage you to check out the additional documentation for more detailed API usage and to learn about all of the things that are possible.
