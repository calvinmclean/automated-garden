---
name: mock-generation
description: Generate Go interface mocks using mockery. Use when creating or updating mocks.
---

# Mock Generation

Skill for generating Go mocks using mockery.

## Tool

**mockery** - Interface mock generator for Go

## Command

```bash
cd ./garden-app/pkg && mockery --all --inpackage
```

Or via Task:
```bash
task generate-mocks
```

## Configuration

- Run from `garden-app/pkg/` directory
- `--all`: Generate mocks for all interfaces
- `--inpackage`: Place mocks in same package as interfaces (not `mocks/` subdirectory)

## Generated Files

Mock files follow the pattern: `mock_InterfaceName.go`

Examples:
- `mock_Client.go` - Mock for `Client` interface
- `mock_Storage.go` - Mock for `Storage` interface

## Important Rules

**DO NOT hand-edit `mock_*.go` files!**

These files are auto-generated. Any manual changes will be lost when mocks are regenerated.

## When to Regenerate

Regenerate mocks when:
1. Interface definitions change (add/remove methods)
2. Method signatures change
3. New interfaces are added
4. Pulling changes that modified interfaces

## Common Interfaces with Mocks

Located in `garden-app/pkg/`:
- `weather.Client` - Weather API clients
- `mqtt.Client` - MQTT messaging
- `influxdb.Client` - Time-series database
- `storage.Client` - Data persistence
- `notifications.Client` - Notification services

## Usage in Tests

```go
// Create mock
mockWeather := &weather.MockClient{}

// Set expectations
mockWeather.On("GetCurrentWeather", mock.Anything).Return(&WeatherData{...}, nil)

// Use in code under test
service := NewService(mockWeather)

// Assert expectations
mockWeather.AssertExpectations(t)
```
