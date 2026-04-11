# AGENTS.md

## Repository Structure

Monorepo with two independent projects:
- **`garden-app/`** — Go backend (REST API + web UI + scheduler). This is the primary codebase.
- **`garden-controller/`** — ESP32/Arduino firmware (PlatformIO project). Uses `pio` CLI.

All Go commands must run from `garden-app/` (the Taskfile sets `dir: ./garden-app`).

## Commands (via Taskfile)

| Task | Alias | Command |
|------|-------|---------|
| `task unit-test` | `task ut` | `go test -short -race -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./...` |
| `task integration-test` | `task it` | Starts Docker Compose (`--profile test`), waits 15s, runs integration tests, cleans up |
| `task lint` | — | `golangci-lint run` |
| `task gofumpt` | — | `gofumpt -w .` |
| `task run-server` | `task rs` | `go run main.go serve --config config.yaml` |
| `task run-dev` | `task dev` | Same as `run-server` but sets `DEV_TEMPLATE=server/templates/*` to read templates from local FS |
| `task run-controller` | `task rc` | Runs mock controller |
| `task generate-mocks` | — | `mockery --all --inpackage` in `garden-app/pkg` |

Without Taskfile, run directly from `garden-app/`:
```shell
go test -short -race -coverpkg=./... ./...
golangci-lint run
go run main.go serve --config config.yaml
```

## Testing

- **Unit tests** use `-short` flag. Run with `task ut` or `go test -short -race ./...` from `garden-app/`.
- **Integration tests** require Docker (starts InfluxDB, MQTT, etc. via `deploy/docker-compose.yml --profile test`). Run with `task it`.
- **Single test**: `go test -short -race -run TestName ./path/to/package`
- Mocks are generated with `mockery` in `pkg/` — do not hand-edit `mock_*.go` files.
- VCR (cassette) tests use `gopkg.in/dnaeon/go-vcr.v4`. Fixtures live in `server/vcr/testdata/`. Run with `task run-vcr` or set `VCR_CASSETTE` env var.

## Architecture

- Built on the [babyapi](https://github.com/calvinmclean/babyapi) framework. API resources (Garden, Zone, WaterSchedule, WaterRoutine, WeatherClient, NotificationClient) are registered as nested babyapi APIs.
- Entry point: `garden-app/main.go` → `cmd.Execute()` (Cobra CLI with `serve`, `controller`, `migrate`, `storage-migrate` subcommands).
- Storage backends: **sqlite** (recommended), **hashmap** (file-based YAML), **redis**. Selected via `storage.driver` in config.
- Worker package (`worker/`) handles scheduled watering, health checks, and notifications.
- Config uses Viper with env prefix `GARDEN_APP_` (dots become underscores, e.g. `GARDEN_APP_MQTT_BROKER`).

## Key Conventions

- Go module: `github.com/calvinmclean/automated-garden/garden-app`
- Go version: 1.24
- Linters configured in `garden-app/.golangci.yaml` (includes gofumpt, gosec, revive, etc.)
- `render.Render` and `viper.BindPFlag` error returns are intentionally excluded from errcheck — see `.golangci.yaml` excludes.
- Test files exclude `revive` linter.
- The `DEV_TEMPLATE` env var controls whether HTML templates are read from the embedded FS or the local filesystem (for live editing).

## Docker Compose Profiles

- `--profile demo`: Full demo stack with mock controller
- `--profile test`: Integration test dependencies (InfluxDB, MQTT)
- `--profile run-local`: Local development dependencies