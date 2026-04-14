---
name: docker-compose-profiles
description: Run Docker Compose with demo, test, or run-local profiles. Use when starting services.
---

# Docker Compose Profiles

Skill for running Docker Compose with different profiles for demo, testing, and local development.

## Compose File Location

`./deploy/docker-compose.yml`

## Available Profiles

### 1. Demo Profile (`--profile demo`)

Full demo stack with mock controller:
```bash
docker compose -f deploy/docker-compose.yml --profile demo up
```

Services included:
- garden-app (backend)
- mock garden-controller
- InfluxDB
- MQTT broker
- Telegraf
- Grafana

Access points:
- API: http://localhost:8080
- Grafana: http://localhost:3000 (admin/adminadmin)
- UI: http://localhost:8080/gardens

### 2. Test Profile (`--profile test`)

Integration test dependencies only:
```bash
docker compose -f deploy/docker-compose.yml --profile test up
```

Services included:
- InfluxDB
- MQTT broker

Used by `task integration-test` (alias `task it`).

### 3. Run Local Profile (`--profile run-local`)

Local development dependencies:
```bash
docker compose -f deploy/docker-compose.yml --profile run-local up
```

## Common Commands

```bash
# Start services in foreground
docker compose -f deploy/docker-compose.yml --profile demo up

# Start in background
docker compose -f deploy/docker-compose.yml --profile demo up -d

# Stop and remove
docker compose -f deploy/docker-compose.yml --profile demo down

# View logs
docker compose -f deploy/docker-compose.yml --profile demo logs -f

# Rebuild after changes
docker compose -f deploy/docker-compose.yml --profile demo up --build
```

## Integration Test Flow

When running `task it`:
1. Docker Compose starts with `--profile test`
2. Waits 15 seconds for services to be healthy
3. Runs `go test -v ./...` (integration tests)
4. Runs `docker compose down` to cleanup

## Environment Variables

The compose file uses environment variables from `.env` file or shell. Key variables:
- `INFLUXDB_ADMIN_USER`, `INFLUXDB_ADMIN_PASSWORD`
- `MQTT_BROKER`
- Grafana credentials
