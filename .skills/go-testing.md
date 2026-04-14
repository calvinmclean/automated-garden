---
name: go-testing
description: Run Go tests with project-specific flags including race detection and coverage. Use when testing Go code.
---

# Go Testing & Coverage

Skill for running Go tests with proper flags and understanding test output.

## Commands

### Run All Unit Tests
```bash
cd ./garden-app && go test -short -race -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./...
```

### Run Specific Test
```bash
cd ./garden-app && go test -short -race -run TestName ./path/to/package
```

### Run Tests for Specific Package
```bash
cd ./garden-app && go test -short -race ./pkg/...
```

## Key Flags

- `-short`: Skip long-running tests (always use for unit tests)
- `-race`: Enable race condition detection
- `-covermode=atomic`: Thread-safe coverage counting
- `-coverprofile=coverage.out`: Output coverage report
- `-coverpkg=./...`: Include all packages in coverage
- `-run TestName`: Run only tests matching pattern

## Test Types

1. **Unit Tests** (`-short`): Fast tests without external dependencies
2. **Integration Tests**: Require Docker services (InfluxDB, MQTT), run via Taskfile

## Coverage

Coverage report is written to `coverage.out`. View with:
```bash
go tool cover -html=coverage.out
```

## Conventions

- All test files use `*_test.go` naming
- Test files exclude `revive` linter (configured in `.golangci.yaml`)
- Mock files (`mock_*.go`) should not be hand-edited
- Test files should be in same package as code being tested
