---
name: vcr-testing
description: Record and replay HTTP interactions using go-vcr. Use when testing external APIs.
---

# VCR Testing

Skill for using go-vcr to record and replay HTTP interactions in tests.

## Tool

**gopkg.in/dnaeon/go-vcr.v4** - Record and replay HTTP interactions

## How It Works

1. **Record Mode**: HTTP requests are made to real services and responses are saved to "cassettes"
2. **Replay Mode**: HTTP requests are intercepted and responses are served from cassettes
3. Tests can run offline using recorded cassettes

## Cassette Location

Fixtures live in: `garden-app/server/vcr/testdata/`

Cassettes are YAML files containing recorded HTTP interactions.

## Running VCR Tests

### Via Task
```bash
task run-vcr
```

### Via Environment Variable
```bash
cd ./garden-app && VCR_CASSETTE=cassette_name go test -v ./server/...
```

## Test Patterns

Tests using VCR typically:
1. Load cassette from `testdata/`
2. Configure HTTP client to use VCR recorder
3. Make requests (replayed from cassette)
4. Assert on responses

## Recording New Cassettes

To record new HTTP interactions:
1. Set recorder to record mode (or use `RECORD=1` env var if implemented)
2. Run test against real API
3. Cassette is saved to `testdata/`
4. Commit cassette to git for reproducible tests

## Best Practices

1. **Commit cassettes**: Treat cassettes as test fixtures, commit them to version control
2. **Sanitize sensitive data**: Remove API keys, tokens, PII from cassettes before committing
3. **Name cassettes descriptively**: Use test name or feature being tested
4. **Re-record periodically**: APIs change; re-record cassettes to catch breaking changes
5. **Keep cassettes small**: Record only the specific requests needed for the test

## Example Test Structure

```go
func TestWeatherClient(t *testing.T) {
    // Load cassette
    recorder, err := vcr.New("testdata/weather_api")
    require.NoError(t, err)
    defer recorder.Stop()
    
    // Create client with VCR transport
    client := &http.Client{
        Transport: recorder,
    }
    
    // Test code using client...
}
```
