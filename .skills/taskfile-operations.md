---
name: taskfile-operations
description: Execute tasks via Taskfile aliases like ut, it, lint, dev. Use when running project tasks.
---

# Taskfile Operations

Skill for executing tasks defined in the project's Taskfile.yml.

## Location

All tasks run from `garden-app/` directory (configured via `dir: ./garden-app` in Taskfile.yml).

## Common Tasks

| Task | Alias | Command |
|------|-------|---------|
| `task unit-test` | `task ut` | Run unit tests with coverage |
| `task integration-test` | `task it` | Run integration tests with Docker |
| `task lint` | — | Run golangci-lint |
| `task gofumpt` | — | Format code with gofumpt |
| `task run-server` | `task rs` | Run the server |
| `task run-dev` | `task dev` | Run server with live template reloading |
| `task run-controller` | `task rc` | Run mock controller |
| `task generate-mocks` | — | Generate mocks with mockery |

## Usage Examples

```bash
# From repo root
task ut
task it
task lint
task dev
```

## Integration Tests

The `task it` command:
1. Starts Docker Compose with `--profile test`
2. Waits 15 seconds for services to be ready
3. Runs integration tests
4. Cleans up Docker containers

Required services: InfluxDB, MQTT broker

## Development Mode

`task dev` sets `DEV_TEMPLATE=server/templates/*` to read HTML templates from local filesystem instead of embedded FS, enabling live template editing.

## Without Taskfile

If Task is not installed, run commands directly from `garden-app/`:
```bash
cd ./garden-app
go test -short -race -coverpkg=./... ./...
golangci-lint run
go run main.go serve --config config.yaml
```
